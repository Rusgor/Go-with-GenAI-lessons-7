package repository

import "errors"

var (
	ErrNotFound         = errors.New("record not found")
	ErrConflict         = errors.New("record conflicts with existing data")
	ErrInvalidReference = errors.New("referenced record does not exist")
)
