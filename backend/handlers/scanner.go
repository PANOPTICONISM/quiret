package handlers

import (
	"quiret/db"
	"quiret/models"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ScanDirectory scans a directory for book files and adds them to the database.
// Returns the list of added books and any error encountered.
func ScanDirectory(booksDir string) ([]models.Book, error) {
	entries, err := os.ReadDir(booksDir)
	if err != nil {
		return nil, err
	}

	var addedBooks []models.Book

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		filePath := filepath.Join(booksDir, filename)

		// Check if it's a supported format
		supportedTypes := map[string]string{
			".epub": "epub",
			".pdf":  "pdf",
			".fb2":  "fb2",
			".cbz":  "cbz",
		}
		ext := strings.ToLower(filepath.Ext(filename))
		fileType, ok := supportedTypes[ext]
		if !ok {
			continue
		}

		// Check if book already exists in database by file path
		var existingID string
		err := db.DB.QueryRow(
			"SELECT id FROM books WHERE file_path = ?",
			filePath,
		).Scan(&existingID)

		if err == nil {
			// Book already exists
			continue
		}

		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Failed to stat file %s: %v", filename, err)
			continue
		}

		// Generate unique ID for the book
		bookID := uuid.New().String()

		// Extract metadata using shared functions
		var title, author, coverPath string

		originalName := strings.TrimSuffix(filename, filepath.Ext(filename))

		storageDir := filepath.Join(DataPath, "books", bookID)

		switch fileType {
		case "epub":
			title, author, coverPath = ExtractEPUBMetadata(filePath, storageDir, originalName)
		case "pdf":
			title, author = ExtractPDFMetadata(filePath, originalName)
			coverPath = ExtractPDFCover(filePath, storageDir, bookID)
		case "cbz":
			title = originalName
			author = ""
			coverPath = ExtractCBZCover(filePath, storageDir)
		case "fb2":
			title, author, coverPath = ExtractFB2Metadata(filePath, storageDir, originalName)
		default:
			title = originalName
			author = ""
		}

		book := models.Book{
			ID:        bookID,
			Title:     title,
			Author:    author,
			CoverPath: coverPath,
			FilePath:  filePath,
			FileSize:  info.Size(),
			FileType:  fileType,
			AddedAt:   time.Now(),
		}

		// Insert into database
		_, err = db.DB.Exec(
			"INSERT INTO books (id, title, author, cover_path, file_path, file_size, file_type, added_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			book.ID,
			book.Title,
			book.Author,
			book.CoverPath,
			book.FilePath,
			book.FileSize,
			book.FileType,
			book.AddedAt,
		)

		if err != nil {
			log.Printf("Failed to insert book %s: %v", title, err)
			continue
		}

		addedBooks = append(addedBooks, book)
		log.Printf("Added book: %s by %s", title, author)
	}

	return addedBooks, nil
}
