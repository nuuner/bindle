package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// The request does not have a user, return a new id
			authHeader = utils.GenerateAccountId()
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

		// Update last login time
		user.LastLogin = time.Now()
		db.Save(&user)

		// Check if IP connection already exists
		ipAddress := c.IP()
		var existingConnection models.AccountIpConnection
		result = db.Where("account_id = ? AND ip_address = ?", user.ID, ipAddress).First(&existingConnection)

		// Only create new connection if it doesn't exist
		if result.Error == gorm.ErrRecordNotFound {
			db.Create(&models.AccountIpConnection{AccountID: user.ID, IPAddress: ipAddress})
		}

		c.Locals("user", user)
		return c.Next()
	}
}
