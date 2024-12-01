package storage

import (
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/utils"
)

type FilesystemStorage struct {
	config config.Config
}

func NewFilesystemStorage(config config.Config) (*FilesystemStorage, error) {
	return &FilesystemStorage{config: config}, nil
}

func (s *FilesystemStorage) SaveFile(file *multipart.FileHeader, filePath string) (string, error) {
	// Ensure filesystem path exists
	err := utils.EnsureFileDirectory(s.config)
	if err != nil {
		return "", err
	}

	// Open source file
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Read file bytes
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return "", err
	}

	fullPath := s.config.FilesystemPath + "/" + filePath
	log.Println("saving file to", fullPath)
	if err := os.WriteFile(fullPath, fileBytes, 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (s *FilesystemStorage) GetFile(filePath string) ([]byte, error) {
	fullPath := s.config.FilesystemPath + "/" + filePath
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read file contents
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func (s *FilesystemStorage) DeleteFile(filePath string) error {
	fullPath := s.config.FilesystemPath + "/" + filePath
	return os.Remove(fullPath)
}
