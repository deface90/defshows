// Package crud provides a small generic gorm CRUD helper for admin-managed
// reference tables, so new dictionaries can be wired up with minimal code.
package crud

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("crud: not found")

// Repository is a generic CRUD repository for model T.
type Repository[T any] struct {
	db *gorm.DB
}

// NewRepository creates a Repository for model T.
func NewRepository[T any](db *gorm.DB) *Repository[T] {
	return &Repository[T]{db: db}
}

// List returns rows ordered by id with pagination.
func (r *Repository[T]) List(ctx context.Context, limit, offset int) ([]T, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []T
	err := r.db.WithContext(ctx).Order("id").Limit(limit).Offset(offset).Find(&out).Error
	return out, err
}

// Get returns a row by id or ErrNotFound.
func (r *Repository[T]) Get(ctx context.Context, id int64) (*T, error) {
	var v T
	err := r.db.WithContext(ctx).First(&v, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Create inserts a new row.
func (r *Repository[T]) Create(ctx context.Context, v *T) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// Save updates an existing row (full save).
func (r *Repository[T]) Save(ctx context.Context, v *T) error {
	return r.db.WithContext(ctx).Save(v).Error
}

// Delete removes a row by id. Returns ErrNotFound if nothing was deleted.
func (r *Repository[T]) Delete(ctx context.Context, id int64) error {
	var v T
	res := r.db.WithContext(ctx).Delete(&v, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
