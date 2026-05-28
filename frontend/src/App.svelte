<script lang="ts">
  import {onMount} from 'svelte';
  import {Events} from '@wailsio/runtime';
  import {LibraryService} from './lib/api/library';
  import Toaster from './lib/components/ui/toast/toaster.svelte';
  import {toast} from './lib/components/ui/toast/toast.svelte';
  import {emptyImporterStatus, importLabelFor, importLampFor} from './lib/importer/status';
  import {
    bookFormatBuckets,
    bookLanguageBuckets,
    defaultSearchFilters,
    filterBooks,
    hasAnyResults,
    setSort as setSearchSort,
    setStringFilter,
  } from './lib/library/search';
  import {viewLabelFor} from './lib/library/views';

  import BookInspector from './views/BookInspector.svelte';
  import ImporterSheet from './views/ImporterSheet.svelte';
  import LibraryMain from './views/LibraryMain.svelte';
  import LibrarySidebar from './views/LibrarySidebar.svelte';
  import SettingsView from './views/SettingsView.svelte';
  import WindowChrome from './views/WindowChrome.svelte';

  // Reader and Chat keep their pre-Mittsu internals for now; the
  // legacy CSS variable aliases in app.css cover their styling.
  import Chat from './components/Chat.svelte';
  import Reader from './components/Reader.svelte';

  import type {
    AggregateGroup,
    Book,
    ImporterStatus,
    LibraryView,
    Passage,
    ResultType,
    SearchFilters,
    SearchResults as SearchResultsT,
    SortKey,
    Stats,
    TOCEntry,
  } from './types';
  import {
    isWailsAvailable,
    mockSnapshot,
    mockPassages,
    mockRunningImport,
    mockDiscoveringImport,
    mockIdleImport,
    mockTOC,
    mockReaderPassages,
    type Snapshot,
  } from './devMocks';

  // ─── Library state ─────────────────────────────────────────────────────────
  let books = $state<Book[]>([]);
  let selected = $state<Book | null>(null);
  let passages = $state<Passage[]>([]);
  let passageQuery = $state('');
  let toc = $state<TOCEntry[]>([]);
  let stats = $state<Stats>({books: 0, passages: 0, needsMetadata: 0});
  let status = $state('Ready');
  let loading = $state(false);
  let hydrating = $state(false);

  // ─── Overlays ──────────────────────────────────────────────────────────────
  let inspectorOpen = $state(false);
  let readerOpen = $state(false);
  let readerPassages = $state<Passage[]>([]);
  let readerInitialIdx = $state(0);
  let chatOpen = $state(false);
  let importerSheetOpen = $state(false);

  // ─── MCP ───────────────────────────────────────────────────────────────────
  let mcpRunning = $state(false);
  let mcpURL = $state('');
  let mcpPort = $state(8765);

  // ─── App version (rendered in the sidebar footer) ─────────────────────────
  let appVersion = $state('');

  // ─── Importer ──────────────────────────────────────────────────────────────
  let importer = $state<ImporterStatus>(emptyImporterStatus());
  let pollTimer: number | undefined;

  // ─── Demo / runtime detection ──────────────────────────────────────────────
  let demoMode = $state(false);
  let searchEl: HTMLInputElement | null = $state(null);

  // ─── View navigation ───────────────────────────────────────────────────────
  let view = $state<LibraryView>('library');
  let aggGroups = $state<AggregateGroup[]>([]);
  let aggLoading = $state(false);

  // ─── Search ────────────────────────────────────────────────────────────────
  let query = $state('');
  let searchResults = $state<SearchResultsT | null>(null);
  let searchOpen = $state(false);
  let searchTimer: number | undefined;
  const SEARCH_DEBOUNCE_MS = 250;

  let filters = $state<SearchFilters>(defaultSearchFilters());

  let mode = $state<'grid' | 'list'>('grid');

  function resetFilters() {
    filters = defaultSearchFilters();
  }
  function toggleFormat(f: string) {
    filters = setStringFilter(filters, 'formats', f);
  }
  function toggleLanguage(l: string) {
    filters = setStringFilter(filters, 'languages', l);
  }
  function setType(next: ResultType) {
    filters = {...filters, type: next};
  }
  function setSort(next: SortKey) {
    filters = setSearchSort(filters, next);
  }

  const filteredBooks = $derived.by(() => {
    if (!searchResults) return [] as Book[];
    return filterBooks(searchResults.books, filters);
  });

  const formatBuckets = $derived.by(() => {
    return bookFormatBuckets(searchResults?.books ?? []);
  });
  const languageBuckets = $derived.by(() => {
    return bookLanguageBuckets(searchResults?.books ?? []);
  });

  const viewLabel = $derived(viewLabelFor(view));

  // ─── Importer lamp ─────────────────────────────────────────────────────────
  const importLamp = $derived(importLampFor(importer));
  const importLabel = $derived(importLabelFor(importer));

  // ─── Snapshot loading ──────────────────────────────────────────────────────
  async function load() {
    loading = true;
    const forceDemo =
      typeof window !== 'undefined' && new URLSearchParams(window.location.search).has('demo');
    if (forceDemo) {
      demoMode = true;
      applySnapshot(mockSnapshot());
      status = 'Preview mode';
      loading = false;
      return;
    }
    try {
      const snap = (await LibraryService.Snapshot(query)) as Snapshot;
      applySnapshot(snap);
    } catch (error) {
      if (!isWailsAvailable()) {
        demoMode = true;
        applySnapshot(mockSnapshot());
        status = 'Preview (no Wails runtime)';
      } else {
        status = error instanceof Error ? error.message : String(error);
      }
    } finally {
      loading = false;
    }
  }

  function applySnapshot(snap: Snapshot) {
    books = snap.books ?? [];
    stats = snap.stats ?? stats;
    hydrating = snap.hydrating ?? false;
    mcpRunning = snap.mcp?.running ?? false;
    mcpURL = snap.mcp?.url ?? '';
    if (snap.mcp?.port) mcpPort = snap.mcp.port;
    if (selected) {
      const fresh = books.find((b) => b.id === selected!.id);
      if (fresh) selected = fresh;
    }
  }

  // ─── Search runners ────────────────────────────────────────────────────────
  function clearSearchTimer() {
    if (searchTimer !== undefined) {
      window.clearTimeout(searchTimer);
      searchTimer = undefined;
    }
  }
  function onQueryChange(next: string) {
    query = next;
  }
  function onQueryInput() {
    clearSearchTimer();
    if (!query.trim()) {
      searchResults = null;
      searchOpen = false;
      void load();
      return;
    }
    searchTimer = window.setTimeout(() => {
      void runSearch();
    }, SEARCH_DEBOUNCE_MS);
  }

  async function runSearch() {
    if (!query.trim()) return;
    passages = [];
    if (demoMode) {
      searchResults = {
        query,
        books: books.slice(0, 6),
        passages: mockPassages(),
        authors: [
          {name: 'Charles Petzold', count: 1},
          {name: 'Christopher Alexander', count: 1},
        ],
        subjects: [
          {name: 'Computability', count: 1},
          {name: 'Mathematical logic', count: 1},
        ],
      };
      searchOpen = true;
      return;
    }
    try {
      const results = (await LibraryService.Search(query)) as SearchResultsT;
      searchResults = results;
      searchOpen = true;
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
      toast.error({title: 'Search failed', description: status});
    }
  }

  function search() {
    clearSearchTimer();
    if (!query.trim()) {
      searchResults = null;
      searchOpen = false;
      void load();
      return;
    }
    void runSearch();
  }

  function clearSearch() {
    query = '';
    searchResults = null;
    searchOpen = false;
    void load();
    searchEl?.focus();
  }
  function onSearchFocus() {
    if (query.trim() && searchResults && hasAnyResults(searchResults)) {
      searchOpen = true;
    }
  }
  function onAuthor(name: string) {
    query = `author:"${name}"`;
    void runSearch();
  }
  function onSubject(name: string) {
    query = `subject:"${name}"`;
    void runSearch();
  }
  function onPassageJump(p: Passage) {
    const target = books.find((b) => b.id === p.bookId);
    if (target) {
      selectBook(target);
      searchOpen = false;
    } else {
      LibraryService.GetBook(p.bookId)
        .then((b) => {
          if (b) {
            selectBook(b as Book);
            searchOpen = false;
          }
        })
        .catch(() => {});
    }
  }

  // ─── View nav ──────────────────────────────────────────────────────────────
  function switchView(next: LibraryView) {
    if (view === next) return;
    view = next;
    aggGroups = [];
    if (next === 'settings') {
      // Settings view manages its own data; nothing to preload here.
      return;
    }
    if (next === 'library' || next === 'recent' || next === 'unprocessed') {
      void loadViewBooks(next);
    } else {
      void loadAggregations(next);
    }
  }

  async function loadViewBooks(v: LibraryView) {
    if (demoMode) return;
    aggLoading = true;
    try {
      const list =
        v === 'unprocessed'
          ? ((await LibraryService.ListUnprocessed(200, 0)) as Book[])
          : v === 'recent'
            ? ((await LibraryService.ListRecentlyAdded(200, 0)) as Book[])
            : ((await LibraryService.ListBooks(query, 200, 0)) as Book[]);
      books = list ?? [];
      if (selected) {
        const stillSelected = books.find((b) => b.id === selected!.id);
        if (!stillSelected) selected = null;
      }
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
    } finally {
      aggLoading = false;
    }
  }

  async function loadAggregations(v: LibraryView) {
    aggLoading = true;
    try {
      let groups: AggregateGroup[] = [];
      if (demoMode) {
        groups =
          v === 'byauthor'
            ? [
                {name: 'Charles Petzold', count: 1},
                {name: 'Christopher Alexander', count: 1},
                {name: 'Yuval Noah Harari', count: 1},
                {name: 'William Zinsser', count: 1},
                {name: 'Bill Watterson', count: 1},
              ]
            : v === 'bysubject'
              ? [
                  {name: 'Computability', count: 1},
                  {name: 'Turing machines', count: 1},
                  {name: 'Mathematical logic', count: 1},
                  {name: 'History of computing', count: 1},
                ]
              : [
                  {name: 'Computer science', count: 1},
                  {name: 'Architecture', count: 1},
                  {name: 'History', count: 1},
                ];
      } else if (v === 'byauthor') {
        groups = (await LibraryService.ListAuthors(200)) as AggregateGroup[];
      } else if (v === 'bysubject') {
        groups = (await LibraryService.ListSubjects(200)) as AggregateGroup[];
      } else if (v === 'categories') {
        groups = (await LibraryService.ListCategories(200)) as AggregateGroup[];
      }
      aggGroups = groups ?? [];
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
    } finally {
      aggLoading = false;
    }
  }

  function selectGroup(name: string) {
    const key = view === 'byauthor' ? 'author' : view === 'bysubject' ? 'subject' : 'genre';
    query = `${key}:"${name}"`;
    view = 'library';
    onQueryInput();
  }

  // ─── Book selection ────────────────────────────────────────────────────────
  let refreshTimer: number | undefined;
  function selectBook(book: Book) {
    selected = book;
    inspectorOpen = true;
    passages = [];
    toc = [];
    void loadTOC(book.id);
    if (refreshTimer !== undefined) {
      window.clearTimeout(refreshTimer);
      refreshTimer = undefined;
    }
    if (demoMode) return;
    if (book.coverUrl && book.description) return;
    LibraryService.RequestMetadata(book.id).catch(() => {});
    refreshTimer = window.setTimeout(async () => {
      refreshTimer = undefined;
      if (selected?.id !== book.id) return;
      try {
        const fresh = (await LibraryService.GetBook(book.id)) as Book;
        const idx = books.findIndex((b) => b.id === book.id);
        if (idx >= 0) books[idx] = fresh;
        if (selected?.id === fresh.id) selected = fresh;
      } catch {
        /* swallow */
      }
    }, 3500);
  }

  async function loadTOC(bookID: string) {
    if (demoMode) {
      toc = bookID === '1' ? mockTOC() : [];
      return;
    }
    try {
      toc = (await LibraryService.BookTOC(bookID)) as TOCEntry[];
    } catch {
      toc = [];
    }
  }

  async function searchPassages() {
    if (!selected || !passageQuery.trim()) return;
    try {
      passages = (await LibraryService.SearchPassages(passageQuery, selected.id, 30)) as Passage[];
    } catch (error) {
      if (demoMode) {
        passages = mockPassages();
      } else {
        status = error instanceof Error ? error.message : String(error);
      }
    }
  }

  // ─── Reader / Chat ─────────────────────────────────────────────────────────
  async function openReader(chunkIndex?: number) {
    if (!selected) return;
    readerOpen = true;
    readerInitialIdx = chunkIndex ?? 0;
    if (demoMode) {
      readerPassages = mockReaderPassages();
      return;
    }
    try {
      readerPassages = (await LibraryService.BookPassages(selected.id, 0, 1000)) as Passage[];
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
    }
  }
  function closeReader() {
    readerOpen = false;
    readerPassages = [];
  }
  function openChat() {
    if (selected) chatOpen = true;
  }
  function closeChat() {
    chatOpen = false;
  }

  // ─── Import ────────────────────────────────────────────────────────────────
  async function pickAndImport() {
    toast({title: 'Opening file picker…', duration: 1500});
    if (demoMode) {
      importer = mockRunningImport();
      importerSheetOpen = true;
      return;
    }
    try {
      const path = (await LibraryService.PickImportPath()) as string;
      if (!path) {
        toast({title: 'No path chosen', duration: 1500});
        return;
      }
      await startImport(path);
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
      toast.error({title: 'Import failed', description: status});
    }
  }

  async function startImport(path: string) {
    const wasRunning = importer.running;
    try {
      const next = (await LibraryService.ImportPath(path)) as ImporterStatus;
      importer = next;
      if (!wasRunning) importerSheetOpen = true;
      schedulePoll();
      if (wasRunning) {
        const depth = next.queuedPaths?.length ?? 0;
        toast({
          title: 'Queued',
          description:
            depth > 0 ? `${depth} folder${depth === 1 ? '' : 's'} waiting` : 'Will run after the current import.',
        });
      }
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
      toast.error({title: 'Import failed', description: status});
    }
  }

  async function cancelImport() {
    if (demoMode) {
      importer = {...importer, running: false, done: true, cancelled: true};
      return;
    }
    try {
      await LibraryService.CancelImport();
    } catch {
      /* swallow */
    }
  }

  function schedulePoll() {
    if (pollTimer !== undefined) return;
    pollTimer = window.setInterval(pollImporter, 500);
  }
  function stopPoll() {
    if (pollTimer !== undefined) {
      window.clearInterval(pollTimer);
      pollTimer = undefined;
    }
  }
  async function pollImporter() {
    try {
      const next = (await LibraryService.ImporterStatus()) as ImporterStatus;
      importer = next;
      if (!next.running && next.done) {
        stopPoll();
        const s = next.summary;
        if (s) {
          if (next.cancelled) {
            toast.warn({title: 'Import cancelled', description: `${s.imported} imported, ${s.updated} updated`});
          } else if (s.failed > 0) {
            toast.warn({title: 'Import complete with errors', description: `${s.imported} imported · ${s.failed} failed`});
          } else {
            toast.success({title: 'Import complete', description: `${s.imported} imported, ${s.updated} updated`});
          }
        } else if (next.error) {
          toast.error({title: 'Import failed', description: next.error});
        }
        await load();
      }
    } catch (error) {
      stopPoll();
      if (!demoMode) status = error instanceof Error ? error.message : String(error);
    }
  }

  async function hydrate() {
    if (hydrating) return;
    hydrating = true;
    try {
      const count = (await LibraryService.HydrateMetadata(80)) as number;
      toast.success({title: 'Metadata refresh queued', description: `${count} books in flight`});
      await load();
    } catch (error) {
      if (!demoMode) {
        status = error instanceof Error ? error.message : String(error);
        toast.error({title: 'Refresh failed', description: status});
      }
    } finally {
      hydrating = false;
    }
  }

  async function toggleMCP() {
    try {
      const next = mcpRunning
        ? await LibraryService.StopMCPServer()
        : await LibraryService.StartMCPServer(Number(mcpPort) || 8765);
      const v = next as {running: boolean; url: string; port: number};
      mcpRunning = v.running;
      mcpURL = v.url;
      if (v.port) mcpPort = v.port;
      toast({title: mcpRunning ? `MCP at ${mcpURL}` : 'MCP stopped'});
    } catch (error) {
      if (demoMode) {
        mcpRunning = !mcpRunning;
        mcpURL = mcpRunning ? `http://127.0.0.1:${mcpPort}/mcp` : '';
      } else {
        status = error instanceof Error ? error.message : String(error);
        toast.error({title: 'MCP toggle failed', description: status});
      }
    }
  }

  // ─── Keyboard ──────────────────────────────────────────────────────────────
  function onGlobalKey(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K' || e.key === 'f' || e.key === 'F')) {
      e.preventDefault();
      searchEl?.focus();
      searchEl?.select();
      return;
    }
    if (e.key === 'Escape') {
      const active = document.activeElement;
      if (active === searchEl && query) {
        e.preventDefault();
        clearSearch();
        return;
      }
      if (searchOpen) {
        e.preventDefault();
        searchOpen = false;
        return;
      }
      if (inspectorOpen) {
        e.preventDefault();
        inspectorOpen = false;
      }
    }
  }

  function onTitleBarDblClick(e: MouseEvent) {
    if (e.target === e.currentTarget || (e.target as HTMLElement).closest('.uin-titlebar-text')) {
      LibraryService.ToggleMaximise().catch(() => {});
    }
  }

  // ─── Mount ─────────────────────────────────────────────────────────────────
  onMount(() => {
    load().then(async () => {
      if (demoMode) {
        const params = new URLSearchParams(window.location.search);
        if (params.has('discovering')) {
          importer = mockDiscoveringImport();
          if (params.has('sheet')) importerSheetOpen = true;
        } else if (params.has('importing')) {
          importer = mockRunningImport();
          if (params.has('sheet')) importerSheetOpen = true;
        } else {
          importer = mockIdleImport();
        }
        if (params.has('reader')) await openReader();
        if (params.has('chat')) openChat();
        return;
      }
      hydrate();
      try {
        const s = (await LibraryService.ImporterStatus()) as ImporterStatus;
        importer = s;
        if (s.running) schedulePoll();
      } catch {
        /* swallow */
      }
    });
    window.addEventListener('keydown', onGlobalKey);
    // Best-effort version fetch for the sidebar footer. Failure leaves
    // the field blank — not worth surfacing to the user.
    LibraryService.Version()
      .then((v) => {
        appVersion = (v as string) ?? '';
      })
      .catch(() => {});
    const offOpenSettings = Events.On('app:open-settings', () => {
      switchView('settings');
    });
    return () => {
      window.removeEventListener('keydown', onGlobalKey);
      stopPoll();
      offOpenSettings();
    };
  });

  const showInspector = $derived(inspectorOpen && selected !== null);
</script>

<div class="window">
  <WindowChrome
    {query}
    {hydrating}
    onQueryChange={onQueryChange}
    onQueryInput={onQueryInput}
    onSearchFocus={onSearchFocus}
    onSearchEnter={search}
    onClear={clearSearch}
    onHydrate={hydrate}
    onDoubleClick={onTitleBarDblClick}
    bindEl={(el) => (searchEl = el)}
  />

  <div class="layout" class:layout-with-inspector={showInspector}>
    <LibrarySidebar
      {view}
      {stats}
      {importer}
      {importLamp}
      {importLabel}
      {mcpRunning}
      {mcpURL}
      {mcpPort}
      version={appVersion}
      onSwitchView={switchView}
      onAddBooks={pickAndImport}
      onOpenImporter={() => (importerSheetOpen = true)}
      onToggleMCP={toggleMCP}
      onMCPPortChange={(p) => (mcpPort = p)}
    />

    {#if view === 'settings'}
      <SettingsView />
    {:else}
      <LibraryMain
        {view}
        {viewLabel}
        {query}
        {searchOpen}
        {loading}
        {aggLoading}
        books={searchOpen ? filteredBooks : books}
        selectedId={selected?.id}
        {aggGroups}
        {searchResults}
        {filters}
        {formatBuckets}
        {languageBuckets}
        {mode}
        onModeChange={(m) => (mode = m)}
        onSortChange={setSort}
        onSelectBook={selectBook}
        onSelectGroup={selectGroup}
        onTypeChange={setType}
        onToggleFormat={toggleFormat}
        onToggleLanguage={toggleLanguage}
        onResetFilters={resetFilters}
        onSelectPassage={onPassageJump}
        onSelectAuthor={onAuthor}
        onSelectSubject={onSubject}
        onPickAndImport={pickAndImport}
      />
    {/if}

    {#if showInspector}
      <BookInspector
        book={selected!}
        {toc}
        {passages}
        {passageQuery}
        onclose={() => (inspectorOpen = false)}
        onPassageQuery={(v) => (passageQuery = v)}
        onSearchPassages={searchPassages}
        onRead={openReader}
        onChat={openChat}
      />
    {/if}
  </div>

  <ImporterSheet
    open={importerSheetOpen}
    {importer}
    {importLamp}
    onOpenChange={(v) => (importerSheetOpen = v)}
    onCancel={cancelImport}
  />

  {#if readerOpen && selected}
    <Reader book={selected} passages={readerPassages} {toc} initialChunkIndex={readerInitialIdx} onclose={closeReader} />
  {/if}

  {#if chatOpen && selected}
    <Chat book={selected} demoMode={demoMode} onclose={closeChat} />
  {/if}
</div>

<Toaster position="bottom-right" />

<style>
  .window {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--uin-bg-base);
  }
  .layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: 244px 1fr;
  }
  .layout-with-inspector {
    grid-template-columns: 244px 1fr 340px;
  }

  @media (max-width: 1100px) {
    .layout {
      grid-template-columns: 200px 1fr;
    }
    .layout-with-inspector {
      grid-template-columns: 200px 1fr 320px;
    }
  }
  @media (max-width: 820px) {
    .layout,
    .layout-with-inspector {
      grid-template-columns: 1fr;
    }
  }
</style>
