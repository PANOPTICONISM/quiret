<script>
  import SidePanel from "./SidePanel.svelte";
  import { HIGHLIGHT_COLOR_NAMES } from "../lib/constants.js";

  let {
    selectedText,
    annotationNote = $bindable(""),
    annotationColor = $bindable("yellow"),
    onSave,
    onClose,
  } = $props();
</script>

<SidePanel title="Add highlight" labelId="annotation-panel-title" {onClose}>
  <div class="selected-text">
    "{selectedText?.slice(0, 100)}{selectedText?.length > 100 ? '...' : ''}"
  </div>
  <div class="color-picker">
    {#each HIGHLIGHT_COLOR_NAMES as color}
      <button
        class="color-btn"
        class:selected={annotationColor === color}
        style="background-color: {color};"
        onclick={() => (annotationColor = color)}
        aria-label="Select {color} highlight"
      ></button>
    {/each}
  </div>
  <textarea
    class="annotation-note"
    bind:value={annotationNote}
    placeholder="Add a note (optional)..."
    rows="6"
  ></textarea>
  <button class="save-annotation-btn" onclick={onSave}>
    Save highlight
  </button>
</SidePanel>

<style>
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
