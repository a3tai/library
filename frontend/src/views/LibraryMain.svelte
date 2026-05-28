<!--
  LibraryMain — center panel.

  PageHeader (eyebrow + count) · Toolbar (view-mode toggle, sort) ·
  body that switches between book grid, aggregation grid, and search
  results. The body scrolls independently inside its own ScrollArea.
-->
<script lang="ts">
  import Empty from '../lib/components/ui/empty/empty.svelte';
  import NativeSelect from '../lib/components/ui/native-select/native-select.svelte';
  import PageHeader from '../lib/components/ui/page-header/page-header.svelte';
  import ScrollArea from '../lib/components/ui/scroll-area/scroll-area.svelte';
  import SegmentedControl from '../lib/components/ui/segmented-control/segmented-control.svelte';
  import Spinner from '../lib/components/ui/spinner/spinner.svelte';

  import AggregationGrid from './AggregationGrid.svelte';
  import BookGrid from './BookGrid.svelte';
  import SearchResultsView from './SearchResultsView.svelte';
  import type {
    AggregateGroup,
    Book,
    LibraryView,
    Passage,
    ResultType,
    SearchFilters,
    SearchResults,
    SortKey,
  } from '../types';

  type Props = {
    view: LibraryView;
    viewLabel: string;
    query: string;
    searchOpen: boolean;

    loading: boolean;
    aggLoading: boolean;

    books: Book[];
    selectedId?: string;
    aggGroups: AggregateGroup[];

    searchResults: SearchResults | null;
    filters: SearchFilters;
    formatBuckets: Array<[string, number]>;
    languageBuckets: Array<[string, number]>;

    mode: 'grid' | 'list';
    onModeChange: (next: 'grid' | 'list') => void;
    onSortChange: (next: SortKey) => void;

    onSelectBook: (book: Book) => void;
    onSelectGroup: (name: string) => void;
    onTypeChange: (next: ResultType) => void;
    onToggleFormat: (value: string) => void;
    onToggleLanguage: (value: string) => void;
    onResetFilters: () => void;
    onSelectPassage: (p: Passage) => void;
    onSelectAuthor: (name: string) => void;
    onSelectSubject: (name: string) => void;
    onPickAndImport: () => void;
  };

  let {
    view,
    viewLabel,
    query,
    searchOpen,
    loading,
    aggLoading,
    books,
    selectedId,
    aggGroups,
    searchResults,
    filters,
    formatBuckets,
    languageBuckets,
    mode,
    onModeChange,
    onSortChange,
    onSelectBook,
    onSelectGroup,
    onTypeChange,
    onToggleFormat,
    onToggleLanguage,
    onResetFilters,
    onSelectPassage,
    onSelectAuthor,
    onSelectSubject,
    onPickAndImport,
  }: Props = $props();

  const modeOptions = [
    {value: 'grid' as const, label: 'Covers'},
    {value: 'list' as const, label: 'List'},
  ];

  const sortOptions = [
    {value: 'relevance' as SortKey, label: 'Relevance'},
    {value: 'newest' as SortKey, label: 'Newest'},
    {value: 'title' as SortKey, label: 'Title'},
    {value: 'author' as SortKey, label: 'Author'},
  ];

  const isAggView = $derived(
    view === 'byauthor' || view === 'bysubject' || view === 'categories'
  );
</script>

<section class="main" aria-label="Library">
  <ScrollArea class="main-scroll">
    <PageHeader
      eyebrow={searchOpen && query ? 'Search' : viewLabel}
      title={searchOpen && query ? `“${query}”` : viewLabel}
      description={searchOpen
        ? `Showing matches for ${query}`
        : view === 'library'
          ? 'Everything in your library.'
          : view === 'recent'
            ? 'The most recently imported books.'
            : view === 'unprocessed'
              ? 'Books that still need metadata or indexing.'
              : `Drill into a ${view === 'byauthor' ? 'name' : view === 'bysubject' ? 'subject' : 'category'} to filter.`}
      sticky
    >
      {#snippet actions()}
        {#if !searchOpen && !isAggView}
          <SegmentedControl
            options={modeOptions}
            value={mode}
            onChange={onModeChange}
            ariaLabel="View mode"
            size="sm"
          />
          <NativeSelect
            value={filters.sort}
            options={sortOptions}
            aria-label="Sort"
            size="sm"
            onchange={(e) => onSortChange((e.currentTarget as HTMLSelectElement).value as SortKey)}
          />
        {/if}
      {/snippet}
    </PageHeader>

    {#if searchOpen && searchResults}
      <SearchResultsView
        results={searchResults}
        {filters}
        {formatBuckets}
        {languageBuckets}
        {selectedId}
        {mode}
        {onTypeChange}
        {onToggleFormat}
        {onToggleLanguage}
        {onResetFilters}
        {onSelectBook}
        {onSelectPassage}
        {onSelectAuthor}
        {onSelectSubject}
      />
    {:else if isAggView}
      <AggregationGrid
        groups={aggGroups}
        loading={aggLoading}
        eyebrow={view === 'byauthor' ? 'Authors' : view === 'bysubject' ? 'Subjects' : 'Categories'}
        onSelect={onSelectGroup}
      />
    {:else if loading}
      <div class="centered"><Spinner /></div>
    {:else if books.length === 0}
      <Empty
        title={view === 'unprocessed'
          ? 'Nothing pending'
          : view === 'recent'
            ? 'Nothing added yet'
            : 'Your library is empty'}
        description={view === 'unprocessed'
          ? 'Every book has been processed.'
          : view === 'recent'
            ? 'Books you import will appear here, newest first.'
            : 'Click Import in the sidebar to choose a folder of EPUB or PDF books.'}
      >
        {#snippet action()}
          {#if view === 'library'}
            <button type="button" class="cta" onclick={onPickAndImport}>Import…</button>
          {/if}
        {/snippet}
      </Empty>
    {:else}
      <BookGrid {books} {selectedId} {mode} onSelect={onSelectBook} />
    {/if}
  </ScrollArea>
</section>

<style>
  .main {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    height: 100%;
    background: var(--uin-bg-base);
  }
  .centered {
    display: grid;
    place-items: center;
    padding: var(--uin-s-8) 0;
  }
  .cta {
    -webkit-app-region: no-drag;
    border: 0;
    background: var(--uin-accent);
    color: var(--uin-accent-fg);
    padding: 7px var(--uin-s-3);
    border-radius: var(--uin-r-sm);
    font-size: 12.5px;
    font-weight: 500;
    cursor: default;
  }
  .cta:hover {
    filter: brightness(1.06);
  }
  .cta:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
</style>
