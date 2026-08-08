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
			return nil, fmt.Errorf("vault: key file %s is %d bytes, want 32 bytes", path, info.Size())
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
