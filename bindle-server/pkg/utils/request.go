package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/models"
)

func GetUser(c *fiber.Ctx) models.User {
	return c.Locals("user").(models.User)
}
