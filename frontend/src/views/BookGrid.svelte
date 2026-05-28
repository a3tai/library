<!--
  BookGrid — auto-fit cover grid. Tiles size themselves to a target
  cover width via `grid-template-columns: repeat(auto-fill, minmax(...))`.

  Switches to a single-column dense list when `mode === 'list'`.
-->
<script lang="ts">
  import BookCoverCard from './BookCoverCard.svelte';
  import BookListRow from './BookListRow.svelte';
  import type {Book} from '../types';

  type Props = {
    books: Book[];
    selectedId?: string;
    mode?: 'grid' | 'list';
    onSelect: (book: Book) => void;
    /** Target tile width when in grid mode (px). */
    tileSize?: number;
  };

  let {books, selectedId, mode = 'grid', onSelect, tileSize = 156}: Props = $props();
</script>

<div class="wrap" class:wrap-list={mode === 'list'} style="--tile-size: {tileSize}px;">
  {#if mode === 'grid'}
    <div class="grid">
      {#each books as book (book.id)}
        <BookCoverCard {book} selected={book.id === selectedId} {onSelect} />
      {/each}
    </div>
  {:else}
    <ul class="rows" role="list">
      {#each books as book (book.id)}
        <li><BookListRow {book} selected={book.id === selectedId} {onSelect} /></li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .wrap {
    padding: var(--uin-s-4) 0;
  }
  .wrap-list {
    padding: var(--uin-s-2) 0;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(var(--tile-size), 1fr));
    gap: var(--uin-s-3);
    container-type: inline-size;
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
</style>
