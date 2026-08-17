package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

// lastLoginResolution is how far LastLogin is allowed to lag behind the real last
// request. Account expiration is the only thing reading it, in days.
const lastLoginResolution = time.Minute

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

		// Deliberately not preloading Files. This runs on every authenticated request,
		// including every chunk of an upload, and only GetMe wants the file list - which
		// loads it itself rather than making every other request pay for it.
		var user models.User
		result := db.Where("account_id = ?", authHeader).First(&user)
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

		// Only touched when it has gone stale. An upload is a long run of requests from
		// one account, and writing this on each of them put a synchronous database write
		// on the critical path of every chunk to move a timestamp by milliseconds. It
		// feeds account expiration, which is measured in days.
		if time.Since(user.LastLogin) > lastLoginResolution {
			user.LastLogin = time.Now()
			db.Save(&user)
		}

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
