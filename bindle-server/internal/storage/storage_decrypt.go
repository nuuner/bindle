package storage

import (
	"io"

	localconfig "github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

// decryptStream wraps an encrypted object body in the reader matching the format it was
// written in, and returns the plaintext length to advertise to the client. Shared by
// both backends so the three formats are dispatched in exactly one place.
func decryptStream(body io.ReadCloser, encryptedSize int64, cfg *localconfig.Config, file StoredFile) (io.ReadCloser, int64, error) {
	if file.EncryptionVersion >= utils.EncryptionVersionStream {
		reader, err := utils.NewDecryptingReader(body, cfg.EncryptionKey, file.PlainSize)
		if err != nil {
			return nil, 0, err
		}
		return reader, file.PlainSize, nil
	}

	// Formats predating the streaming one. Both buffer more than a frame - the whole
	// file for the oldest, one chunk for the other - which is why they are reachable
	// only for files already stored that way.
	if file.ChunkCount == 0 {
		return utils.NewLegacyDecryptionReader(body, cfg), encryptedSize, nil
	}

	return utils.NewChunkedDecryptionReader(body, cfg, file.ChunkCount),
		utils.CalculateDecryptedSize(encryptedSize, file.ChunkCount, cfg.ChunkSizeMB),
		nil
}
