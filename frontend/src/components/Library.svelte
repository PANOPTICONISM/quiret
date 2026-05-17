<script>
  import { onMount } from "svelte";
  import {
    SUPPORTED_EXTENSIONS,
    FILE_ACCEPT,
    FOLIATE_FORMATS,
  } from "../lib/constants.js";

  let { onOpenBook } = $props();

  let books = $state([]);
  let uploading = $state(false);
  let darkMode = $state(false);
  let searchQuery = $state("");
  let sortBy = $state("added");
  let dragDepth = $state(0);

  const isDragging = $derived(dragDepth > 0);

  const filteredBooks = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase();
    let list = books;
    if (q) {
      list = list.filter(
        (b) =>
          b.title?.toLowerCase().includes(q) ||
          b.author?.toLowerCase().includes(q),
      );
    }
    if (sortBy === "title") {
      return [...list].sort((a, b) =>
        (a.title || "").localeCompare(b.title || ""),
      );
    }
    if (sortBy === "author") {
      return [...list].sort((a, b) =>
        (a.author || "").localeCompare(b.author || ""),
      );
    }
    return list;
  });

  onMount(() => {
    darkMode = localStorage.getItem("darkMode") === "true";
    applyDarkMode(darkMode);
    fetchBooks();

    document.addEventListener("dragenter", onDocDragEnter);
    document.addEventListener("dragleave", onDocDragLeave);
    document.addEventListener("dragover", onDocDragOver);
    document.addEventListener("drop", onDocDrop);

    return () => {
      document.removeEventListener("dragenter", onDocDragEnter);
      document.removeEventListener("dragleave", onDocDragLeave);
      document.removeEventListener("dragover", onDocDragOver);
      document.removeEventListener("drop", onDocDrop);
    };
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

  const isFileDrag = (e) => e.dataTransfer?.types?.includes?.("Files");

  const onDocDragEnter = (e) => {
    if (isFileDrag(e)) {
      e.preventDefault();
      dragDepth++;
    }
  };

  const onDocDragLeave = () => {
    if (dragDepth > 0) dragDepth--;
  };

  const onDocDragOver = (e) => {
    if (isFileDrag(e)) e.preventDefault();
  };

  const onDocDrop = (e) => {
    e.preventDefault();
    dragDepth = 0;
    const file = e.dataTransfer?.files?.[0];
    if (file) tryUploadFile(file);
  };

  const handleFilePicker = async (event) => {
    const file = event.target.files?.[0];
    if (file) {
      await tryUploadFile(file);
      event.target.value = "";
    }
  };

  const tryUploadFile = async (file) => {
    const filename = file.name.toLowerCase();
    const isSupported = SUPPORTED_EXTENSIONS.some((ext) =>
      filename.endsWith(ext),
    );
    if (!isSupported) {
      alert("Supported formats: EPUB, PDF, FB2, CBZ");
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

  const getReadingProgress = (book) => {
    if (!book.readingProgress) return 0;
    try {
      const progress = JSON.parse(book.readingProgress);
      if (
        FOLIATE_FORMATS.includes(progress.type) &&
        progress.fraction !== undefined
      ) {
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
    if (!confirm(`Delete "${bookTitle}"?`)) return;

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

  const triggerUpload = () => {
    document.getElementById("file-input")?.click();
  };
</script>

<input
  type="file"
  accept={FILE_ACCEPT}
  onchange={handleFilePicker}
  id="file-input"
  hidden
/>

<div class="container">
  <header class="library-header">
    <div class="brand">
      <h1>Quiret</h1>
      {#if books.length > 0}
        <span class="count">{books.length} {books.length === 1 ? "book" : "books"}</span>
      {/if}
    </div>
    <div class="header-actions">
      {#if books.length > 0}
        <div class="search">
          <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="7" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            type="search"
            placeholder="Search"
            bind:value={searchQuery}
            class="search-input"
          />
        </div>
        <div class="select-wrapper">
          <select class="sort-select" bind:value={sortBy} aria-label="Sort books">
            <option value="added">Recently added</option>
            <option value="title">Title</option>
            <option value="author">Author</option>
          </select>
          <svg class="select-chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </div>
      {/if}
      <button
        class="upload-btn"
        onclick={triggerUpload}
        disabled={uploading}
      >
        {#if uploading}
          <span class="spinner-sm"></span>
          Uploading
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="17 8 12 3 7 8" />
            <line x1="12" y1="3" x2="12" y2="15" />
          </svg>
          Upload
        {/if}
      </button>
      <button
        class="icon-btn"
        onclick={toggleDarkMode}
        aria-label="Toggle dark mode"
      >
        {#if darkMode}
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          </svg>
        {/if}
      </button>
    </div>
  </header>

  {#if filteredBooks.length > 0}
    <div class="books-grid">
      {#each filteredBooks as book (book.id)}
        <article class="book-card">
          <button
            type="button"
            class="book-card-main"
            onclick={() => onOpenBook(book.id)}
          >
            <div class="cover-container">
              {#if book.coverPath}
                <img
                  src="/api/books/{book.id}/cover"
                  alt={book.title}
                  loading="lazy"
                />
              {:else}
                <div class="no-cover">
                  <span class="no-cover-title">{book.title}</span>
                </div>
              {/if}
              <span class="file-type-tag">
                {book.fileType?.toUpperCase() || "EPUB"}
              </span>
              {#if getReadingProgress(book) > 0}
                <div
                  class="progress-indicator"
                  aria-label="Reading progress"
                >
                  <div
                    class="progress-fill"
                    style="width: {getReadingProgress(book)}%"
                  ></div>
                </div>
              {/if}
            </div>
            <div class="book-info">
              <h3>{book.title}</h3>
              <p>{book.author || "Unknown"}</p>
            </div>
          </button>
          <button
            type="button"
            class="delete-btn"
            onclick={(e) => deleteBook(e, book.id, book.title)}
            aria-label="Delete book"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </article>
      {/each}
    </div>
  {:else if books.length > 0}
    <div class="empty-state">
      <p>No books match "{searchQuery}"</p>
    </div>
  {:else}
    <div class="empty-state empty-state-onboard">
      <h2>Your library is empty</h2>
      <p>Drop an EPUB, PDF, FB2, or CBZ anywhere on this page — or pick one to upload.</p>
      <button class="upload-btn primary" onclick={triggerUpload}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="17 8 12 3 7 8" />
          <line x1="12" y1="3" x2="12" y2="15" />
        </svg>
        Upload your first book
      </button>
    </div>
  {/if}
</div>

{#if isDragging}
  <div class="drop-overlay" aria-hidden="true">
    <div class="drop-overlay-inner">
      <svg width="42" height="42" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
        <polyline points="17 8 12 3 7 8" />
        <line x1="12" y1="3" x2="12" y2="15" />
      </svg>
      <p>Drop to add to your library</p>
    </div>
  </div>
{/if}

<style>
  .container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2.5rem 2rem 4rem;
  }

  .library-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1.5rem;
    margin-bottom: 2.5rem;
    flex-wrap: wrap;
  }

  .brand {
    display: flex;
    align-items: baseline;
    gap: 0.85rem;
  }

  h1 {
    font-family: var(--font-serif);
    font-size: 2.25rem;
    font-weight: 500;
    letter-spacing: -0.01em;
    color: var(--text);
    line-height: 1;
  }

  .count {
    font-size: 0.85rem;
    color: var(--text-faint);
    font-variant-numeric: tabular-nums;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .search {
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 0.7rem;
    color: var(--text);
    pointer-events: none;
  }

  .search-input {
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    padding: 0.5rem 0.85rem 0.5rem 2.1rem;
    border-radius: var(--radius);
    font-size: 0.9rem;
    line-height: 1.25;
    width: 180px;
    font-family: inherit;
    transition: border-color 0.15s, box-shadow 0.15s;
  }

  .search-input::placeholder {
    color: var(--text);
  }

  .search-input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .select-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
  }

  .sort-select {
    appearance: none;
    -webkit-appearance: none;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    padding: 0.5rem 2.1rem 0.5rem 0.85rem;
    border-radius: var(--radius);
    font-size: 0.9rem;
    line-height: 1.25;
    font-family: inherit;
    cursor: pointer;
  }

  .sort-select:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .select-chevron {
    position: absolute;
    right: 0.7rem;
    color: var(--text-muted);
    pointer-events: none;
  }

  .upload-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    background: var(--accent);
    color: white;
    border: none;
    padding: 0.5rem 0.95rem;
    border-radius: var(--radius);
    font-size: 0.9rem;
    line-height: 1.25;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s;
  }

  .upload-btn:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .upload-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .upload-btn.primary {
    padding: 0.7rem 1.2rem;
    font-size: 1rem;
    margin-top: 1.5rem;
  }

  .icon-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    cursor: pointer;
    padding: 0.5rem;
    border-radius: var(--radius);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, color 0.15s;
  }

  .icon-btn:hover {
    background: var(--surface);
    color: var(--text);
  }

  .spinner-sm {
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    display: inline-block;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .books-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 2.25rem 1.25rem;
  }

  .book-card {
    position: relative;
    transition: transform 0.15s;
  }

  .book-card:hover {
    transform: translateY(-2px);
  }

  .book-card-main {
    display: block;
    width: 100%;
    cursor: pointer;
    border: none;
    padding: 0;
    background: none;
    text-align: left;
    font-family: inherit;
    color: inherit;
  }

  .cover-container {
    position: relative;
    aspect-ratio: 2 / 3;
    border-radius: var(--radius);
    overflow: hidden;
    background: var(--surface-muted);
    box-shadow: var(--shadow);
    border: 1px solid var(--border);
  }

  .book-card img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .no-cover {
    width: 100%;
    height: 100%;
    background: var(--surface-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    color: var(--text);
  }

  .no-cover-title {
    font-family: var(--font-serif);
    font-size: 1rem;
    line-height: 1.3;
    text-align: center;
    display: -webkit-box;
    -webkit-line-clamp: 6;
    line-clamp: 6;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .file-type-tag {
    position: absolute;
    bottom: 0.5rem;
    left: 0.5rem;
    background: #D85940;
    color: #FFFFFF;
    font-size: 0.6rem;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 3px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .progress-indicator {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: var(--progress-track);
  }

  .progress-fill {
    height: 100%;
    background: var(--accent);
    transition: width 0.3s ease;
  }

  .delete-btn {
    position: absolute;
    top: 0.4rem;
    right: 0.4rem;
    background: rgba(28, 26, 23, 0.75);
    color: white;
    border: none;
    border-radius: 50%;
    width: 24px;
    height: 24px;
    padding: 0;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, background 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .book-card:hover .delete-btn,
  .delete-btn:focus-visible {
    opacity: 1;
  }

  .delete-btn:hover {
    background: var(--danger);
  }

  .book-info {
    padding: 0.85rem 0.25rem 0;
  }

  .book-info h3 {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text);
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    margin-bottom: 0.2rem;
  }

  .book-info p {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-style: italic;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty-state {
    text-align: center;
    padding: 3rem 1rem;
    color: var(--text-muted);
  }

  .empty-state-onboard {
    padding: 6rem 1rem;
  }

  .empty-state-onboard h2 {
    font-family: var(--font-serif);
    font-size: 1.6rem;
    font-weight: 500;
    color: var(--text);
    margin-bottom: 0.5rem;
  }

  .empty-state-onboard p {
    color: var(--text-muted);
  }

  .drop-overlay {
    position: fixed;
    inset: 0;
    background: var(--drop-overlay-bg);
    backdrop-filter: blur(2px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
    pointer-events: none;
    animation: fadeIn 0.15s ease-out;
  }

  .drop-overlay-inner {
    background: var(--surface);
    color: var(--accent);
    padding: 2rem 3rem;
    border-radius: var(--radius-lg);
    border: 2px dashed var(--accent);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
    box-shadow: var(--shadow);
  }

  .drop-overlay-inner p {
    font-family: var(--font-serif);
    font-size: 1.1rem;
    color: var(--text);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @media (max-width: 600px) {
    .container {
      padding: 1.5rem 1rem 3rem;
    }
    .library-header {
      margin-bottom: 1.75rem;
    }
    .search-input { width: 130px; }
    h1 { font-size: 1.85rem; }
    .books-grid {
      grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
      gap: 1.75rem 1rem;
    }
  }
</style>
