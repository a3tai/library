<!--
  BookListRow — list-mode entry. Compact horizontal cover + meta + format
  badge for users who prefer dense rows over the cover grid.
-->
<script lang="ts">
  import Badge from '../lib/components/ui/badge/badge.svelte';
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
  class="row"
  class:row-selected={selected}
  onclick={() => onSelect(book)}
  aria-pressed={selected}
>
  <span class="row-thumb"><CoverArt {book} width="36px" /></span>
  <span class="row-text">
    <span class="row-title">{book.title || 'Untitled'}</span>
    {#if book.authors}
      <span class="row-authors">{book.authors}</span>
    {/if}
  </span>
  {#if book.format}
    <span class="row-format">
      <Badge variant="outline" size="sm">{book.format.toUpperCase()}</Badge>
    </span>
  {/if}
  {#if book.passageCount > 0}
    <span class="row-passages">{book.passageCount.toLocaleString()}</span>
  {/if}
</button>

<style>
  .row {
    -webkit-app-region: no-drag;
    width: 100%;
    display: grid;
    grid-template-columns: 36px 1fr auto auto;
    align-items: center;
    gap: var(--uin-s-3);
    padding: var(--uin-s-2) var(--uin-s-3);
    border: 0;
    background: transparent;
    color: var(--uin-fg);
    text-align: left;
    border-radius: var(--uin-r-sm);
    cursor: default;
    transition: background-color var(--uin-dur-1) var(--uin-ease-standard);
  }
  .row:hover {
    background: var(--uin-mat-hover);
  }
  .row-selected {
    background: var(--uin-mat-selected);
  }
  .row:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
  .row-text {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .row-title {
    font-size: 12.5px;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-authors {
    font-size: 11px;
    color: var(--uin-fg-mute);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-passages {
    font-family: var(--uin-font-mono);
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-variant-numeric: tabular-nums;
  }
</style>
