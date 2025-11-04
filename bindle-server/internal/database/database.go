package database

import (
	"os"

	"github.com/nuuner/bindle-server/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDatabase() (*gorm.DB, error) {
	// ensure storage directory exists
	os.MkdirAll("storage", os.ModePerm)

	db, err := gorm.Open(sqlite.Open("./storage/bindle.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.UploadedFile{}, &models.User{}, &models.AccountIpConnection{}, &models.UploadSession{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
