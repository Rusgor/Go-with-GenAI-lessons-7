package model

import "time"

const (
	UserReader    = "reader"
	UserMember    = UserReader // Deprecated: use UserReader.
	UserLibrarian = "librarian"
	UserAdmin     = "admin"
)

type User struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"not null;uniqueIndex"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	Role         string    `json:"role" gorm:"not null;default:reader"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
