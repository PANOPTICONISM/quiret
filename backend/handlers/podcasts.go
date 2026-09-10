package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"quiret/db"
	"quiret/models"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	maxFeedBytes  = 30 << 20  // 30 MB
	maxImageBytes = 10 << 20  // 10 MB
	maxAudioBytes = 1 << 30   // 1 GB
	maxEpisodes   = 500       // cap episodes returned to the client
)

var (
	// feedClient has an overall timeout; fine for small feed/image fetches.
	feedClient = &http.Client{Timeout: 30 * time.Second}
	// audioClient has no overall timeout (large downloads); bounded by context.
	audioClient = &http.Client{}
)

// --- RSS parsing (matches by local element name, ignoring namespace prefixes) ---

type rssImage struct {
	URL  string `xml:"url"`       // standard <image><url>
	Href string `xml:"href,attr"` // <itunes:image href="...">
}

type rssItem struct {
	Title       string   `xml:"title"`
	PubDate     string   `xml:"pubDate"`
	Duration    string   `xml:"duration"` // itunes:duration
	Description string   `xml:"description"`
	Summary     string   `xml:"summary"` // itunes:summary
	Image       rssImage `xml:"image"`
	Enclosure   struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure"`
}

type rssFeed struct {
	Channel struct {
		Title  string    `xml:"title"`
		Author string    `xml:"author"` // itunes:author
		Image  rssImage  `xml:"image"`
		Items  []rssItem `xml:"item"`
	} `xml:"channel"`
}

type podcastEpisode struct {
	Title       string `json:"title"`
	AudioURL    string `json:"audioUrl"`
	PubDate     string `json:"pubDate"`
	Duration    string `json:"duration"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

var htmlTagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// cleanDescription strips HTML tags/entities and collapses whitespace to plain
// text, capped to a reasonable length.
func cleanDescription(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 2000 {
		s = strings.TrimSpace(string(r[:2000])) + "…"
	}
	return strings.TrimSpace(s)
}

// validatePublicURL parses raw, requires http(s), and rejects hosts that resolve
// to loopback/private/link-local addresses (basic SSRF protection).
func validatePublicURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme")
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("missing host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("cannot resolve host")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
			return "", fmt.Errorf("host not allowed")
		}
	}
	return u.String(), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func httpGet(client *http.Client, ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Quiret/1.0")
	return client.Do(req)
}

// loadFeed validates, fetches and parses an RSS feed, returning the canonical URL.
func loadFeed(ctx context.Context, rawURL string) (string, *rssFeed, error) {
	feedURL, err := validatePublicURL(rawURL)
	if err != nil {
		return "", nil, err
	}
	resp, err := httpGet(feedClient, ctx, feedURL)
	if err != nil {
		return feedURL, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return feedURL, nil, fmt.Errorf("feed status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
	if err != nil {
		return feedURL, nil, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return feedURL, nil, err
	}
	return feedURL, &feed, nil
}

// GetPodcastEpisodes fetches and parses an RSS feed, returning show info and a
// capped list of episodes with playable audio enclosures.
func GetPodcastEpisodes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	feedURL, feed, err := loadFeed(ctx, r.URL.Query().Get("url"))
	if err != nil {
		http.Error(w, "Could not load feed. Check the URL.", http.StatusBadRequest)
		return
	}

	showImage := firstNonEmpty(feed.Channel.Image.Href, feed.Channel.Image.URL)
	episodes := make([]podcastEpisode, 0)
	for _, it := range feed.Channel.Items {
		audio := strings.TrimSpace(it.Enclosure.URL)
		if audio == "" {
			continue
		}
		episodes = append(episodes, podcastEpisode{
			Title:       strings.TrimSpace(it.Title),
			AudioURL:    audio,
			PubDate:     strings.TrimSpace(it.PubDate),
			Duration:    strings.TrimSpace(it.Duration),
			Description: cleanDescription(firstNonEmpty(it.Description, it.Summary)),
			Image:       firstNonEmpty(it.Image.Href, showImage),
		})
		if len(episodes) >= maxEpisodes {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"feedUrl":  feedURL,
		"title":    strings.TrimSpace(feed.Channel.Title),
		"author":   strings.TrimSpace(feed.Channel.Author),
		"image":    showImage,
		"episodes": episodes,
	})
}

// ListFeeds returns the saved podcast subscriptions.
func ListFeeds(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, url, title, image, added_at FROM feeds ORDER BY title COLLATE NOCASE")
	if err != nil {
		http.Error(w, "Failed to fetch feeds", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	feeds := make([]models.Feed, 0)
	for rows.Next() {
		var f models.Feed
		var title, image sql.NullString
		if err := rows.Scan(&f.ID, &f.URL, &title, &image, &f.AddedAt); err != nil {
			continue
		}
		f.Title = title.String
		f.Image = image.String
		feeds = append(feeds, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

// AddFeed saves (or refreshes) a podcast subscription, fetching its title/image.
func AddFeed(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	feedURL, feed, err := loadFeed(ctx, payload.URL)
	if err != nil {
		http.Error(w, "Could not load feed. Check the URL.", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(feed.Channel.Title)
	image := firstNonEmpty(feed.Channel.Image.Href, feed.Channel.Image.URL)

	_, err = db.DB.Exec(
		`INSERT INTO feeds (id, url, title, image, added_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(url) DO UPDATE SET title = excluded.title, image = excluded.image`,
		uuid.New().String(), feedURL, title, image, time.Now(),
	)
	if err != nil {
		http.Error(w, "Failed to save feed", http.StatusInternalServerError)
		return
	}

	var f models.Feed
	var t, im sql.NullString
	if err := db.DB.QueryRow(
		"SELECT id, url, title, image, added_at FROM feeds WHERE url = ?", feedURL,
	).Scan(&f.ID, &f.URL, &t, &im, &f.AddedAt); err != nil {
		http.Error(w, "Failed to load saved feed", http.StatusInternalServerError)
		return
	}
	f.Title = t.String
	f.Image = im.String

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f)
}

// DeleteFeed removes a saved subscription.
func DeleteFeed(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if _, err := db.DB.Exec("DELETE FROM feeds WHERE id = ?", id); err != nil {
		http.Error(w, "Failed to delete feed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// DownloadPodcastEpisode downloads an episode's audio into the library as an
// audiobook, using the supplied title/author and (optionally) cover image.
func DownloadPodcastEpisode(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AudioURL string `json:"audioUrl"`
		Title    string `json:"title"`
		Author   string `json:"author"`
		Image    string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	audioURL, err := validatePublicURL(payload.AudioURL)
	if err != nil {
		http.Error(w, "Invalid audio URL", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = "Podcast episode"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	resp, err := httpGet(audioClient, ctx, audioURL)
	if err != nil {
		http.Error(w, "Failed to download episode", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Episode download failed", http.StatusBadGateway)
		return
	}

	ext := audioExtFromURL(audioURL, resp.Header.Get("Content-Type"))
	fileType := strings.TrimPrefix(ext, ".")

	bookID := uuid.New().String()
	storageDir := filepath.Join(DataPath, "books", bookID)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		http.Error(w, "Failed to create storage directory", http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(storageDir, safeFileName(title)+ext)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save episode", http.StatusInternalServerError)
		return
	}
	n, err := io.Copy(dst, io.LimitReader(resp.Body, maxAudioBytes+1))
	dst.Close()
	if err != nil || n == 0 {
		os.RemoveAll(storageDir)
		http.Error(w, "Failed to save episode", http.StatusInternalServerError)
		return
	}
	if n > maxAudioBytes {
		os.RemoveAll(storageDir)
		http.Error(w, "Episode too large", http.StatusRequestEntityTooLarge)
		return
	}

	coverPath := ""
	if payload.Image != "" {
		if imgURL, err := validatePublicURL(payload.Image); err == nil {
			if data, err := fetchLimited(imgURL, maxImageBytes); err == nil {
				coverPath = writeCoverImage(storageDir, data)
			}
		}
	}

	book := models.Book{
		ID:        bookID,
		Title:     title,
		Author:    strings.TrimSpace(payload.Author),
		CoverPath: coverPath,
		FilePath:  filePath,
		FileSize:  n,
		FileType:  fileType,
		AddedAt:   time.Now(),
	}

	_, err = db.DB.Exec(
		"INSERT INTO books (id, title, author, cover_path, file_path, file_size, file_type, added_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		book.ID, book.Title, book.Author, book.CoverPath, book.FilePath, book.FileSize, book.FileType, book.AddedAt,
	)
	if err != nil {
		os.RemoveAll(storageDir)
		http.Error(w, "Failed to save episode metadata", http.StatusInternalServerError)
		log.Println("Podcast DB error:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func fetchLimited(rawURL string, max int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := httpGet(feedClient, ctx, rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func audioExtFromURL(rawURL, contentType string) string {
	supported := map[string]bool{
		".mp3": true, ".m4a": true, ".m4b": true, ".aac": true, ".ogg": true, ".opus": true,
	}
	if u, err := url.Parse(rawURL); err == nil {
		if ext := strings.ToLower(path.Ext(u.Path)); supported[ext] {
			return ext
		}
	}
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "mpeg"):
		return ".mp3"
	case strings.Contains(ct, "mp4"), strings.Contains(ct, "m4a"), strings.Contains(ct, "aac"):
		return ".m4a"
	case strings.Contains(ct, "ogg"), strings.Contains(ct, "opus"):
		return ".ogg"
	}
	return ".mp3"
}

func safeFileName(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteByte('_')
		default:
			if r < 32 {
				b.WriteByte('_')
			} else {
				b.WriteRune(r)
			}
		}
	}
	out := strings.TrimSpace(b.String())
	if len(out) > 120 {
		out = strings.TrimSpace(out[:120])
	}
	if out == "" {
		out = "episode"
	}
	return out
}
