package entity

// User is meant to be immutable
type User struct {
	ID            int    `gorm:"primaryKey"`
	SubUUID       string `gorm:"not null"`
	Username      string `gorm:"not null"`
	EmailVerified bool   `gorm:"not null"`
	IsAdmin       bool   `gorm:"not null"`
	CreatedAt     int64  `gorm:"not null"`
	UpdatedAt     int64  `gorm:"not null"`
}
