package entity

import "time"

const (
	HeartbeatPeriod    = 60 * time.Second
	HeartbeatTolerance = 10 * time.Second

	HeartbeatPeriodMillis    = int64(60 * 1000)
	HeartbeatToleranceMillis = int64(10 * 1000)
)

type Connection struct {
	ConnectionID    string `gorm:"primaryKey;autoIncrement:false"`
	UserID          int64  `gorm:"not null;index"`
	ExpiresAt       int64  `gorm:"not null"`
	LastHeartbeatAt int64  `gorm:"not null;index"`
	CreatedAt       int64  `gorm:"not null"`
}
