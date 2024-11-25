package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}

		if !utils.AccountIdIsValid(authHeader) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid Authorization header format",
			})
		}

		var user models.User
		result := db.Preload("Files").Where("account_id = ?", authHeader).First(&user)
		if result.Error == gorm.ErrRecordNotFound {
			user = models.User{AccountId: authHeader}
			if err := db.Create(&user).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create user",
				})
			}
		} else if result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database error",
			})
		}

		c.Locals("user", user)
		return c.Next()
	}
}
