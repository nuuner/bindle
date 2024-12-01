package handlers

import (
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

	if limiter.ShouldThrottle(c, db, cfg) {
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
	uploadedFile := &models.UploadedFile{}
	db.Where("file_id = ?", fileId).First(uploadedFile)
	filePath := uploadedFile.FilePath

	db.Delete(&models.UploadedFile{}, "file_id = ?", fileId)

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

func GetFile(c *fiber.Ctx, storage storage.Storage, filePath string) error {
	fileBytes, err := storage.GetFile(filePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
	}

	mimeType := utils.BytesToMimeType(fileBytes)
	c.Set("Content-Type", mimeType)

	return c.Send(fileBytes)
}
