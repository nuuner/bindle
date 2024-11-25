package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/pkg/utils"
	"gorm.io/gorm"
)

func GetMe(c *fiber.Ctx, db *gorm.DB) error {
	user := utils.GetUser(c)
	println("GetMe: User:", user.ID)
	return c.JSON(user)
}
