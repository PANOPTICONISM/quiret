<script>
  import { FOLIATE_FORMATS, TEXT_FORMATS } from "../lib/constants.js";

  let {
    visible,
    fileType,
    annotationsCount = 0,
    fontSize,
    minFontSize,
    maxFontSize,
    isFullscreen,
    isTouchDevice,
    onClose,
    onToggleAnnotations,
    onIncreaseFontSize,
    onDecreaseFontSize,
    onToggleFullscreen,
    onMouseEnter,
    onMouseLeave,
  } = $props();

  const handleMouseEnter = () => {
    if (!isTouchDevice) onMouseEnter?.();
  };

  const handleMouseLeave = () => {
    if (!isTouchDevice) onMouseLeave?.();
  };

  const fullscreenSupported =
    typeof document !== "undefined" &&
    (document.fullscreenEnabled || document.webkitFullscreenEnabled);
</script>

<div
  class="reader-header"
  class:hidden={!visible}
  onmouseenter={handleMouseEnter}
  onmouseleave={handleMouseLeave}
  role="banner"
>
  <button class="close-btn" onclick={onClose}>
    <svg
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
    >
      <path d="M19 12H5M12 19l-7-7 7-7" />
    </svg>
    Back to Library
  </button>
  <div class="header-controls">
    {#if FOLIATE_FORMATS.includes(fileType)}
      <button
        class="annotations-btn"
        onclick={onToggleAnnotations}
        aria-label="View annotations"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 20h9M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" />
        </svg>
        {#if annotationsCount > 0}
          <span class="annotation-count">{annotationsCount}</span>
        {/if}
      </button>
    {/if}
    {#if TEXT_FORMATS.includes(fileType)}
      <div class="font-size-controls">
        <button
          class="font-btn"
          onclick={onDecreaseFontSize}
          disabled={fontSize <= minFontSize}
          aria-label="Decrease font size"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </button>
        <span class="font-size-label">{fontSize}px</span>
        <button
          class="font-btn"
          onclick={onIncreaseFontSize}
          disabled={fontSize >= maxFontSize}
          aria-label="Increase font size"
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </button>
      </div>
    {/if}
    {#if fullscreenSupported}
      <button
        class="fullscreen-btn"
        onclick={onToggleFullscreen}
        aria-label={isFullscreen ? "Exit fullscreen" : "Enter fullscreen"}
      >
        {#if isFullscreen}
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3"
            />
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
            <path
              d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"
            />
          </svg>
        {/if}
      </button>
    {/if}
  </div>
</div>

<style>
  .reader-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.9rem 1.75rem;
    background: var(--reader-header-bg);
    backdrop-filter: blur(8px);
    border-bottom: 1px solid var(--border);
    z-index: 1001;
    transform: translateY(0);
    transition: transform 0.3s ease-in-out;
  }

  .reader-header.hidden {
    transform: translateY(-100%);
  }

  .close-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 0.95rem;
    color: var(--text-muted);
    padding: 0.45rem 0.85rem;
    border-radius: var(--radius);
    font-family: inherit;
    transition: background 0.15s, color 0.15s;
  }

  .close-btn:hover {
    background: var(--tint);
    color: var(--text);
  }

  .header-controls {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .annotations-btn,
  .fullscreen-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.5rem;
    border-radius: var(--radius);
    color: var(--text-muted);
    transition: background 0.15s, color 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .annotations-btn {
    position: relative;
  }

  .annotations-btn:hover,
  .fullscreen-btn:hover {
    background: var(--tint);
    color: var(--text);
  }

  .annotation-count {
    position: absolute;
    top: 0;
    right: 0;
    background: var(--accent);
    color: white;
    font-size: 0.65rem;
    font-weight: 600;
    padding: 0.1rem 0.35rem;
    border-radius: 10px;
    min-width: 16px;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  .font-size-controls {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    background: var(--tint);
    padding: 0.25rem;
    border-radius: var(--radius);
  }

  .font-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.35rem;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, color 0.15s;
  }

  .font-btn:hover:not(:disabled) {
    background: var(--tint-strong);
    color: var(--text);
  }

  .font-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .font-size-label {
    font-size: 0.8rem;
    color: var(--text-muted);
    min-width: 40px;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }
</style>
