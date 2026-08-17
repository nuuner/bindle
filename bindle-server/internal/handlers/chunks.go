package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/internal/storage"
	"github.com/nuuner/bindle-server/pkg/limiter"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

// sessionFilePath derives where a session's object will live. It is decided at init
// rather than on completion so the storage backend can write straight to the final
// location, which is what lets an S3 upload finish without copying the assembled object
// out of a temporary key.
func sessionFilePath(sessionID, fileName string) (hash string, path string) {
	hasher := sha256.New()
	hasher.Write([]byte(sessionID + fileName))
	hash = hex.EncodeToString(hasher.Sum(nil))
	return hash, hash + filepath.Ext(fileName)
}

// InitChunkedUpload initializes a new chunked upload session
func InitChunkedUpload(c *fiber.Ctx, db *gorm.DB, cfg *config.Config, st storage.Storage) error {
	type InitRequest struct {
		FileName    string `json:"fileName"`
		FileSize    int64  `json:"fileSize"`
		MimeType    string `json:"mimeType"`
		TotalChunks int    `json:"totalChunks"`
	}

	req := new(InitRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate input
	if req.FileName == "" || req.FileSize <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file metadata"})
	}

	if req.FileSize > cfg.MaxFileSizeMB*1000*1000 {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "File exceeds maximum allowed size"})
	}

	// Check upload limits
	if limiter.ShouldThrottle(c, db, cfg, req.FileSize) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Upload limit exceeded"})
	}

	// The chunk layout is derived from the declared size rather than taken from the
	// request: it is what bounds every subsequent chunk, so the client must not get
	// to pick it. req.TotalChunks is accepted for compatibility but ignored.
	chunkSize := cfg.ChunkSizeMB * 1024 * 1024
	totalChunks := int((req.FileSize + chunkSize - 1) / chunkSize)

	// Generate session ID
	sessionID := uuid.New().String()
	hash, filePath := sessionFilePath(sessionID, req.FileName)

	// Create upload session in database
	user := utils.GetUser(c)
	uploadSession := &models.UploadSession{
		SessionID:   sessionID,
		AccountID:   user.ID,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		MimeType:    req.MimeType,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		FilePath:    filePath,
		FileHash:    hash,
		Status:      models.UploadSessionStatusActive,
		ExpiresAt:   time.Now().Add(24 * time.Hour), // 24 hour expiration
	}

	result := db.Create(uploadSession)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload session"})
	}

	// Initialize storage for chunked upload
	err := st.InitChunkedUpload(sessionID, filePath, totalChunks, chunkSize)
	if err != nil {
		log.Printf("Failed to initialize chunked upload: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initialize upload"})
	}

	log.Printf("Initialized upload session %s for file %s (%d bytes, %d chunks)", sessionID, req.FileName, req.FileSize, totalChunks)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"sessionId":   sessionID,
		"chunkSize":   chunkSize,
		"totalChunks": totalChunks,
	})
}

// maxChunkBytes returns how many bytes chunk chunkNumber is allowed to carry: the
// slice of the declared file that lands at that index. Bounding every chunk this way
// is what keeps a session from storing more than the size that was quota-checked at
// init - without it a client could declare one byte and then upload chunks of any
// size. The bound depends only on the chunk index, so retrying a chunk re-checks the
// same value instead of drifting the way a running total would.
func maxChunkBytes(session *models.UploadSession, chunkNumber int) int64 {
	remaining := session.FileSize - int64(chunkNumber)*session.ChunkSize
	if remaining < 0 {
		return 0
	}
	if remaining > session.ChunkSize {
		return session.ChunkSize
	}
	return remaining
}

// loadActiveSession fetches the session a chunk request names, scoped to the account
// that opened it so a leaked session id is not on its own enough to write into someone
// else's upload.
func loadActiveSession(c *fiber.Ctx, db *gorm.DB, sessionID string) (*models.UploadSession, error) {
	var session models.UploadSession
	user := utils.GetUser(c)
	err := db.Where("session_id = ? AND account_id = ? AND status = ?",
		sessionID, user.ID, models.UploadSessionStatusActive).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UploadChunk handles uploading a single chunk
func UploadChunk(c *fiber.Ctx, db *gorm.DB, st storage.Storage) error {
	sessionID := c.Params("sessionId")
	chunkNumberStr := c.Params("chunkNumber")

	chunkNumber, err := strconv.Atoi(chunkNumberStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid chunk number"})
	}

	uploadSession, err := loadActiveSession(c, db, sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Upload session not found or expired"})
	}

	// Check if session has expired
	if time.Now().After(uploadSession.ExpiresAt) {
		uploadSession.Status = models.UploadSessionStatusExpired
		db.Save(uploadSession)
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "Upload session expired"})
	}

	// Validate chunk number
	if chunkNumber < 0 || chunkNumber >= uploadSession.TotalChunks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid chunk number"})
	}

	// The chunk has to be exactly the slice the layout assigns to this index. It was an
	// upper bound before, but the encrypted length is now declared to the storage
	// backend before a byte is read - S3 needs the part length up front - so the length
	// has to be known rather than discovered, and a client sending less is rejected
	// instead of silently storing a hole.
	expected := maxChunkBytes(uploadSession, chunkNumber)
	if int64(c.Request().Header.ContentLength()) != expected {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chunk does not match the size declared for this upload",
		})
	}

	// Save chunk to storage straight from the request body: it is encrypted and
	// forwarded as the backend pulls, so a chunk is never held in memory whole and
	// back pressure from storage reaches the client's socket.
	if err := st.SaveChunk(sessionID, chunkNumber, requestBodyReader(c, expected), expected); err != nil {
		log.Printf("Failed to save chunk %d for session %s: %v", chunkNumber, sessionID, err)
		if errors.Is(err, utils.ErrShortSource) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Chunk body ended early"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save chunk"})
	}

	// No progress is written here on purpose. This runs once per chunk on the upload's
	// critical path, and which chunks have landed is answered by the storage backend at
	// finalize - a counter could not answer it anyway now that chunks arrive
	// concurrently and may be retried.
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"chunkNumber": chunkNumber,
		"totalChunks": uploadSession.TotalChunks,
	})
}

// requestBodyReader returns the request body as a stream bounded to limit bytes.
// StreamRequestBody is on, so this is normally the connection itself; the fallback
// covers the case where fasthttp has already buffered the body.
func requestBodyReader(c *fiber.Ctx, limit int64) io.Reader {
	if stream := c.Context().RequestBodyStream(); stream != nil {
		return io.LimitReader(stream, limit)
	}
	return bytes.NewReader(c.Body())
}

// CompleteChunkedUpload finalizes the chunked upload
func CompleteChunkedUpload(c *fiber.Ctx, db *gorm.DB, st storage.Storage) error {
	sessionID := c.Params("sessionId")

	uploadSession, err := loadActiveSession(c, db, sessionID)
	if err != nil {
		log.Printf("Session not found: %s, error: %v", sessionID, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Upload session not found"})
	}

	// Publishes the object where it was already written. Whether every chunk arrived is
	// the storage backend's answer, since it is the only place that knows which indexes
	// it holds.
	finalPath, err := st.FinalizeChunkedUpload(sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrIncompleteUpload) {
			log.Printf("Complete failed for session %s: %v", sessionID, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		log.Printf("Failed to finalize upload for session %s: %v", sessionID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize upload"})
	}

	// Create file record in database
	guid, err := uuid.NewV7()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate file ID"})
	}

	fileToCreate := &models.UploadedFile{
		FileId:            guid.String(),
		FilePath:          uploadSession.FilePath,
		FileName:          uploadSession.FileName,
		Size:              uploadSession.FileSize,
		Type:              utils.GetFileType(uploadSession.MimeType),
		MimeType:          uploadSession.MimeType,
		ChunkCount:        uploadSession.TotalChunks,
		EncryptionVersion: utils.EncryptionVersionStream,
		OwnerID:           uploadSession.AccountID,
	}

	result := db.Create(fileToCreate)
	if result.Error != nil {
		log.Printf("Failed to create file record: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create file record"})
	}

	// Update session status
	uploadSession.Status = models.UploadSessionStatusCompleted
	db.Save(uploadSession)

	log.Printf("Completed chunked upload for session %s: %s (%d bytes)", sessionID, finalPath, uploadSession.FileSize)

	return c.Status(fiber.StatusOK).JSON(fileToCreate)
}

// AbortChunkedUpload cancels an upload session
func AbortChunkedUpload(c *fiber.Ctx, db *gorm.DB, st storage.Storage) error {
	sessionID := c.Params("sessionId")

	uploadSession, err := loadActiveSession(c, db, sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Upload session not found"})
	}

	// Abort the upload in storage
	if err := st.AbortChunkedUpload(sessionID); err != nil {
		log.Printf("Failed to abort upload for session %s: %v", sessionID, err)
		// Continue anyway to update database
	}

	// Update session status
	uploadSession.Status = models.UploadSessionStatusCancelled
	db.Save(uploadSession)

	log.Printf("Aborted upload session %s", sessionID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Upload cancelled"})
}
