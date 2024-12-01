package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/database"
	"github.com/nuuner/bindle-server/internal/handlers"
	"github.com/nuuner/bindle-server/internal/middleware"
	"github.com/nuuner/bindle-server/internal/storage"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("failed to load environment variables:", err)
	}

	config := config.GetConfig()

	var storageInstance storage.Storage

	if config.S3Enabled {
		storageInstance, err = storage.NewS3Storage(config)
		if err != nil {
			log.Fatal("failed to create S3 storage:", err)
		}
	} else {
		storageInstance, err = storage.NewFilesystemStorage(config)
		if err != nil {
			log.Fatal("failed to create filesystem storage:", err)
		}
	}

	// Initialize database
	db, err := database.InitDatabase()
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// Initialize Fiber with config
	app := fiber.New(fiber.Config{
		BodyLimit: int(config.RequestSizeLimitMB) * 1024 * 1024,
	})

	// Add middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     config.ClientOrigin,
		AllowCredentials: true,
	}))
	app.Use(logger.New())

	// Setup static file serving
	app.Get("/files/:filePath", func(c *fiber.Ctx) error {
		if !config.S3Enabled {
			return c.SendFile(config.FilesystemPath + "/" + c.Params("filePath"))
		} else {
			return handlers.GetFile(c, storageInstance, c.Params("filePath"))
		}
	})

	app.Use(middleware.AuthMiddleware(db))

	// Setup routes
	app.Get("/api/me", func(c *fiber.Ctx) error {
		return handlers.GetMe(c, db)
	})
	app.Post("/api/file", func(c *fiber.Ctx) error {
		return handlers.UploadFile(c, db, &config, storageInstance)
	})
	app.Delete("/api/file/:fileId", func(c *fiber.Ctx) error {
		return handlers.DeleteFile(c, db, storageInstance, c.Params("fileId"))
	})
	app.Put("/api/file", func(c *fiber.Ctx) error {
		return handlers.UpdateFile(c, db)
	})

	// Start server
	log.Fatal(app.Listen(":3000"))
}
