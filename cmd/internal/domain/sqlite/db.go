package sqlite

import (
	"os"
	"path/filepath"
	"simplenotes/cmd/internal/domain/entity"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Init() (*gorm.DB, error) {
	dataDir := os.Getenv("SQLITE_PATH")
	if dataDir == "" {
		dataDir = "/data"
	}
	dbPath := filepath.Join(dataDir, "database.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&entity.Note{},
		&entity.User{},
		&entity.Connection{},
		&entity.Company{},
	)
	if err != nil {
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
