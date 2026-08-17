package storage

import (
	"errors"
	"io"
	"mime/multipart"
)

// ErrIncompleteUpload reports a finalize attempt on a session that is missing chunks.
// Completeness is answered by the backend holding the chunks rather than by a counter in
// the database: chunks are uploaded concurrently and may be retried, so only the store
// that actually holds them knows which indexes are present.
var ErrIncompleteUpload = errors.New("upload is missing chunks")

// StoredFile describes how an object was encrypted so a reader can be built for it.
type StoredFile struct {
	// EncryptionVersion is utils.EncryptionVersionStream for the framed streaming
	// format and 0 for everything written before it.
	EncryptionVersion int
	// ChunkCount is meaningful only at version 0: 0 means the whole file was sealed in
	// one call, greater than 0 means it was sealed one upload chunk at a time.
	ChunkCount int
	// PlainSize is the decrypted length. The framed format needs it to place frame
	// boundaries, and it is the length served to the client.
	PlainSize int64
}

type Storage interface {
	SaveFile(file *multipart.FileHeader, filePath string) (string, error)
	// GetFileStream returns a reader over the decrypted file and its plaintext length.
	// Nothing larger than one frame is held in memory at any point.
	GetFileStream(filePath string, file StoredFile) (io.ReadCloser, int64, error)
	DeleteFile(filePath string) error

	// Chunked upload. The session is opened against its final destination up front so
	// that no byte has to be moved, copied or re-encrypted once the last chunk lands.
	InitChunkedUpload(sessionID string, filePath string, totalChunks int, chunkSize int64) error
	// SaveChunk encrypts exactly plainSize bytes from r and stores them as chunk
	// chunkNumber. r is the request body, so the chunk is never held whole in memory.
	SaveChunk(sessionID string, chunkNumber int, r io.Reader, plainSize int64) error
	// FinalizeChunkedUpload publishes the session at the path it was opened with, and
	// returns ErrIncompleteUpload if any chunk is missing.
	FinalizeChunkedUpload(sessionID string) (string, error)
	AbortChunkedUpload(sessionID string) error
}
