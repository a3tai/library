<!--
  BookInspector — right-rail metadata panel for the selected book.

  Mirrors the macOS Mail inspector pattern: cover at top, big title,
  authors, action buttons, then sectioned metadata + table of contents
  + passage search. Pulls `book.coverUrl/title/authors/...` straight off
  the type. Empty fields are simply omitted from the inspector list.
-->
<script lang="ts">
  import {X} from '@lucide/svelte';
  import Button from '../lib/components/ui/button/button.svelte';
  import Badge from '../lib/components/ui/badge/badge.svelte';
  import Inspector from '../lib/components/ui/inspector/inspector.svelte';
  import Input from '../lib/components/ui/input/input.svelte';
  import Notch from '../lib/components/ui/notch/notch.svelte';
  import ScrollArea from '../lib/components/ui/scroll-area/scroll-area.svelte';
  import Separator from '../lib/components/ui/separator/separator.svelte';
  import Typography from '../lib/components/ui/typography/typography.svelte';
  import CoverArt from './CoverArt.svelte';
  import type {Book, Passage, TOCEntry} from '../types';

  type Props = {
    book: Book;
    toc: TOCEntry[];
    passages: Passage[];
    passageQuery: string;
    onclose: () => void;
    onPassageQuery: (next: string) => void;
    onSearchPassages: () => void;
    onRead: (chunkIndex?: number) => void;
    onChat: () => void;
  };

  let {
    book,
    toc,
    passages,
    passageQuery,
    onclose,
    onPassageQuery,
    onSearchPassages,
    onRead,
    onChat,
  }: Props = $props();

  let localQuery = $state('');
  $effect(() => {
    localQuery = passageQuery;
  });

  function commitQuery() {
    onPassageQuery(localQuery);
    onSearchPassages();
  }

  const sections = $derived.by(() => {
    const rows: {label: string; value: string}[] = [];
    if (book.publisher) rows.push({label: 'Publisher', value: book.publisher});
    if (book.publishedDate) rows.push({label: 'Published', value: book.publishedDate});
    if (book.language) rows.push({label: 'Language', value: book.language.toUpperCase()});
    if (book.isbn13) rows.push({label: 'ISBN', value: book.isbn13});
    else if (book.isbn10) rows.push({label: 'ISBN', value: book.isbn10});
    if (book.format) rows.push({label: 'Format', value: book.format.toUpperCase()});
    if (book.passageCount) rows.push({label: 'Passages', value: book.passageCount.toLocaleString()});
    if (book.metadataSource) rows.push({label: 'Source', value: book.metadataSource});
    return rows.length > 0 ? [{title: 'Details', rows}] : [];
  });

  // Highlight <b>...</b> from the FTS snippet without dangerously
  // injecting unfiltered HTML — split on the well-known marker.
  function snippetParts(snippet: string): {bold: boolean; text: string}[] {
    const parts: {bold: boolean; text: string}[] = [];
    const re = /<b>(.*?)<\/b>/g;
    let last = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(snippet)) !== null) {
      if (m.index > last) parts.push({bold: false, text: snippet.slice(last, m.index)});
      parts.push({bold: true, text: m[1]});
      last = m.index + m[0].length;
    }
    if (last < snippet.length) parts.push({bold: false, text: snippet.slice(last)});
    return parts;
  }
</script>

<aside class="inspector" aria-label="Book details">
  <Notch class="inspector-head">
    {#snippet leading()}
      <Typography variant="eyebrow" tone="dim">Detail</Typography>
    {/snippet}
    {#snippet trailing()}
      <button class="close-btn" type="button" aria-label="Close inspector" onclick={onclose}>
        <X size={12} strokeWidth={1.8} />
      </button>
    {/snippet}
  </Notch>

  <ScrollArea class="inspector-body">
    <div class="hero">
      <CoverArt {book} width="160px" />
      <Typography variant="h3" class="hero-title">{book.title || 'Untitled'}</Typography>
      {#if book.authors}
        <Typography variant="body" tone="muted" class="hero-authors">{book.authors}</Typography>
      {/if}
      <div class="hero-tags">
        {#if book.format}
          <Badge variant="outline" size="sm">{book.format.toUpperCase()}</Badge>
        {/if}
        {#if book.indexStatus === 'indexed'}
          <Badge variant="success" size="sm">Indexed</Badge>
        {:else if book.indexStatus === 'failed'}
          <Badge variant="danger" size="sm">Index failed</Badge>
        {/if}
        {#if book.textStatus === 'text_unavailable'}
          <Badge variant="warn" size="sm">Scanned PDF</Badge>
        {/if}
      </div>
      <div class="actions">
        <Button variant="primary" size="sm" onclick={() => onRead()} disabled={book.passageCount === 0}>
          Open reader
        </Button>
        <Button variant="ghost" size="sm" onclick={onChat} disabled={book.passageCount === 0}>
          Chat
        </Button>
      </div>
    </div>

    {#if book.description}
      <section class="block">
        <Typography variant="eyebrow" tone="dim">About</Typography>
        <Typography variant="body" class="block-body">{book.description}</Typography>
      </section>
    {/if}

    {#if sections.length > 0}
      <Inspector {sections} class="inspector-meta" />
    {/if}

    <Separator />

    <section class="block">
      <Typography variant="eyebrow" tone="dim">Search inside</Typography>
      <Input
        size="sm"
        bind:value={localQuery}
        placeholder="Find a passage…"
        onkeydown={(e: KeyboardEvent) => {
          if (e.key === 'Enter') commitQuery();
        }}
      />
      {#if passages.length > 0}
        <ul class="passage-hits" role="list">
          {#each passages as p (p.id)}
            <li>
              <button type="button" class="passage" onclick={() => onRead(p.chunkIndex)}>
                <span class="passage-label">{p.label || `Passage ${p.chunkIndex + 1}`}</span>
                <span class="passage-snippet">
                  {#each snippetParts(p.snippet || p.text) as part, i (i)}
                    {#if part.bold}<mark>{part.text}</mark>{:else}{part.text}{/if}
                  {/each}
                </span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    {#if toc.length > 0}
      <Separator />
      <section class="block">
        <Typography variant="eyebrow" tone="dim">Contents</Typography>
        <ul class="toc" role="list">
          {#each toc as entry, i (i)}
            <li>
              <button type="button" class="toc-row" onclick={() => onRead(entry.chunkIndex)}>
                <span class="toc-label">{entry.label}</span>
                {#if entry.pages > 0}
                  <span class="toc-pages">{entry.pages.toLocaleString()}</span>
                {/if}
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </ScrollArea>
</aside>

<style>
  .inspector {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    border-left: 1px solid var(--uin-line);
    background: var(--uin-mat-panel);
    backdrop-filter: blur(20px) saturate(1.4);
    -webkit-backdrop-filter: blur(20px) saturate(1.4);
  }
  .close-btn {
    width: 22px;
    height: 22px;
    display: grid;
    place-items: center;
    border: 0;
    background: transparent;
    color: var(--uin-fg-mute);
    border-radius: var(--uin-r-sm);
    cursor: default;
    transition: background-color var(--uin-dur-1) var(--uin-ease-standard);
  }
  .close-btn:hover {
    background: var(--uin-mat-hover);
    color: var(--uin-fg);
  }
  .close-btn:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }

  .hero {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--uin-s-2);
    padding: var(--uin-s-4);
  }
  .hero :global(.hero-title) {
    margin: var(--uin-s-2) 0 0;
  }
  .hero :global(.hero-authors) {
    margin: 0;
  }
  .hero-tags {
    display: flex;
    flex-wrap: wrap;
    gap: var(--uin-s-1);
  }
  .actions {
    display: flex;
    gap: var(--uin-s-2);
    margin-top: var(--uin-s-2);
  }

  .block {
    padding: 0 var(--uin-s-4) var(--uin-s-4);
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
  }
  .block :global(.block-body) {
    white-space: pre-wrap;
  }

  .inspector :global(.inspector-meta) {
    width: auto;
    border: 0;
    background: transparent;
    padding: 0 var(--uin-s-4) var(--uin-s-4);
  }

  .toc,
  .passage-hits {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .toc-row {
    -webkit-app-region: no-drag;
    width: 100%;
    display: flex;
    justify-content: space-between;
    gap: var(--uin-s-3);
    padding: 6px var(--uin-s-2);
    border: 0;
    background: transparent;
    color: var(--uin-fg);
    text-align: left;
    border-radius: var(--uin-r-sm);
    cursor: default;
    font-size: 12px;
    transition: background-color var(--uin-dur-1) var(--uin-ease-standard);
  }
  .toc-row:hover {
    background: var(--uin-mat-hover);
  }
  .toc-row:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
  .toc-pages {
    font-family: var(--uin-font-mono);
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-variant-numeric: tabular-nums;
  }

  .passage {
    -webkit-app-region: no-drag;
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--uin-s-2);
    border: 0;
    background: transparent;
    color: var(--uin-fg);
    text-align: left;
    border-radius: var(--uin-r-sm);
    cursor: default;
    transition: background-color var(--uin-dur-1) var(--uin-ease-standard);
  }
  .passage:hover {
    background: var(--uin-mat-hover);
  }
  .passage:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
  .passage-label {
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .passage-snippet {
    font-size: 12px;
    line-height: 1.4;
    color: var(--uin-fg);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .passage-snippet :global(mark) {
    background: color-mix(in srgb, var(--uin-accent) 25%, transparent);
    color: inherit;
    border-radius: 2px;
    padding: 0 2px;
  }
</style>
