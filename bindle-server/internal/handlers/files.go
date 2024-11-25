package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func UploadFile(c *fiber.Ctx, db *gorm.DB) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No Content-Type header"})
	}

	hash, err := utils.GetFileHash(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash file"})
	}

	existingFile := db.Where("hash = ?", hash).First(&models.UploadedFile{})
	if existingFile.Error != nil {
		if err := c.SaveFile(file, "files/"+hash); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
		}
	}

	guid, err := uuid.NewV7()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate file ID"})
	}

	fileToCreate := &models.UploadedFile{
		FileId:   guid.String(),
		Hash:     hash,
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
