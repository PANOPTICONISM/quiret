<script>
  import { onMount } from "svelte";
  import SidePanel from "./SidePanel.svelte";

  let { onClose, onAdded } = $props();

  let feedUrl = $state("");
  let loading = $state(false);
  let error = $state(null);
  let show = $state(null); // { feedUrl, title, author, image, episodes }
  let loadedUrl = $state(null); // canonical URL of the loaded show
  let savingUrl = $state(null);
  let saved = $state(new Set());

  let savedFeeds = $state([]);
  let savingFeed = $state(false);
  let savedFilter = $state("");

  const currentSaved = $derived(
    !!loadedUrl && savedFeeds.some((f) => f.url === loadedUrl),
  );

  const filteredSaved = $derived.by(() => {
    const q = savedFilter.trim().toLowerCase();
    if (!q) return savedFeeds;
    return savedFeeds.filter((f) => (f.title || f.url).toLowerCase().includes(q));
  });

  const clearShow = () => {
    show = null;
    loadedUrl = null;
  };

  const loadSavedFeeds = async () => {
    try {
      const res = await fetch("/api/podcasts/feeds");
      if (res.ok) savedFeeds = await res.json();
    } catch {}
  };

  onMount(loadSavedFeeds);

  const fetchFeed = async (url) => {
    const target = (url ?? feedUrl).trim();
    if (!target) return;
    feedUrl = target;
    loading = true;
    error = null;
    show = null;
    try {
      const res = await fetch(
        `/api/podcasts/episodes?url=${encodeURIComponent(target)}`,
      );
      if (!res.ok) throw new Error("Couldn't load that feed. Check the URL.");
      show = await res.json();
      loadedUrl = show.feedUrl || target;
      if (!show.episodes?.length) error = "No episodes found in this feed.";
    } catch (e) {
      error = e.message || "Failed to load feed";
    } finally {
      loading = false;
    }
  };

  const saveFeed = async () => {
    if (!loadedUrl || savingFeed) return;
    savingFeed = true;
    try {
      const res = await fetch("/api/podcasts/feeds", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: loadedUrl }),
      });
      if (res.ok) await loadSavedFeeds();
    } catch {
    } finally {
      savingFeed = false;
    }
  };

  const removeFeed = async (id) => {
    try {
      const res = await fetch(`/api/podcasts/feeds/${id}`, { method: "DELETE" });
      if (res.ok) savedFeeds = savedFeeds.filter((f) => f.id !== id);
    } catch {}
  };

  const addEpisode = async (ep) => {
    if (savingUrl) return;
    savingUrl = ep.audioUrl;
    error = null;
    try {
      const res = await fetch("/api/podcasts/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          audioUrl: ep.audioUrl,
          title: ep.title,
          author: show.author || show.title,
          image: ep.image,
        }),
      });
      if (!res.ok) throw new Error("Download failed");
      const book = await res.json();
      saved = new Set(saved).add(ep.audioUrl);
      onAdded?.(book);
    } catch (e) {
      error = e.message || "Download failed";
    } finally {
      savingUrl = null;
    }
  };

  const fmtDate = (s) => {
    const d = new Date(s);
    return isNaN(d)
      ? ""
      : d.toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
        });
  };
</script>

<SidePanel title="Podcasts" labelId="podcast-title" {onClose}>
  {#if error}
    <p class="error">{error}</p>
  {/if}

  {#if show}
    <button class="back-btn" onclick={clearShow}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M15 18l-6-6 6-6" />
      </svg>
      Shows
    </button>

    <div class="show-head">
      <div class="show-meta">
        <h4>{show.title}</h4>
        {#if show.author}<p class="show-author">{show.author}</p>{/if}
      </div>
      {#if currentSaved}
        <span class="saved-badge">Saved</span>
      {:else}
        <button class="save-feed-btn" onclick={saveFeed} disabled={savingFeed}>
          {savingFeed ? "Saving..." : "Save show"}
        </button>
      {/if}
    </div>

    <ul class="episodes">
      {#each show.episodes as ep (ep.audioUrl)}
        <li>
          <div class="ep-info">
            <span class="ep-title">{ep.title}</span>
            <span class="ep-meta">
              {[fmtDate(ep.pubDate), ep.duration].filter(Boolean).join(" · ")}
            </span>
          </div>
          {#if saved.has(ep.audioUrl)}
            <span class="added">Added</span>
          {:else}
            <button
              class="add-btn"
              onclick={() => addEpisode(ep)}
              disabled={!!savingUrl}
            >
              {savingUrl === ep.audioUrl ? "Downloading..." : "Add"}
            </button>
          {/if}
        </li>
      {/each}
    </ul>
  {:else}
    <form
      class="feed-row"
      onsubmit={(e) => {
        e.preventDefault();
        fetchFeed();
      }}
    >
      <input
        type="url"
        bind:value={feedUrl}
        placeholder="Paste an RSS feed URL"
        aria-label="RSS feed URL"
      />
      <button type="submit" disabled={loading || !feedUrl.trim()}>
        {loading ? "Loading..." : "Load"}
      </button>
    </form>

    {#if savedFeeds.length > 0}
      <h5 class="saved-label">Saved shows</h5>
      {#if savedFeeds.length > 6}
        <input
          class="saved-filter"
          bind:value={savedFilter}
          placeholder="Filter saved shows"
          aria-label="Filter saved shows"
        />
      {/if}
      <div class="saved-list">
        {#each filteredSaved as f (f.id)}
          <div class="saved-item">
            <button class="saved-load" onclick={() => fetchFeed(f.url)}>
              {f.title || f.url}
            </button>
            <button
              class="saved-remove"
              onclick={() => removeFeed(f.id)}
              aria-label="Remove saved show"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <path d="M18 6L6 18M6 6l12 12" />
              </svg>
            </button>
          </div>
        {/each}
        {#if filteredSaved.length === 0}
          <p class="saved-empty">No saved shows match.</p>
        {/if}
      </div>
    {/if}
  {/if}
</SidePanel>

<style>
  .feed-row {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .feed-row input {
    flex: 1;
    padding: 0.6rem 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: var(--radius);
    font-size: 0.9rem;
    font-family: inherit;
  }

  .feed-row input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .feed-row button {
    background: var(--accent);
    color: white;
    border: none;
    padding: 0 0.95rem;
    border-radius: var(--radius);
    font-size: 0.9rem;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s;
  }

  .feed-row button:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .feed-row button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error {
    color: var(--danger);
    font-size: 0.85rem;
    margin-bottom: 1rem;
  }

  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.85rem;
    padding: 0.35rem 0.5rem 0.35rem 0.35rem;
    margin-bottom: 0.85rem;
    border-radius: var(--radius-sm);
    transition: background 0.15s, color 0.15s;
  }

  .back-btn:hover {
    background: var(--tint);
    color: var(--text);
  }

  .saved-label {
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-faint);
    margin-bottom: 0.5rem;
  }

  .saved-filter {
    width: 100%;
    padding: 0.5rem 0.7rem;
    margin-bottom: 0.5rem;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    border-radius: var(--radius);
    font-size: 0.85rem;
    font-family: inherit;
  }

  .saved-filter:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .saved-list {
    max-height: 46vh;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .saved-empty {
    color: var(--text-faint);
    font-size: 0.85rem;
    padding: 0.5rem;
  }

  .saved-item {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    border-radius: var(--radius);
  }

  .saved-load {
    flex: 1;
    min-width: 0;
    text-align: left;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.55rem 0.6rem;
    border-radius: var(--radius);
    color: var(--text);
    font-family: inherit;
    font-size: 0.9rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: background 0.15s;
  }

  .saved-load:hover {
    background: var(--tint);
  }

  .saved-remove {
    flex-shrink: 0;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-faint);
    padding: 0.4rem;
    border-radius: var(--radius-sm);
    display: flex;
    transition: background 0.15s, color 0.15s;
  }

  .saved-remove:hover {
    background: var(--tint);
    color: var(--danger);
  }

  .show-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    margin-bottom: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border);
  }

  .show-meta {
    min-width: 0;
  }

  .show-head h4 {
    font-family: var(--font-serif);
    font-size: 1rem;
    color: var(--text);
    line-height: 1.2;
  }

  .show-author {
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-top: 0.15rem;
  }

  .save-feed-btn {
    flex-shrink: 0;
    padding: 0.4rem 0.8rem;
    font-size: 0.82rem;
    border: 1px solid var(--accent);
    background: var(--accent-soft);
    color: var(--accent);
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-family: inherit;
    transition: background 0.15s;
  }

  .save-feed-btn:hover:not(:disabled) {
    background: var(--accent);
    color: white;
  }

  .save-feed-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .saved-badge {
    flex-shrink: 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .episodes {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .episodes li {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.7rem 0;
    border-bottom: 1px solid var(--border);
  }

  .ep-info {
    flex: 1;
    min-width: 0;
  }

  .ep-title {
    display: block;
    font-size: 0.9rem;
    color: var(--text);
    line-height: 1.35;
  }

  .ep-meta {
    display: block;
    font-size: 0.78rem;
    color: var(--text-faint);
    margin-top: 0.2rem;
    font-variant-numeric: tabular-nums;
  }

  .add-btn {
    flex-shrink: 0;
    padding: 0.4rem 0.8rem;
    font-size: 0.82rem;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-muted);
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-family: inherit;
    transition: background 0.15s, color 0.15s, border-color 0.15s;
  }

  .add-btn:hover:not(:disabled) {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
  }

  .add-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .added {
    flex-shrink: 0;
    font-size: 0.82rem;
    color: var(--accent);
    font-weight: 500;
  }
</style>
