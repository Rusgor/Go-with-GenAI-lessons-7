package model

import "time"

type Book struct {
	ID            uint64    `json:"id" gorm:"primaryKey"`
	Title         string    `json:"title" gorm:"not null"`
	ISBN          string    `json:"isbn" gorm:"not null;uniqueIndex"`
	CategoryID    uint64    `json:"category_id" gorm:"not null;index"`
	Category      Category  `json:"category" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Authors       []Author  `json:"authors" gorm:"many2many:book_authors;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	PublishedYear *int      `json:"published_year"`
	Description   *string   `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
