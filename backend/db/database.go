package db

import (
	"database/sql"
	"log"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dataPath string) error {
	var err error
	// Enable WAL mode and set busy timeout for better concurrency
	DB, err = sql.Open("sqlite", dataPath+"/books.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}

	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT,
		cover_path TEXT,
		file_path TEXT NOT NULL UNIQUE,
		file_size INTEGER,
		file_type TEXT DEFAULT 'epub',
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_added_at ON books(added_at DESC);
	CREATE INDEX IF NOT EXISTS idx_file_path ON books(file_path);
	`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// Migration: Add file_type column if it doesn't exist
	_, err = DB.Exec(`ALTER TABLE books ADD COLUMN file_type TEXT DEFAULT 'epub'`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		log.Printf("Migration warning: %v", err)
	}

	// Migration: Add reading_progress column if it doesn't exist
	_, err = DB.Exec(`ALTER TABLE books ADD COLUMN reading_progress TEXT`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		log.Printf("Migration warning: %v", err)
	}

	// Migration: Add progress_updated_at column (used to order the "Continue" shelf)
	_, err = DB.Exec(`ALTER TABLE books ADD COLUMN progress_updated_at DATETIME`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		log.Printf("Migration warning: %v", err)
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS feeds (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL UNIQUE,
			title TEXT,
			image TEXT,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Feeds table warning: %v", err)
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS annotations (
			id TEXT PRIMARY KEY,
			book_id TEXT NOT NULL,
			cfi TEXT NOT NULL,
			text TEXT NOT NULL,
			note TEXT,
			color TEXT DEFAULT 'yellow',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_annotations_book ON annotations(book_id);
	`)
	if err != nil {
		log.Printf("Annotations table warning: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}
