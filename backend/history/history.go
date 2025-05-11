package history

import (
	"database/sql"
)

// NewService creates a new history service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Initialize creates the search history table if it doesn't exist
func (s *Service) Initialize() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// AddToHistory adds a search query to history if it doesn't already exist
func (s *Service) AddToHistory(query string) error {
	// Check if query already exists (case insensitive)
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM search_history WHERE query COLLATE NOCASE = ?", query).Scan(&count)
	if err != nil {
		return err
	}

	// Only insert if query doesn't exist
	if count == 0 {
		_, err = s.db.Exec("INSERT INTO search_history (query) VALUES (?)", query)
		return err
	}

	// Update timestamp if query exists (case insensitive)
	_, err = s.db.Exec("UPDATE search_history SET timestamp = CURRENT_TIMESTAMP WHERE query COLLATE NOCASE = ?", query)
	return err
}

// GetSearchHistory returns the search history, optionally filtered by a prefix
func (s *Service) GetSearchHistory(prefix string) []SearchHistory {
	history := make([]SearchHistory, 0)

	query := "SELECT id, query, timestamp FROM search_history WHERE query COLLATE NOCASE LIKE ? COLLATE NOCASE ORDER BY timestamp DESC LIMIT 10"
	rows, err := s.db.Query(query, prefix+"%")
	if err != nil {
		return history
	}
	defer rows.Close()

	for rows.Next() {
		var h SearchHistory
		if err := rows.Scan(&h.ID, &h.Query, &h.Timestamp); err != nil {
			continue
		}
		history = append(history, h)
	}

	return history
}
