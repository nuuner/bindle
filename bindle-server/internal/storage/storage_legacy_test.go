package storage

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/nuuner/bindle-server/pkg/utils"
)

// Files already in the bucket were written before the streaming format, so the reader
// has to keep decoding both older layouts. These build objects the old way and read them
// back through the current code path.

func TestLegacyWholeFileStillDecrypts(t *testing.T) {
	st := newTestStorage(t)
	plain := testPayload(200000)

	sealed, err := utils.EncryptFile(&st.config, plain)
	if err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}
	if err := os.WriteFile(st.config.FilesystemPath+"/legacy.bin", sealed, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// EncryptionVersion 0 with no chunks is how the oldest uploads are recorded.
	reader, _, err := st.GetFileStream("legacy.bin", StoredFile{})
	if err != nil {
		t.Fatalf("GetFileStream: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Error("a legacy whole-file upload no longer reads back correctly")
	}
}

func TestVersionOneChunkedFileStillDecrypts(t *testing.T) {
	st := newTestStorage(t)

	chunkSize := int(st.config.ChunkSizeMB * 1024 * 1024)
	plain := testPayload(2*chunkSize + 1234)

	// Version 1 sealed each upload chunk whole and concatenated the results.
	var object bytes.Buffer
	chunkCount := 0
	for offset := 0; offset < len(plain); offset += chunkSize {
		end := offset + chunkSize
		if end > len(plain) {
			end = len(plain)
		}
		sealed, err := utils.EncryptChunk(&st.config, plain[offset:end], chunkCount)
		if err != nil {
			t.Fatalf("EncryptChunk: %v", err)
		}
		object.Write(sealed)
		chunkCount++
	}

	if err := os.WriteFile(st.config.FilesystemPath+"/v1.bin", object.Bytes(), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	reader, size, err := st.GetFileStream("v1.bin", StoredFile{ChunkCount: chunkCount})
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
		t.Error("a version 1 chunked upload no longer reads back correctly")
	}
}
