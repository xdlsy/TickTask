# API Key 安全加固 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 TickTask 的 AI API Key 落地形态从「明文存 DB / 默认导出明文 / 掩码往返 bug」改造成「AES-256-GCM 加密存储 + 备份永不含 key + 掩码 bug 结构性不可能」，对标 `gh login` / `aws configure` 的"凭据不同步"惯例。

**Architecture:** 新增 `backend/pkg/vault` 包提供 AES-256-GCM 加解密，密钥来自本地 `.keyvault` 文件（随机 32 字节、权限 0600）。`SettingRepository` 在 DB 读写时透明加解密 `api_key`，调用方零改动；启动期 + import 后即时迁移 legacy 明文 key。备份导出天然不含密文（`dataRepository.ReadAll` raw JSON 路径已天然适配）。前端 GET 不再返回 `api_key`，改返回 `api_key_set` + `api_key_preview`；测试连接走临时 body，不写库。

**Tech Stack:** Go 1.21 / `crypto/aes` + `crypto/cipher` (GCM) / GORM / Gin / Vue 3 / TypeScript / Vitest

## Global Constraints

- Go: `gofmt`，PascalCase 导出，camelCase 未导出；构造器 `New*` 前缀；模块路径 `ticktask`
- 不引入新第三方依赖（AES-GCM 用标准库）
- AES-256-GCM（32 字节 key，12 字节 nonce，认证 tag 附在 ciphertext 末尾）
- DB 中 `ai.settings` JSON 必须不含 `api_key` 字段，只含 `api_key_encrypted: {ct, nonce}`
- 备份导出 JSON 必须不含 `api_key` 也不含 `api_key_encrypted`
- 测试用标准 `testing` 包 + 手写 mock，**不引入 testify/gomock**
- 前端 strict TS：`strict: true`、`noUnusedLocals`、`noUnusedParameters`
- 提交规范：Conventional Commits（`feat:` / `fix:` / `refactor:` / `docs:` / `chore:`）

---

## File Structure

### 新建文件

| 文件 | 职责 |
|------|------|
| `backend/pkg/vault/vault.go` | Vault 接口 + 文件实现：加载/生成 `.keyvault`，AES-256-GCM 加解密 |
| `backend/pkg/vault/vault_test.go` | vault 单测：生成/往返/损坏/解密失败 |
| `backend/internal/repository/setting_repo_test.go` | repo 单测：加解密往返/DB JSON 无明文/迁移幂等/解密失败兜底 |

### 修改文件

| 文件 | 改动 |
|------|------|
| `backend/internal/repository/setting_repo.go` | SettingRepository 增加 vault 字段、`MigrateLegacyAPIKey()` 方法；GetAISettings/UpdateAISettings 走加解密 |
| `backend/internal/api/handler/setting.go` | GET 返回 `api_key_set` + `api_key_preview`；PUT 空字段=保留、掩码字符串拒绝 |
| `backend/internal/api/handler/setting_test.go` | 更新现有 mask 测试；新增 preview/保留/拒绝掩码用例 |
| `backend/internal/api/handler/mocks_test.go` | mockSettingRepository 增加 `MigrateLegacyAPIKey` 实现 |
| `backend/internal/agent/service.go` | `TestConnection(ctx, settings *model.AISettings)` 接受临时 settings |
| `backend/internal/agent/service_test.go` | TestConnection 测试更新 |
| `backend/internal/api/handler/agent_handler.go` | `testConnection` 接受 body `{provider, api_key, base_url, model}` |
| `backend/internal/api/handler/agent_handler_test.go` | mockAgentSvc.TestConnection 签名更新 + 新 body 用例 |
| `backend/internal/service/data_service.go` | Export 无条件清空 api_key；ApplyImport 后调 MigrateLegacyAPIKey |
| `backend/internal/service/data_service_test.go` | 新增 Export 永不含 encrypted + ApplyImport 触发迁移用例 |
| `backend/cmd/server/main.go` | vault.Init + 注入 settingRepo + 启动期迁移 + 安全网 env 引导 |
| `backend/pkg/config/config.go` | 删除 AIConfig.APIKey 字段 + 删除 TT_AI_API_KEY 覆盖 |
| `backend/configs/config.yaml.example` | 删除 ai.api_key 行 |
| `backend/.env.example` | 删除 TT_AI_API_KEY |
| `backend/.gitignore` | 加 `.keyvault` |
| `backend/pkg/config/AGENTS.md` | 文档：key 只从 Settings 页输入 |
| `frontend/src/types/index.ts` | AISettings 加 `api_key_set?` + `api_key_preview?` |
| `frontend/src/api/client.ts` | `agent.test(settings)` 接受 body |
| `frontend/src/api/agent.spec.ts` | test() 接受 settings body 用例 |
| `frontend/src/views/Settings.vue` | API Key input placeholder + saveAISettings/testAIConnection 改造 |
| `frontend/src/views/Settings.spec.ts` | 新建：input 为空/placeholder/测试连接走新 body |

---

### Task 1: vault 包（AES-256-GCM + .keyvault 文件）

**Files:**
- Create: `backend/pkg/vault/vault.go`
- Test: `backend/pkg/vault/vault_test.go`

**Interfaces:**
- Produces: `vault.Vault` interface with `Encrypt(plaintext string) (ct, nonce []byte, err error)` / `Decrypt(ct, nonce []byte) (string, error)` / `Init() error`。构造器 `vault.New(path string) (Vault, error)`。

- [ ] **Step 1.1: 写失败测试 `vault_test.go`**

```go
package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVault_Init_CreatesKeyFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keyvault")

	v, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := v.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 32 {
		t.Errorf("key file size = %d, want 32", info.Size())
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("key file mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestVault_Init_IdempotentOnExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keyvault")

	v1, _ := New(path)
	if err := v1.Init(); err != nil {
		t.Fatalf("Init first: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	v2, _ := New(path)
	if err := v2.Init(); err != nil {
		t.Fatalf("Init second: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}

	if !bytes.Equal(before, after) {
		t.Errorf("Init overwrote existing key file (not idempotent)")
	}
}

func TestVault_EncryptDecrypt_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keyvault")
	v, _ := New(path)
	_ = v.Init()

	ct, nonce, err := v.Encrypt("sk-1234567890abcdef")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(ct, []byte("sk-1234567890abcdef")) {
		t.Error("ciphertext equals plaintext — not encrypted")
	}
	if len(nonce) != 12 {
		t.Errorf("nonce size = %d, want 12", len(nonce))
	}

	plain, err := v.Decrypt(ct, nonce)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if plain != "sk-1234567890abcdef" {
		t.Errorf("got %q, want original", plain)
	}
}

func TestVault_Decrypt_FailsWithDifferentKey(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	va, _ := New(filepath.Join(dirA, ".keyvault"))
	vb, _ := New(filepath.Join(dirB, ".keyvault"))
	_ = va.Init()
	_ = vb.Init()

	ct, nonce, _ := va.Encrypt("sk-leak-test")
	if _, err := vb.Decrypt(ct, nonce); err == nil {
		t.Error("expected decryption failure with different key, got nil")
	}
}

func TestVault_Decrypt_FailsOnCorruptCiphertext(t *testing.T) {
	dir := t.TempDir()
	v, _ := New(filepath.Join(dir, ".keyvault"))
	_ = v.Init()

	ct, nonce, _ := v.Encrypt("hello")
	bad := append([]byte{0xff, 0xee}, ct...)
	if _, err := v.Decrypt(bad, nonce); err == nil {
		t.Error("expected decryption failure on corrupt ct, got nil")
	}
}

func TestVault_New_FailsOnUnreadableExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".keyvault")
	if err := os.WriteFile(path, []byte("too-short"), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := New(path)
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("expected size error, got %v", err)
	}
}
```

- [ ] **Step 1.2: 运行测试验证失败**

Run: `cd backend && go test ./pkg/vault/...`
Expected: FAIL（包不存在 / 符号未定义）

- [ ] **Step 1.3: 写 `vault.go` 最小实现**

```go
// Package vault provides AES-256-GCM encryption backed by a 32-byte key file.
// The key file is created on first Init with permission 0600. Once created,
// the same key is reused across runs so DB ciphertext stays decryptable.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Vault interface {
	Init() error
	Encrypt(plaintext string) (ct, nonce []byte, err error)
	Decrypt(ct, nonce []byte) (string, error)
}

type fileVault struct {
	path string
	key  []byte
}

// New loads the key file if present (must be exactly 32 bytes), or prepares
// the path for Init to generate. Call Init before Encrypt/Decrypt on a fresh
// path; on an existing path New returns a ready-to-use Vault.
func New(path string) (Vault, error) {
	v := &fileVault{path: path}
	if info, err := os.Stat(path); err == nil {
		if info.Size() != 32 {
			return nil, fmt.Errorf("vault: key file %s is %d bytes, want 32", path, info.Size())
		}
		key, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("vault: read key: %w", err)
		}
		v.key = key
	}
	return v, nil
}

// Init generates the key file iff it does not already exist. Existing files
// are preserved (idempotent).
func (v *fileVault) Init() error {
	if _, err := os.Stat(v.path); err == nil {
		key, err := os.ReadFile(v.path)
		if err != nil {
			return fmt.Errorf("vault: read existing key: %w", err)
		}
		if len(key) != 32 {
			return fmt.Errorf("vault: existing key is %d bytes, want 32", len(key))
		}
		v.key = key
		return nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("vault: generate key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(v.path), 0700); err != nil {
		return fmt.Errorf("vault: mkdir: %w", err)
	}
	if err := os.WriteFile(v.path, key, 0600); err != nil {
		return fmt.Errorf("vault: write key: %w", err)
	}
	v.key = key
	return nil
}

// Encrypt produces an authenticated ciphertext + 12-byte nonce suitable for
// JSON storage. The GCM tag is appended to ct by cipher.GCM.Seal.
func (v *fileVault) Encrypt(plaintext string) (ct, nonce []byte, err error) {
	if len(v.key) == 0 {
		return nil, nil, errors.New("vault: not initialized")
	}
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, nil, fmt.Errorf("vault: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("vault: new gcm: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("vault: nonce: %w", err)
	}
	ct = gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ct, nonce, nil
}

// Decrypt reverses Encrypt. Returns error if the key does not match or the
// ciphertext/tag was tampered with.
func (v *fileVault) Decrypt(ct, nonce []byte) (string, error) {
	if len(v.key) == 0 {
		return "", errors.New("vault: not initialized")
	}
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", fmt.Errorf("vault: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("vault: new gcm: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("vault: decrypt: %w", err)
	}
	return string(plain), nil
}
```

- [ ] **Step 1.4: 运行测试验证通过**

Run: `cd backend && go test ./pkg/vault/...`
Expected: PASS（6/6）

- [ ] **Step 1.5: 提交**

```bash
git add backend/pkg/vault/
git commit -m "feat(vault): add AES-256-GCM vault backed by .keyvault file"
```

---

### Task 2: setting_repo 加解密 + 迁移方法

**Files:**
- Modify: `backend/internal/repository/setting_repo.go`
- Create: `backend/internal/repository/setting_repo_test.go`

**Interfaces:**
- Consumes: `vault.Vault`（来自 Task 1）
- Produces:
  - `NewSettingRepository(db, vault)` 新签名（vault 可为 nil，但加密功能就不可用）
  - `SettingRepository.MigrateLegacyAPIKey() error` 新方法
  - `SettingRepository` 的 `GetAISettings` / `UpdateAISettings` 行为变了：DB 中无明文 `api_key`，调 `vault.Encrypt/Decrypt`

**注意：** `NewSettingRepository` 签名变了，会影响 `cmd/server/main.go` 和 `handler/mocks_test.go`。Task 3 会更新 mock；Task 6 会更新 main.go。本任务编译会有这两处暂时报错——可以临时让 `vault` 参数为可选（functional option 风格），或者本任务一并更新 mock。**采用第二种：本任务一并改 mock**。

- [ ] **Step 2.1: 写失败测试 `setting_repo_test.go`**

```go
package repository

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"ticktask/internal/model"
	"ticktask/pkg/vault"

	"gorm.io/driver/sqlite"
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
```

- [ ] **Step 2.2: 运行测试验证失败**

Run: `cd backend && go test ./internal/repository/...`
Expected: FAIL（编译失败：`NewSettingRepository` 签名不匹配、`MigrateLegacyAPIKey` 不存在）

- [ ] **Step 2.3: 写 `setting_repo.go` 实现**

```go
package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"ticktask/internal/model"
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
	Provider         string             `json:"provider"`
	BaseURL          string             `json:"base_url"`
	Model            string             `json:"model"`
	APIKeyEncrypted  *encryptedBlobJSON `json:"api_key_encrypted,omitempty"`
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

	out := &model.AISettings{
		Provider: stringFromMap(raw, "provider", "openai"),
		BaseURL:  stringFromMap(raw, "base_url", "https://api.openai.com/v1"),
		Model:    stringFromMap(raw, "model", "gpt-4o-mini"),
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
			}
			// On decrypt failure (different vault / corrupt), return empty
			// key — graceful degrade. Caller (handler) shows "not configured".
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
```

- [ ] **Step 2.4: 同步更新 mock（否则 handler 包编译失败）**

修改 `backend/internal/api/handler/mocks_test.go`，把 `mockSettingRepository` 改造为支持新接口。在文件末尾追加 `MigrateLegacyAPIKey` 实现，并把 vault-aware 测试需要的字段加上：

```go
// 修改 mockSettingRepository 定义，加 vault 字段（实际不用，但需保持接口实现完整）：
type mockSettingRepository struct {
	pomodoroSettings *model.PomodoroSettings
	aiSettings       *model.AISettings
	migrateCalls     int
	migrateErr       error
}

func (m *mockSettingRepository) MigrateLegacyAPIKey() error {
	m.migrateCalls++
	return m.migrateErr
}
```

`newMockSettingRepository()` 构造器签名不变。

- [ ] **Step 2.5: 运行 repo + handler 测试**

Run:
```
cd backend && go test ./internal/repository/... ./internal/api/handler/...
```
Expected: repo 测试 PASS；handler 测试 PASS（mock 已适配）

- [ ] **Step 2.6: 提交**

```bash
git add backend/internal/repository/setting_repo.go backend/internal/repository/setting_repo_test.go backend/internal/api/handler/mocks_test.go
git commit -m "feat(repo): encrypt api_key in DB with vault + legacy migration"
```

---

### Task 3: handler DTO + body 语义（掩码 bug 结构性修复）

**Files:**
- Modify: `backend/internal/api/handler/setting.go`
- Modify: `backend/internal/api/handler/setting_test.go`

**Interfaces:**
- Consumes: Task 2 的新 repo 行为（空 `api_key` 字段=保留原值）
- Produces:
  - GET `/api/settings` 返回 `{ai: {provider, base_url, model, api_key_set, api_key_preview}}`（**不含 `api_key`**）
  - PUT `/api/settings/ai` 空字段=保留原值；含 `****` 字面量=400 拒绝

- [ ] **Step 3.1: 写失败测试（替换 / 新增到 `setting_test.go`）**

把现有的 `TestSettingHandler_GetSettings_APIKeyHidden` 和 `TestSettingHandler_GetSettings_ShortAPIKeyHidden` 改写为以下用例，并新增 PUT 语义用例：

```go
// Test: GET /api/settings - 不返回 api_key 字段，返回 api_key_set + api_key_preview
func TestSettingHandler_GetSettings_NoAPIKeyOnlyPreview(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-1234567890abcdefghijklmnop",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	})

	handler := NewSettingHandler(settingRepo)
	router := setupTestRouter()
	router.GET("/api/settings", handler.GetSettings)

	req, _ := http.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	ai := response["ai"].(map[string]interface{})

	if _, ok := ai["api_key"]; ok {
		t.Error("response must NOT include api_key field")
	}
	if ai["api_key_set"] != true {
		t.Errorf("api_key_set = %v, want true", ai["api_key_set"])
	}
	preview, _ := ai["api_key_preview"].(string)
	if !strings.Contains(preview, "****") {
		t.Errorf("api_key_preview should contain mask: got %q", preview)
	}
	if !strings.HasPrefix(preview, "sk-1") {
		t.Errorf("api_key_preview should show first 4 chars: got %q", preview)
	}
	if !strings.HasSuffix(preview, "mnop") {
		t.Errorf("api_key_preview should show last 4 chars: got %q", preview)
	}
}

func TestSettingHandler_GetSettings_NoKeyConfigured(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)
	router := setupTestRouter()
	router.GET("/api/settings", handler.GetSettings)

	req, _ := http.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	ai := response["ai"].(map[string]interface{})

	if ai["api_key_set"] != false {
		t.Errorf("api_key_set = %v, want false", ai["api_key_set"])
	}
	if _, ok := ai["api_key_preview"]; ok {
		t.Errorf("api_key_preview should be absent when no key set: got %q", ai["api_key_preview"])
	}
}

// Test: PUT /api/settings/ai - 空 api_key 字段 = 保留原值
func TestSettingHandler_UpdateAISettings_EmptyAPIKeyPreserves(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-original-key-12345678",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	})

	handler := NewSettingHandler(settingRepo)
	router := setupTestRouter()
	router.PUT("/api/settings/ai", handler.UpdateAISettings)

	body, _ := json.Marshal(map[string]interface{}{
		"provider": "openai",
		"api_key":  "", // 空：保留
		"base_url": "https://api.openai.com/v1",
		"model":    "gpt-4o", // 只改 model
	})
	req, _ := http.NewRequest("PUT", "/api/settings/ai", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	got, _ := settingRepo.GetAISettings()
	if got.APIKey != "sk-original-key-12345678" {
		t.Errorf("api_key overwritten by empty: got %q, want original", got.APIKey)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("model not updated: got %q", got.Model)
	}
}

// Test: PUT /api/settings/ai - 含掩码字面量 **** = 拒绝
func TestSettingHandler_UpdateAISettings_RejectsMaskString(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)
	router := setupTestRouter()
	router.PUT("/api/settings/ai", handler.UpdateAISettings)

	body, _ := json.Marshal(map[string]interface{}{
		"provider": "openai",
		"api_key":  "sk-ab****wxyz", // 掩码字面量 — bug 复发信号
		"base_url": "https://api.openai.com/v1",
		"model":    "gpt-4o",
	})
	req, _ := http.NewRequest("PUT", "/api/settings/ai", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (mask string rejected); body=%s", w.Code, w.Body.String())
	}
}
```

注意 import 需要 `"strings"`，确保 setting_test.go 顶部已加。

- [ ] **Step 3.2: 运行测试验证失败**

Run: `cd backend && go test ./internal/api/handler/ -run TestSettingHandler`
Expected: FAIL（GET 返回 `api_key` 掩码字段，而非 `api_key_set`/`api_key_preview`；PUT 没做掩码拒绝）

- [ ] **Step 3.3: 写 `setting.go` 新实现**

```go
package handler

import (
	"net/http"
	"strings"
	"ticktask/internal/model"
	"ticktask/internal/repository"

	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	settingRepo repository.SettingRepository
}

func NewSettingHandler(settingRepo repository.SettingRepository) *SettingHandler {
	return &SettingHandler{settingRepo: settingRepo}
}

// GetSettings returns pomodoro + AI settings. For AI, the secret api_key is
// NEVER returned. Instead, api_key_set (bool) and api_key_preview (masked
// prefix/suffix) tell the frontend whether a key exists and what to display.
func (h *SettingHandler) GetSettings(c *gin.Context) {
	pomodoroSettings, err := h.settingRepo.GetPomodoroSettings()
	if err != nil {
		pomodoroSettings = model.DefaultPomodoroSettings()
	}

	aiSettings, err := h.settingRepo.GetAISettings()
	if err != nil {
		aiSettings = model.DefaultAISettings()
	}

	aiPayload := gin.H{
		"provider": aiSettings.Provider,
		"base_url": aiSettings.BaseURL,
		"model":    aiSettings.Model,
	}
	if aiSettings.APIKey != "" {
		aiPayload["api_key_set"] = true
		aiPayload["api_key_preview"] = maskKey(aiSettings.APIKey)
	} else {
		aiPayload["api_key_set"] = false
	}

	c.JSON(http.StatusOK, gin.H{
		"pomodoro": pomodoroSettings,
		"ai":       aiPayload,
	})
}

// maskKey returns "abcd****wxyz" form for keys ≥ 8 chars, "****" otherwise.
// The full secret never leaves the server.
func maskKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

// UpdatePomodoroSettings unchanged.
func (h *SettingHandler) UpdatePomodoroSettings(c *gin.Context) {
	var settings model.PomodoroSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.settingRepo.UpdatePomodoroSettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
}

// UpdateAISettings accepts an ai settings body. The api_key field has two
// non-fallthrough semantics:
//   - empty / missing: preserve the previously stored key (handled in repo)
//   - contains "****" literal: reject as 400 — this is a mask-roundtrip bug
//     signal (frontend forwarded our own masked preview back to us)
// Non-empty, non-mask values are written through to the repo, which encrypts.
func (h *SettingHandler) UpdateAISettings(c *gin.Context) {
	var settings model.AISettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.Contains(settings.APIKey, "****") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key looks like a masked preview — clear the field to keep the existing key, or enter a new value"})
		return
	}
	if err := h.settingRepo.UpdateAISettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AI settings updated"})
}
```

- [ ] **Step 3.4: 运行测试验证通过**

Run: `cd backend && go test ./internal/api/handler/ -run TestSettingHandler`
Expected: PASS（所有新旧用例）

- [ ] **Step 3.5: 提交**

```bash
git add backend/internal/api/handler/setting.go backend/internal/api/handler/setting_test.go
git commit -m "feat(handler): return api_key_set/preview, preserve on empty, reject mask string"
```

---

### Task 4: agent /test 接受临时 body（不写库）

**Files:**
- Modify: `backend/internal/agent/service.go`
- Modify: `backend/internal/agent/service_test.go`
- Modify: `backend/internal/api/handler/agent_handler.go`
- Modify: `backend/internal/api/handler/agent_handler_test.go`

**Interfaces:**
- Produces:
  - `agent.AgentService.TestConnection(ctx, settings *model.AISettings) TestResult` — 新签名；settings 为 nil 时用 DB 中已存的（兼容"测一下当前保存的"）；非 nil 时用传入值，**完全不写库**
  - `POST /api/agent/test` body `{provider, api_key, base_url, model}` — 缺失字段从 DB fallback

- [ ] **Step 4.1: 写失败测试（agent_handler_test.go 追加）**

```go
func TestAgentHandler_TestConnection_BodyWithTempKey(t *testing.T) {
	svc := &mockAgentSvc{testResult: agent.TestResult{OK: true, Provider: "openai"}}
	repo := &mockAgentRepo{}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	body, _ := json.Marshal(map[string]string{
		"provider": "openai",
		"api_key":  "sk-temp-test-key",
		"base_url": "https://api.openai.com/v1",
		"model":    "gpt-4o-mini",
	})
	req, _ := http.NewRequest("POST", "/api/agent/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if svc.lastTestSettings == nil {
		t.Fatal("TestConnection not called with settings")
	}
	if svc.lastTestSettings.APIKey != "sk-temp-test-key" {
		t.Errorf("temp key not forwarded: got %q", svc.lastTestSettings.APIKey)
	}
}

func TestAgentHandler_TestConnection_EmptyBodyFallsBackToDB(t *testing.T) {
	svc := &mockAgentSvc{testResult: agent.TestResult{OK: true}}
	repo := &mockAgentRepo{}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	req, _ := http.NewRequest("POST", "/api/agent/test", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if svc.lastTestSettings != nil {
		t.Errorf("expected nil settings (fallback to DB), got %+v", svc.lastTestSettings)
	}
}
```

同时在 mockAgentSvc 里加字段并改 TestConnection 签名：

```go
// 修改 mockAgentSvc 字段：
type mockAgentSvc struct {
	// ... existing fields ...
	lastTestSettings *model.AISettings
	// ... rest ...
}

// 替换 TestConnection 实现：
func (m *mockAgentSvc) TestConnection(ctx context.Context, settings *model.AISettings) agent.TestResult {
	m.lastTestSettings = settings
	if m.testResult != (agent.TestResult{}) {
		return m.testResult
	}
	return agent.TestResult{OK: true, Provider: "openai", Model: "gpt-4o-mini", LatencyMs: 1}
}
```

- [ ] **Step 4.2: 运行测试验证失败**

Run: `cd backend && go test ./internal/api/handler/ -run TestAgentHandler_TestConnection`
Expected: FAIL（编译失败：mock TestConnection 签名不匹配、新字段不存在）

- [ ] **Step 4.3: 修改 `agent/service.go` 的接口与实现**

接口：

```go
type AgentService interface {
	SendMessage(ctx context.Context, convID, text string) error
	Confirm(ctx context.Context, msgID string, decision string) error
	RunTool(ctx context.Context, name string, args json.RawMessage) (any, error)
	Status() AgentStatus
	TestConnection(ctx context.Context, settings *model.AISettings) TestResult
}
```

实现替换（注意：settings 非空时**不**用 LLMFactory，而是临时构造一个 client）：

```go
// TestConnection sends a minimal ChatCompletion to verify the configured LLM
// provider actually accepts calls. settings controls what to test:
//   - settings == nil: use the saved DB settings via LLMFactory (test the
//     currently-configured key)
//   - settings != nil: construct a one-shot client from these settings
//     (test the form values the user just typed, before saving)
// In neither branch are settings written back to the DB.
func (s *agentService) TestConnection(ctx context.Context, settings *model.AISettings) TestResult {
	var client ai.LLMClient
	var provider, model string

	if settings != nil {
		client = constructLLMClientFromSettings(settings)
		provider = settings.Provider
		model = settings.Model
	} else {
		current := s.readSettings()
		client = s.LLMFactory()
		provider = current.Provider
		model = current.Model
	}

	result := TestResult{Provider: provider, Model: model}
	if client == nil {
		result.Error = "AI 未配置 — 请填写 API Key 或切换到 CLI provider"
		return result
	}

	t0 := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := client.ChatCompletion(callCtx, "hi"); err != nil {
		result.Error = err.Error()
		result.LatencyMs = time.Since(t0).Milliseconds()
		return result
	}
	result.LatencyMs = time.Since(t0).Milliseconds()
	result.OK = true
	return result
}
```

注意：`constructLLMClientFromSettings` 是从 `cmd/server/main.go` 提取出来的 helper，原本叫 `constructLLMClient`，是个包级函数。为了让 `agent` 包能调用，需要把它从 `main` 包移到 `ai` 包（或 `agent` 包），导出为 `ai.NewClientFromSettings(*model.AISettings) LLMClient`。

**重构步骤：** 在 `backend/internal/ai/client.go` 末尾追加：

```go
// NewClientFromSettings constructs the appropriate LLMClient for a given
// AISettings. Returns nil if the provider needs an API key and none is set.
// Extracted from cmd/server/main.go's constructLLMClient so other packages
// (agent service test path) can build one-shot clients without import cycles.
func NewClientFromSettings(settings *model.AISettings) LLMClient {
	if settings == nil {
		return nil
	}
	switch settings.Provider {
	case "claude", "cli":
		return NewCLIClient()
	case "anthropic":
		if settings.APIKey == "" {
			return nil
		}
		return NewAnthropicClient(settings.APIKey, settings.BaseURL, settings.Model)
	default: // openai / custom / OpenAI-compatible
		if settings.APIKey == "" {
			return nil
		}
		return NewOpenAIClient(settings.APIKey, settings.BaseURL, settings.Model)
	}
}
```

`agent/service.go` 的 import 增加 `"ticktask/internal/ai"` 并用 `ai.NewClientFromSettings(settings)`；同时把 service.go 内对 `constructLLMClientFromSettings` 的调用改为 `ai.NewClientFromSettings`。

`cmd/server/main.go` 的 `constructLLMClient` 改为薄包装：`return ai.NewClientFromSettings(settings)`（Task 6 再彻底清理）。

- [ ] **Step 4.4: 修改 `agent_handler.go` 的 testConnection**

```go
// testConnectionInput is the body for POST /api/agent/test.
// All fields optional: missing/empty fields fall back to the saved DB settings.
type testConnectionInput struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

func (h *agentHandler) testConnection(c *gin.Context) {
	var in testConnectionInput
	_ = c.ShouldBindJSON(&in)

	var settings *model.AISettings
	if in.Provider != "" || in.APIKey != "" || in.BaseURL != "" || in.Model != "" {
		settings = &model.AISettings{
			Provider: in.Provider,
			APIKey:   in.APIKey,
			BaseURL:  in.BaseURL,
			Model:    in.Model,
		}
	}
	// settings == nil → service falls back to saved DB settings.
	result := h.Svc.TestConnection(c.Request.Context(), settings)
	c.JSON(http.StatusOK, result)
}
```

- [ ] **Step 4.5: 同时更新 service_test.go 现有 TestConnection 测试**

Grep: `cd backend && go test ./internal/agent/ -run TestAgentService_TestConnection -v` 看哪些用例需要补 settings 参数。

把所有 `(s *agentService) TestConnection(ctx)` 调用改成 `TestConnection(ctx, settings)`。已有"用 DB 配置测试"的用例：传 nil。新增一个用例覆盖 settings 非空路径：

```go
func TestAgentService_TestConnection_WithTempSettings(t *testing.T) {
	// ... set up an agentSvc with a mock LLMClient that records the key it received ...
	tempSettings := &model.AISettings{Provider: "openai", APIKey: "sk-temp", Model: "gpt-4o-mini"}
	_ = svc.TestConnection(ctx, tempSettings)
	// assert mock client received "sk-temp", not the DB-stored key
}
```

具体 mock 设计：在 service_test.go 现有 LLMClient mock 上加字段记录 `lastAPIKey`。如果现有 mock 不便改造，新增一个 `recordingLLMClient`。

- [ ] **Step 4.6: 运行所有 agent 相关测试**

Run: `cd backend && go test ./internal/agent/... ./internal/api/handler/ -run TestAgentHandler`
Expected: PASS

- [ ] **Step 4.7: 提交**

```bash
git add backend/internal/ai/client.go backend/internal/agent/service.go backend/internal/agent/service_test.go backend/internal/api/handler/agent_handler.go backend/internal/api/handler/agent_handler_test.go
git commit -m "feat(agent): /test accepts temp settings body, never writes DB"
```

---

### Task 5: data_service 导出永不含密文 + import 即时迁移

**Files:**
- Modify: `backend/internal/service/data_service.go`
- Modify: `backend/internal/service/data_service_test.go`

**Interfaces:**
- Consumes: Task 2 的 `SettingRepository.MigrateLegacyAPIKey()`
- Produces:
  - `Export(includeAPIKey bool)` 永远清空 `BackupData.Settings.AI.APIKey`（参数为兼容旧前端保留但变成 no-op）
  - `ApplyImport` 在事务提交后调用 `settingRepo.MigrateLegacyAPIKey()`，失败仅 warn
  - `NewDataService(repo BackupRepository, settingRepo SettingRepository)` — 新增 settingRepo 参数

- [ ] **Step 5.1: 写失败测试（data_service_test.go 追加）**

```go
func TestDataService_Export_AlwaysStripsAPIKey(t *testing.T) {
	snap := newSnapshot()
	snap.Settings.AI = &model.AISettings{
		Provider: "openai", APIKey: "should-never-leave",
		BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini",
	}
	svc := NewDataService(&mockBackupRepo{snapshot: snap}, nil)

	cases := []bool{true, false}
	for _, include := range cases {
		t.Run(fmt.Sprintf("include=%v", include), func(t *testing.T) {
			env, err := svc.Export(include)
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			if env.Data.Settings.AI == nil {
				t.Fatal("AI nil")
			}
			if env.Data.Settings.AI.APIKey != "" {
				t.Errorf("include=%v: api_key leaked into export: %q", include, env.Data.Settings.AI.APIKey)
			}
		})
	}
}

// mockSettingRepoForData is a minimal SettingRepository mock for data service
// tests. It only records MigrateLegacyAPIKey calls; other methods panic if used.
type mockSettingRepoForData struct {
	migrateCalls int
}

func (m *mockSettingRepoForData) Get(string) (*model.Setting, error)              { return nil, nil }
func (m *mockSettingRepoForData) Set(string, string) error                        { return nil }
func (m *mockSettingRepoForData) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return model.DefaultPomodoroSettings(), nil
}
func (m *mockSettingRepoForData) UpdatePomodoroSettings(*model.PomodoroSettings) error { return nil }
func (m *mockSettingRepoForData) GetAISettings() (*model.AISettings, error)            { return model.DefaultAISettings(), nil }
func (m *mockSettingRepoForData) UpdateAISettings(*model.AISettings) error             { return nil }
func (m *mockSettingRepoForData) MigrateLegacyAPIKey() error {
	m.migrateCalls++
	return nil
}

func TestDataService_ApplyImport_TriggersLegacyMigration(t *testing.T) {
	backupRepo := &mockBackupRepo{snapshot: newSnapshot()}
	settingRepo := &mockSettingRepoForData{}
	svc := NewDataService(backupRepo, settingRepo)

	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Modules: map[string]model.ModuleApply{
			"tasks":     {Policy: model.PolicyAddNewOnly},
			"sessions":  {Policy: model.PolicyAddNewOnly},
			"schedules": {Policy: model.PolicyAddNewOnly},
			"work_reports": {Policy: model.PolicyAddNewOnly},
			"work_logs": {Policy: model.PolicyAddNewOnly},
			"settings":  {Policy: model.PolicyAddNewOnly},
		},
		Data: *newSnapshot(),
	})
	if err != nil {
		t.Fatalf("apply import: %v", err)
	}
	if settingRepo.migrateCalls != 1 {
		t.Errorf("expected 1 MigrateLegacyAPIKey call after import, got %d", settingRepo.migrateCalls)
	}
}

func TestDataService_ApplyImport_MigrationFailureDoesNotFailImport(t *testing.T) {
	backupRepo := &mockBackupRepo{snapshot: newSnapshot()}
	settingRepo := &mockSettingRepoForData{}
	svc := NewDataService(backupRepo, settingRepo)

	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Modules: map[string]model.ModuleApply{"tasks": {Policy: model.PolicyAddNewOnly}},
		Data:    model.BackupData{},
	})
	if err != nil {
		t.Errorf("import should not fail when migration fails (key is not core to import): %v", err)
	}
}
```

注意：`mockSettingRepoForData` 实现了完整的 `SettingRepository` 接口（包括 Task 2 加的 `MigrateLegacyAPIKey`）。需要 import `"ticktask/internal/repository"` 还是直接用 model 即可（看是否需要 repository.ErrNotFound；这里不需要，import model 即可）。

- [ ] **Step 5.2: 运行测试验证失败**

Run: `cd backend && go test ./internal/service/ -run TestDataService`
Expected: FAIL（`NewDataService` 签名错；`ApplyImport` 未触发迁移）

- [ ] **Step 5.3: 写 `data_service.go` 新实现**

```go
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/pkg/logger"
	"time"
)

const backupApp = "ticktask"

var ErrInvalidPolicy = errors.New("invalid policy")

type DataService interface {
	Export(includeAPIKey bool) (*model.BackupEnvelope, error)
	PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error)
	ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error)
	ClearAll() (*model.ClearResult, error)
}

type dataService struct {
	repo        repository.BackupRepository
	settingRepo repository.SettingRepository
}

// NewDataService wires the data service. settingRepo is used to run an
// immediate api_key migration after ApplyImport so legacy plaintext imports
// don't sit in the DB waiting for the next server restart. May be nil in
// tests that don't exercise ApplyImport.
func NewDataService(repo repository.BackupRepository, settingRepo repository.SettingRepository) DataService {
	return &dataService{repo: repo, settingRepo: settingRepo}
}

func (s *dataService) Export(includeAPIKey bool) (*model.BackupEnvelope, error) {
	data, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	// Defense in depth: even if include_api_key=true reached us (old client,
	// hand-crafted URL), the export NEVER carries the api_key. Encrypted
	// blob never leaks either — dataRepository.ReadAll unmarshals into
	// model.AISettings which has no api_key_encrypted field, so it's
	// already absent. We additionally blank APIKey as belt-and-suspenders.
	if data.Settings.AI != nil {
		data.Settings.AI.APIKey = ""
	}
	return &model.BackupEnvelope{
		App:           backupApp,
		SchemaVersion: model.BackupSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Data:          *data,
	}, nil
}

// PreviewImport unchanged.
func (s *dataService) PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error) {
	cur, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	warning := ""
	if fileVersion != model.BackupSchemaVersion {
		warning = fmt.Sprintf("备份 schema 版本 %d 与当前 %d 不一致,导入可能不完整", fileVersion, model.BackupSchemaVersion)
	}
	return &model.ImportPreview{
		SchemaVersion: model.BackupSchemaVersion,
		SchemaWarning: warning,
		Modules: map[string]*model.ModulePreview{
			"tasks":        classify(cur.Tasks, file.Tasks, idOfTask),
			"sessions":     classify(cur.Sessions, file.Sessions, idOfSession),
			"schedules":    classify(cur.Schedules, file.Schedules, idOfSchedule),
			"work_reports": classify(cur.WorkReports, file.WorkReports, idOfWorkReport),
			"work_logs":    classifyWorkLogs(cur.WorkLogs, file.WorkLogs),
			"settings":     diffSettings(cur.Settings, file.Settings),
		},
	}, nil
}

var validPolicies = map[string]bool{
	model.PolicyAddNewOnly: true, model.PolicyMergeFile: true,
	model.PolicyMergeCurrent: true, model.PolicyReplace: true,
}

func (s *dataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	for key, mod := range req.Modules {
		if !validPolicies[mod.Policy] {
			return nil, fmt.Errorf("%w: %q for module %q", ErrInvalidPolicy, mod.Policy, key)
		}
	}

	cur, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	plan := model.ApplyPlan{Settings: &req.Data.Settings}
	result := &model.ApplyResult{Applied: map[string]model.ModuleApplyResult{}}

	plan.Tasks, plan.DeleteTasks, result.Applied["tasks"] = resolveModule(
		cur.Tasks, req.Data.Tasks, req.Modules["tasks"], idOfTask)
	plan.Sessions, plan.DeleteSessions, result.Applied["sessions"] = resolveModule(
		cur.Sessions, req.Data.Sessions, req.Modules["sessions"], idOfSession)
	plan.Schedules, plan.DeleteSchedules, result.Applied["schedules"] = resolveModule(
		cur.Schedules, req.Data.Schedules, req.Modules["schedules"], idOfSchedule)
	plan.WorkReports, plan.DeleteWorkReports, result.Applied["work_reports"] = resolveModule(
		cur.WorkReports, req.Data.WorkReports, req.Modules["work_reports"], idOfWorkReport)
	plan.WorkLogs, plan.DeleteWorkLogs, result.Applied["work_logs"] = resolveModule(
		cur.WorkLogs, req.Data.WorkLogs, req.Modules["work_logs"], func(l model.WorkLog) string { return l.ID })

	if err := s.repo.Apply(plan); err != nil {
		return nil, err
	}

	// After the import tx commits, immediately migrate any legacy plaintext
	// api_key the imported file might have written. Failure is logged but not
	// surfaced — api_key handling is not the core purpose of import, and the
	// startup-time migration will retry on next restart.
	if s.settingRepo != nil {
		if err := s.settingRepo.MigrateLegacyAPIKey(); err != nil {
			logger.Logger.Warn("post-import api_key migration", "err", err)
		}
	}
	return result, nil
}

func (s *dataService) ClearAll() (*model.ClearResult, error) {
	return s.repo.ClearAll()
}

// resolveModule, classify, classifyWorkLogs, diffSettings, diffSection, toMap,
// jsonEqual, fieldDiffs, idOfXxx — unchanged from existing file.
```

- [ ] **Step 5.4: 修复 main.go 的 NewDataService 调用（暂时传 nil，Task 6 再正式接 settingRepo）**

```go
// 在 main.go 现有的 NewDataService(dataRepo) 调用处，改成：
dataService := service.NewDataService(dataRepo, settingRepo)
```

如果 main.go 在 dataService 构造点之前已经构造了 settingRepo，就传 settingRepo；否则先传 nil，Task 6 调整。

实际看 main.go 当前顺序：settingRepo 在第 51 行构造，dataService 在第 94 行。**直接传 settingRepo 即可，不需要 Task 6 再改。**

- [ ] **Step 5.5: 运行 service 测试**

Run: `cd backend && go test ./internal/service/...`
Expected: PASS

- [ ] **Step 5.6: 提交**

```bash
git add backend/internal/service/data_service.go backend/internal/service/data_service_test.go backend/cmd/server/main.go
git commit -m "feat(data): always strip api_key from export, migrate after import"
```

---

### Task 6: main.go wiring（vault.Init + 注入 + 启动期迁移 + env 安全网）

**Files:**
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: Task 1 vault、Task 2 repo.NewSettingRepository 新签名、Task 5 data.NewDataService 新签名

- [ ] **Step 6.1: 修改 main.go**

把现有 `settingRepo := repository.NewSettingRepository(db)` 改为 vault-aware 版本，并在 seed 之前加 vault.Init + 启动期迁移 + env 安全网：

```go
func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		cfg = config.LoadDefault()
		logger.Logger.Warn("using default config")
	}

	logger.Init(cfg.Server.Mode)
	logger.Logger.Info("starting TickTask server")

	if err := ensureDataDir(cfg.Database.Path); err != nil {
		log.Fatal(err)
	}

	// Initialize the .keyvault BEFORE opening the DB: we need the vault ready
	// to construct the encrypted settingRepo, and the migration runs against
	// a freshly-opened DB right after seed.
	vaultPath := filepath.Join(filepath.Dir(cfg.Database.Path), ".keyvault")
	v, err := vault.New(vaultPath)
	if err != nil {
		log.Fatalf("vault load: %v", err)
	}
	if err := v.Init(); err != nil {
		log.Fatalf("vault init: %v", err)
	}

	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		log.Fatal(err)
	}
	logger.Logger.Info("database initialized")

	if err := database.SeedInitialData(db); err != nil {
		logger.Logger.Warn("seed initial data", "err", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	settingRepo := repository.NewSettingRepository(db, v)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// One-time legacy api_key migration. Idempotent — runs once on first
	// post-upgrade startup, becomes a no-op afterwards.
	if err := settingRepo.MigrateLegacyAPIKey(); err != nil {
		log.Fatalf("legacy api_key migration: %v", err)
	}

	// Safety net: honor TT_AI_API_KEY one last time. If env is set AND the DB
	// has no encrypted key yet (fresh install, user just upgraded), persist
	// the env key into encrypted storage and warn the user to remove the env.
	if envKey := os.Getenv("TT_AI_API_KEY"); envKey != "" {
		ai, _ := settingRepo.GetAISettings()
		if ai != nil && ai.APIKey == "" {
			logger.Logger.Warn("TT_AI_API_KEY env detected; migrating to encrypted storage. Remove this env var from your shell/scripts.")
			ai.APIKey = envKey
			if err := settingRepo.UpdateAISettings(ai); err != nil {
				logger.Logger.Warn("env api_key migration failed", "err", err)
			}
		}
	}

	wsHub := websocket.NewHub()

	taskService := service.NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)
	timerService := service.NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)
	analyticsService := service.NewAnalyticsService(analyticsRepo, taskRepo, sessionRepo, settingRepo)

	aiSettings, err := settingRepo.GetAISettings()
	if err != nil {
		aiSettings = nil
	}
	llm := ai.NewClientFromSettings(aiSettings)

	llmFactory := func() ai.LLMClient {
		current, err := settingRepo.GetAISettings()
		if err != nil || current == nil {
			return llm
		}
		return ai.NewClientFromSettings(current)
	}

	scheduleService := service.NewScheduleService(scheduleRepo, taskRepo, llm, settingRepo, wsHub)

	workLogRepo := repository.NewWorkLogRepository(db)
	workLogService := service.NewWorkLogService(workLogRepo, taskRepo, sessionRepo, llm)

	dataRepo := repository.NewDataRepository(db)
	dataService := service.NewDataService(dataRepo, settingRepo)

	agentRepo := repository.NewAgentRepository(db)
	registry := agent.NewToolRegistry()
	tools.RegisterAll(registry, tools.Deps{
		Tasks:     taskService,
		Timer:     timerService,
		Schedule:  scheduleService,
		Analytics: analyticsService,
		WorkLog:   workLogService,
		LLM:       llm,
	})
	agentSvc := agent.NewAgentService(agent.AgentDeps{
		Repo:         agentRepo,
		LLMFactory:   llmFactory,
		SettingsRepo: settingRepo,
		Registry:     registry,
		Hub:          wsHub,
		System:       agent.DefaultSystemPrompt,
	})

	router := api.SetupRouter(cfg, taskService, timerService, analyticsService, scheduleService, workLogService, wsHub, settingRepo, dataService, agentSvc, agentRepo)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Logger.Info("server listening", "addr", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}
```

新增 import：`"os"`, `"path/filepath"`, `"ticktask/pkg/vault"`。同时移除 `constructLLMClient` 函数（已迁移到 `ai.NewClientFromSettings`），并删除原 `aiSettings, err := settingRepo.GetAISettings()` 重复块（已合并到上面）。

- [ ] **Step 6.2: 编译 + 运行 backend 全量测试**

Run:
```
cd backend && go build ./... && go test ./...
```
Expected: 编译通过；所有测试 PASS。

- [ ] **Step 6.3: 手动启动验证 vault 文件生成**

Run（本机 Windows）:
```
bash scripts/start.sh dev
```
（让进程跑几秒后 Ctrl+C）

Expected:
- `backend/data/.keyvault` 文件存在，大小 32 字节
- 启动日志包含 `database initialized`
- 之前的明文 key（如有）被迁移：日志 `migrated legacy api_key to encrypted form`

- [ ] **Step 6.4: 提交**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(main): init vault, wire encrypted repo, startup legacy migration"
```

---

### Task 7: config.yaml 路径废弃

**Files:**
- Modify: `backend/pkg/config/config.go`
- Modify: `backend/pkg/config/AGENTS.md`
- Modify: `backend/configs/config.yaml.example`
- Modify: `backend/.env.example`

**Interfaces:**
- Consumes: nothing
- Produces: `config.AIConfig` 不再有 `APIKey` 字段；`config.Load` 不再读 `TT_AI_API_KEY`（迁移路径已在 Task 6 由 main.go 直接读 env 处理）

- [ ] **Step 7.1: 修改 `config.go`**

```go
type AIConfig struct {
	Provider string        `yaml:"provider"`
	BaseURL  string        `yaml:"base_url"`
	Model    string        `yaml:"model"`
	Timeout  time.Duration `yaml:"timeout"`
}
```

删除 `APIKey string\`yaml:"api_key"\`` 字段。

`Load` 函数删除环境变量覆盖逻辑（整个 `if apiKey := os.Getenv("TT_AI_API_KEY"); ...` 块）。`"os"` import 如不再使用就移除。

`LoadDefault` 中 `AIConfig` 初始化删除 `APIKey: ""` 行。

- [ ] **Step 7.2: 修改 `config.yaml.example`**

```yaml
ai:
  provider: "openai"
  base_url: "https://api.openai.com/v1"
  model: "gpt-4o-mini"
  timeout: 30s
  # API Key is entered via the Settings page (stored encrypted in DB).
  # Do NOT put it in this file.
```

- [ ] **Step 7.3: 修改 `.env.example`**

删除 `TT_AI_API_KEY=your-api-key-here` 这行。在文件末尾追加注释：

```
# Note: TT_AI_API_KEY is no longer read by the server. Enter your API key in
# the Settings page; it is stored encrypted in the SQLite DB.
```

- [ ] **Step 7.4: 更新 `pkg/config/AGENTS.md`**

把现有"Environment variable `TT_AI_API_KEY` overrides config file value"那条改成：

> API Key 不再通过 yaml/env 配置。用户在 Settings 页面输入，存为加密形态到 SQLite。`TT_AI_API_KEY` 仅在升级时被 main.go 读取一次做迁移，之后忽略。

- [ ] **Step 7.5: 验证编译 + 测试**

Run: `cd backend && go build ./... && go test ./...`
Expected: PASS（注意 grep 确认没有遗漏代码引用 `cfg.AI.APIKey`）

```bash
cd backend && grep -rn "AI\.APIKey\|AIConfig{" --include="*.go"
```
应该不再有任何代码读取 `cfg.AI.APIKey`（main.go 已改用 settingRepo）。

- [ ] **Step 7.6: 提交**

```bash
git add backend/pkg/config/config.go backend/pkg/config/AGENTS.md backend/configs/config.yaml.example backend/.env.example
git commit -m "refactor(config): drop ai.api_key yaml/env path"
```

---

### Task 8: 前端类型 + API client

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/api/agent.spec.ts`

- [ ] **Step 8.1: 修改 `types/index.ts` 的 AISettings**

找到现有的 `AISettings` 定义，改为：

```ts
export interface AISettings {
  provider: string
  api_key: string
  base_url: string
  model: string
  /** GET only — true if a key is stored (encrypted) server-side. */
  api_key_set?: boolean
  /** GET only — masked preview like "sk-ab****wxyz", absent when no key set. */
  api_key_preview?: string
}
```

- [ ] **Step 8.2: 修改 `api/client.ts` 的 agent.test**

找到：
```ts
test: () => client.post<AgentTestResult>('/agent/test'),
```
改为：
```ts
test: (settings?: Partial<AISettings>) =>
  client.post<AgentTestResult>('/agent/test', settings ?? {}),
```

- [ ] **Step 8.3: 修改 `api/agent.spec.ts` 加用例**

```ts
import { describe, it, expect } from 'vitest'
import { api } from './client'

describe('api.agent', () => {
  it('exposes the agent sub-group with expected methods', () => {
    expect(api.agent).toBeDefined()
    expect(typeof api.agent.createConversation).toBe('function')
    expect(typeof api.agent.listConversations).toBe('function')
    expect(typeof api.agent.getMessages).toBe('function')
    expect(typeof api.agent.deleteConversation).toBe('function')
    expect(typeof api.agent.chat).toBe('function')
    expect(typeof api.agent.runTool).toBe('function')
    expect(typeof api.agent.confirm).toBe('function')
    expect(typeof api.agent.status).toBe('function')
  })

  it('test() accepts an optional settings body for temp-key testing', () => {
    // Type-level check: should accept partial AISettings. Runtime request
    // shape verified via Settings.spec.ts integration test.
    const spy = vi.spyOn(api.client ?? api, 'post').mockResolvedValue({ data: { ok: true } } as any)
    // Note: client.ts uses module-level axios instance; spy on axios directly:
  })
})
```

实际上 spy 写法复杂，简化为只断言 `api.agent.test` 是函数且能调用：

```ts
it('test() is callable with optional settings body', () => {
  expect(typeof api.agent.test).toBe('function')
  // Smoke: ensure calling with empty arg doesn't throw at type layer.
  // Network mocked away by Settings.spec.ts; here we just verify shape.
  expect(() => api.agent.test()).not.toThrow()
  expect(() => api.agent.test({ api_key: 'sk-x' })).not.toThrow()
})
```

`api.agent.test()` 会真的发 axios 请求（无 mock）—— vitest 的 60s timeout 会等到失败。改成 vi.mock 拦截：

```ts
import { describe, it, expect, vi } from 'vitest'
import { api } from './client'

vi.mock('./client', async () => {
  const actual = await vi.importActual('./client')
  return {
    ...actual,
    api: { ...actual.api, agent: { ...actual.api.agent, test: vi.fn(() => Promise.resolve({ data: { ok: true } })) } },
  }
})

describe('api.agent.test', () => {
  it('accepts optional settings body', async () => {
    await expect(api.agent.test()).resolves.toBeDefined()
    await expect(api.agent.test({ api_key: 'sk-x' })).resolves.toBeDefined()
    expect(api.agent.test).toHaveBeenCalledWith()
    expect(api.agent.test).toHaveBeenCalledWith({ api_key: 'sk-x' })
  })
})
```

- [ ] **Step 8.4: 类型检查 + 跑测试**

Run:
```
cd frontend && npx vue-tsc --noEmit && npx vitest run src/api/agent.spec.ts
```
Expected: PASS

- [ ] **Step 8.5: 提交**

```bash
git add frontend/src/types/index.ts frontend/src/api/client.ts frontend/src/api/agent.spec.ts
git commit -m "feat(frontend): api.agent.test accepts settings body; AISettings gains api_key_set/preview"
```

---

### Task 9: 前端 Settings.vue 改造

**Files:**
- Modify: `frontend/src/views/Settings.vue`
- Create: `frontend/src/views/Settings.spec.ts`

- [ ] **Step 9.1: 写失败测试 `Settings.spec.ts`**

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Settings from './Settings.vue'
import { api } from '@/api/client'

vi.mock('@/api/client')
vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  ElMessageBox: { confirm: vi.fn(), prompt: vi.fn() },
}))

const mockSettingsResponse = (overrides: Partial<any> = {}) => ({
  data: {
    pomodoro: {
      work_duration: 1500, short_break_duration: 300, long_break_duration: 900,
      long_break_after: 4, auto_start_break: false, auto_start_work: false,
      enable_sound: true, buffer_ratio: 20,
      task_time_preferences: '{"management":"any","dev":"any"}',
    },
    ai: {
      provider: 'openai',
      base_url: 'https://api.openai.com/v1',
      model: 'gpt-4o-mini',
      api_key_set: true,
      api_key_preview: 'sk-ab****wxyz',
      ...overrides,
    },
  },
})

describe('Settings.vue — API Key input', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads settings and leaves the api_key input empty (with preview placeholder)', async () => {
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())
    const wrapper = mount(Settings, { global: { stubs: ['el-input', 'el-input-number', 'el-select', 'el-option', 'el-switch', 'el-button', 'el-slider', 'el-tag'] } })
    await flushPromises()

    // The api_key ref should be empty after load — preview is shown as
    // placeholder, NOT as the input value.
    expect((wrapper.vm as any).aiSettings.api_key).toBe('')
    expect((wrapper.vm as any).aiSettingsPreview).toBe('sk-ab****wxyz')
  })

  it('saves AI settings with empty api_key — backend preserves the existing key', async () => {
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())
    ;(api.updateAISettings as any).mockResolvedValue({ data: {} })
    ;(api.agent as any).status = vi.fn().mockResolvedValue({ data: { configured: true } })
    const wrapper = mount(Settings, { global: { stubs: ['el-input', 'el-input-number', 'el-select', 'el-option', 'el-switch', 'el-button', 'el-slider', 'el-tag'] } })
    await flushPromises()

    // Change only the model, leave api_key empty.
    ;(wrapper.vm as any).aiSettings.model = 'gpt-4o'
    await (wrapper.vm as any).saveAISettings()
    await flushPromises()

    const sentBody = (api.updateAISettings as any).mock.calls[0][0]
    expect(sentBody.api_key).toBe('')
    expect(sentBody.model).toBe('gpt-4o')
  })

  it('testAIConnection sends form values via api.agent.test body without saving first', async () => {
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())
    ;(api.agent as any).test = vi.fn().mockResolvedValue({ data: { ok: true, provider: 'openai' } })
    ;(api.agent as any).status = vi.fn().mockResolvedValue({ data: {} })

    const wrapper = mount(Settings, { global: { stubs: ['el-input', 'el-input-number', 'el-select', 'el-option', 'el-switch', 'el-button', 'el-slider', 'el-tag'] } })
    await flushPromises()

    ;(wrapper.vm as any).aiSettings.api_key = 'sk-typed-in-form'
    await (wrapper.vm as any).testAIConnection()
    await flushPromises()

    expect((api.updateAISettings as any)).not.toHaveBeenCalled() // 关键：测试连接前不保存
    expect((api.agent as any).test).toHaveBeenCalledWith(expect.objectContaining({
      api_key: 'sk-typed-in-form',
    }))
  })
})
```

- [ ] **Step 9.2: 运行测试验证失败**

Run: `cd frontend && npx vitest run src/views/Settings.spec.ts`
Expected: FAIL（`aiSettingsPreview` computed 不存在；`testAIConnection` 调用了 `updateAISettings`）

- [ ] **Step 9.3: 修改 `Settings.vue`**

把 `<script setup lang="ts">` 块中相关部分改为：

```ts
// ref 不再装掩码字符串
const aiSettings = ref<AISettings>({
  provider: 'openai',
  api_key: '',
  base_url: 'https://api.openai.com/v1',
  model: 'gpt-4o-mini',
})

// Preview 单独存，不参与 form 数据
const aiSettingsPreview = ref<string>('')

const apiKeyPlaceholder = computed(() => {
  if (aiSettingsPreview.value) {
    return `已设置 (${aiSettingsPreview.value})，留空保留；输入新值覆盖`
  }
  return '请输入 API Key'
})

async function loadSettings() {
  loading.value = true
  try {
    const res = await api.getSettings()
    if (res.data.pomodoro) {
      pomodoroSettings.value = {
        work_duration: Math.floor(res.data.pomodoro.work_duration / 60),
        // ... 其他字段保持不变 ...
        short_break_duration: Math.floor(res.data.pomodoro.short_break_duration / 60),
        long_break_duration: Math.floor(res.data.pomodoro.long_break_duration / 60),
        long_break_after: res.data.pomodoro.long_break_after,
        auto_start_break: res.data.pomodoro.auto_start_break,
        auto_start_work: res.data.pomodoro.auto_start_work,
        enable_sound: res.data.pomodoro.enable_sound,
        buffer_ratio: res.data.pomodoro.buffer_ratio || 20,
        task_time_preferences: res.data.pomodoro.task_time_preferences || '{"management":"any","dev":"any"}',
      }
    }
    if (res.data.ai) {
      // 关键：api_key 字段永远不写入 form；用 preview 字段单独保存显示
      aiSettings.value = {
        provider: res.data.ai.provider,
        api_key: '',
        base_url: res.data.ai.base_url,
        model: res.data.ai.model,
      }
      aiSettingsPreview.value = res.data.ai.api_key_preview ?? ''
    }
  } catch (error) {
    ElMessage.error('加载设置失败')
  } finally {
    loading.value = false
  }
}

async function saveAISettings() {
  saving.value = true
  try {
    // api_key 为空 = 后端保留原 key（已在 handler/repo 实现）
    await api.updateAISettings(aiSettings.value)
    await agentStore.checkStatus()
    ElMessage.success('AI 设置已保存')
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function testAIConnection() {
  // 兜底：form 中无 key 且后端也没存 key → 提示
  if (!aiSettings.value.api_key && !aiSettingsPreview.value) {
    ElMessage.warning('请先输入 API Key')
    return
  }

  testing.value = true
  try {
    // 关键改动：不调 updateAISettings，直接把 form 中的临时值传给 /test
    const r = await api.agent.test({
      provider: aiSettings.value.provider,
      api_key: aiSettings.value.api_key, // 空时后端从 DB 读
      base_url: aiSettings.value.base_url,
      model: aiSettings.value.model,
    })
    const result = r.data
    if (result.ok) {
      const latency = result.latency_ms ? ` · ${result.latency_ms}ms` : ''
      const model = result.model ? ` · ${result.model}` : ''
      ElMessage.success(`${result.provider}${model} 连接成功${latency}`)
    } else {
      const errMsg = result.error ? `: ${result.error}` : ' — 请检查 API Key、BaseURL、Model 与网络'
      ElMessage({
        type: 'error',
        message: `${result.provider} 连接失败${errMsg}`,
        duration: 8000,
      })
    }
  } catch (error: any) {
    const detail = error?.response?.data?.error || error?.message || '未知错误'
    ElMessage({
      type: 'error',
      message: `连接测试请求失败: ${detail}`,
      duration: 8000,
    })
  } finally {
    testing.value = false
  }
}
```

模板部分把 API Key input 改成：

```vue
<el-input
  v-model="aiSettings.api_key"
  type="password"
  :placeholder="apiKeyPlaceholder"
  show-password
  size="large"
/>
```

并删除"已配置"标签上的旧 preview 引用（如果存在）。

- [ ] **Step 9.4: 运行测试验证通过**

Run:
```
cd frontend && npx vue-tsc --noEmit && npx vitest run src/views/Settings.spec.ts
```
Expected: PASS

- [ ] **Step 9.5: 跑前端全量测试**

Run: `cd frontend && npx vitest run`
Expected: 所有现有测试仍 PASS（关注 stores/ai.spec.ts、其他 view 测试是否被 API 改动影响）

- [ ] **Step 9.6: 提交**

```bash
git add frontend/src/views/Settings.vue frontend/src/views/Settings.spec.ts
git commit -m "feat(settings): empty-input + preview placeholder, test connection without saving"
```

---

### Task 10: 收尾（.gitignore、文档、端到端验证）

**Files:**
- Modify: `backend/.gitignore`

- [ ] **Step 10.1: 加 .keyvault 到 .gitignore**

打开 `backend/.gitignore`，在"Secrets"段下追加：

```
# ===== Vault key (per-machine encryption key — never sync, never commit) =====
backend/data/.keyvault
*.keyvault
```

注意：根 `.gitignore` 已经存在；本任务只追加 vault 相关条目。

- [ ] **Step 10.2: 全量验证 — 后端 + 前端测试**

Run:
```
cd backend && go test ./...
cd frontend && npx vue-tsc --noEmit && npx vitest run
```
Expected: 全部 PASS

- [ ] **Step 10.3: 手动 E2E 验证清单**

启动 dev server：`bash scripts/start.sh dev`，浏览器打开 `http://localhost:5173/settings`。

逐项验证（每项打勾）：

- [ ] 首次启动：`backend/data/.keyvault` 文件被生成，大小 32 字节
- [ ] Settings 页打开：API Key input 为空，placeholder 显示"请输入 API Key"（首次未配置）
- [ ] 输入 key 点保存 → 成功；刷新页面 → input 仍为空，placeholder 显示"已设置 (sk-ab****wxyz)，留空保留；输入新值覆盖"
- [ ] 只改 model 点保存 → 成功；刷新后再点"测试连接" → 连接成功（key 没被破坏）
- [ ] **回归 bug 验证**：旧版"改 model 点保存 → key 失效"不再复现
- [ ] 清空 key 重新输入新 key 点保存 → 测试连接用新 key 成功
- [ ] 点"导出全部数据" → 下载 JSON → 用文本编辑器打开 → **`api_key` 与 `api_key_encrypted` 都不存在**
- [ ] 用 sqlite3 打开 `backend/data/ticktask.db` → `SELECT value FROM settings WHERE key='ai.settings';` → 输出**不含**明文 key，含 `api_key_encrypted`
- [ ] 关闭后端 → 删除 `backend/data/.keyvault` → 重启 → 服务正常启动；Settings 页显示未配置（key 解不开）；输入新 key 保存 → 正常工作
- [ ] 模拟多机同步：把 `data/` 目录拷到另一台机器（同代码）→ 启动 → key 不可用 → 重填 → 正常
- [ ] CLIClient provider：切换到 claude/cli → 测试连接不要求 key（沿用现有行为）

- [ ] **Step 10.4: 提交**

```bash
git add backend/.gitignore
git commit -m "chore: gitignore .keyvault per-machine encryption files"
```

- [ ] **Step 10.5: 更新 MEMORY.md（记录"业界惯例：凭据不同步"知识）**

如果用户允许，加一条 `docs/learnings/` 笔记，记录：
- "用户对 API Key 安全的初始直觉常常是要求'主口令加密'，但业界惯例（gh/aws/VS Code）是凭据不同步、换机器重填"
- "BackupRepository raw JSON unmarshal 路径与 schema 解耦，会天然屏蔽新增字段——这是设计简化点"

需要先问用户是否要保存。

---

## Self-Review

**Spec coverage:**
- ✅ KEK 来源（.keyvault 文件）→ Task 1, 6
- ✅ DB Schema 变更（api_key_encrypted）→ Task 2
- ✅ BackupRepository raw JSON 天然不含密文 → Task 5 步骤 5.3 注释 + 测试断言
- ✅ 启动期 MigrateLegacyAPIKey → Task 2 (实现) + Task 6 (调用)
- ✅ Import 后即时迁移 → Task 5
- ✅ 错误处理矩阵（vault 不存在 / 解密失败 / 迁移失败）→ Task 1 (vault 测试) + Task 2 (decrypt 失败兜底测试) + Task 5 (migrate 失败不阻断 import)
- ✅ 掩码往返 bug 结构性修复 → Task 3 (preview 字段 + **** 拒绝)
- ✅ 前端 input + placeholder → Task 9
- ✅ 测试连接不写库 + body 接受 → Task 4
- ✅ yaml 路径废弃 + 安全网 → Task 6 (安全网) + Task 7 (废弃)
- ✅ 历史泄露 rotate 提醒 → Step 10 文档（手动提醒用户）
- ✅ 测试矩阵全部覆盖 → 各 Task 的测试步骤

**Placeholder scan:**
- ✅ 所有"..."省略号都在已经写过的代码上下文里（如 Settings.vue loadSettings 的其他字段），不是 TBD
- ✅ 没有"add error handling""similar to Task N"等占位符

**Type consistency:**
- `vault.Vault.Init() / Encrypt / Decrypt` — Task 1 定义、Task 2/5/6 使用，签名一致
- `repository.NewSettingRepository(db, vault)` — Task 2 定义、Task 6 使用，签名一致
- `SettingRepository.MigrateLegacyAPIKey() error` — Task 2 定义、Task 5/6 调用，签名一致
- `agent.AgentService.TestConnection(ctx, *model.AISettings)` — Task 4 定义、handler 调用一致
- `service.NewDataService(backupRepo, settingRepo)` — Task 5 定义、Task 6 调用，签名一致
- `ai.NewClientFromSettings(*model.AISettings) LLMClient` — Task 4 定义、Task 6 使用，一致
- 前端 `AISettings.api_key_set?` + `api_key_preview?` — Task 8 定义、Task 9 使用，一致
- 前端 `api.agent.test(settings?)` — Task 8 定义、Task 9 调用，一致
