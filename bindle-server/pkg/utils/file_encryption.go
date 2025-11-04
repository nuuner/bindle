package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"

	"github.com/nuuner/bindle-server/internal/config"
)

// EncryptChunk encrypts a single chunk independently with a unique nonce.
// This allows for streaming encryption without loading the entire file into memory.
// Format: [chunkNumber(8bytes)][nonce(12bytes)][encryptedData][authTag(16bytes)]
func EncryptChunk(c *config.Config, chunkData []byte, chunkNumber int) ([]byte, error) {
	block, err := aes.NewCipher([]byte(c.EncryptionKey))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Prepend chunk number for integrity verification
	chunkNumberBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(chunkNumberBytes, uint64(chunkNumber))

	// Encrypt: [chunkNumber][nonce][encrypted data]
	result := make([]byte, 0, 8+len(nonce)+len(chunkData)+gcm.Overhead())
	result = append(result, chunkNumberBytes...)
	result = append(result, nonce...)

	// Seal encrypts and authenticates
	ciphertext := gcm.Seal(nil, nonce, chunkData, chunkNumberBytes)
	result = append(result, ciphertext...)

	return result, nil
}

// DecryptChunk decrypts a single encrypted chunk.
// Verifies chunk number integrity and GCM authentication.
func DecryptChunk(c *config.Config, encryptedChunk []byte, expectedChunkNumber int) ([]byte, error) {
	block, err := aes.NewCipher(c.EncryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	minSize := 8 + nonceSize + gcm.Overhead()

	if len(encryptedChunk) < minSize {
		return nil, errors.New("encrypted chunk too short")
	}

	// Extract components
	chunkNumberBytes := encryptedChunk[:8]
	nonce := encryptedChunk[8 : 8+nonceSize]
	ciphertext := encryptedChunk[8+nonceSize:]

	// Verify chunk number
	chunkNumber := binary.BigEndian.Uint64(chunkNumberBytes)
	if int(chunkNumber) != expectedChunkNumber {
		return nil, errors.New("chunk number mismatch")
	}

	// Decrypt and verify authentication
	plaintext, err := gcm.Open(nil, nonce, ciphertext, chunkNumberBytes)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptFile encrypts an entire file at once (legacy method for backward compatibility)
func EncryptFile(c *config.Config, bytes []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(c.EncryptionKey))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, bytes, nil)

	return ciphertext, nil
}

func DecryptFile(c *config.Config, bytes []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.EncryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(bytes) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := bytes[:nonceSize], bytes[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
