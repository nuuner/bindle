package storage

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

// fakeS3 implements just enough of the S3 multipart API to drive the real AWS SDK, which
// is what these tests are really exercising: prod runs on S3, so the part upload has to
// be verified against the SDK's own request handling rather than a stub of our own.
type fakeS3 struct {
	mu sync.Mutex
	// parts is uploadID -> part number -> body, in the order the parts were declared
	// complete rather than the order they were uploaded.
	parts map[string]map[int32][]byte
	// completeOrder records the part numbers in the order CompleteMultipartUpload
	// listed them, which S3 requires to be ascending.
	completeOrder []int32
	// partContentLengths records the Content-Length each part arrived with, to prove
	// the streaming body declared its length instead of falling back to chunked
	// transfer encoding, which S3 rejects.
	partContentLengths map[int32]int64
	// partHeaders records what a part upload actually put on the wire. The bucket is
	// Backblaze B2, whose S3 API rejects aws-chunked bodies and trailing checksums, so
	// "the SDK streamed it as a plain body" is a property worth asserting rather than
	// assuming.
	partHeaders  map[int32]http.Header
	partEncoding map[int32][]string
	objects      map[string][]byte
	copies       int
	aborted      []string
	// failFirstAttempt makes every part fail once with the transient 500 the real
	// bucket returns, so the recovery path can be exercised.
	failFirstAttempt bool
	attempts         map[int32]int
}

func newFakeS3() *fakeS3 {
	return &fakeS3{
		parts:              make(map[string]map[int32][]byte),
		partContentLengths: make(map[int32]int64),
		partHeaders:        make(map[int32]http.Header),
		partEncoding:       make(map[int32][]string),
		attempts:           make(map[int32]int),
		objects:            make(map[string][]byte),
	}
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := r.URL.Path
	if len(key) > 0 && key[0] == '/' {
		key = key[1:]
	}
	// Path-style addressing: /{bucket}/{key}
	if i := len(testBucket) + 1; len(key) >= i {
		key = key[i:]
	}

	q := r.URL.Query()
	uploadID := q.Get("uploadId")

	switch {
	case r.Method == http.MethodPost && q.Has("uploads"):
		f.parts["upload-1"] = make(map[int32][]byte)
		writeXML(w, fmt.Sprintf(
			`<InitiateMultipartUploadResult><Bucket>%s</Bucket><Key>%s</Key><UploadId>upload-1</UploadId></InitiateMultipartUploadResult>`,
			testBucket, key))

	case r.Method == http.MethodPut && uploadID != "":
		partNumber, _ := strconv.Atoi(q.Get("partNumber"))
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.attempts[int32(partNumber)]++
		if f.failFirstAttempt && f.attempts[int32(partNumber)] == 1 {
			// What B2 actually returns: the whole body is read, then a 500.
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, xml.Header+`<Error><Code>InternalError</Code><Message>internal incident</Message></Error>`)
			return
		}
		f.parts[uploadID][int32(partNumber)] = body
		f.partContentLengths[int32(partNumber)] = r.ContentLength
		f.partHeaders[int32(partNumber)] = r.Header.Clone()
		f.partEncoding[int32(partNumber)] = r.TransferEncoding
		w.Header().Set("ETag", fmt.Sprintf(`"etag-%d"`, partNumber))
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodPost && uploadID != "":
		var complete struct {
			Parts []struct {
				PartNumber int32 `xml:"PartNumber"`
			} `xml:"Part"`
		}
		body, _ := io.ReadAll(r.Body)
		if err := xml.Unmarshal(body, &complete); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var assembled []byte
		for _, part := range complete.Parts {
			f.completeOrder = append(f.completeOrder, part.PartNumber)
			assembled = append(assembled, f.parts[uploadID][part.PartNumber]...)
		}
		f.objects[key] = assembled
		writeXML(w, fmt.Sprintf(
			`<CompleteMultipartUploadResult><Bucket>%s</Bucket><Key>%s</Key><ETag>"final"</ETag></CompleteMultipartUploadResult>`,
			testBucket, key))

	case r.Method == http.MethodDelete && uploadID != "":
		f.aborted = append(f.aborted, uploadID)
		w.WriteHeader(http.StatusNoContent)

	case r.Method == http.MethodPut && r.Header.Get("x-amz-copy-source") != "":
		f.copies++
		writeXML(w, `<CopyObjectResult><ETag>"copied"</ETag></CopyObjectResult>`)

	case r.Method == http.MethodGet:
		object, ok := f.objects[key]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(object)))
		w.WriteHeader(http.StatusOK)
		w.Write(object)

	default:
		http.Error(w, "unexpected request "+r.Method+" "+r.URL.String(), http.StatusBadRequest)
	}
}

func writeXML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, xml.Header+body)
}

const testBucket = "bindle-test"

func newFakeS3Storage(t *testing.T) (*S3Storage, *fakeS3) {
	t.Helper()

	fake := newFakeS3()
	// TLS, because that is what an S3 endpoint is in production and it is what decides
	// how the SDK signs a body it cannot rewind.
	server := httptest.NewTLSServer(fake)
	t.Cleanup(server.Close)

	cfg := config.Config{
		ChunkSizeMB:   testChunkSizeMB,
		EncryptionKey: bytes.Repeat([]byte{0x51}, 32),
		S3Bucket:      testBucket,
	}

	client := s3.NewFromConfig(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
		HTTPClient:  server.Client(),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(server.URL)
		o.UsePathStyle = true
	})

	return &S3Storage{
		client:  client,
		bucket:  testBucket,
		config:  cfg,
		uploads: make(map[string]*s3Upload),
	}, fake
}

// The whole S3 path in one pass: parts stream out of the request body with a declared
// length, arrive out of order, get listed in ascending order at completion, land at the
// final key without a copy, and read back byte for byte.
func TestS3ChunkedUploadRoundTrip(t *testing.T) {
	st, fake := newFakeS3Storage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	plain := testPayload(int(2*chunkSize) + 5000)
	totalChunks := 3
	const path = "abc123.bin"

	if err := st.InitChunkedUpload("session", path, totalChunks, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}

	// Last chunk first, the way the client's worker pool can finish them.
	for i := totalChunks - 1; i >= 0; i-- {
		start := int64(i) * chunkSize
		end := start + chunkSize
		if end > int64(len(plain)) {
			end = int64(len(plain))
		}
		slice := plain[start:end]
		if err := st.SaveChunk("session", i, bytes.NewReader(slice), int64(len(slice))); err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
	}

	// A part sent without a declared length would arrive chunked, which S3 refuses.
	for i := 0; i < totalChunks; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize
		if end > int64(len(plain)) {
			end = int64(len(plain))
		}
		want := utils.EncryptedSize(end - start)
		if got := fake.partContentLengths[int32(i+1)]; got != want {
			t.Errorf("part %d arrived with Content-Length %d, want %d", i+1, got, want)
		}
	}

	// B2 refuses aws-chunked bodies, trailing checksums and chunked transfer encoding.
	for partNumber, header := range fake.partHeaders {
		if encoding := header.Get("Content-Encoding"); encoding == "aws-chunked" {
			t.Errorf("part %d was sent with Content-Encoding %q", partNumber, encoding)
		}
		if trailer := header.Get("X-Amz-Trailer"); trailer != "" {
			t.Errorf("part %d was sent with a trailing checksum (%s)", partNumber, trailer)
		}
		if len(fake.partEncoding[partNumber]) != 0 {
			t.Errorf("part %d was sent with transfer encoding %v", partNumber, fake.partEncoding[partNumber])
		}
	}

	if _, err := st.FinalizeChunkedUpload("session"); err != nil {
		t.Fatalf("FinalizeChunkedUpload: %v", err)
	}

	if !sort.SliceIsSorted(fake.completeOrder, func(i, j int) bool {
		return fake.completeOrder[i] < fake.completeOrder[j]
	}) {
		t.Errorf("parts were completed out of order: %v", fake.completeOrder)
	}

	if fake.copies != 0 {
		t.Errorf("the object was copied %d times; it should be assembled at its final key", fake.copies)
	}
	if _, ok := fake.objects[path]; !ok {
		t.Fatalf("nothing was written at %s; keys present: %v", path, keysOf(fake.objects))
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
		t.Error("the object read back differs from what was uploaded")
	}
}

func TestS3FinalizeRefusesMissingParts(t *testing.T) {
	st, _ := newFakeS3Storage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	if err := st.InitChunkedUpload("session", "hole.bin", 2, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}
	if err := st.SaveChunk("session", 0, bytes.NewReader(testPayload(int(chunkSize))), chunkSize); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}

	if _, err := st.FinalizeChunkedUpload("session"); err == nil {
		t.Error("finalizing with a part missing succeeded")
	}
}

// A retried chunk replaces its part rather than adding a second one, which would
// otherwise make an incomplete upload look complete and corrupt the assembled object.
func TestS3RetriedChunkReplacesItsPart(t *testing.T) {
	st, fake := newFakeS3Storage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	chunk := testPayload(int(chunkSize))

	if err := st.InitChunkedUpload("session", "retried.bin", 2, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := st.SaveChunk("session", 0, bytes.NewReader(chunk), chunkSize); err != nil {
			t.Fatalf("SaveChunk: %v", err)
		}
	}

	if _, err := st.FinalizeChunkedUpload("session"); err == nil {
		t.Fatal("finalizing with chunk 1 missing succeeded")
	}

	st.uploadsMutex.RLock()
	parts := len(st.uploads["session"].parts)
	st.uploadsMutex.RUnlock()
	if parts != 1 {
		t.Errorf("three uploads of the same chunk produced %d parts, want 1", parts)
	}

	if _, ok := fake.objects["retried.bin"]; ok {
		t.Error("an incomplete upload published an object")
	}
}

func TestS3AbortReleasesTheUpload(t *testing.T) {
	st, fake := newFakeS3Storage(t)

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	if err := st.InitChunkedUpload("session", "aborted.bin", 2, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}
	if err := st.SaveChunk("session", 0, bytes.NewReader(testPayload(int(chunkSize))), chunkSize); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if err := st.AbortChunkedUpload("session"); err != nil {
		t.Fatalf("AbortChunkedUpload: %v", err)
	}

	// Without this the parts already uploaded stay in the bucket, billed, indefinitely.
	if len(fake.aborted) != 1 {
		t.Errorf("the multipart upload was aborted %d times, want 1", len(fake.aborted))
	}
}

func keysOf(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// The bucket returns transient 500s often enough that a part failing is routine. The part
// is spooled as it streams past precisely so that this is recovered inside the server,
// rather than costing the client a re-upload of the whole chunk over its own uplink.
func TestS3PartRecoversFromTransientFailureWithoutTheClient(t *testing.T) {
	st, fake := newFakeS3Storage(t)
	fake.failFirstAttempt = true

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	plain := testPayload(int(chunkSize) + 777)
	totalChunks := 2
	const path = "flaky.bin"

	if err := st.InitChunkedUpload("session", path, totalChunks, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}

	// The reader is handed over exactly once, as the request body would be. If the
	// recovery needed the client to send again, there would be nothing left to read.
	for i := 0; i < totalChunks; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize
		if end > int64(len(plain)) {
			end = int64(len(plain))
		}
		slice := plain[start:end]
		if err := st.SaveChunk("session", i, bytes.NewReader(slice), int64(len(slice))); err != nil {
			t.Fatalf("SaveChunk(%d) did not recover from the transient failure: %v", i, err)
		}
	}

	if _, err := st.FinalizeChunkedUpload("session"); err != nil {
		t.Fatalf("FinalizeChunkedUpload: %v", err)
	}

	for part, count := range fake.attempts {
		if count < 2 {
			t.Errorf("part %d was sent %d times, so the failure was never retried", part, count)
		}
	}

	reader, _, err := st.GetFileStream(path, StoredFile{
		EncryptionVersion: utils.EncryptionVersionStream,
		PlainSize:         int64(len(plain)),
	})
	if err != nil {
		t.Fatalf("GetFileStream: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Error("the object retried from the spool does not match what was uploaded")
	}
}

// If the connection breaks partway through the body there is no complete copy to replay,
// and the chunk has to go back to the client rather than a short part being sent.
func TestS3PartialBodyIsNotReplayed(t *testing.T) {
	st, fake := newFakeS3Storage(t)
	fake.failFirstAttempt = true

	const chunkSize = int64(testChunkSizeMB * 1024 * 1024)
	if err := st.InitChunkedUpload("session", "short.bin", 1, chunkSize); err != nil {
		t.Fatalf("InitChunkedUpload: %v", err)
	}

	// A body that ends early: encryption fails, so the spool never holds a whole part.
	err := st.SaveChunk("session", 0, bytes.NewReader(testPayload(100)), chunkSize)
	if err == nil {
		t.Fatal("a truncated chunk was accepted")
	}

	// The handler tells a short body apart from a storage fault by this, answering the
	// first with a 400, so the cause has to survive the failed replay attempt.
	if !errors.Is(err, utils.ErrShortSource) {
		t.Errorf("error lost the short-body cause: %v", err)
	}

	st.uploadsMutex.RLock()
	parts := len(st.uploads["session"].parts)
	st.uploadsMutex.RUnlock()
	if parts != 0 {
		t.Errorf("a truncated chunk recorded %d parts, want 0", parts)
	}
}
