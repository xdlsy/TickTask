package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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
	// Windows ignores the 0600 perm bits passed to os.WriteFile — typical Perm() is 0666.
	// The plan's spec explicitly accepts this limitation. Only assert 0600 on non-Windows.
	if runtime.GOOS != "windows" {
		if info.Mode().Perm() != 0600 {
			t.Errorf("key file mode = %o, want 0600", info.Mode().Perm())
		}
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
