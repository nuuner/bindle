package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

func AdminAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		adminPassword := os.Getenv("ADMIN_PASSWORD")

		// If no admin password is set, deny access
		if adminPassword == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access not configured",
			})
		}

		// Get password from header
		providedPassword := c.Get("X-Admin-Password")

		if providedPassword == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Admin password required",
			})
		}

		// Check if password matches
		if providedPassword != adminPassword {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid admin password",
			})
		}

		return c.Next()
	}
}
