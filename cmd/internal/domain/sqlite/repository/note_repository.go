package repository

import (
	"errors"
	"gorm.io/gorm"
	"simplenotes/internal/domain/entity"
)

type DefaultNoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) *DefaultNoteRepository {
	return &DefaultNoteRepository{db: db}
}

func (d *DefaultNoteRepository) FindAll() ([]*entity.Note, error) {
	var notes []*entity.Note
	err := d.db.Find(notes).Error
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
