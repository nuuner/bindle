package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	localconfig "github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

// s3Upload is the in-flight state of one chunked upload.
type s3Upload struct {
	// key is the object's final location. The multipart upload is opened directly
	// against it, so completing the upload is the only step left at the end - the
	// earlier design assembled at a temp key and then copied, which meant a full
	// server-side copy of the file after the client had already finished (and failed
	// outright above the 5 GB CopyObject limit).
	key         string
	uploadID    string
	totalChunks int
	chunkSize   int64
	// parts is keyed by part number rather than appended in arrival order, because
	// chunks arrive concurrently and may be retried: CompleteMultipartUpload requires
	// ascending part numbers, and a retried chunk has to replace its earlier ETag
	// instead of adding a duplicate.
	parts map[int32]types.CompletedPart
}

type S3Storage struct {
	client       *s3.Client
	bucket       string
	config       localconfig.Config
	uploads      map[string]*s3Upload // sessionID -> upload
	uploadsMutex sync.RWMutex
}

func NewS3Storage(cfg localconfig.Config) (*S3Storage, error) {
	var options []func(*s3.Options)

	if cfg.S3Endpoint != "" {
		options = append(options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		})
	}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.S3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3KeyId,
			cfg.S3AppKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, options...)

	return &S3Storage{
		client:  client,
		bucket:  cfg.S3Bucket,
		config:  cfg,
		uploads: make(map[string]*s3Upload),
	}, nil
}

func (s *S3Storage) SaveFile(file *multipart.FileHeader, filePath string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	encrypted, err := utils.NewEncryptingReader(src, s.config.EncryptionKey, file.Size, 0)
	if err != nil {
		return "", fmt.Errorf("failed to set up encryption: %w", err)
	}

	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(filePath),
		Body:          encrypted,
		ContentLength: aws.Int64(utils.EncryptedSize(file.Size)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return filePath, nil
}

// GetFileStream returns a streaming reader over the decrypted object. S3 hands back a
// streaming body, and the decryption reader consumes it a frame at a time, so a download
// costs one frame of memory no matter how large the file is.
func (s *S3Storage) GetFileStream(filePath string, file StoredFile) (io.ReadCloser, int64, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get file from S3: %w", err)
	}

	reader, size, err := decryptStream(result.Body, *result.ContentLength, &s.config, file)
	if err != nil {
		result.Body.Close()
		return nil, 0, err
	}

	log.Printf("Streaming S3 file %s (%d bytes)\n", filePath, size)
	return reader, size, nil
}

func (s *S3Storage) DeleteFile(filePath string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// Chunked upload

func (s *S3Storage) InitChunkedUpload(sessionID string, filePath string, totalChunks int, chunkSize int64) error {
	// Opened here rather than lazily on the first chunk: chunks now arrive
	// concurrently, and creating the multipart upload up front keeps the first
	// arrivals from queueing behind one another to do it.
	createResp, err := s.client.CreateMultipartUpload(context.TODO(), &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	s.uploadsMutex.Lock()
	s.uploads[sessionID] = &s3Upload{
		key:         filePath,
		uploadID:    *createResp.UploadId,
		totalChunks: totalChunks,
		chunkSize:   chunkSize,
		parts:       make(map[int32]types.CompletedPart, totalChunks),
	}
	s.uploadsMutex.Unlock()

	log.Printf("Initialized S3 multipart upload %s for session %s at %s (%d parts)\n",
		*createResp.UploadId, sessionID, filePath, totalChunks)
	return nil
}

func (s *S3Storage) SaveChunk(sessionID string, chunkNumber int, r io.Reader, plainSize int64) error {
	s.uploadsMutex.RLock()
	upload, exists := s.uploads[sessionID]
	s.uploadsMutex.RUnlock()
	if !exists {
		return fmt.Errorf("upload session %s not found", sessionID)
	}

	// The encrypted length follows from the plaintext length, so it can be declared
	// before a byte is read. That is what lets the chunk go straight from the request
	// body into the S3 request: encryption happens as the SDK pulls, and back pressure
	// from S3 reaches the client's socket instead of a buffer growing in between.
	encrypted, err := utils.NewEncryptingReader(
		r, s.config.EncryptionKey, plainSize, int64(chunkNumber)*utils.FramesPerChunk(upload.chunkSize))
	if err != nil {
		return fmt.Errorf("failed to set up encryption: %w", err)
	}

	// Part numbers are 1-indexed in S3.
	partNumber := int32(chunkNumber + 1)

	uploadResp, err := s.client.UploadPart(context.TODO(), &s3.UploadPartInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(upload.key),
		UploadId:      aws.String(upload.uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          encrypted,
		ContentLength: aws.Int64(utils.EncryptedSize(plainSize)),
	}, func(o *s3.Options) {
		// The body is the client's connection, so it cannot be rewound and the SDK
		// cannot retry this call - left at the default it burns the retry delay only to
		// fail with "failed to rewind transport stream". Failing immediately hands the
		// chunk back to the client, which retries it; re-uploading the part replaces it
		// rather than adding a duplicate, so a retry is safe.
		o.RetryMaxAttempts = 1
	})
	if err != nil {
		return fmt.Errorf("failed to upload part %d to S3: %w", partNumber, err)
	}

	s.uploadsMutex.Lock()
	upload.parts[partNumber] = types.CompletedPart{
		ETag:       uploadResp.ETag,
		PartNumber: aws.Int32(partNumber),
	}
	s.uploadsMutex.Unlock()

	return nil
}

func (s *S3Storage) FinalizeChunkedUpload(sessionID string) (string, error) {
	s.uploadsMutex.Lock()
	upload, exists := s.uploads[sessionID]
	if !exists {
		s.uploadsMutex.Unlock()
		return "", fmt.Errorf("upload session %s not found", sessionID)
	}

	parts := make([]types.CompletedPart, 0, len(upload.parts))
	for _, part := range upload.parts {
		parts = append(parts, part)
	}
	key, uploadID, totalChunks := upload.key, upload.uploadID, upload.totalChunks
	s.uploadsMutex.Unlock()

	if len(parts) != totalChunks {
		return "", fmt.Errorf("%w: have %d of %d", ErrIncompleteUpload, len(parts), totalChunks)
	}

	sort.Slice(parts, func(i, j int) bool { return *parts[i].PartNumber < *parts[j].PartNumber })

	_, err := s.client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	// Only dropped once the object exists, so a failed completion can still be retried
	// or aborted with the upload id intact.
	s.uploadsMutex.Lock()
	delete(s.uploads, sessionID)
	s.uploadsMutex.Unlock()

	log.Printf("Completed S3 multipart upload for session %s at %s (%d parts)\n", sessionID, key, totalChunks)
	return key, nil
}

func (s *S3Storage) AbortChunkedUpload(sessionID string) error {
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

	// Without this the parts already uploaded stay in the bucket, billed, forever.
	_, err := s.client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(upload.key),
		UploadId: aws.String(upload.uploadID),
	})
	if err != nil {
		log.Printf("Warning: failed to abort multipart upload %s: %v\n", upload.uploadID, err)
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	log.Printf("Aborted S3 multipart upload %s for session %s\n", upload.uploadID, sessionID)
	return nil
}
