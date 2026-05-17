<script>
  import SidePanel from "./SidePanel.svelte";

  let { annotations, onGoTo, onDelete, onClose } = $props();
</script>

<SidePanel
  title="Highlights ({annotations.length})"
  labelId="annotations-list-title"
  {onClose}
>
  {#if annotations.length === 0}
    <p class="no-annotations">No highlights yet. Select text in the book to create a highlight.</p>
  {:else}
    {#each annotations as annotation (annotation.id)}
      <div
        class="annotation-item"
        style="border-left-color: {annotation.color};"
      >
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
</SidePanel>

<style>
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
