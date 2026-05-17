<script>
  import { onMount } from "svelte";

  let { title, labelId, onClose, children } = $props();

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
  class="side-panel"
  role="dialog"
  aria-modal="true"
  aria-labelledby={labelId}
>
  <div class="side-panel-header">
    <h3 id={labelId}>{title}</h3>
    <button
      class="close-btn"
      onclick={onClose}
      aria-label="Close panel"
      bind:this={initialFocusEl}
    >
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M18 6L6 18M6 6l12 12" />
      </svg>
    </button>
  </div>
  <div class="side-panel-body">
    {@render children?.()}
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

  .side-panel {
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

  .side-panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .side-panel-header h3 {
    margin: 0;
    font-family: var(--font-serif);
    font-size: 1.15rem;
    font-weight: 500;
    color: var(--text);
  }

  .close-btn {
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

  .close-btn:hover {
    background: var(--tint);
    color: var(--text);
  }

  .side-panel-body {
    flex: 1;
    overflow-y: auto;
    padding: 1.25rem 1.5rem 1.5rem;
  }
</style>
