package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"homework/internal/model"
)

type BookFilter struct {
	CategoryID uint64
	// Query is a case-insensitive substring matched against title, ISBN and author names.
	Query string
	Page  int
	Limit int
}

type BookRepository interface {
	Create(context.Context, *model.Book) error
	FindByID(context.Context, uint64) (*model.Book, error)
	Update(context.Context, *model.Book) error
	Delete(context.Context, uint64) error
	List(context.Context, BookFilter) ([]model.Book, error)
}

type GormBookRepository struct {
	db *gorm.DB
}

func NewGormBookRepository(db *gorm.DB) *GormBookRepository {
	return &GormBookRepository{db: db}
}

var _ BookRepository = (*GormBookRepository)(nil)

func (r *GormBookRepository) Create(ctx context.Context, book *model.Book) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		authors, err := r.loadAuthors(ctx, tx, book.Authors)
		if err != nil {
			return err
		}
		book.Authors = authors
		if err := tx.Omit(clause.Associations).Create(book).Error; err != nil {
			return translateDBError(err)
		}
		if len(authors) > 0 {
			if err := tx.Model(book).Association("Authors").Append(&authors); err != nil {
				return translateDBError(err)
			}
		}
		return nil
	})
}

func (r *GormBookRepository) FindByID(ctx context.Context, id uint64) (*model.Book, error) {
	var book model.Book
	result := r.db.WithContext(ctx).Preload("Authors").Preload("Category").First(&book, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if result.Error != nil {
		return nil, translateDBError(result.Error)
	}
	return &book, nil
}

func (r *GormBookRepository) Update(ctx context.Context, book *model.Book) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Book
		if err := tx.First(&current, book.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return translateDBError(err)
		}
		authors, err := r.loadAuthors(ctx, tx, book.Authors)
		if err != nil {
			return err
		}
		updates := map[string]any{
			"title":          book.Title,
			"isbn":           book.ISBN,
			"category_id":    book.CategoryID,
			"published_year": book.PublishedYear,
			"description":    book.Description,
		}
		if err := tx.Model(&current).Updates(updates).Error; err != nil {
			return translateDBError(err)
		}
		if err := tx.Model(&current).Association("Authors").Replace(&authors); err != nil {
			return translateDBError(err)
		}
		return nil
	})
}

func (r *GormBookRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&model.Book{}, id)
	if result.Error != nil {
		return translateDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormBookRepository) List(ctx context.Context, filter BookFilter) ([]model.Book, error) {
	page, limit := normalizePage(filter.Page, filter.Limit)
	query := r.db.WithContext(ctx).Model(&model.Book{}).
		Preload("Authors").Preload("Category").
		Order("books.id ASC")
	if filter.CategoryID != 0 {
		query = query.Where("books.category_id = ?", filter.CategoryID)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + likeEscaper.Replace(q) + "%"
		query = query.Where(`books.title ILIKE ? OR books.isbn ILIKE ? OR EXISTS (
			SELECT 1 FROM book_authors ba JOIN authors a ON a.id = ba.author_id
			WHERE ba.book_id = books.id AND a.full_name ILIKE ?)`, like, like, like)
	}
	var books []model.Book
	if err := query.Offset((page - 1) * limit).Limit(limit).Find(&books).Error; err != nil {
		return nil, translateDBError(err)
	}
	if books == nil {
		books = make([]model.Book, 0)
	}
	return books, nil
}

func (r *GormBookRepository) loadAuthors(ctx context.Context, tx *gorm.DB, input []model.Author) ([]model.Author, error) {
	if len(input) == 0 {
		return []model.Author{}, nil
	}
	ids := make([]uint64, 0, len(input))
	seen := make(map[uint64]struct{}, len(input))
	for _, author := range input {
		if author.ID == 0 {
			return nil, ErrInvalidReference
		}
		if _, ok := seen[author.ID]; !ok {
			ids = append(ids, author.ID)
			seen[author.ID] = struct{}{}
		}
	}
	var authors []model.Author
	if err := tx.WithContext(ctx).Where("id IN ?", ids).Order("id ASC").Find(&authors).Error; err != nil {
		return nil, translateDBError(err)
	}
	if len(authors) != len(ids) {
		return nil, fmt.Errorf("%w: author id", ErrInvalidReference)
	}
	return authors, nil
}

// likeEscaper neutralises LIKE wildcards so user input is matched literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func translateDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	var sqlState interface{ SQLState() string }
	if errors.As(err, &sqlState) {
		switch sqlState.SQLState() {
		case "23505":
			return fmt.Errorf("%w: %v", ErrConflict, err)
		case "23503", "23514":
			return fmt.Errorf("%w: %v", ErrInvalidReference, err)
		}
	}
	return err
}
