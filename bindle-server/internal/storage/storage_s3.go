package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	localconfig "github.com/nuuner/bindle-server/internal/config"
)

type S3Storage struct {
	client *s3.Client
	bucket string
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
		client: client,
		bucket: cfg.S3Bucket,
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

	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return filePath, nil
}

func (s *S3Storage) GetFile(filePath string) ([]byte, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from S3: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, nil
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
