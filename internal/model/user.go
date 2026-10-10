package model

import "time"

const (
	UserMember    = "member"
	UserLibrarian = "librarian"
	UserAdmin     = "admin"
)

type User struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"not null;uniqueIndex"`
	Role      string    `json:"role" gorm:"not null;default:member"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
