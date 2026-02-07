package entity

const HeartbeatPeriodMillis = 60 * 1000
const HeartbeatMissToleranceMillis = 10 * 1000

type Connection struct {
	ConnectionID    string `gorm:"primaryKey;autoIncrement:false"`
	UserID          int    `gorm:"not null;index"`
	ExpiresAt       int64  `gorm:"not null"`
	LastHeartbeatAt int64  `gorm:"not null;index"`
	CreatedAt       int64  `gorm:"not null"`
}
