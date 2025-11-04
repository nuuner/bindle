package config

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
)

type Config struct {
	FileHost           string
	ClientOrigin       string
	RequestSizeLimitMB int64
	// S3
	S3Enabled  bool
	S3KeyId    string
	S3AppKey   string
	S3Bucket   string
	S3Region   string
	S3Endpoint string
	// Filesystem
	FilesystemPath string
	// Account
	AccountExpirationDays int
	// Upload limits
	UploadLimitMBPerDay int64
	ChunkSizeMB         int64
	MaxFileSizeMB       int64
	// Encryption
	EncryptionKey []byte
}

var cfg Config

func GetConfig() Config {
	if cfg.FileHost != "" {
		return cfg
	}

	requestSizeLimitMB, err := strconv.ParseInt(os.Getenv("REQUEST_SIZE_LIMIT_MB"), 10, 64)
	if err != nil {
		log.Fatal("failed to parse REQUEST_SIZE_LIMIT_MB:", err)
	}

	accountExpirationDays, err := strconv.Atoi(os.Getenv("ACCOUNT_EXPIRATION_DAYS"))
	if err != nil {
		log.Fatal("failed to parse ACCOUNT_EXPIRATION_DAYS:", err)
	}

	uploadLimitMBPerDay, err := strconv.ParseInt(os.Getenv("UPLOAD_LIMIT_MB_PER_DAY"), 10, 64)
	if err != nil {
		log.Println("No UPLOAD_LIMIT_MB_PER_DAY environment variable found, using default value of 1000MB")
		uploadLimitMBPerDay = 1000
	}

	chunkSizeMB, err := strconv.ParseInt(os.Getenv("CHUNK_SIZE_MB"), 10, 64)
	if err != nil {
		log.Println("No CHUNK_SIZE_MB environment variable found, using default value of 10MB")
		chunkSizeMB = 10
	}

	maxFileSizeMB, err := strconv.ParseInt(os.Getenv("MAX_FILE_SIZE_MB"), 10, 64)
	if err != nil {
		log.Println("No MAX_FILE_SIZE_MB environment variable found, using default value of 20480MB (20GB)")
		maxFileSizeMB = 20480
	}

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Fatal("ENCRYPTION_KEY environment variable is not set")
	}
	encryptionKeyBytes, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		log.Fatal("ENCRYPTION_KEY is not valid base64:", err)
	}
	if len(encryptionKeyBytes) != 32 {
		log.Fatal("ENCRYPTION_KEY must be 32 bytes long")
	}

	cfg = Config{
		FileHost:              os.Getenv("FILE_HOST"),
		ClientOrigin:          os.Getenv("CLIENT_ORIGIN"),
		RequestSizeLimitMB:    requestSizeLimitMB,
		S3Enabled:             os.Getenv("S3_BUCKET") != "",
		S3KeyId:               os.Getenv("S3_KEY_ID"),
		S3AppKey:              os.Getenv("S3_APP_KEY"),
		S3Bucket:              os.Getenv("S3_BUCKET"),
		S3Region:              os.Getenv("S3_REGION"),
		S3Endpoint:            os.Getenv("S3_ENDPOINT"),
		FilesystemPath:        os.Getenv("FILESYSTEM_PATH"),
		AccountExpirationDays: accountExpirationDays,
		UploadLimitMBPerDay:   uploadLimitMBPerDay,
		ChunkSizeMB:           chunkSizeMB,
		MaxFileSizeMB:         maxFileSizeMB,
		EncryptionKey:         encryptionKeyBytes,
	}

	return cfg
}
