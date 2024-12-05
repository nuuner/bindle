package utils

import (
	"strings"
	"testing"

	"github.com/nuuner/bindle-server/internal/config"
)

func TestEncryptFile(t *testing.T) {
	bytes := []byte("Hello, world!")
	config := config.Config{
		// this is a test key, not a real key
		EncryptionKey: []byte("11111111111111111111111111111111"),
	}

	encrypted, err := EncryptFile(&config, bytes)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(string(encrypted), string(bytes)) == 0 {
		t.Fatal("Encrypted file is the same as the original file")
	}

	decrypted, err := DecryptFile(&config, encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(string(decrypted), string(bytes)) != 0 {
		t.Fatal("Decrypted file does not match original file")
	}
}
