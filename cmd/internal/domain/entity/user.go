package entity

// User is the general basic structure of all users across the platform
type User struct {
	ID            int64      `gorm:"primaryKey"`
	SubUUID       string     `gorm:"not null"`
	Username      string     `gorm:"not null"`
	Email         string     `gorm:"not null"`
	EmailVerified bool       `gorm:"not null"`
	Permissions   Permission `gorm:"not null;type:bigint;default:0"`
	Active        bool       `gorm:"not null;default:true"`
	Suspended     bool       `gorm:"not null;default:false"`
	CreatedAt     int64      `gorm:"not null"`
	UpdatedAt     int64      `gorm:"not null;autoUpdateTime:false"`
}
