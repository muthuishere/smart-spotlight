package history

import (
	"database/sql"
	"time"
)

// SearchHistory represents a historical search query
type SearchHistory struct {
	ID        int64     `json:"id"`
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
}

// Service handles search history operations
type Service struct {
	db *sql.DB
}
