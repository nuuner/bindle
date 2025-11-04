package handlers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/internal/storage"
	"github.com/nuuner/bindle-server/pkg/limiter"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func UploadFile(c *fiber.Ctx, db *gorm.DB, cfg *config.Config, storage storage.Storage) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No Content-Type header"})
	}

	if limiter.ShouldThrottle(c, db, cfg, file.Size) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Upload limit exceeded"})
	}

	hash, err := utils.GetFileHash(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash file"})
	}

	filePath := hash + filepath.Ext(file.Filename)

	existingFile := db.Where("file_path = ?", filePath).First(&models.UploadedFile{})
	if existingFile.Error != nil {
		_, err := storage.SaveFile(file, filePath)
		if err != nil {
			log.Println("error saving file", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
		}
	}

	guid, err := uuid.NewV7()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate file ID"})
	}

	fileToCreate := &models.UploadedFile{
		FileId:   guid.String(),
		FilePath: filePath,
		FileName: file.Filename,
		Size:     file.Size,
		Type:     utils.GetFileType(mimeType),
		MimeType: mimeType,
		Owner:    utils.GetUser(c),
	}

	result := db.Create(fileToCreate)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create file"})
	}

	return c.Status(fiber.StatusOK).JSON(fileToCreate)
}

func DeleteFile(c *fiber.Ctx, db *gorm.DB, storage storage.Storage, fileId string) error {
	user := utils.GetUser(c)
	uploadedFile := &models.UploadedFile{}
	db.Where("file_id = ? AND owner_id = ?", fileId, user.ID).First(uploadedFile)
	filePath := uploadedFile.FilePath

	db.Delete(&models.UploadedFile{}, "file_id = ? AND owner_id = ?", fileId, user.ID)
	log.Printf("Deleted file %s for user %d", filePath, user.ID)

	existingFile := db.Where("file_path = ?", filePath).First(&models.UploadedFile{})
	if existingFile.Error != nil {
		if err := storage.DeleteFile(filePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete file"})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "File deleted"})
}

func UpdateFile(c *fiber.Ctx, db *gorm.DB) error {
	file := &models.UploadedFileDTO{}
	if err := c.BodyParser(file); err != nil {
		log.Println("error parsing file", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse file"})
	}

	existingFile := &models.UploadedFile{}
	if err := db.First(existingFile, "file_id = ? AND owner_id = ?", file.FileId, utils.GetUser(c).ID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
	}

	// Only allow updating the file name
	existingFile.FileName = file.FileName

	result := db.Save(existingFile)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update file"})
	}

	return c.Status(fiber.StatusOK).JSON(file)
}

func GetFile(c *fiber.Ctx, db *gorm.DB, storage storage.Storage, filePath string) error {
	// Query database to get file metadata (including chunk count and MIME type)
	var uploadedFile models.UploadedFile
	result := db.Where("file_path = ?", filePath).First(&uploadedFile)

	var reader io.ReadCloser
	var fileSize int64
	var mimeType string
	var fileName string
	var err error

	if result.Error != nil {
		// File not in database, but might exist in storage (backward compatibility)
		// Try to retrieve with chunkCount=0 (legacy single-file upload)
		reader, fileSize, err = storage.GetFileStream(filePath, 0)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
		}
		fileName = filePath
		mimeType = "" // Will detect from content
	} else {
		// Get file stream with chunk count for proper decryption
		reader, fileSize, err = storage.GetFileStream(filePath, uploadedFile.ChunkCount)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
		}
		// Use stored metadata from database
		fileName = uploadedFile.FileName
		mimeType = uploadedFile.MimeType
	}
	// Note: We don't defer close here because SendStream will close the reader when done

	// If MIME type not known, detect from first bytes
	if mimeType == "" {
		// Read first 512 bytes for MIME detection
		header := make([]byte, 512)
		n, readErr := io.ReadFull(reader, header)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			reader.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read file"})
		}

		mimeType = utils.BytesToMimeType(header[:n])

		// Create new reader that includes the header we read + rest of original
		reader = &combinedReadCloser{
			Reader: io.MultiReader(bytes.NewReader(header[:n]), reader),
			Closer: reader,
		}
	}

	// Set response headers
	c.Set("Content-Type", mimeType)
	if fileName != "" {
		c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fileName))
	}

	// Test read to catch decryption errors early
	testBuf := make([]byte, 1)
	_, testErr := reader.Read(testBuf)
	if testErr != nil && testErr != io.EOF {
		log.Printf("ERROR: Failed to read from decryption reader: %v", testErr)
		reader.Close()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to decrypt file: %v", testErr)})
	}

	// Create new reader that includes the test byte + rest of stream
	reader = &combinedReadCloser{
		Reader: io.MultiReader(bytes.NewReader(testBuf), reader),
		Closer: reader,
	}

	// Stream file to client (memory-efficient - only ~20MB for any file size)
	return c.SendStream(reader, int(fileSize))
}

// combinedReadCloser combines a Reader and Closer for MIME detection
type combinedReadCloser struct {
	io.Reader
	io.Closer
}
