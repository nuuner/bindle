package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/limiter"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func GetMe(c *fiber.Ctx, db *gorm.DB) error {
	user := utils.GetUser(c)
	cfg := config.GetConfig()

	// Get uploaded size for the IP
	uploadedBytes, err := limiter.GetUploadedSizeForIP(db, c.IP())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get upload usage",
		})
	}

	limitBytes := int64(cfg.UploadLimitMBPerDay * 1000 * 1000) // Convert MB to bytes

	return c.JSON(fiber.Map{
		"user":             user,
		"uploadedBytes":    uploadedBytes,
		"uploadLimitBytes": limitBytes,
	})
}
