package internal

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)


// DBManager handles all database operations
type DBManager struct {
	db *sql.DB
}

// NewDBManager creates a new database manager
func NewDBManager(databasePath string) (*DBManager, error) {
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	manager := &DBManager{db: db}

	err = manager.createTables()
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return manager, nil
}

func (dm *DBManager) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS feed_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guid TEXT UNIQUE NOT NULL,
		title TEXT,
		description TEXT,
		link TEXT,
		published_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		feed_url TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_guid ON feed_items(guid);
	CREATE INDEX IF NOT EXISTS idx_feed_url ON feed_items(feed_url);
	CREATE INDEX IF NOT EXISTS idx_created_at ON feed_items(created_at);
	`

	_, err := dm.db.Exec(query)
	return err
}

func (dm *DBManager) SaveFeedItem(item FeedItem) error {
	query := `
	INSERT OR IGNORE INTO feed_items (guid, title, description, link, published_at, feed_url)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := dm.db.Exec(query, item.GUID, item.Title, item.Description, item.Link, item.PublishedAt, item.FeedURL)
	if err != nil {
		return fmt.Errorf("failed to save feed item: %v", err)
	}

	return nil
}

func (dm *DBManager) IsFeedItemPosted(guid string, feedURL string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM feed_items WHERE guid = ? AND feed_url = ?`
	err := dm.db.QueryRow(query, guid, feedURL).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if feed item exists: %v", err)
	}

	return count > 0, nil
}

func (dm *DBManager) CleanupOldItems(retentionDays int) error {
	thresholdDate := time.Now().AddDate(0, 0, -retentionDays)
	query := `DELETE FROM feed_items WHERE created_at < ?`

	result, err := dm.db.Exec(query, thresholdDate)
	if err != nil {
		return fmt.Errorf("failed to cleanup old items: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	log.Printf("Cleaned up %d old feed items", rowsAffected)
	return nil
}

func (dm *DBManager) Close() error {
	return dm.db.Close()
}