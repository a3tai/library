<script lang="ts">
  import {ChevronLeft, List, Maximize2, Minimize2} from '@lucide/svelte';
  import type {Book} from '../../types';

  type Props = {
    book: Book;
    chapterLabel: string;
    tocOpen: boolean;
    tocDisabled: boolean;
    settingsOpen: boolean;
    fullscreen: boolean;
    onClose: () => void;
    onToggleToc: () => void;
    onToggleSettings: () => void;
    onToggleFullscreen: () => void;
    onDoubleClick: () => void;
  };

  let {
    book,
    chapterLabel,
    tocOpen,
    tocDisabled,
    settingsOpen,
    fullscreen,
    onClose,
    onToggleToc,
    onToggleSettings,
    onToggleFullscreen,
    onDoubleClick,
  }: Props = $props();
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<header class="bar" ondblclick={onDoubleClick}>
  <button class="icon" type="button" onclick={onClose} title="Back to library (Esc)" aria-label="Close reader">
    <ChevronLeft size={16} strokeWidth={1.8} />
  </button>
  <div class="title-block">
    <strong>{book.title}</strong>
    {#if chapterLabel}
      <span class="chap">{chapterLabel}</span>
    {/if}
  </div>
  <div class="controls">
    <button class="icon" type="button" class:active={tocOpen} onclick={onToggleToc} title="Contents (T)" aria-label="Contents" disabled={tocDisabled}>
      <List size={16} strokeWidth={1.7} />
    </button>
    <button class="icon aa" type="button" class:active={settingsOpen} onclick={onToggleSettings} title="Display settings" aria-label="Display settings">
      <span aria-hidden="true">A<small>a</small></span>
    </button>
    <button class="icon" type="button" onclick={onToggleFullscreen} title="Fullscreen (F)" aria-label="Fullscreen">
      {#if fullscreen}
        <Minimize2 size={15} strokeWidth={1.7} />
      {:else}
        <Maximize2 size={15} strokeWidth={1.7} />
      {/if}
    </button>
  </div>
</header>

<style>
  .bar {
    flex-shrink: 0;
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--s-3);
    padding: 6px var(--s-4) 6px 80px;
    border-bottom: 1px solid var(--reader-line);
    background: color-mix(in srgb, var(--reader-bg) 85%, transparent);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    -webkit-app-region: drag;
  }
  .title-block {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 0;
    line-height: 1.2;
  }
  .title-block strong,
  .title-block .chap {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 60vw;
  }
  .title-block strong {
    font-size: 12.5px;
    font-weight: 600;
  }
  .title-block .chap {
    font-size: 11px;
    color: var(--reader-mute);
  }
  .controls {
    display: flex;
    gap: 4px;
  }
  .icon {
    -webkit-app-region: no-drag;
    width: 28px;
    height: 28px;
    display: grid;
    place-items: center;
    border: 0;
    background: transparent;
    color: var(--reader-fg);
    border-radius: var(--r-sm);
    cursor: default;
    transition: background-color var(--dur-1) var(--ease-standard);
  }
  .icon:hover:not(:disabled) {
    background: color-mix(in srgb, var(--reader-fg) 10%, transparent);
  }
  .icon.active {
    background: color-mix(in srgb, var(--reader-accent) 16%, transparent);
    color: var(--reader-accent);
  }
  .icon:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--reader-accent) 35%, transparent);
  }
  .icon:disabled {
    opacity: 0.35;
  }
  .icon.aa {
    font-family: var(--font-display);
    font-size: 14px;
    font-weight: 500;
  }
  .icon.aa small {
    font-size: 9px;
    vertical-align: -2px;
  }
</style>
