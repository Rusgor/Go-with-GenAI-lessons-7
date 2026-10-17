package repository

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"homework/internal/model"
)

func postgresFixture(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := ApplyMigrations(databaseURL, "../../migrations"); err != nil {
		t.Fatalf("apply test migrations: %v", err)
	}
	db, err := OpenDB(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db, time.Now().UTC().Format("20060102150405.000000000")
}

var integrationISBNSequence = uint64(time.Now().UnixNano() % 10_000_000_000)

func nextIntegrationISBN() string {
	return fmt.Sprintf("978%010d", atomic.AddUint64(&integrationISBNSequence, 1)%10_000_000_000)
}

func createTestBook(t *testing.T, db *gorm.DB, suffix string) model.Book {
	t.Helper()
	category := model.Category{Name: "Integration category " + suffix}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	author := model.Author{FullName: "Integration author " + suffix}
	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	book := model.Book{
		Title:      "Integration book " + suffix,
		ISBN:       nextIntegrationISBN(),
		CategoryID: category.ID,
		Authors:    []model.Author{{ID: author.ID}},
	}
	if err := NewGormBookRepository(db).Create(context.Background(), &book); err != nil {
		t.Fatalf("create book: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Delete(&book).Error
		_ = db.Delete(&author).Error
		_ = db.Delete(&category).Error
	})
	return book
}

func TestPostgresBookRepositoryCreateReadUpdateDelete(t *testing.T) {
	db, suffix := postgresFixture(t)
	book := createTestBook(t, db, suffix)
	repo := NewGormBookRepository(db)

	loaded, err := repo.FindByID(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("find book: %v", err)
	}
	if loaded.Category.ID != book.CategoryID || len(loaded.Authors) != 1 {
		t.Fatalf("preloaded relations = category %d, authors %d", loaded.Category.ID, len(loaded.Authors))
	}

	loaded.Title = "Updated " + loaded.Title
	if err := repo.Update(context.Background(), loaded); err != nil {
		t.Fatalf("update book: %v", err)
	}
	updated, err := repo.FindByID(context.Background(), book.ID)
	if err != nil || !strings.HasPrefix(updated.Title, "Updated ") {
		t.Fatalf("updated book = %#v, error %v", updated, err)
	}

	if err := repo.Delete(context.Background(), book.ID); err != nil {
		t.Fatalf("delete book: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), book.ID); err != ErrNotFound {
		t.Fatalf("FindByID after delete error = %v, want ErrNotFound", err)
	}
}

func TestPostgresBookRepositoryRejectsUnknownAuthor(t *testing.T) {
	db, suffix := postgresFixture(t)
	var category model.Category
	if err := db.Where("name = ?", "Fiction").First(&category).Error; err != nil {
		t.Fatalf("find seeded category: %v", err)
	}
	book := model.Book{Title: "Invalid relation " + suffix, ISBN: nextIntegrationISBN(), CategoryID: category.ID, Authors: []model.Author{{ID: 999999999}}}
	if err := NewGormBookRepository(db).Create(context.Background(), &book); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("Create error = %v, want ErrInvalidReference", err)
	}
}

func TestPostgresBookListUsesConstantPreloadQueries(t *testing.T) {
	db, suffix := postgresFixture(t)
	createTestBook(t, db, suffix+"1")
	counter := &selectCounter{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	countListSelects := func() int {
		t.Helper()
		counter.reset()
		if _, err := NewGormBookRepository(db.Session(&gorm.Session{Logger: counter})).List(context.Background(), BookFilter{Page: 1, Limit: 100}); err != nil {
			t.Fatalf("list books: %v", err)
		}
		return counter.count()
	}
	selectsWithOneExtraBook := countListSelects()
	if selectsWithOneExtraBook != 4 {
		t.Fatalf("list SELECT count with one extra book = %d, want 4 (books, book_authors, authors, categories)", selectsWithOneExtraBook)
	}
	var base model.Book
	if err := db.Preload("Authors").First(&base).Error; err != nil {
		t.Fatalf("load seed book: %v", err)
	}
	for i := 0; i < 5; i++ {
		book := model.Book{Title: fmt.Sprintf("Scale %s %d", suffix, i), ISBN: nextIntegrationISBN(), CategoryID: base.CategoryID, Authors: []model.Author{{ID: base.Authors[0].ID}}}
		if err := NewGormBookRepository(db).Create(context.Background(), &book); err != nil {
			t.Fatalf("create scale book: %v", err)
		}
		created := book
		t.Cleanup(func() { _ = db.Delete(&created).Error })
	}
	selectsWithSixExtraBooks := countListSelects()
	if selectsWithSixExtraBooks != selectsWithOneExtraBook {
		t.Fatalf("list SELECT count changed as books were added: one extra=%d, six extra=%d", selectsWithOneExtraBook, selectsWithSixExtraBooks)
	}
}

func TestPostgresHomeworkNormalizationMigrationRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	baseDB, err := OpenDB(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	baseSQLDB, err := baseDB.DB()
	if err != nil {
		t.Fatalf("get base database pool: %v", err)
	}

	schema := fmt.Sprintf("libratrack_migration_%d", time.Now().UnixNano())
	if err := baseDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)).Error; err != nil {
		t.Fatalf("create isolated migration schema: %v", err)
	}
	t.Cleanup(func() {
		if err := baseDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schema)).Error; err != nil {
			t.Errorf("drop isolated migration schema: %v", err)
		}
		_ = baseSQLDB.Close()
	})

	schemaURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	query := schemaURL.Query()
	query.Set("options", "-c search_path="+schema)
	schemaURL.RawQuery = query.Encode()
	legacyURL := schemaURL.String()

	migrationDir, err := filepath.Abs("../../migrations-homework/migrations")
	if err != nil {
		t.Fatalf("resolve homework migrations: %v", err)
	}
	sourceURL := (&url.URL{Scheme: "file", Opaque: filepath.ToSlash(migrationDir)}).String()
	m, err := migrate.New(sourceURL, legacyURL)
	if err != nil {
		t.Fatalf("create homework migrator: %v", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Steps(1); err != nil {
		t.Fatalf("apply legacy schema: %v", err)
	}

	legacyDB, err := OpenDB(ctx, legacyURL)
	if err != nil {
		t.Fatalf("open isolated legacy schema: %v", err)
	}
	legacySQLDB, err := legacyDB.DB()
	if err != nil {
		t.Fatalf("get isolated database pool: %v", err)
	}
	t.Cleanup(func() { _ = legacySQLDB.Close() })
	const originalAuthor = "Lovelace, Ada"
	const originalCategory = "   "
	var bookID uint64
	if err := legacyDB.Raw(`INSERT INTO books (title, isbn, author, category) VALUES (?, ?, ?, ?) RETURNING id`, "Migration round-trip", "L8-MIGRATION-ROUNDTRIP", originalAuthor, originalCategory).Row().Scan(&bookID); err != nil {
		t.Fatalf("insert legacy book: %v", err)
	}
	if err := m.Steps(1); err != nil {
		t.Fatalf("apply normalization migration: %v", err)
	}

	var category model.Category
	if err := legacyDB.Where("name = ?", "Uncategorized").First(&category).Error; err != nil {
		t.Fatalf("find fallback category: %v", err)
	}
	var authorCount int64
	if err := legacyDB.Model(&model.Author{}).Where("full_name IN ?", []string{"Lovelace", "Ada"}).Count(&authorCount).Error; err != nil || authorCount != 2 {
		t.Fatalf("normalized author count = %d, error %v; want 2", authorCount, err)
	}

	if err := m.Steps(-1); err != nil {
		t.Fatalf("roll back normalization migration: %v", err)
	}
	var restoredAuthor, restoredCategory string
	if err := legacyDB.Raw("SELECT author, category FROM books WHERE id = ?", bookID).Row().Scan(&restoredAuthor, &restoredCategory); err != nil {
		t.Fatalf("read restored legacy values: %v", err)
	}
	if restoredAuthor != originalAuthor || restoredCategory != originalCategory {
		t.Fatalf("restored author/category = %q/%q; want %q/%q", restoredAuthor, restoredCategory, originalAuthor, originalCategory)
	}
	if err := m.Steps(1); err != nil {
		t.Fatalf("reapply normalization migration: %v", err)
	}
}

func TestPostgresBookListNPlusOneComparison(t *testing.T) {
	db, suffix := postgresFixture(t)
	categoryID := createNPlusOneDataset(t, db, suffix)
	capture := &queryCaptureLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	capturedDB := db.Session(&gorm.Session{Logger: capture})
	ctx := context.Background()

	naive := func() error {
		capture.reset()
		_, err := naiveBookList(ctx, capturedDB, categoryID, 100)
		return err
	}
	preloaded := func() error {
		capture.reset()
		_, err := NewGormBookRepository(capturedDB).List(ctx, BookFilter{CategoryID: categoryID, Page: 1, Limit: 100})
		return err
	}

	if err := naive(); err != nil {
		t.Fatalf("warm up naive list: %v", err)
	}
	if err := preloaded(); err != nil {
		t.Fatalf("warm up preloaded list: %v", err)
	}

	capture.reset()
	started := time.Now()
	if err := naive(); err != nil {
		t.Fatalf("naive list: %v", err)
	}
	naiveOnce := time.Since(started)
	naiveQueries := capture.snapshot()

	capture.reset()
	started = time.Now()
	if err := preloaded(); err != nil {
		t.Fatalf("preloaded list: %v", err)
	}
	preloadedOnce := time.Since(started)
	preloadedQueries := capture.snapshot()

	if len(naiveQueries) != 21 {
		t.Fatalf("naive SELECT count = %d, want 21 for 10 books", len(naiveQueries))
	}
	if countSelectsFromTable(naiveQueries, "books") != 1 ||
		countSelectsFromTable(naiveQueries, "authors") != 10 ||
		countSelectsFromTable(naiveQueries, "categories") != 10 {
		t.Fatalf("naive SELECT breakdown = books:%d authors:%d categories:%d, want 1/10/10", countSelectsFromTable(naiveQueries, "books"), countSelectsFromTable(naiveQueries, "authors"), countSelectsFromTable(naiveQueries, "categories"))
	}
	if len(preloadedQueries) != 4 {
		t.Fatalf("preloaded SELECT count = %d, want 4 (books, book_authors, authors, categories)", len(preloadedQueries))
	}
	for _, table := range []string{"books", "book_authors", "authors", "categories"} {
		if countSelectsFromTable(preloadedQueries, table) != 1 {
			t.Fatalf("preloaded SELECT count from %s = %d, want 1: %v", table, countSelectsFromTable(preloadedQueries, table), preloadedQueries)
		}
	}
	for _, table := range []string{"book_authors", "authors"} {
		if !hasBatchedSelectFromTable(preloadedQueries, table) {
			t.Fatalf("preloaded SQL lacks a batched SELECT from %s: %v", table, preloadedQueries)
		}
	}

	naiveMedian := measureMedian(t, naive, 7)
	preloadedMedian := measureMedian(t, preloaded, 7)
	t.Logf("N+1 comparison: books=10 authors=5 categories=3 warmups=1 samples=7")
	t.Logf("BEFORE naive: SELECTs=%d first-run=%.3fms median=%.3fms", len(naiveQueries), float64(naiveOnce.Microseconds())/1000, float64(naiveMedian.Microseconds())/1000)
	for _, query := range naiveQueries {
		t.Logf("BEFORE SQL duration=%s: %s", query.duration, query.sql)
	}
	t.Logf("AFTER Preload: SELECTs=%d first-run=%.3fms median=%.3fms", len(preloadedQueries), float64(preloadedOnce.Microseconds())/1000, float64(preloadedMedian.Microseconds())/1000)
	for _, query := range preloadedQueries {
		t.Logf("AFTER SQL duration=%s: %s", query.duration, query.sql)
	}
}

func TestPostgresBookRepositoryDeleteHonorsContext(t *testing.T) {
	db, suffix := postgresFixture(t)
	book := createTestBook(t, db, suffix)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := NewGormBookRepository(db).Delete(ctx, book.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete with canceled context error = %v, want context.Canceled", err)
	}
	if _, err := NewGormBookRepository(db).FindByID(context.Background(), book.ID); err != nil {
		t.Fatalf("book was deleted despite canceled context: %v", err)
	}
}

func TestPostgresCopyAvailabilityCountsAvailableCopies(t *testing.T) {
	db, suffix := postgresFixture(t)
	book := createTestBook(t, db, suffix)
	repo := NewCopyRepository(db)
	copies := []model.Copy{
		{BookID: book.ID, InventoryNo: "AV-" + strings.ReplaceAll(suffix, ".", "") + "1", Status: model.CopyAvailable},
		{BookID: book.ID, InventoryNo: "AV-" + strings.ReplaceAll(suffix, ".", "") + "2", Status: model.CopyCheckedOut},
	}
	for i := range copies {
		if err := repo.Create(context.Background(), &copies[i]); err != nil {
			t.Fatalf("create copy: %v", err)
		}
	}
	total, available, err := repo.Availability(context.Background(), book.ID)
	if err != nil || total != 2 || available != 1 {
		t.Fatalf("Availability = (%d, %d, %v), want (2, 1, nil)", total, available, err)
	}
}

type capturedQuery struct {
	sql      string
	duration time.Duration
}

type queryCaptureLogger struct {
	gormlogger.Interface
	mu      sync.Mutex
	queries []capturedQuery
}

func (c *queryCaptureLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return c }
func (c *queryCaptureLogger) Info(context.Context, string, ...interface{})     {}
func (c *queryCaptureLogger) Warn(context.Context, string, ...interface{})     {}
func (c *queryCaptureLogger) Error(context.Context, string, ...interface{})    {}

func (c *queryCaptureLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), _ error) {
	query, _ := fc()
	c.mu.Lock()
	c.queries = append(c.queries, capturedQuery{sql: query, duration: time.Since(begin)})
	c.mu.Unlock()
}

func (c *queryCaptureLogger) reset() {
	c.mu.Lock()
	c.queries = nil
	c.mu.Unlock()
}

func (c *queryCaptureLogger) snapshot() []capturedQuery {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]capturedQuery(nil), c.queries...)
}

func createNPlusOneDataset(t *testing.T, db *gorm.DB, suffix string) uint64 {
	t.Helper()
	categories := make([]model.Category, 3)
	for i := range categories {
		categories[i] = model.Category{Name: fmt.Sprintf("N+1 category %s %d", suffix, i)}
		if err := db.Create(&categories[i]).Error; err != nil {
			t.Fatalf("create N+1 category: %v", err)
		}
	}
	authors := make([]model.Author, 5)
	for i := range authors {
		authors[i] = model.Author{FullName: fmt.Sprintf("N+1 author %s %d", suffix, i)}
		if err := db.Create(&authors[i]).Error; err != nil {
			t.Fatalf("create N+1 author: %v", err)
		}
	}

	bookIDs := make([]uint64, 10)
	repo := NewGormBookRepository(db)
	for i := range bookIDs {
		book := model.Book{
			Title:      fmt.Sprintf("N+1 book %s %d", suffix, i),
			ISBN:       nextIntegrationISBN(),
			CategoryID: categories[0].ID,
			Authors:    []model.Author{authors[i%len(authors)], authors[(i+1)%len(authors)]},
		}
		if err := repo.Create(context.Background(), &book); err != nil {
			t.Fatalf("create N+1 book: %v", err)
		}
		bookIDs[i] = book.ID
	}
	t.Cleanup(func() {
		for _, id := range bookIDs {
			_ = db.Delete(&model.Book{}, id).Error
		}
		for i := range authors {
			_ = db.Delete(&authors[i]).Error
		}
		for i := range categories {
			_ = db.Delete(&categories[i]).Error
		}
	})
	return categories[0].ID
}

func naiveBookList(ctx context.Context, db *gorm.DB, categoryID uint64, limit int) ([]model.Book, error) {
	var books []model.Book
	if err := db.WithContext(ctx).Where("books.category_id = ?", categoryID).Order("books.id ASC").Limit(limit).Find(&books).Error; err != nil {
		return nil, err
	}
	for i := range books {
		if err := db.WithContext(ctx).Model(&books[i]).Association("Authors").Find(&books[i].Authors); err != nil {
			return nil, err
		}
		if err := db.WithContext(ctx).First(&books[i].Category, books[i].CategoryID).Error; err != nil {
			return nil, err
		}
	}
	return books, nil
}

func measureMedian(t *testing.T, run func() error, samples int) time.Duration {
	t.Helper()
	durations := make([]time.Duration, samples)
	for i := range durations {
		started := time.Now()
		if err := run(); err != nil {
			t.Fatalf("benchmark list query: %v", err)
		}
		durations[i] = time.Since(started)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return durations[len(durations)/2]
}

func countSelectsFromTable(queries []capturedQuery, table string) int {
	count := 0
	for _, query := range queries {
		sql := normalizeCapturedSQL(query.sql)
		if strings.HasPrefix(sql, "select ") && strings.Contains(sql, " from "+table) {
			count++
		}
	}
	return count
}

func hasBatchedSelectFromTable(queries []capturedQuery, table string) bool {
	for _, query := range queries {
		sql := normalizeCapturedSQL(query.sql)
		if strings.HasPrefix(sql, "select ") && strings.Contains(sql, " from "+table) && strings.Contains(sql, " in ") {
			return true
		}
	}
	return false
}

func normalizeCapturedSQL(query string) string {
	return strings.ReplaceAll(strings.ToLower(query), `"`, "")
}

type selectCounter struct {
	gormlogger.Interface
	mu      sync.Mutex
	selects int
}

func (c *selectCounter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	query, _ := fc()
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		c.mu.Lock()
		c.selects++
		c.mu.Unlock()
	}
	c.Interface.Trace(ctx, begin, fc, err)
}

func (c *selectCounter) reset() {
	c.mu.Lock()
	c.selects = 0
	c.mu.Unlock()
}

func (c *selectCounter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selects
}

func TestPostgresBookSearchAndBatchAvailability(t *testing.T) {
	db, suffix := postgresFixture(t)
	book := createTestBook(t, db, suffix)
	other := createTestBook(t, db, "other-"+suffix)
	ctx := context.Background()
	repo := NewGormBookRepository(db)

	// Copies: book has 2 (1 available), other has none.
	copyRepo := NewCopyRepository(db)
	tag := strings.ReplaceAll(suffix, ".", "")
	for i, status := range []string{model.CopyAvailable, model.CopyCheckedOut} {
		c := model.Copy{BookID: book.ID, InventoryNo: fmt.Sprintf("SR-%s-%d", tag, i), Status: status}
		if err := copyRepo.Create(ctx, &c); err != nil {
			t.Fatalf("create copy: %v", err)
		}
	}

	tests := []struct {
		name    string
		query   string
		wantIDs []uint64
		wantNot []uint64
	}{
		{"by title, case-insensitive", "INTEGRATION BOOK " + suffix, []uint64{book.ID}, []uint64{other.ID}},
		{"by ISBN", book.ISBN, []uint64{book.ID}, []uint64{other.ID}},
		{"by author name", "integration author " + suffix, []uint64{book.ID}, []uint64{other.ID}},
		{"LIKE wildcards are literal", "%", nil, []uint64{book.ID, other.ID}},
		{"SQL metacharacters are inert", "'; DROP TABLE books;--", nil, []uint64{book.ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.List(ctx, BookFilter{Query: tt.query, Limit: 100})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			seen := map[uint64]bool{}
			for _, b := range got {
				seen[b.ID] = true
			}
			for _, id := range tt.wantIDs {
				if !seen[id] {
					t.Errorf("book %d missing from results", id)
				}
			}
			for _, id := range tt.wantNot {
				if seen[id] {
					t.Errorf("book %d unexpectedly present", id)
				}
			}
		})
	}

	counts, err := copyRepo.AvailabilityByBooks(ctx, []uint64{book.ID, other.ID})
	if err != nil {
		t.Fatalf("AvailabilityByBooks: %v", err)
	}
	if got := counts[book.ID]; got.Total != 2 || got.Available != 1 {
		t.Errorf("book availability = %+v, want total 2 available 1", got)
	}
	if got := counts[other.ID]; got.Total != 0 || got.Available != 0 {
		t.Errorf("book without copies = %+v, want zeros", got)
	}
}
