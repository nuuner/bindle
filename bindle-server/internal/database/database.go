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

	// WAL lets reads run while a write is in flight, which matters because uploads read
	// the session row on every chunk. synchronous=NORMAL drops the fsync per commit -
	// under WAL that risks losing the last commits on a machine crash, never a corrupt
	// database, which is the right trade for rows describing transient uploads.
	// busy_timeout keeps concurrent writers waiting briefly instead of failing outright.
	dsn := "./storage/bindle.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000"

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
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
