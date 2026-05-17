# Quiret

A minimal, low-footprint ebook library for your home server.

![alt text](image.png)

## Pros

- **~30MB Docker image**, low idle RAM — designed for a Pi, NAS, or any low-power box
- **One static Go binary, one SQLite file** — no external services, no background workers
- **No Calibre, no JVM, no Node at runtime** — `poppler-utils` is the only system dependency
- **Works for EPUB, FB2, CBZ, and PDF** — using Foliate-js for ebooks and comics, PDF.js for PDFs
- **Auto-extracted covers and metadata** for EPUB, PDF, FB2, and CBZ
- **Annotations, reading progress, drag-and-drop upload, folder auto-scan**

## Tech Stack

**Backend:**
- Go
- SQLite
- Gorilla Mux

**Frontend:**
- Svelte
- Foliate-js
- Vite

## Self-hosting with Docker

1. Clone this repository:
   ```bash
   git clone <repo-url>
   cd quiret
   ```

2. Create an `.env` file at the root, using `.env.example` as an example

3. Start the application:
   ```bash
   docker-compose build
   docker-compose up -d
   ```

4. Open your browser:
   ```
   http://localhost:8080
   ```

That's it! Your books are stored in a Docker volume and persist between restarts.

## Usage

1. **Upload Books**: Drag and drop EPUB or PDF files, or click the upload area
2. **Auto-import**: Place books in `BOOKS_PATH` and they'll be scanned on startup
3. **View Library**: See all your books with covers in a single place
4. **Read**: Click any book to open the reader
5. **Navigate**: Use arrow keys or swipe to turn pages
6. **Annotations**: Select text to highlight content, add notes and return to it later


## Development

**Backend:**
```bash
cd backend
go run .
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

## Configuration

**Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `DATA_PATH` | Where database and books/covers are stored | `./data` |
| `BOOKS_PATH` | Where to scan for book files (can be read-only) | `/path/to/your/books`

## License

MIT
