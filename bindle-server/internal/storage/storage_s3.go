package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	localconfig "github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

type S3Upload struct {
	UploadID string
	FilePath string
	Parts    []types.CompletedPart
}

type S3Storage struct {
	client       *s3.Client
	bucket       string
	config       localconfig.Config
	uploads      map[string]*S3Upload // sessionID -> S3Upload
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
		uploads: make(map[string]*S3Upload),
	}, nil
}

func (s *S3Storage) SaveFile(file *multipart.FileHeader, filePath string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	encryptedFile, err := utils.EncryptFile(&s.config, content)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt file: %w", err)
	}

	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
		Body:   bytes.NewReader(encryptedFile),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return filePath, nil
}

func (s *S3Storage) GetFile(filePath string, chunkCount int) ([]byte, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from S3: %w", err)
	}
	defer result.Body.Close()

	encryptedFile, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// For legacy single-file uploads (chunkCount == 0), use old decryption
	if chunkCount == 0 {
		decryptedFile, err := utils.DecryptFile(&s.config, encryptedFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt file: %w", err)
		}
		return decryptedFile, nil
	}

	// For chunked uploads, stream decrypt each chunk
	log.Printf("Streaming decrypt %d chunks for S3 file %s\n", chunkCount, filePath)

	// Standard chunk size from config (10MB by default)
	standardChunkSize := int(s.config.ChunkSizeMB * 1024 * 1024)
	// Encrypted chunk overhead: 8 bytes chunkNumber + 12 bytes nonce + 16 bytes authTag
	chunkOverhead := 8 + 12 + 16

	// Decrypt each chunk and accumulate
	decryptedData := make([]byte, 0)
	offset := 0

	for i := 0; i < chunkCount; i++ {
		// Calculate expected encrypted chunk size
		var expectedEncryptedSize int
		if i < chunkCount-1 {
			// Not the last chunk - standard size
			expectedEncryptedSize = standardChunkSize + chunkOverhead
		} else {
			// Last chunk - whatever remains
			expectedEncryptedSize = len(encryptedFile) - offset
		}

		if offset+expectedEncryptedSize > len(encryptedFile) {
			return nil, fmt.Errorf("encrypted file truncated at chunk %d", i)
		}

		encryptedChunk := encryptedFile[offset : offset+expectedEncryptedSize]
		decryptedChunk, err := utils.DecryptChunk(&s.config, encryptedChunk, i)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt chunk %d: %w", i, err)
		}

		decryptedData = append(decryptedData, decryptedChunk...)
		offset += expectedEncryptedSize
	}

	log.Printf("Decrypted %d chunks (%d bytes) for S3 file %s\n", chunkCount, len(decryptedData), filePath)
	return decryptedData, nil
}

// GetFileStream returns a streaming reader for file download from S3 (memory-efficient)
// This is the preferred method for downloading files as it doesn't load entire file into memory
func (s *S3Storage) GetFileStream(filePath string, chunkCount int) (io.ReadCloser, int64, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get file from S3: %w", err)
	}

	// S3 GetObject already returns a streaming reader (result.Body)
	encryptedSize := *result.ContentLength

	// For legacy single-file uploads (chunkCount == 0)
	if chunkCount == 0 {
		reader := utils.NewLegacyDecryptionReader(result.Body, &s.config)
		// For legacy files, we can't know exact decrypted size without decrypting
		// Return encrypted size as approximation
		return reader, encryptedSize, nil
	}

	// For chunked uploads - wrap S3 body in streaming decryption reader
	reader := utils.NewChunkedDecryptionReader(result.Body, &s.config, chunkCount)

	// Calculate actual decrypted size (remove encryption overhead)
	decryptedSize := utils.CalculateDecryptedSize(encryptedSize, chunkCount, s.config.ChunkSizeMB)

	log.Printf("Streaming S3 file %s (%d chunks, %d bytes decrypted)\n", filePath, chunkCount, decryptedSize)

	return reader, decryptedSize, nil
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

// Chunked upload methods

func (s *S3Storage) InitChunkedUpload(sessionID string, fileName string, totalChunks int) error {
	// Use S3's native multipart upload API for efficient streaming
	// This avoids downloading chunks back to assemble them
	s.uploadsMutex.Lock()
	defer s.uploadsMutex.Unlock()

	// Generate file path for S3 object key (will be set in SaveChunk on first call)
	s.uploads[sessionID] = &S3Upload{
		FilePath: "",  // Will be set in SaveChunk
		UploadID: "",  // Will be created when first chunk arrives
		Parts:    make([]types.CompletedPart, 0, totalChunks),
	}

	log.Printf("Initialized S3 chunked upload session %s with %d chunks (multipart API)\n", sessionID, totalChunks)
	return nil
}

func (s *S3Storage) SaveChunk(sessionID string, chunkNumber int, chunkData []byte) error {
	s.uploadsMutex.Lock()
	upload, exists := s.uploads[sessionID]
	if !exists {
		s.uploadsMutex.Unlock()
		return fmt.Errorf("upload session %s not found", sessionID)
	}

	// Create multipart upload on first chunk (we don't know filePath until FinalizeChunkedUpload)
	// So we use a temporary key based on sessionID
	if upload.UploadID == "" {
		tempKey := fmt.Sprintf("temp-uploads/%s", sessionID)
		upload.FilePath = tempKey

		createResp, err := s.client.CreateMultipartUpload(context.TODO(), &s3.CreateMultipartUploadInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(tempKey),
		})
		if err != nil {
			s.uploadsMutex.Unlock()
			return fmt.Errorf("failed to create multipart upload: %w", err)
		}

		upload.UploadID = *createResp.UploadId
		log.Printf("Created S3 multipart upload %s for session %s\n", upload.UploadID, sessionID)
	}
	s.uploadsMutex.Unlock()

	// Encrypt chunk independently (10MB - acceptable in memory)
	encryptedChunk, err := utils.EncryptChunk(&s.config, chunkData, chunkNumber)
	if err != nil {
		return fmt.Errorf("failed to encrypt chunk: %w", err)
	}

	// Upload part using S3 multipart API
	// Part numbers are 1-indexed in S3
	partNumber := int32(chunkNumber + 1)

	uploadResp, err := s.client.UploadPart(context.TODO(), &s3.UploadPartInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(upload.FilePath),
		UploadId:   aws.String(upload.UploadID),
		PartNumber: aws.Int32(partNumber),
		Body:       bytes.NewReader(encryptedChunk),
	})
	if err != nil {
		return fmt.Errorf("failed to upload part to S3: %w", err)
	}

	// Track completed part
	s.uploadsMutex.Lock()
	upload.Parts = append(upload.Parts, types.CompletedPart{
		ETag:       uploadResp.ETag,
		PartNumber: aws.Int32(partNumber),
	})
	s.uploadsMutex.Unlock()

	log.Printf("Uploaded encrypted chunk %d (part %d) for session %s to S3 (%d bytes)\n",
		chunkNumber, partNumber, sessionID, len(encryptedChunk))
	return nil
}

func (s *S3Storage) FinalizeChunkedUpload(sessionID string, filePath string) (string, error) {
	s.uploadsMutex.Lock()
	upload, exists := s.uploads[sessionID]
	if !exists {
		s.uploadsMutex.Unlock()
		return "", fmt.Errorf("upload session %s not found", sessionID)
	}
	uploadID := upload.UploadID
	tempKey := upload.FilePath
	parts := upload.Parts
	delete(s.uploads, sessionID)
	s.uploadsMutex.Unlock()

	log.Printf("Completing S3 multipart upload for session %s (%d parts)\n", sessionID, len(parts))

	// Complete multipart upload - S3 assembles parts server-side (zero memory!)
	_, err := s.client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(tempKey),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	log.Printf("S3 multipart upload completed for session %s at temp key: %s\n", sessionID, tempKey)

	// Copy from temp location to final location
	_, err = s.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", s.bucket, tempKey)),
		Key:        aws.String(filePath),
	})
	if err != nil {
		return "", fmt.Errorf("failed to copy to final location: %w", err)
	}

	// Delete temp object
	_, err = s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(tempKey),
	})
	if err != nil {
		log.Printf("Warning: failed to delete temp object %s: %v\n", tempKey, err)
	}

	log.Printf("Finalized S3 upload to: %s\n", filePath)
	return filePath, nil
}

func (s *S3Storage) AbortChunkedUpload(sessionID string) error {
	s.uploadsMutex.Lock()
	upload, exists := s.uploads[sessionID]
	if !exists {
		s.uploadsMutex.Unlock()
		log.Printf("Upload session %s not found for abort\n", sessionID)
		return nil
	}

	uploadID := upload.UploadID
	filePath := upload.FilePath
	delete(s.uploads, sessionID)
	s.uploadsMutex.Unlock()

	// Abort multipart upload if it was started
	if uploadID != "" {
		_, err := s.client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(s.bucket),
			Key:      aws.String(filePath),
			UploadId: aws.String(uploadID),
		})
		if err != nil {
			log.Printf("Warning: failed to abort multipart upload %s: %v\n", uploadID, err)
			return fmt.Errorf("failed to abort multipart upload: %w", err)
		}

		log.Printf("Aborted S3 multipart upload %s for session %s\n", uploadID, sessionID)
	} else {
		log.Printf("Aborted S3 upload session %s (no multipart upload started)\n", sessionID)
	}

	return nil
}
