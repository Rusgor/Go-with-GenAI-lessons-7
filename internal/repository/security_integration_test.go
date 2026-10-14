package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"homework/internal/model"
)

func TestPostgresLoanCheckoutReturnAndReaderDefault(t *testing.T) {
	db, suffix := postgresFixture(t)
	user := model.User{Email: fmt.Sprintf("reader-%s@example.test", suffix), PasswordHash: "bcrypt-hash"}
	if err := NewUserRepository(db).Create(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.Role != model.UserReader {
		t.Fatalf("database default role = %q, want reader", user.Role)
	}
	book := createTestBook(t, db, suffix)
	copy := model.Copy{BookID: book.ID, InventoryNo: "SEC-" + suffix, Status: model.CopyAvailable}
	if err := db.Create(&copy).Error; err != nil {
		t.Fatalf("create copy: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("user_id = ?", user.ID).Delete(&model.Loan{}).Error
		_ = db.Delete(&copy).Error
		_ = db.Delete(&user).Error
	})
	repo := NewLoanRepository(db)
	loan, err := repo.Create(context.Background(), user.ID, copy.ID, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("checkout copy: %v", err)
	}
	var checkedOut model.Copy
	if err := db.First(&checkedOut, copy.ID).Error; err != nil || checkedOut.Status != model.CopyCheckedOut {
		t.Fatalf("copy status after checkout = %q, err=%v; want checked_out", checkedOut.Status, err)
	}
	loans, err := repo.ListByUser(context.Background(), user.ID)
	if err != nil || len(loans) != 1 || loans[0].ID != loan.ID {
		t.Fatalf("reader loans = %#v, err=%v; want created loan", loans, err)
	}
	if _, err := repo.Create(context.Background(), user.ID, copy.ID, time.Now().UTC().Add(48*time.Hour)); !errors.Is(err, ErrConflict) {
		t.Fatalf("second active checkout error = %v, want conflict", err)
	}
	returned, err := repo.Return(context.Background(), loan.ID)
	if err != nil || returned.ReturnedAt == nil {
		t.Fatalf("return loan = %#v, err=%v", returned, err)
	}
	if _, err := repo.Return(context.Background(), loan.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("second return error = %v, want conflict", err)
	}
	if err := db.First(&checkedOut, copy.ID).Error; err != nil || checkedOut.Status != model.CopyAvailable {
		t.Fatalf("copy status after return = %q, err=%v; want available", checkedOut.Status, err)
	}
}
