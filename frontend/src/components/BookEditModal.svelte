<script>
  import SidePanel from "./SidePanel.svelte";

  let { book, onClose, onSaved } = $props();

  let title = $state(book.title || "");
  let author = $state(book.author || "");
  let coverFile = $state(null);
  let coverPreview = $state(null);
  let saving = $state(false);
  let error = $state(null);

  const existingCover = $derived(
    book.coverPath ? `/api/books/${book.id}/cover` : null,
  );

  const onCoverPick = (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      error = "Please choose an image file.";
      return;
    }
    error = null;
    coverFile = file;
    if (coverPreview) URL.revokeObjectURL(coverPreview);
    coverPreview = URL.createObjectURL(file);
  };

  const save = async () => {
    const trimmed = title.trim();
    if (!trimmed) {
      error = "Title is required.";
      return;
    }
    saving = true;
    error = null;
    try {
      const res = await fetch(`/api/books/${book.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ title: trimmed, author: author.trim() }),
      });
      if (!res.ok) throw new Error("Failed to save details");

      let coverPath;
      if (coverFile) {
        const fd = new FormData();
        fd.append("cover", coverFile);
        const cr = await fetch(`/api/books/${book.id}/cover`, {
          method: "POST",
          body: fd,
        });
        if (!cr.ok) throw new Error("Failed to upload cover");
        coverPath = (await cr.json()).coverPath;
      }

      onSaved({ title: trimmed, author: author.trim(), coverPath });
    } catch (e) {
      error = e.message || "Something went wrong";
      saving = false;
    }
  };
</script>

<SidePanel title="Edit details" labelId="edit-book-title" {onClose}>
  <div class="cover-row">
    <div class="cover-preview">
      {#if coverPreview || existingCover}
        <img src={coverPreview || existingCover} alt="Cover preview" />
      {:else}
        <div class="cover-placeholder">No cover</div>
      {/if}
    </div>
    <label class="cover-upload">
      <input type="file" accept="image/*" onchange={onCoverPick} hidden />
      <span>Change cover</span>
    </label>
  </div>

  <label class="field">
    <span class="field-label">Title</span>
    <input type="text" bind:value={title} placeholder="Title" />
  </label>

  <label class="field">
    <span class="field-label">Author</span>
    <input type="text" bind:value={author} placeholder="Author" />
  </label>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  <button class="save-btn" onclick={save} disabled={saving}>
    {saving ? "Saving..." : "Save changes"}
  </button>
</SidePanel>

<style>
  .cover-row {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .cover-preview {
    width: 90px;
    aspect-ratio: 2 / 3;
    flex-shrink: 0;
    border-radius: var(--radius);
    overflow: hidden;
    background: var(--surface-muted);
    border: 1px solid var(--border);
  }

  .cover-preview img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .cover-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-faint);
    font-size: 0.8rem;
  }

  .cover-upload {
    display: inline-flex;
    cursor: pointer;
  }

  .cover-upload span {
    display: inline-block;
    padding: 0.5rem 0.85rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-muted);
    font-size: 0.85rem;
    transition: background 0.15s, color 0.15s;
  }

  .cover-upload:hover span {
    background: var(--tint);
    color: var(--text);
  }

  .field {
    display: block;
    margin-bottom: 1rem;
  }

  .field-label {
    display: block;
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-bottom: 0.35rem;
  }

  .field input {
    width: 100%;
    padding: 0.6rem 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: var(--radius);
    font-size: 0.95rem;
    font-family: inherit;
  }

  .field input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .error {
    color: var(--danger);
    font-size: 0.85rem;
    margin-bottom: 1rem;
  }

  .save-btn {
    width: 100%;
    padding: 0.7rem;
    background: var(--accent);
    color: white;
    border: none;
    border-radius: var(--radius);
    font-size: 0.95rem;
    cursor: pointer;
    font-weight: 500;
    font-family: inherit;
    transition: background 0.15s;
  }

  .save-btn:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .save-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
