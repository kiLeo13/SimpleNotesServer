package entity

// User is the general basic structure of all users across the platform
type User struct {
	ID            int    `gorm:"primaryKey"`
	SubUUID       string `gorm:"not null"`
	Username      string `gorm:"not null"`
	Email         string `gorm:"not null"`
	EmailVerified bool   `gorm:"not null"`
	IsAdmin       bool   `gorm:"not null"`
	CreatedAt     int64  `gorm:"not null"`
	UpdatedAt     int64  `gorm:"not null"`
}
