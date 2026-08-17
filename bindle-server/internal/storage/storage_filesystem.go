package storage

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"sync"

	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

// fsUpload is the in-flight state of one chunked upload.
type fsUpload struct {
	// file is the destination, opened once and written at computed offsets. Chunks
	// used to be spooled to a temp directory and then read back, encrypted and written
	// out again after the last one arrived, which wrote every byte twice and left a
	// full pass over the file happening while the client waited on 100%.
	file *os.File
	// tempPath is where that destination lives until every chunk is in, so an
	// abandoned session never leaves a half-written file at the real path.
	tempPath    string
	finalPath   string
	totalChunks int
	chunkSize   int64
	// received records chunk indexes rather than counting them: chunks arrive
	// concurrently and may be retried, and a retry must not count twice.
	received map[int]bool
	mu       sync.Mutex
}

type FilesystemStorage struct {
	config       config.Config
	uploads      map[string]*fsUpload
	uploadsMutex sync.RWMutex
}

func NewFilesystemStorage(config config.Config) (*FilesystemStorage, error) {
	return &FilesystemStorage{
		config:  config,
		uploads: make(map[string]*fsUpload),
	}, nil
}

func (s *FilesystemStorage) SaveFile(file *multipart.FileHeader, filePath string) (string, error) {
	if err := utils.EnsureFileDirectory(s.config); err != nil {
		return "", err
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	encrypted, err := utils.NewEncryptingReader(src, s.config.EncryptionKey, file.Size, 0)
	if err != nil {
		return "", err
	}

	fullPath := s.config.FilesystemPath + "/" + filePath
	out, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, encrypted); err != nil {
		return "", err
	}

	return fullPath, nil
}

// GetFileStream returns a streaming reader over the decrypted file, holding at most one
// frame in memory regardless of the file's size.
func (s *FilesystemStorage) GetFileStream(filePath string, file StoredFile) (io.ReadCloser, int64, error) {
	fullPath := s.config.FilesystemPath + "/" + filePath
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}

	reader, size, err := decryptStream(f, stat.Size(), &s.config, file)
	if err != nil {
		f.Close()
		return nil, 0, err
	}

	log.Printf("Streaming file %s (%d bytes)\n", filePath, size)
	return reader, size, nil
}

func (s *FilesystemStorage) DeleteFile(filePath string) error {
	fullPath := s.config.FilesystemPath + "/" + filePath
	return os.Remove(fullPath)
}

// Chunked upload

func (s *FilesystemStorage) InitChunkedUpload(sessionID string, filePath string, totalChunks int, chunkSize int64) error {
	if err := utils.EnsureFileDirectory(s.config); err != nil {
		return err
	}

	finalPath := s.config.FilesystemPath + "/" + filePath
	tempPath := finalPath + ".part"

	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	s.uploadsMutex.Lock()
	s.uploads[sessionID] = &fsUpload{
		file:        file,
		tempPath:    tempPath,
		finalPath:   finalPath,
		totalChunks: totalChunks,
		chunkSize:   chunkSize,
		received:    make(map[int]bool, totalChunks),
	}
	s.uploadsMutex.Unlock()

	log.Printf("Initialized chunked upload session %s at %s (%d chunks)\n", sessionID, tempPath, totalChunks)
	return nil
}

func (s *FilesystemStorage) SaveChunk(sessionID string, chunkNumber int, r io.Reader, plainSize int64) error {
	s.uploadsMutex.RLock()
	upload, exists := s.uploads[sessionID]
	s.uploadsMutex.RUnlock()
	if !exists {
		return fmt.Errorf("upload session %s not found", sessionID)
	}

	// Every chunk but the last is full, so a chunk's encrypted length - and therefore
	// where it belongs in the file - follows from its index alone. Writing each chunk
	// straight to its final offset is what removes the assembly pass at the end;
	// concurrent chunks land at disjoint offsets, which WriteAt handles directly.
	offset := int64(chunkNumber) * utils.EncryptedSize(upload.chunkSize)

	encrypted, err := utils.NewEncryptingReader(
		r, s.config.EncryptionKey, plainSize, int64(chunkNumber)*utils.FramesPerChunk(upload.chunkSize))
	if err != nil {
		return err
	}

	if _, err := io.Copy(io.NewOffsetWriter(upload.file, offset), encrypted); err != nil {
		return fmt.Errorf("failed to write chunk %d: %w", chunkNumber, err)
	}

	upload.mu.Lock()
	upload.received[chunkNumber] = true
	upload.mu.Unlock()

	return nil
}

func (s *FilesystemStorage) FinalizeChunkedUpload(sessionID string) (string, error) {
	s.uploadsMutex.RLock()
	upload, exists := s.uploads[sessionID]
	s.uploadsMutex.RUnlock()
	if !exists {
		return "", fmt.Errorf("upload session %s not found", sessionID)
	}

	upload.mu.Lock()
	missing := upload.totalChunks - len(upload.received)
	upload.mu.Unlock()

	if missing > 0 {
		return "", fmt.Errorf("%w: have %d of %d",
			ErrIncompleteUpload, upload.totalChunks-missing, upload.totalChunks)
	}

	if err := upload.file.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(upload.tempPath, upload.finalPath); err != nil {
		return "", err
	}

	s.uploadsMutex.Lock()
	delete(s.uploads, sessionID)
	s.uploadsMutex.Unlock()

	log.Printf("Finalized chunked upload session %s at %s\n", sessionID, upload.finalPath)
	return upload.finalPath, nil
}

func (s *FilesystemStorage) AbortChunkedUpload(sessionID string) error {
	s.uploadsMutex.Lock()
	upload, exists := s.uploads[sessionID]
	if exists {
		delete(s.uploads, sessionID)
	}
	s.uploadsMutex.Unlock()

	if !exists {
		log.Printf("Upload session %s not found for abort\n", sessionID)
		return nil
	}

	upload.file.Close()
	if err := os.Remove(upload.tempPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove %s: %v\n", upload.tempPath, err)
		return err
	}

	log.Printf("Aborted chunked upload session %s\n", sessionID)
	return nil
}
