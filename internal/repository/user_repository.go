package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"homework/internal/model"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return translateDBError(err)
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, translateDBError(err)
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, translateDBError(err)
	}
	return &user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).Select("id", "email", "role", "created_at", "updated_at").Order("id ASC").Find(&users).Error; err != nil {
		return nil, translateDBError(err)
	}
	if users == nil {
		users = make([]model.User, 0)
	}
	return users, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, id uint64, role string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("role", role)
	if result.Error != nil {
		return translateDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
