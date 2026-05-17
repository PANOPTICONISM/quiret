package main

import (
	"quiret/db"
	"quiret/handlers"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		dataPath = "./data"
	}

	booksPath := os.Getenv("BOOKS_PATH")
	if booksPath == "" {
		booksPath = filepath.Join(dataPath, "books")
	}

	// Convert to absolute paths
	absDataPath, err := filepath.Abs(dataPath)
	if err != nil {
		log.Fatal("Failed to get absolute data path:", err)
	}
	dataPath = absDataPath

	absBooksPath, err := filepath.Abs(booksPath)
	if err != nil {
		log.Fatal("Failed to get absolute books path:", err)
	}
	booksPath = absBooksPath

	handlers.DataPath = dataPath
	handlers.BooksPath = booksPath

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	err = db.InitDB(dataPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Scan books directory on startup
	log.Printf("Scanning books directory: %s", booksPath)
	scanBooksOnStartup(booksPath)

	r := mux.NewRouter()
	r.Use(securityMiddleware)
	r.Use(corsMiddleware)

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/books", handlers.GetBooks).Methods("GET")
	api.HandleFunc("/books", handlers.UploadBook).Methods("POST")
	api.HandleFunc("/books/{id}", handlers.GetBook).Methods("GET")
	api.HandleFunc("/books/{id}/file", handlers.ServeBookFile).Methods("GET")
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

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
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

func scanBooksOnStartup(booksPath string) {
	addedBooks, err := handlers.ScanDirectory(booksPath)
	if err != nil {
		log.Printf("Warning: Failed to scan books directory: %v", err)
		return
	}

	if len(addedBooks) > 0 {
		log.Printf("Scan complete: Added %d books from directory", len(addedBooks))
	} else {
		log.Println("Scan complete: No new books found")
	}
}
