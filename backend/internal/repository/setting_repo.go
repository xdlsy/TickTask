package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"ticktask/internal/model"
	"ticktask/pkg/logger"
	"ticktask/pkg/vault"

	"gorm.io/gorm"
)

type SettingRepository interface {
	Get(key string) (*model.Setting, error)
	Set(key, value string) error
	GetPomodoroSettings() (*model.PomodoroSettings, error)
	UpdatePomodoroSettings(settings *model.PomodoroSettings) error
	GetAISettings() (*model.AISettings, error)
	UpdateAISettings(settings *model.AISettings) error
	MigrateLegacyAPIKey() error
}

type settingRepository struct {
	db    *gorm.DB
	vault vault.Vault
}

// NewSettingRepository wires the repo to a DB handle and an optional vault.
// vault may be nil in tests that don't touch api_key encryption; production
// callers should always pass a non-nil vault.
func NewSettingRepository(db *gorm.DB, v vault.Vault) SettingRepository {
	return &settingRepository{db: db, vault: v}
}

func (r *settingRepository) Get(key string) (*model.Setting, error) {
	var setting model.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingRepository) Set(key, value string) error {
	return r.db.Save(&model.Setting{Key: key, Value: value}).Error
}

func (r *settingRepository) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	settings := model.DefaultPomodoroSettings()

	setting, err := r.Get("pomodoro.settings")
	if err != nil {
		return settings, nil
	}

	if err := json.Unmarshal([]byte(setting.Value), settings); err != nil {
		return model.DefaultPomodoroSettings(), nil
	}
	return settings, nil
}

func (r *settingRepository) UpdatePomodoroSettings(settings *model.PomodoroSettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return r.Set("pomodoro.settings", string(data))
}

// aiSettingsDBShape is the on-disk JSON form. Plaintext api_key is NEVER
// written by new code; api_key_encrypted holds the AES-GCM ciphertext and
// nonce, both base64-encoded so they round-trip cleanly through JSON.
type aiSettingsDBShape struct {
	Provider        string             `json:"provider"`
	BaseURL         string             `json:"base_url"`
	Model           string             `json:"model"`
	APIKeyEncrypted *encryptedBlobJSON `json:"api_key_encrypted,omitempty"`
}

type encryptedBlobJSON struct {
	CT    string `json:"ct"`
	Nonce string `json:"nonce"`
}

func (r *settingRepository) GetAISettings() (*model.AISettings, error) {
	setting, err := r.Get("ai.settings")
	if err != nil {
		return model.DefaultAISettings(), nil
	}

	// Probe both shapes via map to detect legacy plaintext form.
	var raw map[string]any
	if err := json.Unmarshal([]byte(setting.Value), &raw); err != nil {
		return model.DefaultAISettings(), nil
	}

	// Defaults mirror model.DefaultAISettings() so a missing/corrupt row gives
	// the same result as a fresh install.
	defaults := model.DefaultAISettings()
	out := &model.AISettings{
		Provider: stringFromMap(raw, "provider", defaults.Provider),
		BaseURL:  stringFromMap(raw, "base_url", defaults.BaseURL),
		Model:    stringFromMap(raw, "model", defaults.Model),
		// APIKey intentionally not filled from raw — legacy plaintext is
		// ignored on read; encrypted form is decrypted below.
	}

	if encStr, ok := raw["api_key_encrypted"].(map[string]any); ok && r.vault != nil {
		ctStr, _ := encStr["ct"].(string)
		nonceStr, _ := encStr["nonce"].(string)
		ct, errCT := base64.StdEncoding.DecodeString(ctStr)
		nonce, errNonce := base64.StdEncoding.DecodeString(nonceStr)
		if errCT == nil && errNonce == nil {
			plain, err := r.vault.Decrypt(ct, nonce)
			if err == nil {
				out.APIKey = plain
			} else {
				// Decrypt failure typically means the row was encrypted with a
				// different .keyvault (e.g. multi-machine sync without copying
				// the vault). Gracefully degrade to "not configured" but warn
				// so the user can re-enter the key in Settings instead of
				// debugging a silent empty key.
				logger.Logger.Warn("ai api_key decrypt failed — please re-enter API Key in Settings", "err", err)
			}
		}
	}
	return out, nil
}

func (r *settingRepository) UpdateAISettings(settings *model.AISettings) error {
	shape := aiSettingsDBShape{
		Provider: settings.Provider,
		BaseURL:  settings.BaseURL,
		Model:    settings.Model,
	}

	if settings.APIKey != "" && r.vault != nil {
		ct, nonce, err := r.vault.Encrypt(settings.APIKey)
		if err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
		shape.APIKeyEncrypted = &encryptedBlobJSON{
			CT:    base64.StdEncoding.EncodeToString(ct),
			Nonce: base64.StdEncoding.EncodeToString(nonce),
		}
	}

	// Read existing api_key_encrypted so an UpdateAISettings call that omits
	// api_key (empty string) does NOT wipe the stored key. This is the
	// backend half of the mask-roundtrip fix.
	if settings.APIKey == "" {
		if existing, err := r.Get("ai.settings"); err == nil {
			var prev aiSettingsDBShape
			if json.Unmarshal([]byte(existing.Value), &prev) == nil {
				shape.APIKeyEncrypted = prev.APIKeyEncrypted
			}
		}
	}

	data, err := json.Marshal(shape)
	if err != nil {
		return err
	}
	return r.Set("ai.settings", string(data))
}

// MigrateLegacyAPIKey finds any ai.settings row still carrying a plaintext
// api_key field, encrypts it, removes the plaintext field, and writes the
// new shape back. Idempotent — if no legacy api_key is present, it's a no-op.
func (r *settingRepository) MigrateLegacyAPIKey() error {
	if r.vault == nil {
		return fmt.Errorf("migrate requires a vault")
	}
	setting, err := r.Get("ai.settings")
	if err != nil {
		return nil // no row yet → nothing to migrate
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(setting.Value), &raw); err != nil {
		return fmt.Errorf("migrate: parse ai.settings: %w", err)
	}

	legacyKey, ok := raw["api_key"].(string)
	if !ok || strings.TrimSpace(legacyKey) == "" {
		return nil // already migrated or never had a key
	}

	ct, nonce, err := r.vault.Encrypt(legacyKey)
	if err != nil {
		return fmt.Errorf("migrate: encrypt: %w", err)
	}

	delete(raw, "api_key")
	raw["api_key_encrypted"] = map[string]string{
		"ct":    base64.StdEncoding.EncodeToString(ct),
		"nonce": base64.StdEncoding.EncodeToString(nonce),
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("migrate: marshal: %w", err)
	}
	return r.db.Model(&model.Setting{}).
		Where("key = ?", "ai.settings").
		Update("value", string(data)).Error
}

func stringFromMap(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return def
}
