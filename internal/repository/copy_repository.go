package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"homework/internal/model"
)

type CopyRepository struct{ db *gorm.DB }

func NewCopyRepository(db *gorm.DB) *CopyRepository { return &CopyRepository{db: db} }

func (r *CopyRepository) Create(ctx context.Context, copy *model.Copy) error {
	if err := r.db.WithContext(ctx).Create(copy).Error; err != nil {
		return translateDBError(err)
	}
	return nil
}

func (r *CopyRepository) ListByBook(ctx context.Context, bookID uint64) ([]model.Copy, error) {
	var copies []model.Copy
	if err := r.db.WithContext(ctx).Where("book_id = ?", bookID).Order("id ASC").Find(&copies).Error; err != nil {
		return nil, translateDBError(err)
	}
	if copies == nil {
		copies = make([]model.Copy, 0)
	}
	return copies, nil
}

// BookAvailability summarises copy counts for one book.
type BookAvailability struct {
	Total     int64
	Available int64
}

// AvailabilityByBooks returns copy counts for many books in a single query.
func (r *CopyRepository) AvailabilityByBooks(ctx context.Context, bookIDs []uint64) (map[uint64]BookAvailability, error) {
	result := make(map[uint64]BookAvailability, len(bookIDs))
	if len(bookIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		BookID    uint64
		Total     int64
		Available int64
	}
	err := r.db.WithContext(ctx).Model(&model.Copy{}).
		Select("book_id, COUNT(*) AS total, COUNT(*) FILTER (WHERE status = ?) AS available", model.CopyAvailable).
		Where("book_id IN ?", bookIDs).Group("book_id").Scan(&rows).Error
	if err != nil {
		return nil, translateDBError(err)
	}
	for _, row := range rows {
		result[row.BookID] = BookAvailability{Total: row.Total, Available: row.Available}
	}
	return result, nil
}

func (r *CopyRepository) Availability(ctx context.Context, bookID uint64) (total, available int64, err error) {
	var counts struct {
		Total     int64
		Available int64
	}
	err = r.db.WithContext(ctx).Model(&model.Copy{}).
		Select("COUNT(*) AS total, COUNT(*) FILTER (WHERE status = ?) AS available", model.CopyAvailable).
		Where("book_id = ?", bookID).Scan(&counts).Error
	if err != nil {
		return 0, 0, translateDBError(err)
	}
	var book model.Book
	if err := r.db.WithContext(ctx).Select("id").First(&book, bookID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, ErrNotFound
		}
		return 0, 0, translateDBError(err)
	}
	return counts.Total, counts.Available, nil
}
