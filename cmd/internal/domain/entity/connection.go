package entity

type Connection struct {
	ConnectionID string `gorm:"primaryKey;autoIncrement:false"`
	UserID       int    `gorm:"not null;index"`
	CreatedAt    int64  `gorm:"not null"`
}
