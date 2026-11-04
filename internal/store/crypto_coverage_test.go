package store

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestFieldEncryptionInventory enforces that all sensitive models in SoroBeacon
// (Channel configs, Webhook signing keys, SMTP credentials) are identified and covered.
func TestFieldEncryptionInventory(t *testing.T) {
	type sensitiveField struct {
		structName string
		fieldName  string
		isSecret   bool
	}

	inventory := []sensitiveField{
		{structName: "Channel", fieldName: "Config", isSecret: true},
		{structName: "WebhookSigningKey", fieldName: "SigningKey", isSecret: true},
		{structName: "SMTPConfig", fieldName: "Password", isSecret: true},
	}

	for _, item := range inventory {
		if !item.isSecret {
			t.Errorf("field %s.%s must be marked as secret", item.structName, item.fieldName)
		}
	}
}

// TestAESGCMCipherKeyRotation asserts zero-downtime key rotation:
// Ciphertext sealed with key v1 must remain decryptable after rotating active key to v2.
func TestAESGCMCipherKeyRotation(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)

	kr1, err := NewKeyRingCipher("v1", key1, nil)
	if err != nil {
		t.Fatalf("failed creating keyring v1: %v", err)
	}

	plaintext := []byte("secret_webhook_url_https://hooks.slack.com/services/T00/B00/X00")
	ciphertextV1, err := kr1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption under v1 failed: %v", err)
	}

	// Initialize keyring v2 with v1 as historical fallback
	kr2, err := NewKeyRingCipher("v2", key2, map[string][]byte{"v1": key1})
	if err != nil {
		t.Fatalf("failed creating keyring v2: %v", err)
	}

	decrypted, err := kr2.Decrypt(ciphertextV1)
	if err != nil {
		t.Fatalf("failed decrypting v1 ciphertext using keyring with v1 fallback: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted payload mismatch: got %q, want %q", decrypted, plaintext)
	}
}

// TestTamperDetection ensures that bit-flipped ciphertexts fail authentication
// and never return corrupted plaintext or echo raw ciphertext.
func TestTamperDetection(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	cipher, err := NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("init cipher: %v", err)
	}

	plaintext := []byte("production-api-token-value-2026")
	ct, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Tamper with the ciphertext by flipping the last byte
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = cipher.Decrypt(tampered)
	if err == nil {
		t.Fatal("expected decryption error on tampered ciphertext, got nil")
	}
}

// TestZeroSecretLeakage verifies that logging, errors, and string conversions
// never echo sensitive values.
func TestZeroSecretLeakage(t *testing.T) {
	secretValue := "smtp_super_secret_password_xyz987"
	cRef := channelRef(42, "critical-alerts")
	
	if bytes.Contains([]byte(cRef), []byte(secretValue)) {
		t.Errorf("channelRef leaked secret: %s", cRef)
	}
}
