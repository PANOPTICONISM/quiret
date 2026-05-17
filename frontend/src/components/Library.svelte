<script>
  import { onMount } from "svelte";
  import { SUPPORTED_EXTENSIONS, FILE_ACCEPT, FOLIATE_FORMATS } from "../lib/constants.js";

  let { onOpenBook } = $props();

  let books = $state([]);
  let uploading = $state(false);
  let dragOver = $state(false);
  let darkMode = $state(false);

  onMount(async () => {
    darkMode = localStorage.getItem("darkMode") === "true";
    applyDarkMode(darkMode);
    await fetchBooks();
  });

  const toggleDarkMode = () => {
    darkMode = !darkMode;
    localStorage.setItem("darkMode", darkMode);
    applyDarkMode(darkMode);
  };

  const applyDarkMode = (enabled) => {
    document.documentElement.classList.toggle("dark", enabled);
  };

  const fetchBooks = async () => {
    try {
      const response = await fetch("/api/books");
      const data = await response.json();
      books = Array.isArray(data) ? data : [];
    } catch (error) {
      console.error("Failed to fetch books:", error);
      books = [];
    }
  };

  const handleFileSelect = async (event) => {
    const files = event.target.files || event.dataTransfer?.files;
    if (!files || files.length === 0) {
      return;
    }

    const file = files[0];
    const filename = file.name.toLowerCase();
    const isSupported = SUPPORTED_EXTENSIONS.some((ext) => filename.endsWith(ext));
    if (!isSupported) {
      alert("Supported formats: EPUB, PDF, MOBI, FB2, CBZ");
      return;
    }

    await uploadBook(file);
  };

  const uploadBook = async (file) => {
    uploading = true;
    const formData = new FormData();
    formData.append("book", file);

    try {
      const response = await fetch("/api/books", {
        method: "POST",
        body: formData,
      });

      if (response.ok) {
        await fetchBooks();
      } else {
        alert("Failed to upload book");
      }
    } catch (error) {
      console.error("Upload error:", error);
      alert("Failed to upload book");
    } finally {
      uploading = false;
    }
  };

  const handleDragOver = (event) => {
    event.preventDefault();
    dragOver = true;
  };

  const handleDragLeave = () => {
    dragOver = false;
  };

  const handleDrop = (event) => {
    event.preventDefault();
    dragOver = false;
    handleFileSelect(event);
  };

  const getReadingProgress = (book) => {
    if (!book.readingProgress) return 0;
    try {
      const progress = JSON.parse(book.readingProgress);
      if (FOLIATE_FORMATS.includes(progress.type) && progress.fraction !== undefined) {
        return Math.round(progress.fraction * 100);
      } else if (progress.type === "pdf" && progress.page && progress.totalPages) {
        return Math.round((progress.page / progress.totalPages) * 100);
      }
    } catch (e) {
      return 0;
    }
    return 0;
  };

  const deleteBook = async (event, bookId, bookTitle) => {
    event.stopPropagation();
    if (!confirm(`Delete "${bookTitle}"?`)) {
      return;
    }

    try {
      const response = await fetch(`/api/books/${bookId}`, {
        method: "DELETE",
      });

      if (response.ok) {
        books = books.filter((b) => b.id !== bookId);
      } else {
        alert("Failed to delete book");
      }
    } catch (error) {
      console.error("Delete error:", error);
      alert("Failed to delete book");
    }
  };
</script>

<div class="container">
  <header>
    <h1>Quiret</h1>
    <button
      class="dark-mode-toggle"
      onclick={toggleDarkMode}
      aria-label="Toggle dark mode"
    >
      {#if darkMode}
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
      {:else}
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      {/if}
    </button>
  </header>
  <div
    class="upload-zone"
    class:drag-over={dragOver}
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
    role="button"
    tabindex="0"
  >
    <input
      type="file"
      accept={FILE_ACCEPT}
      onchange={handleFileSelect}
      id="file-input"
      style="display: none;"
    />
    <label for="file-input">
      {#if uploading}
        <div class="spinner"></div>
        <p>Uploading...</p>
      {:else}
        <svg
          width="48"
          height="48"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="17 8 12 3 7 8" />
          <line x1="12" y1="3" x2="12" y2="15" />
        </svg>
        <p>Drop your book here or click to upload</p>
      {/if}
    </label>
  </div>
  {#if books.length > 0}
    <div class="books-grid">
      {#each books as book (book.id)}
        <div class="book-card">
          <button
            type="button"
            class="book-card-main"
            onclick={() => onOpenBook(book.id)}
          >
            <div class="cover-container">
              {#if book.coverPath}
                <img src="/api/books/{book.id}/cover" alt={book.title} />
              {:else}
                <div class="no-cover">
                  <svg
                    width="48"
                    height="48"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
                    <path
                      d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"
                    />
                  </svg>
                </div>
              {/if}
              <span class="file-type-tag" class:pdf={book.fileType === "pdf"}>
                {book.fileType?.toUpperCase() || "EPUB"}
              </span>
              {#if getReadingProgress(book) > 0}
                <div class="progress-indicator">
                  <div
                    class="progress-fill"
                    style="width: {getReadingProgress(book)}%"
                  ></div>
                </div>
              {/if}
            </div>
            <div class="book-info">
              <h3>{book.title}</h3>
              <p>{book.author}</p>
            </div>
          </button>
          <button
            type="button"
            class="delete-btn"
            onclick={(e) => deleteBook(e, book.id, book.title)}
            aria-label="Delete book"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
              />
            </svg>
          </button>
        </div>
      {/each}
    </div>
  {:else}
    <div class="empty-state">
      <svg
        width="64"
        height="64"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
      >
        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
        <path
          d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"
        />
      </svg>
      <p>No books yet. Upload your first book!</p>
    </div>
  {/if}
</div>

<style>
  .container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
  }

  header {
    margin-bottom: 2rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  h1 {
    font-size: 2rem;
    font-weight: 600;
    color: #1a1a1a;
  }

  :global(.dark) h1 {
    color: #f7fafc;
  }

  .dark-mode-toggle {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.5rem;
    border-radius: 8px;
    color: #4a5568;
    transition:
      background 0.2s,
      color 0.2s;
  }

  .dark-mode-toggle:hover {
    background: #e2e8f0;
  }

  :global(.dark) .dark-mode-toggle {
    color: #e2e8f0;
  }

  :global(.dark) .dark-mode-toggle:hover {
    background: #4a5568;
  }

  .upload-zone {
    border: 2px dashed #cbd5e0;
    border-radius: 12px;
    padding: 3rem;
    text-align: center;
    margin-bottom: 3rem;
    background: white;
    transition: all 0.2s;
    cursor: pointer;
  }

  .upload-zone:hover,
  .upload-zone.drag-over {
    border-color: #4299e1;
    background: #ebf8ff;
  }

  .upload-zone label {
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    color: #4a5568;
  }

  .spinner {
    width: 48px;
    height: 48px;
    border: 4px solid #e2e8f0;
    border-top-color: #4299e1;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .books-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 1.5rem;
  }

  .book-card {
    position: relative;
    border-radius: 8px;
    overflow: hidden;
    background: white;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }

  .book-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  }

  .book-card-main {
    cursor: pointer;
    border: none;
    padding: 0;
    background: none;
    width: 100%;
    text-align: left;
  }

  .delete-btn {
    position: absolute;
    top: 8px;
    left: 8px;
    background: rgba(220, 38, 38, 0.9);
    border: none;
    border-radius: 4px;
    padding: 6px;
    cursor: pointer;
    color: white;
    opacity: 0;
    transition: opacity 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .book-card:hover .delete-btn {
    opacity: 1;
  }

  .delete-btn:hover {
    background: rgba(185, 28, 28, 1);
  }

  .cover-container {
    position: relative;
  }

  .book-card img {
    width: 100%;
    height: 240px;
    object-fit: cover;
    display: block;
  }

  .file-type-tag {
    position: absolute;
    top: 8px;
    right: 8px;
    background: rgba(102, 126, 234, 0.9);
    color: white;
    font-size: 0.65rem;
    font-weight: 600;
    padding: 3px 6px;
    border-radius: 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .file-type-tag.pdf {
    background: rgba(229, 62, 62, 0.9);
  }

  .progress-indicator {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: rgba(0, 0, 0, 0.3);
  }

  .progress-fill {
    height: 100%;
    background: #48bb78;
    transition: width 0.3s ease;
  }

  .no-cover {
    width: 100%;
    height: 240px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }

  .book-info {
    padding: 1rem;
  }

  .book-info h3 {
    font-size: 0.95rem;
    font-weight: 600;
    margin-bottom: 0.25rem;
    color: #1a1a1a;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .book-info p {
    font-size: 0.85rem;
    color: #718096;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-state {
    text-align: center;
    padding: 2rem;
    color: #a0aec0;
  }

  .empty-state svg {
    margin-bottom: 1rem;
  }

  .empty-state p {
    font-size: 1.1rem;
  }

  /* Dark mode styles */
  :global(.dark) .upload-zone {
    background: #2d3748;
    border-color: #4a5568;
  }

  :global(.dark) .upload-zone:hover,
  :global(.dark) .upload-zone.drag-over {
    border-color: #4299e1;
    background: #2a4365;
  }

  :global(.dark) .upload-zone label {
    color: #a0aec0;
  }

  :global(.dark) .book-card {
    background: #2d3748;
  }

  :global(.dark) .book-info h3 {
    color: #f7fafc;
  }

  :global(.dark) .book-info p {
    color: #a0aec0;
  }

  :global(.dark) .empty-state {
    color: #718096;
  }
</style>
