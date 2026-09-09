package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"quiret/db"
	"quiret/handlers"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		dataPath = "./data"
	}

	booksPathEnv := os.Getenv("BOOKS_PATH")
	if booksPathEnv == "" {
		booksPathEnv = filepath.Join(dataPath, "books")
	}

	// Convert to absolute paths
	absDataPath, err := filepath.Abs(dataPath)
	if err != nil {
		log.Fatal("Failed to get absolute data path:", err)
	}
	dataPath = absDataPath

	// Scan roots come from BOOKS_PATH plus the optional AUDIOBOOKS_PATH. Each may
	// itself list several folders separated by the OS path-list separator (":").
	var bookPaths []string
	seen := make(map[string]bool)
	addPaths := func(env string) {
		for _, p := range filepath.SplitList(env) {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				log.Printf("Skipping invalid books path %q: %v", p, err)
				continue
			}
			if seen[abs] {
				continue
			}
			seen[abs] = true
			bookPaths = append(bookPaths, abs)
		}
	}
	addPaths(booksPathEnv)
	addPaths(os.Getenv("AUDIOBOOKS_PATH"))

	handlers.DataPath = dataPath
	handlers.BookPaths = bookPaths

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	err = db.InitDB(dataPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Scan books directories on startup
	scanBooksOnStartup(bookPaths)

	r := mux.NewRouter()
	r.Use(securityMiddleware)
	r.Use(corsMiddleware)

	r.HandleFunc("/healthz", healthCheck).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/books", handlers.GetBooks).Methods("GET")
	api.HandleFunc("/books", handlers.UploadBook).Methods("POST")
	api.HandleFunc("/books/{id}", handlers.GetBook).Methods("GET")
	api.HandleFunc("/books/{id}/file", handlers.ServeBookFile).Methods("GET")
	api.HandleFunc("/books/{id}/chapters", handlers.GetChapters).Methods("GET")
	api.HandleFunc("/books/{id}/cover", handlers.ServeCover).Methods("GET")
	api.HandleFunc("/books/{id}/cover", handlers.UploadCover).Methods("POST")
	api.HandleFunc("/books/{id}/progress", handlers.SaveProgress).Methods("PUT")
	api.HandleFunc("/books/{id}", handlers.DeleteBook).Methods("DELETE")

	api.HandleFunc("/books/{id}/annotations", handlers.GetAnnotations).Methods("GET")
	api.HandleFunc("/books/{id}/annotations", handlers.CreateAnnotation).Methods("POST")
	api.HandleFunc("/books/{id}/annotations/{annotationId}", handlers.UpdateAnnotation).Methods("PUT")
	api.HandleFunc("/books/{id}/annotations/{annotationId}", handlers.DeleteAnnotation).Methods("DELETE")

	// Serve static frontend files in production
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath != "" {
		log.Printf("Serving static files from: %s", staticPath)
		spa := spaHandler{staticPath: staticPath, indexPath: "index.html"}
		r.PathPrefix("/").Handler(spa)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("Server failed: %v", err)
	case sig := <-stop:
		log.Printf("Received %s, shutting down", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	if err := db.DB.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}
	log.Println("Shutdown complete")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := db.DB.PingContext(ctx); err != nil {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self' blob:; "+
				"script-src 'self'; "+
				"style-src 'self' blob: 'unsafe-inline'; "+
				"img-src 'self' blob: data:; "+
				"connect-src 'self' blob: data:; "+
				"frame-src blob: data:; "+
				"object-src blob: data:; "+
				"form-action 'none'")
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("CORS_ORIGIN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(filepath.Join(h.staticPath, r.URL.Path))

	if !strings.HasPrefix(path, h.staticPath) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// File does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serve the file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func scanBooksOnStartup(bookPaths []string) {
	total := 0
	for _, booksPath := range bookPaths {
		log.Printf("Scanning books directory: %s", booksPath)
		addedBooks, err := handlers.ScanDirectory(booksPath)
		if err != nil {
			log.Printf("Warning: Failed to scan books directory %s: %v", booksPath, err)
			continue
		}
		total += len(addedBooks)
	}

	if total > 0 {
		log.Printf("Scan complete: Added %d books", total)
	} else {
		log.Println("Scan complete: No new books found")
	}
}
