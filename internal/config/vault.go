package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/scrypt"
)

// Vault stores saved connection passwords encrypted at rest, keyed by
// profile ID. Two key-derivation modes are supported:
//
//   - OVFTOP_MASTER_PASSWORD set: the key is derived from that passphrase
//     via scrypt, using a persisted random salt. Nothing sensitive touches
//     disk unencrypted, but the same passphrase must be supplied every run.
//   - Unset (default): a random 32-byte key is generated once and stored
//     locally with 0600 permissions. This protects against casual copying
//     of the config directory but not against an attacker with read access
//     to the machine's filesystem as the same user — a deliberate trade-off
//     for a single-user admin CLI that must not prompt on every launch.
type Vault struct {
	dir string
	key [32]byte
}

// OpenVault derives or loads the vault encryption key and returns a handle
// ready for Get/Set/Delete calls.
func OpenVault(dir string) (*Vault, error) {
	v := &Vault{dir: dir}
	if pass := os.Getenv("OVFTOP_MASTER_PASSWORD"); pass != "" {
		salt, err := loadOrCreateSalt(vaultSaltPath(dir))
		if err != nil {
			return nil, err
		}
		key, err := scrypt.Key([]byte(pass), salt, 1<<15, 8, 1, 32)
		if err != nil {
			return nil, fmt.Errorf("deriving vault key: %w", err)
		}
		copy(v.key[:], key)
		return v, nil
	}

	key, err := loadOrCreateSalt(machineKeyPath(dir))
	if err != nil {
		return nil, err
	}
	copy(v.key[:], key)
	return v, nil
}

func loadOrCreateSalt(path string) ([]byte, error) {
	if b, err := os.ReadFile(path); err == nil && len(b) == 32 {
		return b, nil
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return nil, err
	}
	return buf, nil
}

// secrets is the plaintext structure encrypted as a single blob.
type secrets map[string]string

func (v *Vault) load() (secrets, error) {
	data, err := os.ReadFile(vaultPath(v.dir))
	if errors.Is(err, os.ErrNotExist) {
		return secrets{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return secrets{}, nil
	}

	block, err := aes.NewCipher(v.key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("vault file is corrupt")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting vault (wrong master password?): %w", err)
	}
	var s secrets
	if err := json.Unmarshal(plaintext, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func (v *Vault) save(s secrets) error {
	plaintext, err := json.Marshal(s)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(v.key[:])
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(vaultPath(v.dir), ciphertext, 0o600)
}

// Get returns the stored password for a profile ID, if any.
func (v *Vault) Get(profileID string) (string, bool, error) {
	s, err := v.load()
	if err != nil {
		return "", false, err
	}
	p, ok := s[profileID]
	return p, ok, nil
}

// Set stores (or overwrites) the password for a profile ID.
func (v *Vault) Set(profileID, password string) error {
	s, err := v.load()
	if err != nil {
		return err
	}
	s[profileID] = password
	return v.save(s)
}

// Delete removes any stored password for a profile ID.
func (v *Vault) Delete(profileID string) error {
	s, err := v.load()
	if err != nil {
		return err
	}
	delete(s, profileID)
	return v.save(s)
}
