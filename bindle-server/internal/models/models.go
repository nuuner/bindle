package models

import (
	"encoding/json"
	"time"

	"github.com/nuuner/bindle-server/internal/config"
	"gorm.io/gorm"
)

// File Types
type FileType string

const (
	FileTypeText    FileType = "text"
	FileTypeImage   FileType = "image"
	FileTypeVideo   FileType = "video"
	FileTypeAudio   FileType = "audio"
	FileTypeUnknown FileType = "unknown"
)

// Upload Session Status
type UploadSessionStatus string

const (
	UploadSessionStatusActive    UploadSessionStatus = "active"
	UploadSessionStatusCompleted UploadSessionStatus = "completed"
	UploadSessionStatusCancelled UploadSessionStatus = "cancelled"
	UploadSessionStatusExpired   UploadSessionStatus = "expired"
)

// Upload Session models
type UploadSession struct {
	gorm.Model
	SessionID      string              `json:"sessionId" gorm:"uniqueIndex"`
	AccountID      uint                `json:"accountId"`
	Account        User                `json:"account"`
	FileName       string              `json:"fileName"`
	FileSize       int64               `json:"fileSize"`
	MimeType       string              `json:"mimeType"`
	TotalChunks    int                 `json:"totalChunks"`
	UploadedChunks int                 `json:"uploadedChunks"`
	FileHash       string              `json:"fileHash"`
	Status         UploadSessionStatus `json:"status"`
	ExpiresAt      time.Time           `json:"expiresAt"`
}

// User related models
type User struct {
	gorm.Model
	AccountId string         `json:"accountId" gorm:"uniqueIndex"`
	Files     []UploadedFile `json:"files" gorm:"foreignKey:OwnerID"`
	LastLogin time.Time      `json:"lastLogin"`
}

type UserDTO struct {
	AccountId string         `json:"accountId"`
	LastLogin time.Time      `json:"lastLogin"`
	Files     []UploadedFile `json:"files"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	dto := UserDTO{
		AccountId: u.AccountId,
		LastLogin: u.LastLogin,
		Files:     u.Files,
	}
	return json.Marshal(dto)
}

// File related models
type UploadedFile struct {
	gorm.Model
	FileId     string   `json:"fileId" gorm:"uniqueIndex"`
	FilePath   string   `json:"filePath"`
	FileName   string   `json:"fileName"`
	Size       int64    `json:"size"`
	Type       FileType `json:"type"`
	MimeType   string   `json:"mimeType"`
	Details    *string  `json:"details,omitempty"`
	ChunkCount int      `json:"chunkCount" gorm:"default:0"` // 0 = single file upload, >0 = chunked upload
	OwnerID    uint     `json:"ownerId"`
	Owner      User
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
	dto := UploadedFileDTO{
		FileId:    uf.FileId,
		FileName:  uf.FileName,
		Size:      uf.Size,
		Type:      uf.Type,
		MimeType:  uf.MimeType,
		URL:       cfg.FileHost + uf.FilePath,
		Details:   uf.Details,
		CreatedAt: uf.CreatedAt,
	}
	return json.Marshal(dto)
}

// Response models
type MeResponse struct {
	User             UserDTO `json:"user"`
	UploadedBytes    int64   `json:"uploadedBytes"`
	UploadLimitBytes int64   `json:"uploadLimitBytes"`
	MaxFileSizeBytes int64   `json:"maxFileSizeBytes"`
}

// Connection tracking models
type AccountIpConnection struct {
	gorm.Model
	AccountID uint
	Account   User
	IPAddress string
}
