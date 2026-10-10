package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"homework/internal/model"
)

type AuthorRepository struct{ db *gorm.DB }

func NewAuthorRepository(db *gorm.DB) *AuthorRepository { return &AuthorRepository{db: db} }

func (r *AuthorRepository) Create(ctx context.Context, author *model.Author) error {
	if err := r.db.WithContext(ctx).Create(author).Error; err != nil {
		return translateDBError(err)
	}
	return nil
}

func (r *AuthorRepository) FindByID(ctx context.Context, id uint64) (*model.Author, error) {
	var author model.Author
	if err := r.db.WithContext(ctx).First(&author, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, translateDBError(err)
	}
	return &author, nil
}

func (r *AuthorRepository) List(ctx context.Context) ([]model.Author, error) {
	var authors []model.Author
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&authors).Error; err != nil {
		return nil, translateDBError(err)
	}
	if authors == nil {
		authors = make([]model.Author, 0)
	}
	return authors, nil
}
