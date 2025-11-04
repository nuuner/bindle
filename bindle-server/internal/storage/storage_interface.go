package storage

import (
	"io"
	"mime/multipart"
)

type Storage interface {
	SaveFile(file *multipart.FileHeader, filePath string) (string, error)
	GetFile(filePath string, chunkCount int) ([]byte, error) // chunkCount=0 for legacy single-file uploads (deprecated - use GetFileStream)
	GetFileStream(filePath string, chunkCount int) (io.ReadCloser, int64, error) // Returns: reader, decryptedSize, error
	DeleteFile(filePath string) error

	// Chunked upload methods
	InitChunkedUpload(sessionID string, fileName string, totalChunks int) error
	SaveChunk(sessionID string, chunkNumber int, chunkData []byte) error
	FinalizeChunkedUpload(sessionID string, filePath string) (string, error)
	AbortChunkedUpload(sessionID string) error
}
