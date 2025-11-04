package utils

import (
	"fmt"
	"io"
	"log"

	"github.com/nuuner/bindle-server/internal/config"
)

// ChunkedDecryptionReader streams decryption of a chunked encrypted file
// It reads encrypted chunks one at a time, decrypts them, and returns decrypted data
// Memory usage: Only one chunk (~10MB) in memory at a time
type ChunkedDecryptionReader struct {
	source         io.ReadCloser  // Encrypted file reader
	config         *config.Config // Config for encryption key
	chunkCount     int            // Total number of chunks
	chunkSizeMB    int64          // Chunk size in MB
	currentChunk   int            // Current chunk being processed
	buffer         []byte         // Current decrypted chunk data
	bufferOffset   int            // Read position in current buffer
	closed         bool           // Whether the reader is closed
}

// NewChunkedDecryptionReader creates a streaming decryption reader for chunked files
func NewChunkedDecryptionReader(source io.ReadCloser, cfg *config.Config, chunkCount int) *ChunkedDecryptionReader {
	return &ChunkedDecryptionReader{
		source:       source,
		config:       cfg,
		chunkCount:   chunkCount,
		chunkSizeMB:  cfg.ChunkSizeMB,
		currentChunk: 0,
		buffer:       nil,
		bufferOffset: 0,
		closed:       false,
	}
}

// Read implements io.Reader interface
// Reads decrypted data progressively, one chunk at a time
func (r *ChunkedDecryptionReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}

	// If no buffer or buffer exhausted, load next chunk
	if r.buffer == nil || r.bufferOffset >= len(r.buffer) {
		if r.currentChunk >= r.chunkCount {
			// All chunks processed
			return 0, io.EOF
		}

		// Read and decrypt next chunk
		err := r.loadNextChunk()
		if err != nil {
			return 0, err
		}
	}

	// Copy from buffer to output
	bytesToCopy := len(p)
	bytesAvailable := len(r.buffer) - r.bufferOffset
	if bytesToCopy > bytesAvailable {
		bytesToCopy = bytesAvailable
	}

	copied := copy(p, r.buffer[r.bufferOffset:r.bufferOffset+bytesToCopy])
	r.bufferOffset += copied

	return copied, nil
}

// loadNextChunk reads one encrypted chunk from source, decrypts it, and stores in buffer
func (r *ChunkedDecryptionReader) loadNextChunk() error {
	log.Printf("Loading chunk %d/%d", r.currentChunk+1, r.chunkCount)

	// Calculate expected encrypted chunk size
	standardChunkSize := int(r.chunkSizeMB * 1024 * 1024)
	chunkOverhead := 8 + 12 + 16 // chunkNumber + nonce + authTag

	var encryptedChunkSize int
	if r.currentChunk < r.chunkCount-1 {
		// Not last chunk - standard size
		encryptedChunkSize = standardChunkSize + chunkOverhead
	} else {
		// Last chunk - read whatever remains
		// We don't know the exact size, so read up to standard size
		// The decryption will validate the actual size
		encryptedChunkSize = standardChunkSize + chunkOverhead
	}

	// Read encrypted chunk from source
	encryptedChunk := make([]byte, encryptedChunkSize)
	bytesRead := 0
	for bytesRead < encryptedChunkSize {
		n, err := r.source.Read(encryptedChunk[bytesRead:])
		bytesRead += n

		if err == io.EOF {
			// Reached end of file (expected for last chunk)
			encryptedChunk = encryptedChunk[:bytesRead]
			break
		} else if err != nil {
			return fmt.Errorf("failed to read encrypted chunk %d: %w", r.currentChunk, err)
		}
	}

	if bytesRead == 0 {
		return io.EOF
	}

	// Decrypt chunk
	decryptedChunk, err := DecryptChunk(r.config, encryptedChunk, r.currentChunk)
	if err != nil {
		// Log detailed error for debugging
		fmt.Printf("ERROR: Failed to decrypt chunk %d (size: %d bytes): %v\n", r.currentChunk, len(encryptedChunk), err)
		return fmt.Errorf("failed to decrypt chunk %d: %w", r.currentChunk, err)
	}

	// Store in buffer
	r.buffer = decryptedChunk
	r.bufferOffset = 0
	r.currentChunk++

	return nil
}

// Close implements io.Closer interface
func (r *ChunkedDecryptionReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.buffer = nil // Free buffer memory
	if r.source != nil {
		return r.source.Close()
	}
	return nil
}

// LegacyDecryptionReader streams decryption of legacy single-file uploads
// Note: Legacy files must be fully decrypted due to encryption format
// But we can still stream the decrypted result
type LegacyDecryptionReader struct {
	source         io.ReadCloser  // Encrypted file reader
	config         *config.Config // Config for encryption key
	decryptedData  []byte         // Decrypted file (loaded once)
	offset         int            // Read position
	closed         bool
	initialized    bool
}

// NewLegacyDecryptionReader creates a reader for legacy single-file uploads
func NewLegacyDecryptionReader(source io.ReadCloser, cfg *config.Config) *LegacyDecryptionReader {
	return &LegacyDecryptionReader{
		source:      source,
		config:      cfg,
		offset:      0,
		closed:      false,
		initialized: false,
	}
}

// Read implements io.Reader interface
func (r *LegacyDecryptionReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}

	// Lazy initialization - decrypt entire file on first read
	if !r.initialized {
		err := r.initialize()
		if err != nil {
			return 0, err
		}
	}

	// Check if EOF
	if r.offset >= len(r.decryptedData) {
		return 0, io.EOF
	}

	// Copy data to output
	bytesToCopy := len(p)
	bytesAvailable := len(r.decryptedData) - r.offset
	if bytesToCopy > bytesAvailable {
		bytesToCopy = bytesAvailable
	}

	copied := copy(p, r.decryptedData[r.offset:r.offset+bytesToCopy])
	r.offset += copied

	return copied, nil
}

// initialize reads and decrypts the entire legacy file
func (r *LegacyDecryptionReader) initialize() error {
	// Read entire encrypted file
	encryptedData, err := io.ReadAll(r.source)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Decrypt entire file using legacy decryption
	decryptedData, err := DecryptFile(r.config, encryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt legacy file: %w", err)
	}

	r.decryptedData = decryptedData
	r.initialized = true

	return nil
}

// Close implements io.Closer interface
func (r *LegacyDecryptionReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.decryptedData = nil // Free memory
	if r.source != nil {
		return r.source.Close()
	}
	return nil
}

// CalculateDecryptedSize calculates the decrypted file size from encrypted size
// For chunked files, removes encryption overhead per chunk
func CalculateDecryptedSize(encryptedSize int64, chunkCount int, chunkSizeMB int64) int64 {
	if chunkCount == 0 {
		// Legacy files - unknown without reading
		return encryptedSize // Approximate (will be wrong, but we'll send correct size)
	}

	// Encrypted overhead per chunk: 8 (chunkNumber) + 12 (nonce) + 16 (authTag) = 36 bytes
	chunkOverhead := int64(36)
	totalOverhead := int64(chunkCount) * chunkOverhead

	return encryptedSize - totalOverhead
}
