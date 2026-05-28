<!--
@component InfiniteScroll — load-more sentinel via IntersectionObserver.

Wrap your scrollable list in this component. It mounts a sentinel at
the bottom; when the sentinel scrolls into view, `onLoadMore` fires.
The `loading` flag (bound by the consumer) suppresses repeat firings
while a request is in flight, and `hasMore` stops firing once the
data is exhausted.

The component is presentational below the sentinel — pass any markup
as children.

CSS lives in `./infinite-scroll.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {cn} from '../../../utils/cn';

  type Props = {
    onLoadMore: () => void | Promise<void>;
    loading?: boolean;
    hasMore?: boolean;
    rootMargin?: string;
    class?: string;
    children: Snippet;
    loader?: Snippet;
  };

  let {
    onLoadMore,
    loading = false,
    hasMore = true,
    rootMargin = '0px 0px 200px 0px',
    class: className,
    children,
    loader,
  }: Props = $props();

  let sentinel: HTMLDivElement | undefined = $state();

  $effect(() => {
    if (!sentinel) return;
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && hasMore && !loading) {
            onLoadMore();
          }
        }
      },
      {rootMargin}
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  });
</script>

<div class={cn('uin-infscroll', className)}>
  {@render children()}
  <div bind:this={sentinel} class="uin-infscroll-sentinel" aria-hidden="true">
    {#if loading}
      {#if loader}{@render loader()}{:else}<span class="uin-infscroll-loading">Loading…</span>{/if}
    {:else if !hasMore}
      <span class="uin-infscroll-end">End of list</span>
    {/if}
  </div>
</div>
