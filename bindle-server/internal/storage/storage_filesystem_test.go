package storage

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

const testChunkSizeMB = 1

func newTestStorage(t *testing.T) *FilesystemStorage {
	t.Helper()
	st, err := NewFilesystemStorage(config.Config{
		FilesystemPath: t.TempDir(),
		ChunkSizeMB:    testChunkSizeMB,
		EncryptionKey:  bytes.Repeat([]byte{0x7f}, 32),
	})
	if err != nil {
		t.Fatalf("NewFilesystemStorage: %v", err)
	}
	return st
}

func testPayload(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i*17 + i/97)
	}
	return data
}

// Chunks are written straight to their final offsets, so the offsets are the only thing
// holding the file together - and they are computed from the chunk index rather than
// from arrival order. Uploading them out of order and concurrently is what the client
// now does, and it has to produce the same file.
func TestChunksUploadedConcurrentlyReassemble(t *testing.T) {
	st := newTestStorage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	plain := testPayload(int(2*chunkSize) + 4096) // two full chunks and a partial one
	totalChunks := 3
	const path = "concurrent.bin"

	if err := st.InitChunkedUpload("session", path, totalChunks, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, totalChunks)
	for i := 0; i < totalChunks; i++ {
		wg.Add(1)
		go func(chunkNumber int) {
			defer wg.Done()
			start := int64(chunkNumber) * chunkSize
			end := start + chunkSize
			if end > int64(len(plain)) {
				end = int64(len(plain))
			}
			slice := plain[start:end]
			errs[chunkNumber] = st.SaveChunk("session", chunkNumber, bytes.NewReader(slice), int64(len(slice)))
		}(totalChunks - 1 - i) // last chunk first
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
	}

	if _, err := st.FinalizeChunkedUpload("session"); err != nil {
		t.Fatalf("FinalizeChunkedUpload: %v", err)
	}

	reader, size, err := st.GetFileStream(path, StoredFile{
		EncryptionVersion: utils.EncryptionVersionStream,
		PlainSize:         int64(len(plain)),
	})
	if err != nil {
		t.Fatalf("GetFileStream: %v", err)
	}
	defer reader.Close()

	if size != int64(len(plain)) {
		t.Errorf("stream reports %d bytes, file is %d", size, len(plain))
	}

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Error("the file read back differs from what was uploaded")
	}
}

// A retried chunk must not be mistaken for a second one, or a session with a hole in it
// would finalize as complete.
func TestRetriedChunkDoesNotCountTwice(t *testing.T) {
	st := newTestStorage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	chunk := testPayload(int(chunkSize))

	if err := st.InitChunkedUpload("session", "retried.bin", 2, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := st.SaveChunk("session", 0, bytes.NewReader(chunk), chunkSize); err != nil {
			t.Fatalf("SaveChunk: %v", err)
		}
	}

	if _, err := st.FinalizeChunkedUpload("session"); !errors.Is(err, ErrIncompleteUpload) {
		t.Errorf("finalizing with chunk 1 missing gave %v, want ErrIncompleteUpload", err)
	}
}

// An abandoned session must not leave anything readable at the destination.
func TestAbortLeavesNoFile(t *testing.T) {
	st := newTestStorage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	const path = "aborted.bin"

	if err := st.InitChunkedUpload("session", path, 2, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}
	if err := st.SaveChunk("session", 0, bytes.NewReader(testPayload(int(chunkSize))), chunkSize); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if err := st.AbortChunkedUpload("session"); err != nil {
		t.Fatalf("AbortChunkedUpload: %v", err)
	}

	if _, _, err := st.GetFileStream(path, StoredFile{
		EncryptionVersion: utils.EncryptionVersionStream,
		PlainSize:         chunkSize,
	}); err == nil {
		t.Error("the destination is readable after the upload was aborted")
	}
}
