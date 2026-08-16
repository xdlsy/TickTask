package repository

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"ticktask/internal/model"
	"ticktask/pkg/vault"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTestVault(t *testing.T) vault.Vault {
	t.Helper()
	v, err := vault.New(t.TempDir() + "/.keyvault")
	if err != nil {
		t.Fatalf("vault new: %v", err)
	}
	if err := v.Init(); err != nil {
		t.Fatalf("vault init: %v", err)
	}
	return v
}

// readRawSettingValue bypasses the repo's decryption layer to inspect the raw
// DB JSON. Used to assert "no plaintext api_key in DB".
func readRawSettingValue(t *testing.T, db *gorm.DB, key string) string {
	t.Helper()
	var s model.Setting
	if err := db.Where("key = ?", key).First(&s).Error; err != nil {
		t.Fatalf("read raw %s: %v", key, err)
	}
	return s.Value
}

func TestSettingRepo_UpdateAISettings_DBHasNoPlaintextAPIKey(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewSettingRepository(db, newTestVault(t))

	if err := repo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-secret-1234567890abcdef",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	raw := readRawSettingValue(t, db, "ai.settings")
	if strings.Contains(raw, "sk-secret-1234567890abcdef") {
		t.Errorf("plaintext api_key leaked into DB JSON: %s", raw)
	}
	if !strings.Contains(raw, "api_key_encrypted") {
		t.Errorf("expected api_key_encrypted in DB JSON: %s", raw)
	}
}

func TestSettingRepo_GetAISettings_RoundTrip(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewSettingRepository(db, newTestVault(t))

	original := &model.AISettings{
		Provider: "openai",
		APIKey:   "sk-round-trip",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	}
	if err := repo.UpdateAISettings(original); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetAISettings()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.APIKey != "sk-round-trip" {
		t.Errorf("api_key round trip: got %q, want %q", got.APIKey, "sk-round-trip")
	}
	if got.Provider != "openai" {
		t.Errorf("provider lost: %s", got.Provider)
	}
}

func TestSettingRepo_GetAISettings_DecryptFailureReturnsDefault(t *testing.T) {
	db := newRepoTestDB(t)
	repoA := NewSettingRepository(db, newTestVault(t))
	_ = repoA.UpdateAISettings(&model.AISettings{
		Provider: "openai", APIKey: "sk-from-machine-A",
		BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini",
	})

	// Machine B opens the same DB but has a different .keyvault → decrypt fails.
	repoB := NewSettingRepository(db, newTestVault(t))
	got, err := repoB.GetAISettings()
	if err != nil {
		t.Fatalf("expected nil err on decrypt failure (graceful degrade), got: %v", err)
	}
	if got.APIKey != "" {
		t.Errorf("expected empty api_key on decrypt failure, got %q", got.APIKey)
	}
}

func TestSettingRepo_GetAISettings_EmptyDBReturnsDefault(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewSettingRepository(db, newTestVault(t))

	got, err := repo.GetAISettings()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Provider != "openai" || got.APIKey != "" {
		t.Errorf("expected default AISettings, got %+v", got)
	}
}

func TestSettingRepo_MigrateLegacyAPIKey_EncryptsAndCleartextRemoved(t *testing.T) {
	db := newRepoTestDB(t)
	// Seed legacy plaintext form directly into DB.
	legacy := `{"provider":"openai","api_key":"sk-legacy-1234567890abcdef","base_url":"https://api.openai.com/v1","model":"gpt-4o-mini"}`
	db.Create(&model.Setting{Key: "ai.settings", Value: legacy})

	repo := NewSettingRepository(db, newTestVault(t))
	if err := repo.MigrateLegacyAPIKey(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	raw := readRawSettingValue(t, db, "ai.settings")
	if strings.Contains(raw, "sk-legacy-1234567890abcdef") {
		t.Errorf("plaintext api_key still in DB after migration: %s", raw)
	}
	if !strings.Contains(raw, "api_key_encrypted") {
		t.Errorf("migration did not write api_key_encrypted: %s", raw)
	}

	// Migration removed the api_key field entirely.
	var probe map[string]any
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := probe["api_key"]; ok {
		t.Errorf("api_key field still present after migration: %s", raw)
	}

	// After migration, repo.GetAISettings should decrypt to the original key.
	got, _ := repo.GetAISettings()
	if got.APIKey != "sk-legacy-1234567890abcdef" {
		t.Errorf("post-migration decrypt: got %q, want original", got.APIKey)
	}
}

func TestSettingRepo_MigrateLegacyAPIKey_Idempotent(t *testing.T) {
	db := newRepoTestDB(t)
	legacy := `{"provider":"openai","api_key":"sk-idem","base_url":"","model":""}`
	db.Create(&model.Setting{Key: "ai.settings", Value: legacy})

	repo := NewSettingRepository(db, newTestVault(t))
	_ = repo.MigrateLegacyAPIKey()
	rawAfter1 := readRawSettingValue(t, db, "ai.settings")

	_ = repo.MigrateLegacyAPIKey()
	rawAfter2 := readRawSettingValue(t, db, "ai.settings")

	if rawAfter1 != rawAfter2 {
		t.Errorf("idempotent migration changed DB JSON:\n1st: %s\n2nd: %s", rawAfter1, rawAfter2)
	}
}

func TestSettingRepo_MigrateLegacyAPIKey_NoLegacyNoOp(t *testing.T) {
	db := newRepoTestDB(t)
	// Already-migrated form.
	alreadyMigrated := `{"provider":"openai","base_url":"https://api.openai.com/v1","model":"gpt-4o-mini","api_key_encrypted":{"ct":"AAAA","nonce":"BBBB"}}`
	db.Create(&model.Setting{Key: "ai.settings", Value: alreadyMigrated})

	repo := NewSettingRepository(db, newTestVault(t))
	if err := repo.MigrateLegacyAPIKey(); err != nil {
		t.Fatalf("migrate on no-op input: %v", err)
	}

	raw := readRawSettingValue(t, db, "ai.settings")
	if raw != alreadyMigrated {
		t.Errorf("no-op migration modified DB JSON:\nbefore: %s\nafter: %s", alreadyMigrated, raw)
	}
}

// Sanity: confirm ciphertext base64 representation is what we expect (so the
// DB JSON shape is stable across versions).
func TestSettingRepo_DBJSONShape(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewSettingRepository(db, newTestVault(t))
	_ = repo.UpdateAISettings(&model.AISettings{
		Provider: "openai", APIKey: "k",
		BaseURL: "u", Model: "m",
	})

	raw := readRawSettingValue(t, db, "ai.settings")
	var probe struct {
		APIKeyEncrypted *struct {
			CT    string `json:"ct"`
			Nonce string `json:"nonce"`
		} `json:"api_key_encrypted"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if probe.APIKeyEncrypted == nil {
		t.Fatal("api_key_encrypted missing")
	}
	if _, err := base64.StdEncoding.DecodeString(probe.APIKeyEncrypted.CT); err != nil {
		t.Errorf("ct not base64: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(probe.APIKeyEncrypted.Nonce); err != nil {
		t.Errorf("nonce not base64: %v", err)
	}
}
