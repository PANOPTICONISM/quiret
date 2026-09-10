<script>
  import SidePanel from "./SidePanel.svelte";

  let { onClose, onAdded } = $props();

  let feedUrl = $state("");
  let loading = $state(false);
  let error = $state(null);
  let show = $state(null); // { title, author, image, episodes }
  let savingUrl = $state(null);
  let saved = $state(new Set());

  const fetchFeed = async () => {
    const url = feedUrl.trim();
    if (!url) return;
    loading = true;
    error = null;
    show = null;
    try {
      const res = await fetch(
        `/api/podcasts/episodes?url=${encodeURIComponent(url)}`,
      );
      if (!res.ok) throw new Error("Couldn't load that feed. Check the URL.");
      show = await res.json();
      if (!show.episodes?.length) error = "No episodes found in this feed.";
    } catch (e) {
      error = e.message || "Failed to load feed";
    } finally {
      loading = false;
    }
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

<SidePanel title="Add from podcast" labelId="podcast-title" {onClose}>
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

  {#if error}
    <p class="error">{error}</p>
  {/if}

  {#if show}
    <div class="show-head">
      <div>
        <h4>{show.title}</h4>
        {#if show.author}<p class="show-author">{show.author}</p>{/if}
      </div>
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

  .show-head {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    margin-bottom: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border);
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
