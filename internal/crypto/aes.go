package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	ivSize  = 12 // AES-GCM standard nonce size
	keySize = 32 // AES-256
)

// Encrypt encrypts plaintext with AES-256-GCM.
// Returns standard base64 of: IV[12] + ciphertext + authTag[16].
// This matches the web app's encryptBlob() in client.ts.
func Encrypt(plaintext, key []byte) (string, error) {
	if len(key) != keySize {
		return "", fmt.Errorf("key must be %d bytes, got %d", keySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	iv := make([]byte, ivSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("generating IV: %w", err)
	}

	// Seal appends ciphertext + authTag to the dst slice
	ciphertextWithTag := gcm.Seal(nil, iv, plaintext, nil)

	// Prepend IV: IV[12] + ciphertext + authTag[16]
	result := make([]byte, ivSize+len(ciphertextWithTag))
	copy(result[:ivSize], iv)
	copy(result[ivSize:], ciphertextWithTag)

	return Base64Encode(result), nil
}

// EncryptWithIV encrypts with a specific IV (for testing cross-platform parity).
// Production code should use Encrypt which generates a random IV.
func EncryptWithIV(plaintext, key, iv []byte) (string, error) {
	if len(key) != keySize {
		return "", fmt.Errorf("key must be %d bytes, got %d", keySize, len(key))
	}
	if len(iv) != ivSize {
		return "", fmt.Errorf("IV must be %d bytes, got %d", ivSize, len(iv))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	ciphertextWithTag := gcm.Seal(nil, iv, plaintext, nil)

	result := make([]byte, ivSize+len(ciphertextWithTag))
	copy(result[:ivSize], iv)
	copy(result[ivSize:], ciphertextWithTag)

	return Base64Encode(result), nil
}

// Decrypt decrypts a standard base64 AES-256-GCM blob.
// Expects format: base64(IV[12] + ciphertext + authTag[16]).
// This matches the web app's decryptBlob() in client.ts.
func Decrypt(encryptedBase64 string, key []byte) ([]byte, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", keySize, len(key))
	}

	data, err := Base64Decode(encryptedBase64)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	if len(data) < ivSize+16 { // IV + at minimum an auth tag
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(data))
	}

	iv := data[:ivSize]
	ciphertextWithTag := data[ivSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertextWithTag, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (invalid key or corrupted data): %w", err)
	}

	return plaintext, nil
}

// DecryptString is a convenience wrapper that returns the plaintext as a string.
func DecryptString(encryptedBase64 string, key []byte) (string, error) {
	plaintext, err := Decrypt(encryptedBase64, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
