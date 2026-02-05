package entity

type Connection struct {
	ConnectionID string `gorm:"primaryKey;autoIncrement:false"`
	UserID       int    `gorm:"not null;index"`
	ExpiresAt    int64  `gorm:"not null"`
	CreatedAt    int64  `gorm:"not null"`
}
