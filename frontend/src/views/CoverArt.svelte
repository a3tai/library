<!--
  CoverArt — rectangular book cover with a graceful fallback.

  Books often arrive with no coverUrl (or one that 404s); this draws a
  generated gradient + initial pair so the grid never has empty slots.
-->
<script lang="ts">
  import type {Book} from '../types';

  type Props = {
    book: Book;
    /** CSS aspect ratio. Default is the typical 2:3 trade book. */
    ratio?: string;
    /** CSS width string applied to the wrapper. */
    width?: string;
    rounded?: boolean;
  };

  let {book, ratio = '2 / 3', width = '100%', rounded = true}: Props = $props();

  let imgFailed = $state(false);
  $effect(() => {
    void book.id;
    imgFailed = false;
  });

  const showImage = $derived(!!book.coverUrl && !imgFailed);
  const initials = $derived.by(() => {
    const t = (book.title || '').trim();
    if (!t) return '?';
    const parts = t.split(/\s+/).filter(Boolean);
    const a = parts[0]?.[0] ?? '';
    const b = parts.length > 1 ? parts[parts.length - 1][0] : '';
    return (a + b).toUpperCase();
  });

  // Stable color from the id so each book has a recognisable hue.
  const hue = $derived.by(() => {
    let h = 0;
    for (let i = 0; i < book.id.length; i++) h = (h * 31 + book.id.charCodeAt(i)) >>> 0;
    return h % 360;
  });
</script>

<div
  class="cover"
  class:rounded
  style="--cover-w: {width}; --cover-ratio: {ratio}; --cover-hue: {hue};"
>
  {#if showImage}
    <img src={book.coverUrl} alt={book.title} loading="lazy" onerror={() => (imgFailed = true)} />
  {:else}
    <div class="fallback" aria-hidden="true">
      <span class="fallback-initials">{initials}</span>
      {#if book.format}
        <span class="fallback-format">{book.format.toUpperCase()}</span>
      {/if}
    </div>
  {/if}
</div>

<style>
  .cover {
    width: var(--cover-w);
    aspect-ratio: var(--cover-ratio);
    background: var(--uin-mat-row);
    overflow: hidden;
    box-shadow:
      0 1px 0 color-mix(in srgb, var(--uin-fg) 6%, transparent),
      0 6px 16px color-mix(in srgb, var(--uin-fg) 8%, transparent);
    position: relative;
  }
  .cover.rounded {
    border-radius: var(--uin-r-md);
  }
  .cover img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .fallback {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--uin-s-2);
    background: linear-gradient(
      155deg,
      hsl(calc(var(--cover-hue) * 1deg) 28% 38%) 0%,
      hsl(calc((var(--cover-hue) + 35) * 1deg) 32% 22%) 100%
    );
    color: rgba(255, 255, 255, 0.92);
    font-family: var(--uin-font-display);
  }
  .fallback-initials {
    font-size: clamp(28px, 22cqw, 56px);
    font-weight: 500;
    letter-spacing: 0.02em;
  }
  .fallback-format {
    font-family: var(--uin-font-mono);
    font-size: 9.5px;
    letter-spacing: 0.18em;
    color: rgba(255, 255, 255, 0.7);
  }
</style>
