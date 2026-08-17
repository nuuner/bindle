package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Streaming AEAD - encryption version 2.
//
// Version 1 sealed a whole upload chunk (10 MB by default) in one AES-GCM call, so an
// upload had to buffer the entire chunk before a byte could be written on, and a
// download had to buffer it again before a byte could be sent. Version 2 splits the
// plaintext into fixed-size frames sealed independently, which lets both directions run
// as io.Reader pipelines holding a single frame at a time, whatever the file size.
//
// Frame layout: [nonce(12)][ciphertext(<=FrameSize)][tag(16)]
//
// The frame grid is absolute over the plaintext - frame i covers bytes
// [i*FrameSize, min((i+1)*FrameSize, size)) - so a reader that knows only the plaintext
// length reproduces exactly the boundaries the writer used, with nothing recorded on
// disk. Chunked uploads keep that grid intact because ChunkSize is a whole number of
// megabytes and FrameSize divides it evenly, so every chunk but the last seals only
// full frames and the next chunk resumes on a frame boundary.
//
// Each frame is authenticated against its absolute index, so frames cannot be reordered
// within a file or spliced in from another one, and a reader stops at the declared
// plaintext length, so a truncated object fails instead of decoding short.
const (
	// FrameSize is the plaintext covered by one frame. It sets the buffer both
	// directions hold: large enough that the per-frame overhead and the AEAD call
	// cost round to nothing, small enough that concurrent transfers stay cheap.
	FrameSize = 256 * 1024

	frameNonceSize = 12
	frameTagSize   = 16

	// FrameOverhead is what each frame adds on top of its plaintext.
	FrameOverhead = frameNonceSize + frameTagSize

	// EncryptionVersionStream marks a file stored in the framed format described above.
	// Files written before it carry version 0 and are read by the v1 paths.
	EncryptionVersionStream = 2
)

// ErrShortSource reports a source that ended before the plaintext length it declared.
// Chunk uploads declare their length up front so the encrypted length can be computed
// before any byte is read, which means a client that sends less has to be rejected
// rather than silently stored.
var ErrShortSource = errors.New("source ended before the declared length")

// EncryptedSize returns how many bytes plainSize occupies once framed and sealed.
// Callers need this before reading anything: S3 wants the part length up front, and the
// filesystem backend places each chunk at a computed offset.
func EncryptedSize(plainSize int64) int64 {
	if plainSize <= 0 {
		return 0
	}
	frames := (plainSize + FrameSize - 1) / FrameSize
	return plainSize + frames*FrameOverhead
}

// FramesPerChunk returns how many frames a full chunk seals, which is also the stride
// between the absolute frame indexes two consecutive chunks start at.
func FramesPerChunk(chunkSize int64) int64 {
	return chunkSize / FrameSize
}

func newFrameGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func frameAAD(frameIndex int64) []byte {
	aad := make([]byte, 8)
	binary.BigEndian.PutUint64(aad, uint64(frameIndex))
	return aad
}

// encryptingReader seals src frame by frame as the consumer pulls.
type encryptingReader struct {
	src        io.Reader
	gcm        cipher.AEAD
	buf        []byte // holds the frame being emitted: [nonce][ciphertext][tag]
	out        []byte // the part of buf not yet handed to the caller
	remaining  int64  // plaintext bytes still to seal
	frameIndex int64  // absolute index of the next frame
	err        error
}

// NewEncryptingReader wraps src so that reading it yields the sealed form of the next
// plainSize bytes. firstFrameIndex is where this reader sits in the file's frame grid:
// 0 for a whole file, chunkNumber*FramesPerChunk(chunkSize) for one chunk of a chunked
// upload.
func NewEncryptingReader(src io.Reader, key []byte, plainSize, firstFrameIndex int64) (io.Reader, error) {
	gcm, err := newFrameGCM(key)
	if err != nil {
		return nil, err
	}
	return &encryptingReader{
		src:        src,
		gcm:        gcm,
		buf:        make([]byte, frameNonceSize+FrameSize+frameTagSize),
		remaining:  plainSize,
		frameIndex: firstFrameIndex,
	}, nil
}

func (r *encryptingReader) Read(p []byte) (int, error) {
	if len(r.out) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		if err := r.sealNextFrame(); err != nil {
			r.err = err
			return 0, err
		}
	}

	n := copy(p, r.out)
	r.out = r.out[n:]
	return n, nil
}

func (r *encryptingReader) sealNextFrame() error {
	if r.remaining == 0 {
		return io.EOF
	}

	n := int64(FrameSize)
	if r.remaining < n {
		n = r.remaining
	}

	// Seal in place: dst is plaintext[:0], which cipher.AEAD permits and which keeps
	// the whole frame in the single buffer allocated for this reader.
	plain := r.buf[frameNonceSize : frameNonceSize+n]
	if _, err := io.ReadFull(r.src, plain); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return fmt.Errorf("%w: %d bytes missing", ErrShortSource, r.remaining)
		}
		return err
	}

	nonce := r.buf[:frameNonceSize]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	sealed := r.gcm.Seal(plain[:0], nonce, plain, frameAAD(r.frameIndex))
	r.out = r.buf[:frameNonceSize+len(sealed)]
	r.remaining -= n
	r.frameIndex++
	return nil
}

// decryptingReader opens a framed object frame by frame as the consumer pulls.
type decryptingReader struct {
	src        io.ReadCloser
	gcm        cipher.AEAD
	buf        []byte
	out        []byte
	remaining  int64
	frameIndex int64
	err        error
	closed     bool
}

// NewDecryptingReader wraps a framed object so that reading it yields plainSize bytes of
// plaintext. plainSize is what pins the frame boundaries, so it must be the size
// recorded when the file was stored.
func NewDecryptingReader(src io.ReadCloser, key []byte, plainSize int64) (io.ReadCloser, error) {
	gcm, err := newFrameGCM(key)
	if err != nil {
		return nil, err
	}
	return &decryptingReader{
		src:       src,
		gcm:       gcm,
		buf:       make([]byte, frameNonceSize+FrameSize+frameTagSize),
		remaining: plainSize,
	}, nil
}

func (r *decryptingReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}

	if len(r.out) == 0 {
		if r.err != nil {
			return 0, r.err
		}
		if err := r.openNextFrame(); err != nil {
			r.err = err
			return 0, err
		}
	}

	n := copy(p, r.out)
	r.out = r.out[n:]
	return n, nil
}

func (r *decryptingReader) openNextFrame() error {
	if r.remaining == 0 {
		return io.EOF
	}

	n := int64(FrameSize)
	if r.remaining < n {
		n = r.remaining
	}

	frame := r.buf[:frameNonceSize+n+frameTagSize]
	if _, err := io.ReadFull(r.src, frame); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return fmt.Errorf("frame %d is truncated: %w", r.frameIndex, err)
		}
		return err
	}

	nonce, sealed := frame[:frameNonceSize], frame[frameNonceSize:]
	plain, err := r.gcm.Open(sealed[:0], nonce, sealed, frameAAD(r.frameIndex))
	if err != nil {
		return fmt.Errorf("failed to decrypt frame %d: %w", r.frameIndex, err)
	}

	r.out = plain
	r.remaining -= n
	r.frameIndex++
	return nil
}

func (r *decryptingReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.out = nil
	if r.src != nil {
		return r.src.Close()
	}
	return nil
}
