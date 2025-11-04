package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// InitChunkedUpload initializes a new chunked upload session
func InitChunkedUpload(c *fiber.Ctx, db *gorm.DB, cfg *config.Config, st storage.Storage) error {
	type InitRequest struct {
		FileName   string `json:"fileName"`
		FileSize   int64  `json:"fileSize"`
		MimeType   string `json:"mimeType"`
		TotalChunks int   `json:"totalChunks"`
	}

	req := new(InitRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate input
	if req.FileName == "" || req.FileSize <= 0 || req.TotalChunks <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file metadata"})
	}

	// Check upload limits
	if limiter.ShouldThrottle(c, db, cfg, req.FileSize) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Upload limit exceeded"})
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Create upload session in database
	user := utils.GetUser(c)
	uploadSession := &models.UploadSession{
		SessionID:      sessionID,
		AccountID:      user.ID,
		FileName:       req.FileName,
		FileSize:       req.FileSize,
		MimeType:       req.MimeType,
		TotalChunks:    req.TotalChunks,
		UploadedChunks: 0,
		Status:         models.UploadSessionStatusActive,
		ExpiresAt:      time.Now().Add(24 * time.Hour), // 24 hour expiration
	}

	result := db.Create(uploadSession)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload session"})
	}

	// Initialize storage for chunked upload
	err := st.InitChunkedUpload(sessionID, req.FileName, req.TotalChunks)
	if err != nil {
		log.Printf("Failed to initialize chunked upload: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initialize upload"})
	}

	log.Printf("Initialized upload session %s for file %s (%d bytes, %d chunks)", sessionID, req.FileName, req.FileSize, req.TotalChunks)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"sessionId": sessionID,
		"chunkSize": cfg.ChunkSizeMB * 1024 * 1024,
	})
}

// UploadChunk handles uploading a single chunk
func UploadChunk(c *fiber.Ctx, db *gorm.DB, st storage.Storage) error {
	sessionID := c.Params("sessionId")
	chunkNumberStr := c.Params("chunkNumber")

	chunkNumber, err := strconv.Atoi(chunkNumberStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid chunk number"})
	}

	// Verify session exists and is active
	var uploadSession models.UploadSession
	result := db.Where("session_id = ? AND status = ?", sessionID, models.UploadSessionStatusActive).First(&uploadSession)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Upload session not found or expired"})
	}

	// Check if session has expired
	if time.Now().After(uploadSession.ExpiresAt) {
		uploadSession.Status = models.UploadSessionStatusExpired
		db.Save(&uploadSession)
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "Upload session expired"})
	}

	// Validate chunk number
	if chunkNumber < 0 || chunkNumber >= uploadSession.TotalChunks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid chunk number"})
	}

	// Read chunk data from request body
	chunkData := c.Body()
	if len(chunkData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Empty chunk data"})
	}

	// Save chunk to storage
	err = st.SaveChunk(sessionID, chunkNumber, chunkData)
	if err != nil {
		log.Printf("Failed to save chunk %d for session %s: %v", chunkNumber, sessionID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save chunk"})
	}

	// Update session progress
	uploadSession.UploadedChunks = chunkNumber + 1
	db.Save(&uploadSession)

	log.Printf("Uploaded chunk %d/%d for session %s (%d bytes)", chunkNumber+1, uploadSession.TotalChunks, sessionID, len(chunkData))

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"chunkNumber":    chunkNumber,
		"uploadedChunks": uploadSession.UploadedChunks,
		"totalChunks":    uploadSession.TotalChunks,
	})
}

// CompleteChunkedUpload finalizes the chunked upload
func CompleteChunkedUpload(c *fiber.Ctx, db *gorm.DB, st storage.Storage) error {
	sessionID := c.Params("sessionId")
	log.Printf("CompleteChunkedUpload called for session: %s", sessionID)

	// Verify session exists and is active
	var uploadSession models.UploadSession
	result := db.Where("session_id = ? AND status = ?", sessionID, models.UploadSessionStatusActive).First(&uploadSession)
	if result.Error != nil {
		log.Printf("Session not found: %s, error: %v", sessionID, result.Error)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Upload session not found"})
	}

	log.Printf("Complete request for session %s: uploaded=%d, total=%d", sessionID, uploadSession.UploadedChunks, uploadSession.TotalChunks)

	// Verify all chunks have been uploaded
	// Note: uploadSession.UploadedChunks tracks the highest chunk number received + 1
	// For sequential uploads, this equals TotalChunks when all chunks are uploaded
	if uploadSession.UploadedChunks < uploadSession.TotalChunks {
		errMsg := fmt.Sprintf("Not all chunks uploaded (%d/%d)", uploadSession.UploadedChunks, uploadSession.TotalChunks)
		log.Printf("Complete failed for session %s: %s", sessionID, errMsg)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": errMsg,
		})
	}

	// Generate file hash from session ID and filename (temporary approach)
	// In a real implementation, we'd calculate the hash from assembled chunks
	hasher := sha256.New()
	hasher.Write([]byte(sessionID + uploadSession.FileName))
	hash := hex.EncodeToString(hasher.Sum(nil))
	filePath := hash + filepath.Ext(uploadSession.FileName)

	// Finalize the upload (assemble chunks, encrypt, save to final location)
	finalPath, err := st.FinalizeChunkedUpload(sessionID, filePath)
	if err != nil {
		log.Printf("Failed to finalize upload for session %s: %v", sessionID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize upload"})
	}

	// Create file record in database
	guid, err := uuid.NewV7()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate file ID"})
	}

	fileToCreate := &models.UploadedFile{
		FileId:     guid.String(),
		FilePath:   filePath,
		FileName:   uploadSession.FileName,
		Size:       uploadSession.FileSize,
		Type:       utils.GetFileType(uploadSession.MimeType),
		MimeType:   uploadSession.MimeType,
		ChunkCount: uploadSession.TotalChunks,
		OwnerID:    uploadSession.AccountID,
	}

	result = db.Create(fileToCreate)
	if result.Error != nil {
		log.Printf("Failed to create file record: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create file record"})
	}

	// Update session status
	uploadSession.Status = models.UploadSessionStatusCompleted
	uploadSession.FileHash = hash
	db.Save(&uploadSession)

	log.Printf("Completed chunked upload for session %s: %s (%d bytes)", sessionID, finalPath, uploadSession.FileSize)

	return c.Status(fiber.StatusOK).JSON(fileToCreate)
}

// AbortChunkedUpload cancels an upload session
func AbortChunkedUpload(c *fiber.Ctx, db *gorm.DB, st storage.Storage) error {
	sessionID := c.Params("sessionId")

	// Verify session exists
	var uploadSession models.UploadSession
	result := db.Where("session_id = ?", sessionID).First(&uploadSession)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Upload session not found"})
	}

	// Abort the upload in storage
	err := st.AbortChunkedUpload(sessionID)
	if err != nil {
		log.Printf("Failed to abort upload for session %s: %v", sessionID, err)
		// Continue anyway to update database
	}

	// Update session status
	uploadSession.Status = models.UploadSessionStatusCancelled
	db.Save(&uploadSession)

	log.Printf("Aborted upload session %s", sessionID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Upload cancelled"})
}
