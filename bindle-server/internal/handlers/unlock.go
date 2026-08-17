package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/pkg/unlock"
)

// UnlockLimits trades the shared password for a cookie that lifts the daily upload
// quota. The route is behind the aggressive rate limiter, which is what keeps a single
// password from being brute forced.
func UnlockLimits(c *fiber.Ctx, cfg *config.Config) error {
	if cfg.UnlockPassword == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Unlocking limits is not configured",
		})
	}

	req := new(struct {
		Password string `json:"password"`
	})
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if !unlock.PasswordMatches(cfg.UnlockPassword, req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Incorrect password"})
	}

	expiresAt := time.Now().Add(unlock.TokenLifetime)
	unlock.SetCookie(c, cfg, expiresAt)

	return c.JSON(fiber.Map{
		"limitsUnlocked": true,
		"expiresAt":      expiresAt,
	})
}

// LockLimits drops the cookie again, putting this browser back under the daily quota.
func LockLimits(c *fiber.Ctx, cfg *config.Config) error {
	unlock.ClearCookie(c, cfg)

	return c.JSON(fiber.Map{"limitsUnlocked": false})
}
