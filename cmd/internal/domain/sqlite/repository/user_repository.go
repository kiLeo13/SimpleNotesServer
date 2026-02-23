package repository

import (
	"errors"
	"gorm.io/gorm"
	"simplenotes/cmd/internal/domain/entity"
	"simplenotes/cmd/internal/utils"
)

type DefaultUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *DefaultUserRepository {
	return &DefaultUserRepository{db: db}
}

// FindAllActive returns only users where Active is true
func (u *DefaultUserRepository) FindAllActive() ([]*entity.User, error) {
	var users []*entity.User
	err := u.db.Where("active = ?", true).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (u *DefaultUserRepository) FindActiveByID(id int) (*entity.User, error) {
	var user entity.User
	err := u.db.Where("active = ?", true).First(&user, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *DefaultUserRepository) FindByID(id int) (*entity.User, error) {
	var user entity.User
	err := u.db.First(&user, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *DefaultUserRepository) FindActiveByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := u.db.Where("email = ? AND active = ?", email, true).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *DefaultUserRepository) FindActiveBySub(sub string) (*entity.User, error) {
	var user entity.User
	err := u.db.Where("sub_uuid = ? AND active = ?", sub, true).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ExistsActiveByEmail check if the given email is held by some user.
func (u *DefaultUserRepository) ExistsActiveByEmail(email string) (bool, error) {
	var exists int64
	err := u.db.Model(&entity.User{}).
		Where("email = ? AND active = ?", email, true).
		Count(&exists).Error

	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// SoftDelete sets the active flag to false.
func (u *DefaultUserRepository) SoftDelete(user *entity.User) error {
	user.Active = false
	user.Email = ""
	user.SubUUID = ""
	user.UpdatedAt = utils.NowUTC()

	return u.db.Model(user).
		Select("Active", "Email", "SubUUID", "UpdatedAt").
		Updates(user).Error
}

func (u *DefaultUserRepository) FetchAllActiveOnline() ([]*entity.User, error) {
	var users []*entity.User
	err := u.db.Table("users").
		Joins("INNER JOIN connections ON connections.user_id = users.id").
		Where("users.active = ?", true).
		Distinct("users.*").
		Find(&users).Error

	if err != nil {
		return nil, err
	}
	return users, nil
}

func (u *DefaultUserRepository) Save(user *entity.User) error {
	return u.db.Save(user).Error
}
