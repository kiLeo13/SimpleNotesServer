package entity

type Note struct {
	ID          int    `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	CreatedByID int    `gorm:"not null"` // References: users(id)
	Content     string `gorm:"not null"`
	Tags        string `gorm:"not null"`
	IsPrivate   bool   `gorm:"not null"`
	S3Key       string `gorm:"not null"`
	CreatedAt   int64  `gorm:"not null"`
	UpdatedAt   int64  `gorm:"not null"`

	// Relations
	CreatedBy User `gorm:"foreignKey:CreatedByID;references:ID"`
}
