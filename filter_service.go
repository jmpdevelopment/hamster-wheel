package main

import (
	"context"

	"hamster-wheel/internal/db"
)

// FilterService handles search filter operations exposed to the frontend.
type FilterService struct {
	db *db.DB
}

// NewFilterService creates a new FilterService.
func NewFilterService(database *db.DB) *FilterService {
	return &FilterService{
		db: database,
	}
}

// GetFilters returns all search filters.
func (s *FilterService) GetFilters() ([]db.SearchFilter, error) {
	return s.db.ListFilters(context.Background())
}

// GetFilter returns a single filter by ID, or nil if not found.
func (s *FilterService) GetFilter(id string) (*db.SearchFilter, error) {
	return s.db.GetFilter(context.Background(), id)
}

// CreateFilter creates a new search filter and returns its ID.
func (s *FilterService) CreateFilter(name, keywords, location, source string) (string, error) {
	return s.db.CreateFilter(context.Background(), name, keywords, location, source)
}

// UpdateFilter updates an existing search filter.
func (s *FilterService) UpdateFilter(id, name, keywords, location, source string, enabled bool) error {
	return s.db.UpdateFilter(context.Background(), id, name, keywords, location, source, enabled)
}

// DeleteFilter removes a search filter by ID.
func (s *FilterService) DeleteFilter(id string) error {
	return s.db.DeleteFilter(context.Background(), id)
}