package repository

import (
	"errors"
	"gorm.io/gorm"
	"simplenotes/cmd/internal/domain/entity"
)

type DefaultCompanyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) *DefaultCompanyRepository {
	return &DefaultCompanyRepository{db: db}
}

func (r *DefaultCompanyRepository) FindByCNPJ(cnpj string) (*entity.Company, error) {
	var company entity.Company
	err := r.db.
		Preload("CompanyPartners").
		Where("cnpj = ?", cnpj).
		First(&company).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *DefaultCompanyRepository) Save(company *entity.Company) error {
	return r.db.Save(company).Error
}

func (r *DefaultCompanyRepository) DeleteExpired(before int64) error {
	return r.db.
		Where("cached_at < ?", before).
		Delete(&entity.Company{}).Error
}
