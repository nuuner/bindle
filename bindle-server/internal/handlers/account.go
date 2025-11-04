package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/internal/storage"
	"github.com/nuuner/bindle-server/pkg/limiter"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func GetMe(c *fiber.Ctx, db *gorm.DB) error {
	user := utils.GetUser(c)
	cfg := config.GetConfig()

	uploadedBytes, err := limiter.GetUploadedSizeForIP(db, c.IP())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get upload usage",
		})
	}

	limitBytes := int64(cfg.UploadLimitMBPerDay * 1000 * 1000)
	maxFileSizeBytes := int64(cfg.MaxFileSizeMB * 1000 * 1000)

	userDTO := models.UserDTO{
		AccountId: user.AccountId,
		LastLogin: user.LastLogin,
		Files:     user.Files,
	}

	meResponse := models.MeResponse{
		User:             userDTO,
		UploadedBytes:    uploadedBytes,
		UploadLimitBytes: limitBytes,
		MaxFileSizeBytes: maxFileSizeBytes,
	}

	return c.JSON(meResponse)
}

func DeleteAccount(c *fiber.Ctx, db *gorm.DB, storage storage.Storage) error {
	user := utils.GetUser(c)
	if user.ID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var files []models.UploadedFile
	if err := db.Where("owner_id = ?", user.ID).Find(&files).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user files",
		})
	}

	for _, file := range files {
		if err := DeleteFile(c, db, storage, file.FileId); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete user files",
			})
		}
	}

	if err := db.Where("account_id = ?", user.ID).Delete(&models.AccountIpConnection{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete IP connections",
		})
	}

	if err := db.Where("id = ?", user.ID).Delete(&models.User{}).Error; err != nil {
		log.Println("Failed to delete user account:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user account",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}
