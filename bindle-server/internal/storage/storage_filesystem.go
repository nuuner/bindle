package storage

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

type FilesystemStorage struct {
	config config.Config
}

func NewFilesystemStorage(config config.Config) (*FilesystemStorage, error) {
	return &FilesystemStorage{config: config}, nil
}

func (s *FilesystemStorage) SaveFile(file *multipart.FileHeader, filePath string) (string, error) {
	// Ensure filesystem path exists
	err := utils.EnsureFileDirectory(s.config)
	if err != nil {
		return "", err
	}

	// Open source file
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Read file bytes
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return "", err
	}

	encryptedBytes, err := utils.EncryptFile(&s.config, fileBytes)
	if err != nil {
		return "", err
	}

	fullPath := s.config.FilesystemPath + "/" + filePath
	log.Println("saving file to", fullPath)
	if err := os.WriteFile(fullPath, encryptedBytes, 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (s *FilesystemStorage) GetFile(filePath string, chunkCount int) ([]byte, error) {
	fullPath := s.config.FilesystemPath + "/" + filePath
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// For legacy single-file uploads (chunkCount == 0), use old decryption
	if chunkCount == 0 {
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		decryptedBytes, err := utils.DecryptFile(&s.config, fileBytes)
		if err != nil {
			return nil, err
		}

		return decryptedBytes, nil
	}

	// For chunked uploads, stream decrypt each chunk
	log.Printf("Streaming decrypt %d chunks for file %s\n", chunkCount, filePath)

	// Read entire encrypted file (contains all encrypted chunks concatenated)
	encryptedFile, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Standard chunk size from config (10MB by default)
	standardChunkSize := int(s.config.ChunkSizeMB * 1024 * 1024)
	// Encrypted chunk overhead: 8 bytes chunkNumber + 12 bytes nonce + 16 bytes authTag
	chunkOverhead := 8 + 12 + 16

	// Decrypt each chunk and accumulate
	decryptedData := make([]byte, 0)
	offset := 0

	for i := 0; i < chunkCount; i++ {
		// Calculate expected encrypted chunk size
		var expectedEncryptedSize int
		if i < chunkCount-1 {
			// Not the last chunk - standard size
			expectedEncryptedSize = standardChunkSize + chunkOverhead
		} else {
			// Last chunk - whatever remains
			expectedEncryptedSize = len(encryptedFile) - offset
		}

		if offset+expectedEncryptedSize > len(encryptedFile) {
			return nil, fmt.Errorf("encrypted file truncated at chunk %d", i)
		}

		encryptedChunk := encryptedFile[offset : offset+expectedEncryptedSize]
		decryptedChunk, err := utils.DecryptChunk(&s.config, encryptedChunk, i)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt chunk %d: %w", i, err)
		}

		decryptedData = append(decryptedData, decryptedChunk...)
		offset += expectedEncryptedSize
	}

	log.Printf("Decrypted %d chunks (%d bytes) for file %s\n", chunkCount, len(decryptedData), filePath)
	return decryptedData, nil
}

// GetFileStream returns a streaming reader for file download (memory-efficient)
// This is the preferred method for downloading files as it doesn't load entire file into memory
func (s *FilesystemStorage) GetFileStream(filePath string, chunkCount int) (io.ReadCloser, int64, error) {
	fullPath := s.config.FilesystemPath + "/" + filePath
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, err
	}

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, err
	}

	encryptedSize := stat.Size()

	// For legacy single-file uploads (chunkCount == 0)
	if chunkCount == 0 {
		reader := utils.NewLegacyDecryptionReader(file, &s.config)
		// For legacy files, we can't know exact decrypted size without decrypting
		// Return encrypted size as approximation (will be slightly larger than actual)
		return reader, encryptedSize, nil
	}

	// For chunked uploads - return streaming decryption reader
	reader := utils.NewChunkedDecryptionReader(file, &s.config, chunkCount)

	// Calculate actual decrypted size (remove encryption overhead)
	decryptedSize := utils.CalculateDecryptedSize(encryptedSize, chunkCount, s.config.ChunkSizeMB)

	log.Printf("Streaming file %s (%d chunks, %d bytes decrypted)\n", filePath, chunkCount, decryptedSize)

	return reader, decryptedSize, nil
}

func (s *FilesystemStorage) DeleteFile(filePath string) error {
	fullPath := s.config.FilesystemPath + "/" + filePath
	return os.Remove(fullPath)
}

// Chunked upload methods

func (s *FilesystemStorage) InitChunkedUpload(sessionID string, fileName string, totalChunks int) error {
	// Create temp directory for chunks
	chunksDir := s.config.FilesystemPath + "/chunks/" + sessionID
	err := os.MkdirAll(chunksDir, os.ModePerm)
	if err != nil {
		return err
	}
	log.Printf("Initialized chunked upload session %s with %d chunks\n", sessionID, totalChunks)
	return nil
}

func (s *FilesystemStorage) SaveChunk(sessionID string, chunkNumber int, chunkData []byte) error {
	chunksDir := s.config.FilesystemPath + "/chunks/" + sessionID
	chunkPath := fmt.Sprintf("%s/%d", chunksDir, chunkNumber)

	// Write chunk to temp file
	err := os.WriteFile(chunkPath, chunkData, 0644)
	if err != nil {
		return err
	}

	log.Printf("Saved chunk %d for session %s (%d bytes)\n", chunkNumber, sessionID, len(chunkData))
	return nil
}

func (s *FilesystemStorage) FinalizeChunkedUpload(sessionID string, filePath string) (string, error) {
	chunksDir := s.config.FilesystemPath + "/chunks/" + sessionID

	// Read all chunk files
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		return "", err
	}

	totalChunks := len(entries)
	log.Printf("Streaming %d chunks for session %s (zero memory accumulation)\n", totalChunks, sessionID)

	// Ensure final directory exists
	err = utils.EnsureFileDirectory(s.config)
	if err != nil {
		return "", err
	}

	// Create final output file
	fullPath := s.config.FilesystemPath + "/" + filePath
	outFile, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	// Stream encrypt each chunk to output file (one at a time - no memory accumulation)
	var totalBytes int64
	for i := 0; i < totalChunks; i++ {
		chunkPath := fmt.Sprintf("%s/%d", chunksDir, i)

		// Read chunk (10MB max)
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return "", fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		// Encrypt chunk independently
		encryptedChunk, err := utils.EncryptChunk(&s.config, chunkData, i)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt chunk %d: %w", i, err)
		}

		// Write encrypted chunk to output file
		n, err := outFile.Write(encryptedChunk)
		if err != nil {
			return "", fmt.Errorf("failed to write encrypted chunk %d: %w", i, err)
		}

		totalBytes += int64(n)
		// Memory freed here - only held one chunk at a time
	}

	log.Printf("Streamed %d encrypted chunks (%d bytes) for session %s\n", totalChunks, totalBytes, sessionID)

	// Clean up chunks directory
	err = os.RemoveAll(chunksDir)
	if err != nil {
		log.Printf("Warning: failed to clean up chunks directory %s: %v\n", chunksDir, err)
	}

	return fullPath, nil
}

func (s *FilesystemStorage) AbortChunkedUpload(sessionID string) error {
	chunksDir := s.config.FilesystemPath + "/chunks/" + sessionID
	err := os.RemoveAll(chunksDir)
	if err != nil {
		log.Printf("Warning: failed to clean up chunks directory %s: %v\n", chunksDir, err)
		return err
	}
	log.Printf("Aborted chunked upload session %s\n", sessionID)
	return nil
}
