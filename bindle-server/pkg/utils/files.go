package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
)

func EnsureFileDirectory(config config.Config) error {
	return os.MkdirAll(config.FilesystemPath, 0755)
}

func GetFileType(mimeType string) models.FileType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return models.FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return models.FileTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return models.FileTypeAudio
	case strings.HasPrefix(mimeType, "text/"):
		return models.FileTypeText
	default:
		return models.FileTypeUnknown
	}
}

func GetFileHash(file *multipart.FileHeader) (string, error) {
	hash := sha256.New()
	multipartFile, err := file.Open()
	if err != nil {
		return "", err
	}
	fileBytes, err := io.ReadAll(multipartFile)
	if err != nil {
		return "", err
	}
	defer multipartFile.Close()

	hash.Write(fileBytes)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func BytesToMimeType(fileBytes []byte) string {
	mimeType := mimetype.Detect(fileBytes)
	return mimeType.String()
}
