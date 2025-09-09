package sqlite

import (
	"path/filepath"
	"simplenotes/cmd/internal/domain/entity"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Init() (*gorm.DB, error) {
	// "/data" is a Docker volume mount defined in docker-compose.yml,
	// so the DB file persists across container restarts and redeploys.
	dbPath := filepath.Join("/data", "database.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&entity.Note{}, &entity.User{})
	if err != nil {
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
