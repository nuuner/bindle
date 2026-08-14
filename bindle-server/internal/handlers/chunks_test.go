package handlers

import (
	"testing"

	"github.com/nuuner/bindle-server/internal/models"
)

func TestMaxChunkBytes(t *testing.T) {
	const chunkSize = 10

	tests := []struct {
		name        string
		fileSize    int64
		chunkNumber int
		want        int64
	}{
		// A file of 25 bytes at a chunk size of 10 is three chunks: 10, 10, 5.
		{"first full chunk", 25, 0, 10},
		{"middle full chunk", 25, 1, 10},
		{"final partial chunk", 25, 2, 5},
		{"exact multiple fills its last chunk", 20, 1, 10},
		{"single chunk smaller than the chunk size", 3, 0, 3},

		// The bypass: declaring a tiny file must not buy the right to upload a full
		// chunk, and indexes past the declared end must buy nothing at all.
		{"one byte file allows one byte", 1, 0, 1},
		{"index past the declared end", 25, 3, 0},
		{"index far past the declared end", 1, 99, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.UploadSession{FileSize: tt.fileSize, ChunkSize: chunkSize}
			if got := maxChunkBytes(session, tt.chunkNumber); got != tt.want {
				t.Errorf("maxChunkBytes(fileSize=%d, chunk=%d) = %d, want %d",
					tt.fileSize, tt.chunkNumber, got, tt.want)
			}
		})
	}
}

// The bounds must sum to exactly the declared size, so an honest client is never
// rejected and a dishonest one can never exceed what it reserved.
func TestMaxChunkBytesSumsToDeclaredSize(t *testing.T) {
	const chunkSize = 10

	for _, fileSize := range []int64{1, 9, 10, 11, 25, 100, 101} {
		session := &models.UploadSession{FileSize: fileSize, ChunkSize: chunkSize}
		totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

		var total int64
		for i := 0; i < totalChunks; i++ {
			total += maxChunkBytes(session, i)
		}

		if total != fileSize {
			t.Errorf("chunk bounds for a %d byte file sum to %d, want %d", fileSize, total, fileSize)
		}
	}
}
