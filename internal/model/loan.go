package model

import "time"

type Loan struct {
	ID           uint64     `json:"id" gorm:"primaryKey"`
	UserID       uint64     `json:"user_id" gorm:"not null;index"`
	CopyID       uint64     `json:"copy_id" gorm:"not null;index"`
	CheckedOutAt time.Time  `json:"checked_out_at" gorm:"not null"`
	DueAt        time.Time  `json:"due_at" gorm:"not null"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
