package database

import (
	"github.com/nuuner/bindle-server/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("bindle.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.UploadedFile{}, &models.User{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
