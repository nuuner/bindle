package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/internal/storage"
	"gorm.io/gorm"
)

// Admin DTOs
type AdminUserDTO struct {
	AccountId    string   `json:"accountId"`
	FileCount    int      `json:"fileCount"`
	StorageUsage int64    `json:"storageUsage"` // in bytes
	LastLogin    string   `json:"lastLogin"`
	IPAddresses  []string `json:"ipAddresses"`
}

type AdminFileDTO struct {
	FileId     string `json:"fileId"`
	FileName   string `json:"fileName"`
	FilePath   string `json:"filePath"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
	MimeType   string `json:"mimeType"`
	OwnerID    uint   `json:"ownerId"`
	AccountId  string `json:"accountId"`
	ChunkCount int    `json:"chunkCount"`
	CreatedAt  string `json:"createdAt"`
}

// ListAllUsers returns all users with their statistics
func ListAllUsers(c *fiber.Ctx, db *gorm.DB) error {
	var users []models.User
	if err := db.Preload("Files").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	adminUsers := make([]AdminUserDTO, 0)

	for _, user := range users {
		// Calculate file count and storage usage
		fileCount := len(user.Files)
		var storageUsage int64
		for _, file := range user.Files {
			storageUsage += file.Size
		}

		// Get IP addresses for this user
		var ipConnections []models.AccountIpConnection
		db.Where("account_id = ?", user.ID).Find(&ipConnections)

		ipAddresses := make([]string, 0)
		for _, conn := range ipConnections {
			ipAddresses = append(ipAddresses, conn.IPAddress)
		}

		adminUsers = append(adminUsers, AdminUserDTO{
			AccountId:    user.AccountId,
			FileCount:    fileCount,
			StorageUsage: storageUsage,
			LastLogin:    user.LastLogin.Format("2006-01-02 15:04:05"),
			IPAddresses:  ipAddresses,
		})
	}

	return c.JSON(adminUsers)
}

// ListAllFiles returns all files in the system with owner information
func ListAllFiles(c *fiber.Ctx, db *gorm.DB) error {
	var files []models.UploadedFile
	if err := db.Preload("Owner").Find(&files).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch files",
		})
	}

	adminFiles := make([]AdminFileDTO, 0)

	for _, file := range files {
		adminFiles = append(adminFiles, AdminFileDTO{
			FileId:     file.FileId,
			FileName:   file.FileName,
			FilePath:   file.FilePath,
			Size:       file.Size,
			Type:       string(file.Type),
			MimeType:   file.MimeType,
			OwnerID:    file.OwnerID,
			AccountId:  file.Owner.AccountId,
			ChunkCount: file.ChunkCount,
			CreatedAt:  file.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return c.JSON(adminFiles)
}

// AdminDeleteFile deletes a specific file (admin version - no owner check)
func AdminDeleteFile(c *fiber.Ctx, db *gorm.DB, storage storage.Storage, fileId string) error {
	uploadedFile := &models.UploadedFile{}
	if err := db.Where("file_id = ?", fileId).First(uploadedFile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	filePath := uploadedFile.FilePath

	// Delete the database record
	if err := db.Delete(uploadedFile).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete file record",
		})
	}

	log.Printf("Admin deleted file %s (ID: %s)", filePath, fileId)

	// Check if any other files reference this physical file
	var count int64
	db.Model(&models.UploadedFile{}).Where("file_path = ?", filePath).Count(&count)

	// If no other files reference it, delete the physical file
	if count == 0 {
		if err := storage.DeleteFile(filePath); err != nil {
			log.Printf("Warning: Failed to delete physical file %s: %v", filePath, err)
			// Don't return error - the database record is already deleted
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "File deleted successfully",
	})
}

// DeleteUserFiles deletes all files for a specific user
func DeleteUserFiles(c *fiber.Ctx, db *gorm.DB, storage storage.Storage, accountId string) error {
	// Find the user
	var user models.User
	if err := db.Where("account_id = ?", accountId).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Get all files for this user
	var files []models.UploadedFile
	if err := db.Where("owner_id = ?", user.ID).Find(&files).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user files",
		})
	}

	deletedCount := 0
	for _, file := range files {
		filePath := file.FilePath

		// Delete the database record
		if err := db.Delete(&file).Error; err != nil {
			log.Printf("Warning: Failed to delete file record %s: %v", file.FileId, err)
			continue
		}

		// Check if any other files reference this physical file
		var count int64
		db.Model(&models.UploadedFile{}).Where("file_path = ?", filePath).Count(&count)

		// If no other files reference it, delete the physical file
		if count == 0 {
			if err := storage.DeleteFile(filePath); err != nil {
				log.Printf("Warning: Failed to delete physical file %s: %v", filePath, err)
			}
		}

		deletedCount++
	}

	log.Printf("Admin deleted %d files for user %s", deletedCount, accountId)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User files deleted successfully",
		"count":   deletedCount,
	})
}

// DeleteAllFiles deletes ALL files in the system (nuclear option)
func DeleteAllFiles(c *fiber.Ctx, db *gorm.DB, storage storage.Storage) error {
	// Get all files
	var files []models.UploadedFile
	if err := db.Find(&files).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch files",
		})
	}

	// Track unique file paths to delete physical files
	uniqueFilePaths := make(map[string]bool)
	for _, file := range files {
		uniqueFilePaths[file.FilePath] = true
	}

	// Delete all file records from database
	if err := db.Where("1 = 1").Delete(&models.UploadedFile{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete file records",
		})
	}

	// Delete all physical files
	deletedCount := 0
	failedCount := 0
	for filePath := range uniqueFilePaths {
		if err := storage.DeleteFile(filePath); err != nil {
			log.Printf("Warning: Failed to delete physical file %s: %v", filePath, err)
			failedCount++
		} else {
			deletedCount++
		}
	}

	log.Printf("Admin deleted ALL files: %d records, %d physical files deleted, %d failed",
		len(files), deletedCount, failedCount)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "All files deleted successfully",
		"recordsDeleted":  len(files),
		"physicalDeleted": deletedCount,
		"physicalFailed":  failedCount,
	})
}
