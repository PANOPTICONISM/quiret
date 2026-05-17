<script>
  import { onMount } from "svelte";

  let { annotations, onGoTo, onDelete, onClose } = $props();

  let initialFocusEl;

  onMount(() => {
    const previouslyFocused = document.activeElement;
    initialFocusEl?.focus();
    return () => {
      if (previouslyFocused instanceof HTMLElement) {
        previouslyFocused.focus();
      }
    };
  });
</script>

<button class="backdrop" onclick={onClose} aria-label="Close" tabindex="-1"></button>
<div
  class="annotations-list-panel"
  role="dialog"
  aria-modal="true"
  aria-labelledby="annotations-list-title"
>
  <div class="panel-header">
    <h3 id="annotations-list-title">Highlights ({annotations.length})</h3>
    <button
      class="close-panel-btn"
      onclick={onClose}
      aria-label="Close annotations list"
      bind:this={initialFocusEl}
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 6L6 18M6 6l12 12" />
      </svg>
    </button>
  </div>
  <div class="annotations-list">
    {#if annotations.length === 0}
      <p class="no-annotations">No highlights yet. Select text in the book to create a highlight.</p>
    {:else}
      {#each annotations as annotation (annotation.id)}
        <div class="annotation-item" style="border-left-color: {annotation.color};">
          <div class="annotation-text">"{annotation.text.slice(0, 150)}{annotation.text.length > 150 ? '...' : ''}"</div>
          {#if annotation.note}
            <div class="annotation-item-note">{annotation.note}</div>
          {/if}
          <div class="annotation-actions">
            <button class="go-to-btn" onclick={() => onGoTo(annotation.cfi)}>
              Go to
            </button>
            <button class="delete-btn" onclick={() => onDelete(annotation.id)}>
              Delete
            </button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.3);
    border: none;
    padding: 0;
    cursor: default;
    z-index: 1002;
    animation: fadeIn 0.15s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .annotations-list-panel {
    position: fixed;
    top: 0;
    right: 0;
    width: 350px;
    max-width: 90vw;
    height: 100vh;
    background: var(--surface);
    border-left: 1px solid var(--border);
    z-index: 1003;
    box-shadow: -4px 0 20px rgba(0, 0, 0, 0.15);
    animation: slideIn 0.2s ease-out;
    display: flex;
    flex-direction: column;
  }

  @keyframes slideIn {
    from { transform: translateX(100%); }
    to { transform: translateX(0); }
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--border);
  }

  .panel-header h3 {
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

  .annotations-list {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
  }

  .no-annotations {
    color: var(--text-muted);
    text-align: center;
    padding: 2rem 1rem;
    font-size: 0.9rem;
  }

  .annotation-item {
    padding: 0.75rem 0.85rem;
    border-left: 3px solid yellow;
    background: var(--surface-muted);
    border-radius: 0 var(--radius) var(--radius) 0;
    margin-bottom: 0.75rem;
  }

  .annotation-text {
    font-family: var(--font-serif);
    font-size: 0.95rem;
    color: var(--text);
    line-height: 1.5;
    font-style: italic;
  }

  .annotation-item-note {
    font-size: 0.85rem;
    color: var(--text-muted);
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px dashed var(--border);
  }

  .annotation-actions {
    display: flex;
    gap: 0.4rem;
    margin-top: 0.75rem;
  }

  .go-to-btn,
  .delete-btn {
    padding: 0.35rem 0.7rem;
    font-size: 0.8rem;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-muted);
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-family: inherit;
    transition: background 0.15s, color 0.15s, border-color 0.15s;
  }

  .go-to-btn:hover {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
  }

  .delete-btn:hover {
    background: var(--danger);
    color: white;
    border-color: var(--danger);
  }
</style>
