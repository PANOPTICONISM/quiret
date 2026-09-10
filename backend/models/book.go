package models

import "time"

type Book struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Author          string    `json:"author"`
	CoverPath       string    `json:"coverPath"`
	FilePath        string    `json:"filePath"`
	FileSize        int64     `json:"fileSize"`
	FileType          string     `json:"fileType"`
	AddedAt           time.Time  `json:"addedAt"`
	ReadingProgress   string     `json:"readingProgress,omitempty"`
	ProgressUpdatedAt *time.Time `json:"progressUpdatedAt,omitempty"`
}

type Chapter struct {
	Title string  `json:"title"`
	Start float64 `json:"start"` // start offset in seconds
}

type Annotation struct {
	ID        string    `json:"id"`
	BookID    string    `json:"bookId"`
	CFI       string    `json:"cfi"`
	Text      string    `json:"text"`
	Note      string    `json:"note,omitempty"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
}
