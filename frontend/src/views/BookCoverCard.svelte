<!--
  BookCoverCard — a single tile in the cover grid.

  Cover + title + authors. The card itself is a button so the whole
  tile is clickable / focusable. The selected state lights the cover
  border with the accent ring (focus-style, not heavy chrome).
-->
<script lang="ts">
  import CoverArt from './CoverArt.svelte';
  import type {Book} from '../types';

  type Props = {
    book: Book;
    selected?: boolean;
    onSelect: (book: Book) => void;
  };

  let {book, selected = false, onSelect}: Props = $props();
</script>

<button
  type="button"
  class="tile"
  class:tile-selected={selected}
  onclick={() => onSelect(book)}
  aria-pressed={selected}
>
  <CoverArt {book} />
  <div class="meta">
    <p class="title">{book.title || 'Untitled'}</p>
    {#if book.authors}
      <p class="authors">{book.authors}</p>
    {/if}
  </div>
</button>

<style>
  .tile {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
    padding: var(--uin-s-2);
    border: 1px solid transparent;
    border-radius: var(--uin-r-md);
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: default;
    transition:
      background-color var(--uin-dur-1) var(--uin-ease-standard),
      border-color var(--uin-dur-1) var(--uin-ease-standard);
    -webkit-app-region: no-drag;
  }
  .tile:hover {
    background: var(--uin-mat-hover);
  }
  .tile-selected {
    background: var(--uin-mat-selected);
    border-color: color-mix(in srgb, var(--uin-accent) 35%, transparent);
  }
  .tile:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
  .meta {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .title {
    margin: 0;
    font-size: 12.5px;
    font-weight: 500;
    line-height: 1.25;
    color: var(--uin-fg);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .authors {
    margin: 0;
    font-size: 11px;
    color: var(--uin-fg-mute);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
