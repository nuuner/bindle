package models

import (
	"encoding/json"
	"time"

	"github.com/nuuner/bindle-server/internal/config"
	"gorm.io/gorm"
)

type FileType string

const (
	FileTypeText    FileType = "text"
	FileTypeImage   FileType = "image"
	FileTypeVideo   FileType = "video"
	FileTypeAudio   FileType = "audio"
	FileTypeUnknown FileType = "unknown"
)

type UploadedFile struct {
	gorm.Model
	FileId   string   `json:"fileId" gorm:"uniqueIndex"`
	FilePath string   `json:"filePath"`
	FileName string   `json:"fileName"`
	Size     int64    `json:"size"`
	Type     FileType `json:"type"`
	MimeType string   `json:"mimeType"`
	Details  *string  `json:"details,omitempty"`
	OwnerID  uint     `json:"ownerId"`
	Owner    User
}

type UploadedFileDTO struct {
	FileId    string    `json:"fileId"`
	FileName  string    `json:"fileName"`
	Size      int64     `json:"size"`
	Type      FileType  `json:"type"`
	MimeType  string    `json:"mimeType"`
	URL       string    `json:"url"`
	Details   *string   `json:"details,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func (uf *UploadedFile) MarshalJSON() ([]byte, error) {
	cfg := config.GetConfig()
	return json.Marshal(UploadedFileDTO{
		FileId:    uf.FileId,
		FileName:  uf.FileName,
		Size:      uf.Size,
		Type:      uf.Type,
		MimeType:  uf.MimeType,
		URL:       cfg.FileHost + uf.FilePath,
		Details:   uf.Details,
		CreatedAt: uf.CreatedAt,
	})
}

type User struct {
	gorm.Model
	AccountId string         `json:"accountId" gorm:"uniqueIndex"`
	Files     []UploadedFile `json:"files" gorm:"foreignKey:OwnerID"`
	LastLogin time.Time      `json:"lastLogin"`
}

type AccountIpConnection struct {
	gorm.Model
	AccountID uint
	Account   User
	IPAddress string
}
