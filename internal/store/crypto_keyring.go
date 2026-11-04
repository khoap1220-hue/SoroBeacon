package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// KeyRingCipher provides zero-downtime key rotation for sensitive column encryption.
// It maintains an active primary key for new writes while supporting a historical list
// of fallback keys for reading existing records encrypted under older keys.
type KeyRingCipher struct {
	mu           sync.RWMutex
	activeKeyID  string
	activeCipher *AESGCMCipher
	ciphers      map[string]*AESGCMCipher
}

// NewKeyRingCipher initializes a KeyRing with an active key and optional fallback keys.
func NewKeyRingCipher(activeKeyID string, activeKey []byte, fallbacks map[string][]byte) (*KeyRingCipher, error) {
	if activeKeyID == "" {
		return nil, errors.New("active key ID cannot be empty")
	}
	primary, err := NewAESGCMCipher(activeKey)
	if err != nil {
		return nil, fmt.Errorf("active key initialization failed: %w", err)
	}

	ciphers := make(map[string]*AESGCMCipher)
	ciphers[activeKeyID] = primary

	for kid, k := range fallbacks {
		if kid == activeKeyID {
			continue
		}
		c, err := NewAESGCMCipher(k)
		if err != nil {
			return nil, fmt.Errorf("fallback key %q initialization failed: %w", kid, err)
		}
		ciphers[kid] = c
	}

	return &KeyRingCipher{
		activeKeyID:  activeKeyID,
		activeCipher: primary,
		ciphers:      ciphers,
	}, nil
}

// Encrypt encrypts plaintext using the active key and prefixes the ciphertext
// with the active key ID (e.g., "v2:<base64-ciphertext>").
func (kr *KeyRingCipher) Encrypt(plaintext []byte) ([]byte, error) {
	kr.mu.RLock()
	activeID := kr.activeKeyID
	c := kr.activeCipher
	kr.mu.RUnlock()

	ct, err := c.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}
	tag := fmt.Sprintf("%s:%s", activeID, base64.StdEncoding.EncodeToString(ct))
	return []byte(tag), nil
}

// Decrypt attempts decryption using the key specified in the version prefix.
// If no prefix is found, it attempts decryption across all registered keys.
func (kr *KeyRingCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	raw := string(ciphertext)
	parts := strings.SplitN(raw, ":", 2)

	kr.mu.RLock()
	defer kr.mu.RUnlock()

	if len(parts) == 2 {
		kid, payload := parts[0], parts[1]
		if c, ok := kr.ciphers[kid]; ok {
			decoded, err := base64.StdEncoding.DecodeString(payload)
			if err == nil {
				pt, err := c.Decrypt(decoded)
				if err == nil {
					return pt, nil
				}
			}
		}
	}

	// Fallback trial across all ciphers
	for _, c := range kr.ciphers {
		pt, err := c.Decrypt(ciphertext)
		if err == nil {
			return pt, nil
		}
	}

	return nil, errors.New("decryption failed: no valid key found in KeyRing or payload corrupted")
}
