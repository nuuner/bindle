package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/nuuner/bindle-server/internal/database"
	"github.com/nuuner/bindle-server/internal/handlers"
	"github.com/nuuner/bindle-server/internal/middleware"
	"github.com/nuuner/bindle-server/pkg/utils"
)

func main() {
	// Initialize database
	db, err := database.InitDatabase()
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// Initialize Fiber with config
	app := fiber.New(fiber.Config{
		BodyLimit: 50 * 1024 * 1024, // 50MB limit
	})

	// Add middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173",
		AllowCredentials: true,
	}))
	app.Use(logger.New())

	// Setup static file serving
	app.Static("/files", "files")

	app.Use(middleware.AuthMiddleware(db))

	// Ensure files directory exists
	if err := utils.EnsureFileDirectory(); err != nil {
		log.Fatal("failed to create files directory:", err)
	}

	// Setup routes
	app.Get("/api/me", func(c *fiber.Ctx) error {
		return handlers.GetMe(c, db)
	})
	app.Post("/api/file", func(c *fiber.Ctx) error {
		return handlers.UploadFile(c, db)
	})
	app.Delete("/api/file/:fileId", func(c *fiber.Ctx) error {
		return handlers.DeleteFile(c, db, c.Params("fileId"))
	})

	// Start server
	log.Fatal(app.Listen(":3000"))
}
