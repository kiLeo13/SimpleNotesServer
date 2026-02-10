package repository

import (
	"errors"
	"gorm.io/gorm"
	"simplenotes/cmd/internal/domain/entity"
)

type DefaultNoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) *DefaultNoteRepository {
	return &DefaultNoteRepository{db: db}
}

func (d *DefaultNoteRepository) FindAll(withPrivate bool) ([]*entity.Note, error) {
	var notes []*entity.Note
	var err error
	if withPrivate {
		err = d.db.Find(&notes).Error
	} else {
		err = d.db.
			Where("visibility != ?", string(entity.VisibilityPrivate)).
			Find(&notes).Error
	}

	if err != nil {
		return nil, err
	}
	return notes, nil
}

func (d *DefaultNoteRepository) FindByID(id int) (*entity.Note, error) {
	var note entity.Note
	err := d.db.First(&note, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (d *DefaultNoteRepository) Save(note *entity.Note) error {
	return d.db.Save(note).Error
}

func (d *DefaultNoteRepository) Delete(note *entity.Note) error {
	return d.db.Delete(note).Error
}
