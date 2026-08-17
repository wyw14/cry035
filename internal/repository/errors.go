package repository

import "errors"

var (
	ErrNotFound        = errors.New("record not found")
	ErrConflict        = errors.New("record conflicts with existing state")
	ErrVersionConflict = errors.New("optimistic version conflict")
	ErrUnauthorized    = errors.New("operation is not permitted")
)
