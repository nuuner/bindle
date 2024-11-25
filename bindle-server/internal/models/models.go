package models

import (
	"encoding/json"

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
	Hash     string   `json:"hash"`
	FileName string   `json:"fileName"`
	Size     int64    `json:"size"`
	Type     FileType `json:"type"`
	MimeType string   `json:"mimeType"`
	Details  *string  `json:"details,omitempty"`
	OwnerID  uint     `json:"ownerId"`
	Owner    User
}

func (uf *UploadedFile) MarshalJSON() ([]byte, error) {
	type fileDTO struct {
		ID       string   `json:"id"`
		FileName string   `json:"fileName"`
		Size     int64    `json:"size"`
		Type     FileType `json:"type"`
		MimeType string   `json:"mimeType"`
		URL      string   `json:"url"`
		Details  *string  `json:"details,omitempty"`
	}

	return json.Marshal(fileDTO{
		ID:       uf.FileId,
		FileName: uf.FileName,
		Size:     uf.Size,
		Type:     uf.Type,
		MimeType: uf.MimeType,
		URL:      config.FileHost + uf.Hash,
		Details:  uf.Details,
	})
}

type User struct {
	gorm.Model
	AccountId string         `json:"accountId" gorm:"uniqueIndex"`
	Files     []UploadedFile `json:"files" gorm:"foreignKey:OwnerID"`
}
