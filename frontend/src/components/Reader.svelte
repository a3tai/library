<script lang="ts">
  import type {Book, Passage, TOCEntry} from '../types';
  import {LibraryService} from '../lib/api/library';
  import {groupPassagesByLabel, readerChapterId} from '../lib/reader/passages';
  import ReaderDisplaySettings from './reader/ReaderDisplaySettings.svelte';
  import ReaderToc from './reader/ReaderToc.svelte';
  import ReaderToolbar from './reader/ReaderToolbar.svelte';
  import type {ReaderFamily, ReaderTheme} from './reader/types';

  type Props = {
    book: Book;
    passages: Passage[];
    toc: TOCEntry[];
    initialChunkIndex?: number;
    onclose: () => void;
  };

  let {book, passages, toc, initialChunkIndex = 0, onclose}: Props = $props();

  let theme = $state<ReaderTheme>('auto');
  let family = $state<ReaderFamily>('serif');
  let osDark = $state(
    typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-color-scheme: dark)').matches,
  );

  const effectiveTheme = $derived<'light' | 'sepia' | 'dark'>(
    theme === 'auto' ? (osDark ? 'dark' : 'light') : theme,
  );

  function tryToggleMaximise() {
    LibraryService.ToggleMaximise().catch(() => {});
  }
  let fontPx = $state(17);
  let widthCh = $state(72);
  let isFullscreen = $state(false);
  let tocOpen = $state(false);
  let settingsOpen = $state(false);

  let scrollEl: HTMLDivElement | undefined = $state();
  let rootEl: HTMLDivElement | undefined = $state();
  let progress = $state(0); // 0..1

  const passagesByLabel = $derived(groupPassagesByLabel(passages, toc));

  function jumpTo(chunkIndex: number) {
    const id = readerChapterId(chunkIndex);
    const el = scrollEl?.querySelector<HTMLElement>(`#${CSS.escape(id)}`);
    if (el) el.scrollIntoView({behavior: 'smooth', block: 'start'});
    tocOpen = false;
  }

  function nextChapter() {
    if (toc.length === 0) return;
    const cur = currentTOCIndex();
    const next = Math.min(toc.length - 1, cur + 1);
    if (next !== cur) jumpTo(toc[next].chunkIndex);
  }

  function prevChapter() {
    if (toc.length === 0) return;
    const cur = currentTOCIndex();
    const prev = Math.max(0, cur - 1);
    if (prev !== cur) jumpTo(toc[prev].chunkIndex);
  }

  function currentTOCIndex(): number {
    if (!scrollEl || toc.length === 0) return 0;
    const top = scrollEl.scrollTop;
    let idx = 0;
    for (let i = 0; i < toc.length; i++) {
      const el = scrollEl.querySelector<HTMLElement>(`#${CSS.escape(readerChapterId(toc[i].chunkIndex))}`);
      if (el && el.offsetTop - 40 <= top) idx = i;
      else break;
    }
    return idx;
  }

  function adjustFont(delta: number) {
    fontPx = Math.max(13, Math.min(28, fontPx + delta));
  }

  async function toggleFullscreen() {
    if (!rootEl) return;
    if (document.fullscreenElement) {
      try { await document.exitFullscreen(); } catch { /* swallow */ }
    } else {
      try { await rootEl.requestFullscreen(); } catch { /* swallow */ }
    }
  }

  function onFsChange() {
    isFullscreen = !!document.fullscreenElement;
  }

  function onScroll() {
    if (!scrollEl) return;
    const max = scrollEl.scrollHeight - scrollEl.clientHeight;
    progress = max > 0 ? Math.min(1, scrollEl.scrollTop / max) : 0;
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (settingsOpen) { settingsOpen = false; return; }
      if (tocOpen) { tocOpen = false; return; }
      onclose();
      return;
    }
    if (e.target && (e.target as HTMLElement).matches('input, textarea, select')) return;
    if (e.key === 'ArrowRight') { e.preventDefault(); nextChapter(); }
    else if (e.key === 'ArrowLeft') { e.preventDefault(); prevChapter(); }
    else if (e.key === 't' || e.key === 'T') { tocOpen = !tocOpen; }
    else if (e.key === 'f' || e.key === 'F') { toggleFullscreen(); }
    else if (e.key === '+' || e.key === '=') { adjustFont(1); }
    else if (e.key === '-' || e.key === '_') { adjustFont(-1); }
    else if (e.key === '0') { fontPx = 17; }
  }

  $effect(() => {
    const el = scrollEl;
    if (!el) return;
    el.addEventListener('scroll', onScroll, {passive: true});
    document.addEventListener('fullscreenchange', onFsChange);
    window.addEventListener('keydown', onKey);

    // Track OS theme changes so 'auto' reflows when the user flips Light/Dark
    // in Control Center / System Settings without leaving the reader.
    const mql = window.matchMedia?.('(prefers-color-scheme: dark)');
    const onMql = (e: MediaQueryListEvent) => { osDark = e.matches; };
    mql?.addEventListener?.('change', onMql);

    if (initialChunkIndex > 0) {
      // jump after layout
      queueMicrotask(() => jumpTo(initialChunkIndex));
    }
    return () => {
      el.removeEventListener('scroll', onScroll);
      document.removeEventListener('fullscreenchange', onFsChange);
      window.removeEventListener('keydown', onKey);
      mql?.removeEventListener?.('change', onMql);
    };
  });

  const chapterLabelNow = $derived.by(() => {
    if (toc.length === 0) return '';
    return toc[currentTOCIndex()]?.label ?? '';
  });
</script>

<div class="reader" class:dark={effectiveTheme === 'dark'} class:sepia={effectiveTheme === 'sepia'} bind:this={rootEl}>
  <ReaderToolbar
    {book}
    chapterLabel={chapterLabelNow}
    {tocOpen}
    tocDisabled={toc.length === 0}
    settingsOpen={settingsOpen}
    fullscreen={isFullscreen}
    onClose={onclose}
    onToggleToc={() => (tocOpen = !tocOpen)}
    onToggleSettings={() => (settingsOpen = !settingsOpen)}
    onToggleFullscreen={toggleFullscreen}
    onDoubleClick={tryToggleMaximise}
  />

  <div class="progress-track" role="progressbar" aria-valuenow={Math.round(progress * 100)} aria-valuemin={0} aria-valuemax={100}>
    <div class="progress-fill" style="width: {progress * 100}%"></div>
  </div>

  <div class="body" class:toc-open={tocOpen && toc.length > 0}>
    {#if tocOpen && toc.length > 0}
      <ReaderToc {toc} currentIndex={currentTOCIndex()} onJump={jumpTo} />
    {/if}

    <div class="page" bind:this={scrollEl} aria-label="Book text">
      <article class="column" style="--reader-font: {fontPx}px; --reader-width: {widthCh}ch; font-family: {family === 'serif' ? 'var(--font-display)' : 'var(--font-ui)'};">
        <header class="title-page">
          <h1>{book.title}</h1>
          <p class="byline">{book.authors || 'Unknown author'}</p>
          {#if book.publisher || book.publishedDate}
            <p class="imprint">
              {book.publisher}{book.publisher && book.publishedDate ? ' · ' : ''}{book.publishedDate}
            </p>
          {/if}
        </header>

        {#each passagesByLabel as group}
          <section id={readerChapterId(group.entry.chunkIndex)} class="chapter">
            {#if group.label}
              <h2>{group.label}</h2>
            {/if}
            {#each group.items as p (p.id)}
              <p>{p.text}</p>
            {/each}
          </section>
        {/each}

        <footer class="end">
          <span>End</span>
        </footer>
      </article>
    </div>
  </div>

  {#if settingsOpen}
    <ReaderDisplaySettings
      {theme}
      {family}
      {fontPx}
      {widthCh}
      onClose={() => (settingsOpen = false)}
      onFontDelta={adjustFont}
      onFontReset={() => (fontPx = 17)}
      onFamilyChange={(next) => (family = next)}
      onThemeChange={(next) => (theme = next)}
      onWidthChange={(next) => (widthCh = next)}
    />
  {/if}
</div>

<style>
  .reader {
    position: fixed;
    inset: 0;
    z-index: 200;
    display: flex;
    flex-direction: column;
    background: var(--reader-bg, #fafaf7);
    color: var(--reader-fg, #1a1a1a);
    --reader-bg: #fafaf7;
    --reader-fg: #1a1a1a;
    --reader-mute: rgba(0,0,0,0.55);
    --reader-line: rgba(0,0,0,0.10);
    --reader-accent: #0a66e6;
  }
  .reader.sepia {
    --reader-bg: #f4ecd8;
    --reader-fg: #3a2e1f;
    --reader-mute: rgba(58,46,31,0.6);
    --reader-line: rgba(58,46,31,0.16);
    --reader-accent: #b1683b;
  }
  .reader.dark {
    --reader-bg: #1c1c1e;
    --reader-fg: #ebebec;
    --reader-mute: rgba(235,235,236,0.6);
    --reader-line: rgba(235,235,236,0.14);
    --reader-accent: #0a84ff;
  }

  .progress-track {
    height: 2px;
    background: color-mix(in srgb, var(--reader-fg) 8%, transparent);
  }
  .progress-fill {
    height: 100%;
    background: var(--reader-accent);
    transition: width 60ms linear;
  }

  /* ---- Body ----
     Default to a single full-width column so the reading area takes the
     entire window and `.column`'s auto-margins center it. When the TOC
     drawer is open we switch to a 280px + 1fr two-column grid; the
     reading column inside `.page` still auto-centers within its 1fr
     track, which keeps the prose visually balanced against the sidebar. */
  .body { flex: 1; min-height: 0; display: grid; grid-template-columns: 1fr; }
  .body.toc-open { grid-template-columns: auto 1fr; }

  /* ---- Reading column ---- */
  .page { overflow: auto; outline: none; }
  .column {
    max-width: var(--reader-width, 72ch);
    margin: var(--s-7) auto;
    padding: 0 var(--s-5) var(--s-7);
    font-size: var(--reader-font, 17px);
    line-height: 1.65;
    color: var(--reader-fg);
  }
  .title-page {
    text-align: center;
    padding: var(--s-7) 0 var(--s-7);
    border-bottom: 1px solid var(--reader-line);
    margin-bottom: var(--s-7);
  }
  .title-page h1 {
    margin: 0 0 var(--s-2);
    font-family: var(--font-display);
    font-weight: 500;
    font-size: clamp(28px, 4vw, 44px);
    line-height: 1.12;
    letter-spacing: -0.01em;
  }
  .title-page .byline { margin: 0; color: var(--reader-mute); font-size: 1em; }
  .title-page .imprint { margin: var(--s-2) 0 0; color: var(--reader-mute); font-size: 0.85em; }

  .chapter { margin-bottom: var(--s-7); }
  .chapter h2 {
    font-family: var(--font-display);
    font-weight: 500;
    font-size: 1.4em;
    margin: var(--s-7) 0 var(--s-4);
    letter-spacing: -0.005em;
  }
  .chapter p {
    margin: 0 0 1.1em;
    text-align: justify;
    hyphens: auto;
    text-indent: 1.2em;
  }
  .chapter p:first-of-type { text-indent: 0; }

  .end {
    text-align: center;
    margin-top: var(--s-7);
    padding-top: var(--s-5);
    border-top: 1px solid var(--reader-line);
    color: var(--reader-mute);
    font-family: var(--font-display);
  }

  /* In fullscreen mode the toolbar still visible; small layout adjustments. */
  .reader:fullscreen .body { background: var(--reader-bg); }

  @media (max-width: 760px) {
    .body { grid-template-columns: 1fr; }
  }
</style>
