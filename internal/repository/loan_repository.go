package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"homework/internal/model"
)

type LoanRepository struct{ db *gorm.DB }

func NewLoanRepository(db *gorm.DB) *LoanRepository { return &LoanRepository{db: db} }

func (r *LoanRepository) Create(ctx context.Context, userID, copyID uint64, dueAt time.Time) (*model.Loan, error) {
	loan := &model.Loan{UserID: userID, CopyID: copyID, CheckedOutAt: time.Now().UTC(), DueAt: dueAt.UTC()}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Copy{}).Where("id = ? AND status = ?", copyID, model.CopyAvailable).Update("status", model.CopyCheckedOut)
		if result.Error != nil {
			return translateDBError(result.Error)
		}
		if result.RowsAffected == 0 {
			var copy model.Copy
			if err := tx.First(&copy, copyID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrNotFound
				}
				return translateDBError(err)
			}
			return ErrConflict
		}
		if err := tx.Create(loan).Error; err != nil {
			return translateDBError(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return loan, nil
}

func (r *LoanRepository) ListByUser(ctx context.Context, userID uint64) ([]model.Loan, error) {
	var loans []model.Loan
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&loans).Error; err != nil {
		return nil, translateDBError(err)
	}
	if loans == nil {
		loans = make([]model.Loan, 0)
	}
	return loans, nil
}

func (r *LoanRepository) Return(ctx context.Context, id uint64) (*model.Loan, error) {
	var loan model.Loan
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return translateDBError(err)
		}
		if loan.ReturnedAt != nil {
			return ErrConflict
		}
		now := time.Now().UTC()
		if err := tx.Model(&loan).Updates(map[string]any{"returned_at": now}).Error; err != nil {
			return translateDBError(err)
		}
		result := tx.Model(&model.Copy{}).Where("id = ? AND status = ?", loan.CopyID, model.CopyCheckedOut).Update("status", model.CopyAvailable)
		if result.Error != nil {
			return translateDBError(result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrConflict
		}
		loan.ReturnedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &loan, nil
}
