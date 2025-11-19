package entity

type Note struct {
	ID          int    `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Content     string `gorm:"not null"`
	CreatedByID int    `gorm:"not null"` // References: users(id)
	Tags        string `gorm:"not null"`
	NoteType    string `gorm:"not null"`
	ContentSize int    `gorm:"not null"`
	Visibility  string `gorm:"not null"`
	CreatedAt   int64  `gorm:"not null"`
	UpdatedAt   int64  `gorm:"not null;autoUpdateTime:false"`

	// Relations
	CreatedBy User `gorm:"foreignKey:CreatedByID;references:ID"`
}
