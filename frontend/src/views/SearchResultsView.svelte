<!--
  SearchResultsView — unified results: books, passages, authors,
  subjects. The four sections come back together from
  LibraryService.Search and are gated by a SegmentedControl.

  Books are rendered through the same cover grid as the main library
  view; passage / author / subject sections are list-shaped.
-->
<script lang="ts">
  import Chip from '../lib/components/ui/chip/chip.svelte';
  import Empty from '../lib/components/ui/empty/empty.svelte';
  import SegmentedControl from '../lib/components/ui/segmented-control/segmented-control.svelte';
  import Typography from '../lib/components/ui/typography/typography.svelte';
  import BookGrid from './BookGrid.svelte';
  import type {Book, Passage, ResultType, SearchFilters, SearchResults} from '../types';

  type Props = {
    results: SearchResults;
    filters: SearchFilters;
    formatBuckets: Array<[string, number]>;
    languageBuckets: Array<[string, number]>;
    selectedId?: string;
    mode: 'grid' | 'list';
    onTypeChange: (next: ResultType) => void;
    onToggleFormat: (value: string) => void;
    onToggleLanguage: (value: string) => void;
    onResetFilters: () => void;
    onSelectBook: (book: Book) => void;
    onSelectPassage: (p: Passage) => void;
    onSelectAuthor: (name: string) => void;
    onSelectSubject: (name: string) => void;
  };

  let {
    results,
    filters,
    formatBuckets,
    languageBuckets,
    selectedId,
    mode,
    onTypeChange,
    onToggleFormat,
    onToggleLanguage,
    onResetFilters,
    onSelectBook,
    onSelectPassage,
    onSelectAuthor,
    onSelectSubject,
  }: Props = $props();

  const counts = $derived({
    books: results.books?.length ?? 0,
    passages: results.passages?.length ?? 0,
    authors: results.authors?.length ?? 0,
    subjects: results.subjects?.length ?? 0,
  });

  const segOpts = $derived([
    {value: 'books' as const, label: `Books · ${counts.books}`},
    {value: 'passages' as const, label: `Passages · ${counts.passages}`},
    {value: 'authors' as const, label: `Authors · ${counts.authors}`},
    {value: 'subjects' as const, label: `Subjects · ${counts.subjects}`},
  ]);

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

  const hasAnyResults = $derived(
    counts.books + counts.passages + counts.authors + counts.subjects > 0
  );

  const filtersActive = $derived(filters.formats.size > 0 || filters.languages.size > 0);
</script>

<div class="results">
  <div class="results-bar">
    <SegmentedControl
      options={segOpts}
      value={filters.type}
      onChange={onTypeChange}
      ariaLabel="Result type"
      size="sm"
    />
    {#if filters.type === 'books' && (formatBuckets.length > 0 || languageBuckets.length > 0)}
      <div class="filter-strip">
        {#each formatBuckets as [fmt, count] (fmt)}
          <Chip active={filters.formats.has(fmt)} {count} onClick={() => onToggleFormat(fmt)}>
            {fmt.toUpperCase()}
          </Chip>
        {/each}
        {#each languageBuckets as [lang, count] (lang)}
          <Chip active={filters.languages.has(lang)} {count} onClick={() => onToggleLanguage(lang)}>
            {lang.toUpperCase()}
          </Chip>
        {/each}
        {#if filtersActive}
          <button class="clear-link" type="button" onclick={onResetFilters}>Clear</button>
        {/if}
      </div>
    {/if}
  </div>

  {#if !hasAnyResults}
    <Empty
      title={`No results for “${results.query}”`}
      description="Try a broader query, or use the field prefix — author:, subject:, or genre:."
    />
  {:else if filters.type === 'books'}
    {#if results.books.length === 0}
      <Empty title="No books matched" description="Adjust filters above." />
    {:else}
      <BookGrid books={results.books} {selectedId} {mode} onSelect={onSelectBook} />
    {/if}
  {:else if filters.type === 'passages'}
    {#if results.passages.length === 0}
      <Empty title="No passages matched" />
    {:else}
      <ul class="passages" role="list">
        {#each results.passages as p (p.id)}
          <li>
            <button type="button" class="passage" onclick={() => onSelectPassage(p)}>
              <Typography variant="caption" tone="dim" class="passage-source">
                {p.bookTitle}{p.label ? ` · ${p.label}` : ''}
              </Typography>
              <Typography variant="body" class="passage-snippet">
                {#each snippetParts(p.snippet || p.text) as part, i (i)}
                  {#if part.bold}<mark>{part.text}</mark>{:else}{part.text}{/if}
                {/each}
              </Typography>
              {#if p.authors}
                <Typography variant="caption" tone="muted">{p.authors}</Typography>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {:else if filters.type === 'authors'}
    {#if results.authors.length === 0}
      <Empty title="No authors matched" />
    {:else}
      <ul class="agg-rows" role="list">
        {#each results.authors as a (a.name)}
          <li>
            <button type="button" class="agg-row" onclick={() => onSelectAuthor(a.name)}>
              <span class="agg-name">{a.name}</span>
              <span class="agg-count">{a.count.toLocaleString()}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {:else}
    {#if results.subjects.length === 0}
      <Empty title="No subjects matched" />
    {:else}
      <ul class="agg-rows" role="list">
        {#each results.subjects as s (s.name)}
          <li>
            <button type="button" class="agg-row" onclick={() => onSelectSubject(s.name)}>
              <span class="agg-name">{s.name}</span>
              <span class="agg-count">{s.count.toLocaleString()}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</div>

<style>
  .results {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .results-bar {
    position: sticky;
    top: 0;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
    padding: var(--uin-s-3) 0;
    background: var(--uin-mat-window);
    border-bottom: 1px solid var(--uin-line);
    backdrop-filter: blur(20px) saturate(1.4);
    -webkit-backdrop-filter: blur(20px) saturate(1.4);
  }
  .filter-strip {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--uin-s-1);
  }
  .clear-link {
    -webkit-app-region: no-drag;
    background: transparent;
    border: 0;
    color: var(--uin-accent);
    font-size: 11px;
    padding: 4px 6px;
    border-radius: var(--uin-r-sm);
    cursor: default;
  }
  .clear-link:hover {
    text-decoration: underline;
  }
  .clear-link:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }

  .passages,
  .agg-rows {
    list-style: none;
    margin: 0;
    padding: var(--uin-s-3) 0;
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-1);
  }
  .agg-rows {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: var(--uin-s-1);
  }
  .passage,
  .agg-row {
    -webkit-app-region: no-drag;
    width: 100%;
    border: 1px solid var(--uin-line);
    background: var(--uin-mat-row);
    color: var(--uin-fg);
    text-align: left;
    border-radius: var(--uin-r-md);
    padding: var(--uin-s-2) var(--uin-s-3);
    cursor: default;
    transition:
      background-color var(--uin-dur-1) var(--uin-ease-standard),
      border-color var(--uin-dur-1) var(--uin-ease-standard);
  }
  .passage {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .agg-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--uin-s-3);
    font-size: 12.5px;
  }
  .passage:hover,
  .agg-row:hover {
    background: var(--uin-mat-hover);
    border-color: var(--uin-line-strong);
  }
  .passage:focus-visible,
  .agg-row:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
  .passage :global(.passage-source) {
    margin: 0;
  }
  .passage :global(.passage-snippet) {
    margin: 0;
    line-height: 1.45;
  }
  .passage :global(.passage-snippet mark) {
    background: color-mix(in srgb, var(--uin-accent) 25%, transparent);
    color: inherit;
    border-radius: 2px;
    padding: 0 2px;
  }

  .agg-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .agg-count {
    font-family: var(--uin-font-mono);
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-variant-numeric: tabular-nums;
  }
</style>
