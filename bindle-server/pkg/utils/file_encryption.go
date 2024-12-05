package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/nuuner/bindle-server/internal/config"
)

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
