<script>
  import SidePanel from "./SidePanel.svelte";

  let { items, onGoTo, onClose } = $props();
</script>

<SidePanel title="Contents" labelId="toc-title" {onClose}>
  {#if items.length === 0}
    <p class="empty">This book has no table of contents.</p>
  {:else}
    <nav>
      {#each items as item, i (i)}
        <button
          class="toc-item"
          style="padding-left: {0.85 + item.level * 0.9}rem"
          onclick={() => onGoTo(item)}
        >
          {item.label || "Untitled"}
        </button>
      {/each}
    </nav>
  {/if}
</SidePanel>

<style>
  .empty {
    color: var(--text-muted);
    text-align: center;
    padding: 2rem 1rem;
    font-size: 0.9rem;
  }

  nav {
    display: flex;
    flex-direction: column;
  }

  .toc-item {
    text-align: left;
    background: none;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    padding: 0.6rem 0.85rem;
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.92rem;
    line-height: 1.4;
    transition: background 0.15s, color 0.15s;
  }

  .toc-item:hover {
    background: var(--tint);
    color: var(--text);
  }
</style>
