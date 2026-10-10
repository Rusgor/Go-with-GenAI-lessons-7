package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"homework/internal/model"
)

type CategoryRepository struct{ db *gorm.DB }

func NewCategoryRepository(db *gorm.DB) *CategoryRepository { return &CategoryRepository{db: db} }

func (r *CategoryRepository) Create(ctx context.Context, category *model.Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return translateDBError(err)
	}
	return nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id uint64) (*model.Category, error) {
	var category model.Category
	if err := r.db.WithContext(ctx).First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, translateDBError(err)
	}
	return &category, nil
}

func (r *CategoryRepository) List(ctx context.Context) ([]model.Category, error) {
	var categories []model.Category
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&categories).Error; err != nil {
		return nil, translateDBError(err)
	}
	if categories == nil {
		categories = make([]model.Category, 0)
	}
	return categories, nil
}
