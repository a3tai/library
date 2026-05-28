<!--
@component VirtualList — windowed list for large datasets.

Renders only a slice of `items` near the visible window — handles
10k+ rows without DOM bloat. Each item is rendered via the `item`
snippet receiving `(item, index)`.

`itemHeight` is fixed (px). Variable-height virtualisation is more
complex and is a future iteration.

CSS lives in `./virtual-list.css`.
-->
<script lang="ts" generics="T">
  import type {Snippet} from 'svelte';
  import {cn} from '../../../utils/cn';

  type Props = {
    items: T[];
    itemHeight: number;
    height?: string;
    overscan?: number;
    item: Snippet<[T, number]>;
    class?: string;
    ariaLabel?: string;
  };

  let {
    items,
    itemHeight,
    height = '320px',
    overscan = 3,
    item,
    class: className,
    ariaLabel,
  }: Props = $props();

  let scrollTop = $state(0);
  let viewportH = $state(0);
  let containerEl: HTMLDivElement | undefined = $state();

  const totalHeight = $derived(items.length * itemHeight);
  const startIdx = $derived(
    Math.max(0, Math.floor(scrollTop / itemHeight) - overscan)
  );
  const endIdx = $derived(
    Math.min(
      items.length,
      Math.ceil((scrollTop + viewportH) / itemHeight) + overscan
    )
  );
  const visible = $derived(items.slice(startIdx, endIdx));
  const offset = $derived(startIdx * itemHeight);

  function onScroll(e: Event) {
    const t = e.currentTarget as HTMLDivElement;
    scrollTop = t.scrollTop;
  }

  $effect(() => {
    if (containerEl) {
      viewportH = containerEl.clientHeight;
      const ro = new ResizeObserver(() => {
        if (containerEl) viewportH = containerEl.clientHeight;
      });
      ro.observe(containerEl);
      return () => ro.disconnect();
    }
  });
</script>

<div
  bind:this={containerEl}
  class={cn('uin-vlist', className)}
  style="height: {height};"
  role="list"
  aria-label={ariaLabel}
  onscroll={onScroll}
>
  <div class="uin-vlist-spacer" style="height: {totalHeight}px;">
    <div class="uin-vlist-window" style="transform: translateY({offset}px);">
      {#each visible as it, i (startIdx + i)}
        <div class="uin-vlist-item" style="height: {itemHeight}px;" role="listitem">
          {@render item(it, startIdx + i)}
        </div>
      {/each}
    </div>
  </div>
</div>
