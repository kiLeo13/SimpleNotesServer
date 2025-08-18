package sqlite

import (
	"path/filepath"
	"simplenotes/internal/domain/entity"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() error {
	dbPath := filepath.Join(".", "database.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&entity.Note{}, &entity.NoteAlias{})
	if err != nil {
		return err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	return nil
}
