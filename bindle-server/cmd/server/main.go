package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/database"
	"github.com/nuuner/bindle-server/internal/handlers"
	"github.com/nuuner/bindle-server/internal/middleware"
	"github.com/nuuner/bindle-server/internal/storage"
)

func main() {
	if os.Getenv("ENVIRONMENT") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("failed to load environment variables:", err)
		}
	}

	config := config.GetConfig()

	var storageInstance storage.Storage
	var err error

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

	// Global rate limiter for all routes
	app.Use(limiter.New(limiter.Config{
		Max:        100, // 100 requests
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // Rate limit by IP address
		},
	}))

	// More aggressive rate limiting for sensitive operations
	sensitiveRateLimiter := limiter.New(limiter.Config{
		Max:        5, // 5 requests
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	// Setup static file serving for uploaded files
	app.Get("/files/:filePath", func(c *fiber.Ctx) error {
		// Disable scripts on possible html files
		c.Set("Content-Security-Policy", "script-src 'none'")

		if !config.S3Enabled {
			return c.SendFile(config.FilesystemPath + "/" + c.Params("filePath"))
		} else {
			return handlers.GetFile(c, storageInstance, c.Params("filePath"))
		}
	})

	// Serve static files from the React build
	app.Static("/", "./static")

	// API routes
	api := app.Group("/api")
	api.Use(middleware.AuthMiddleware(db))

	api.Get("/me", func(c *fiber.Ctx) error {
		return handlers.GetMe(c, db)
	})
	api.Delete("/me", sensitiveRateLimiter, func(c *fiber.Ctx) error {
		return handlers.DeleteAccount(c, db, storageInstance)
	})
	api.Post("/file", sensitiveRateLimiter, func(c *fiber.Ctx) error {
		return handlers.UploadFile(c, db, &config, storageInstance)
	})
	api.Delete("/file/:fileId", sensitiveRateLimiter, func(c *fiber.Ctx) error {
		return handlers.DeleteFile(c, db, storageInstance, c.Params("fileId"))
	})
	api.Put("/file", sensitiveRateLimiter, func(c *fiber.Ctx) error {
		return handlers.UpdateFile(c, db)
	})

	// Handle SPA routing - serve index.html for all non-API routes
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./static/index.html")
	})

	// Start server
	log.Fatal(app.Listen(":3000"))
}
