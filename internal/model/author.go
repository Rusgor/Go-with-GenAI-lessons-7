package model

import "time"

type Author struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	FullName  string    `json:"full_name" gorm:"not null;index"`
	Bio       *string   `json:"bio,omitempty"`
	Books     []Book    `json:"-" gorm:"many2many:book_authors;"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
