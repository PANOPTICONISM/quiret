<script>
  let {
    selectedText,
    annotationNote = $bindable(""),
    annotationColor = $bindable("yellow"),
    onSave,
    onClose,
  } = $props();

  const highlightColors = ["yellow", "green", "blue", "pink", "orange"];
</script>

<div class="annotation-panel">
  <div class="annotation-panel-header">
    <h3>Add Highlight</h3>
    <button class="close-panel-btn" onclick={onClose} aria-label="Close annotation panel">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 6L6 18M6 6l12 12" />
      </svg>
    </button>
  </div>
  <div class="selected-text">
    "{selectedText?.slice(0, 100)}{selectedText?.length > 100 ? '...' : ''}"
  </div>
  <div class="color-picker">
    {#each highlightColors as color}
      <button
        class="color-btn"
        class:selected={annotationColor === color}
        style="background-color: {color};"
        onclick={() => annotationColor = color}
        aria-label="Select {color} highlight"
      ></button>
    {/each}
  </div>
  <textarea
    class="annotation-note"
    bind:value={annotationNote}
    placeholder="Add a note (optional)..."
    rows="3"
  ></textarea>
  <button class="save-annotation-btn" onclick={onSave}>
    Save highlight
  </button>
</div>

<style>
  .annotation-panel {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    background: var(--surface);
    border-top: 1px solid var(--border);
    padding: 1.5rem;
    z-index: 1002;
    box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.08);
    animation: slideUp 0.2s ease-out;
  }

  @keyframes slideUp {
    from { transform: translateY(100%); }
    to { transform: translateY(0); }
  }

  .annotation-panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .annotation-panel-header h3 {
    margin: 0;
    font-family: var(--font-serif);
    font-size: 1.15rem;
    font-weight: 500;
    color: var(--text);
  }

  .close-panel-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.25rem;
    color: var(--text-muted);
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, color 0.15s;
  }

  .close-panel-btn:hover {
    background: var(--tint);
    color: var(--text);
  }

  .selected-text {
    font-family: var(--font-serif);
    font-style: italic;
    color: var(--text-muted);
    padding: 0.75rem 1rem;
    background: var(--surface-muted);
    border-radius: var(--radius);
    margin-bottom: 1rem;
    font-size: 0.95rem;
    line-height: 1.5;
    max-height: 80px;
    overflow: hidden;
  }

  .color-picker {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .color-btn {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: transform 0.15s, border-color 0.15s;
    padding: 0;
  }

  .color-btn:hover {
    transform: scale(1.1);
  }

  .color-btn.selected {
    border-color: var(--text);
    transform: scale(1.1);
  }

  .annotation-note {
    width: 100%;
    padding: 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: var(--radius);
    font-size: 0.95rem;
    resize: none;
    margin-bottom: 1rem;
    font-family: inherit;
  }

  .annotation-note::placeholder {
    color: var(--text-faint);
  }

  .annotation-note:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .save-annotation-btn {
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

  .save-annotation-btn:hover {
    background: var(--accent-hover);
  }
</style>
