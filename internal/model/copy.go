package model

import "time"

const (
	CopyAvailable  = "available"
	CopyCheckedOut = "checked_out"
	CopyLost       = "lost"
)

type Copy struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	BookID      uint64    `json:"book_id" gorm:"not null;index"`
	InventoryNo string    `json:"inventory_no" gorm:"not null;uniqueIndex"`
	Status      string    `json:"status" gorm:"not null;default:available"`
	Book        Book      `json:"-" gorm:"foreignKey:BookID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
