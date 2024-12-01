package storage

import "mime/multipart"

type Storage interface {
	SaveFile(file *multipart.FileHeader, filePath string) (string, error)
	GetFile(filePath string) ([]byte, error)
	DeleteFile(filePath string) error
}
