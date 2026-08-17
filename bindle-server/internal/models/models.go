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
	SessionID string `json:"sessionId" gorm:"uniqueIndex"`
	AccountID uint   `json:"accountId"`
	Account   User   `json:"account"`
	FileName  string `json:"fileName"`
	// FileSize is the size the client declared at init. It is checked against the
	// upload quota up front and then enforced per chunk, so it is an upper bound on
	// what the session can actually store.
	FileSize int64  `json:"fileSize"`
	MimeType string `json:"mimeType"`
	// ChunkSize is pinned at init so that chunk bounds stay stable for the life of
	// the session even if CHUNK_SIZE_MB changes underneath it.
	ChunkSize   int64 `json:"chunkSize"`
	TotalChunks int   `json:"totalChunks"`
	// FilePath is the object the session writes to, decided at init so the storage
	// backend can open the upload against its final destination and never move the
	// data afterwards.
	FilePath  string              `json:"filePath"`
	FileHash  string              `json:"fileHash"`
	Status    UploadSessionStatus `json:"status"`
	ExpiresAt time.Time           `json:"expiresAt"`
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
	// EncryptionVersion selects the reader used to decrypt this file. 0 is everything
	// stored before the framed streaming format, which is why the default matters:
	// rows that predate the column have to keep decoding the way they were written.
	EncryptionVersion int  `json:"-" gorm:"default:0"`
	OwnerID           uint `json:"ownerId" gorm:"index"`
	Owner             User
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
	// Whether this client holds a valid unlock cookie, in which case UploadLimitBytes
	// does not apply to it.
	LimitsUnlocked bool `json:"limitsUnlocked"`
	// Whether the server has an unlock password configured at all. The client hides the
	// unlock option when it does not.
	UnlockAvailable bool `json:"unlockAvailable"`
}

// Connection tracking models
// AccountIpConnection links accounts that shared an IP, which is how the daily quota is
// pooled. Both columns are looked up on their own by the quota walk, and together on
// every authenticated request, so both are indexed.
type AccountIpConnection struct {
	gorm.Model
	AccountID uint `gorm:"index:idx_account_ip,priority:1;index"`
	Account   User
	IPAddress string `gorm:"index:idx_account_ip,priority:2;index"`
}
