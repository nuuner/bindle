package utils

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var testKey = bytes.Repeat([]byte{0x2b}, 32)

func randomish(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i*31 + i/251)
	}
	return data
}

func sealAll(t *testing.T, plain []byte, firstFrameIndex int64) []byte {
	t.Helper()
	r, err := NewEncryptingReader(bytes.NewReader(plain), testKey, int64(len(plain)), firstFrameIndex)
	if err != nil {
		t.Fatalf("NewEncryptingReader: %v", err)
	}
	sealed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read sealed: %v", err)
	}
	return sealed
}

func openAll(t *testing.T, sealed []byte, plainSize int64) ([]byte, error) {
	t.Helper()
	r, err := NewDecryptingReader(io.NopCloser(bytes.NewReader(sealed)), testKey, plainSize)
	if err != nil {
		t.Fatalf("NewDecryptingReader: %v", err)
	}
	defer r.Close()
	return io.ReadAll(r)
}

func TestRoundTrip(t *testing.T) {
	sizes := []int{
		1,
		FrameSize - 1,
		FrameSize,
		FrameSize + 1,
		3*FrameSize + 17,
	}

	for _, size := range sizes {
		plain := randomish(size)
		sealed := sealAll(t, plain, 0)

		if int64(len(sealed)) != EncryptedSize(int64(size)) {
			t.Errorf("size %d sealed to %d bytes, EncryptedSize says %d",
				size, len(sealed), EncryptedSize(int64(size)))
		}

		got, err := openAll(t, sealed, int64(size))
		if err != nil {
			t.Fatalf("size %d: decrypt failed: %v", size, err)
		}
		if !bytes.Equal(got, plain) {
			t.Errorf("size %d: round trip changed the data", size)
		}
	}
}

// A chunked upload seals each chunk on its own, and the frames land in the object back
// to back. The result has to read as one continuous stream, or every multi-chunk file
// is corrupt.
func TestChunkedSealReadsAsOneStream(t *testing.T) {
	const chunkSize = int64(2 * FrameSize)

	// Two full chunks and a partial one, with the last chunk ending mid-frame.
	plain := randomish(int(2*chunkSize) + FrameSize + 123)

	var object bytes.Buffer
	for offset, chunkNumber := int64(0), 0; offset < int64(len(plain)); chunkNumber++ {
		end := offset + chunkSize
		if end > int64(len(plain)) {
			end = int64(len(plain))
		}
		object.Write(sealAll(t, plain[offset:end], int64(chunkNumber)*FramesPerChunk(chunkSize)))
		offset = end
	}

	got, err := openAll(t, object.Bytes(), int64(len(plain)))
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Error("chunked seal did not read back as the original file")
	}
}

// Each chunk's part is placed at an offset computed from its index alone, so the
// encrypted length of a full chunk has to be exactly what every chunk before the last
// occupies.
func TestChunkedSealSizesArePredictable(t *testing.T) {
	const chunkSize = int64(4 * FrameSize)

	sealed := sealAll(t, randomish(int(chunkSize)), 0)
	if int64(len(sealed)) != EncryptedSize(chunkSize) {
		t.Errorf("full chunk sealed to %d bytes, offsets assume %d", len(sealed), EncryptedSize(chunkSize))
	}
}

func TestTamperedFrameIsRejected(t *testing.T) {
	plain := randomish(FrameSize + 500)
	sealed := sealAll(t, plain, 0)

	tampered := bytes.Clone(sealed)
	tampered[frameNonceSize+10] ^= 0xff

	if _, err := openAll(t, tampered, int64(len(plain))); err == nil {
		t.Error("a modified ciphertext byte decrypted without error")
	}
}

// Frames are bound to their absolute index, so a frame moved within a file - or spliced
// in from another one - must not open.
func TestReorderedFramesAreRejected(t *testing.T) {
	plain := randomish(2 * FrameSize)
	sealed := sealAll(t, plain, 0)

	frameLen := FrameSize + FrameOverhead
	swapped := make([]byte, 0, len(sealed))
	swapped = append(swapped, sealed[frameLen:]...)
	swapped = append(swapped, sealed[:frameLen]...)

	if _, err := openAll(t, swapped, int64(len(plain))); err == nil {
		t.Error("swapping two frames decrypted without error")
	}
}

func TestTruncatedObjectIsRejected(t *testing.T) {
	plain := randomish(2*FrameSize + 40)
	sealed := sealAll(t, plain, 0)

	if _, err := openAll(t, sealed[:len(sealed)-100], int64(len(plain))); err == nil {
		t.Error("a truncated object decrypted without error")
	}
}

// A chunk declares its length before its body is read, so a body that ends early has to
// fail rather than be stored short.
func TestShortSourceIsReported(t *testing.T) {
	r, err := NewEncryptingReader(bytes.NewReader(randomish(100)), testKey, 500, 0)
	if err != nil {
		t.Fatalf("NewEncryptingReader: %v", err)
	}

	if _, err := io.ReadAll(r); !errors.Is(err, ErrShortSource) {
		t.Errorf("reading a short source gave %v, want ErrShortSource", err)
	}
}

// The reader is pulled with whatever buffer the consumer happens to use - the S3 SDK and
// fasthttp both pick their own - so it has to be correct for any read size.
func TestReadsSurviveSmallBuffers(t *testing.T) {
	plain := randomish(FrameSize + 777)
	sealed := sealAll(t, plain, 0)

	r, err := NewDecryptingReader(io.NopCloser(bytes.NewReader(sealed)), testKey, int64(len(plain)))
	if err != nil {
		t.Fatalf("NewDecryptingReader: %v", err)
	}
	defer r.Close()

	var got bytes.Buffer
	buf := make([]byte, 7)
	for {
		n, err := r.Read(buf)
		got.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read: %v", err)
		}
	}

	if !bytes.Equal(got.Bytes(), plain) {
		t.Error("reading in small pieces did not reproduce the file")
	}
}

func TestFrameSizeDividesEveryChunkSize(t *testing.T) {
	// Chunk size is configured in whole megabytes, and the frame grid only stays
	// aligned across chunk boundaries if a chunk is a whole number of frames.
	for _, chunkSizeMB := range []int64{1, 5, 8, 10, 16, 64} {
		chunkSize := chunkSizeMB * 1024 * 1024
		if chunkSize%FrameSize != 0 {
			t.Errorf("a %d MB chunk is not a whole number of frames", chunkSizeMB)
		}
		if FramesPerChunk(chunkSize) == 0 {
			t.Errorf("a %d MB chunk holds no whole frame", chunkSizeMB)
		}
	}
}
