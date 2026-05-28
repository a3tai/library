<!--
@component StatusRow — compact card summarising an in-progress operation.

Three slot regions:
- `leading` — typically a `<Spinner />` or `<Lamp />`
- `title` — the operation name (e.g. "Importing")
- `aside` — optional right-side accent (chevron, badge, etc.)
- default snippet — the body line beneath the head (e.g. progress text)
- `meta` — optional secondary line (e.g. "Hydrating 47 in queue")
- `bar` — optional `<ProgressBar />` slotted between body and meta

Renders as a `<button>` when `onClick` is provided (becomes the
"open the importer dialog" entry-point), otherwise a `<div>`.

CSS lives in `./status-row.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {cn} from '../../../utils/cn';

  type Props = {
    title: string;
    leading?: Snippet;
    aside?: Snippet;
    bar?: Snippet;
    meta?: Snippet;
    onClick?: () => void;
    ariaLabel?: string;
    class?: string;
    children?: Snippet;
  };

  let {
    title,
    leading,
    aside,
    bar,
    meta,
    onClick,
    ariaLabel,
    class: className,
    children,
  }: Props = $props();

  const interactive = $derived(onClick !== undefined);
</script>

{#if interactive}
  <button
    type="button"
    class={cn('uin-status-row', className)}
    onclick={onClick}
    aria-label={ariaLabel}
  >
    <div class="uin-status-row-head">
      {#if leading}<span>{@render leading()}</span>{:else}<span></span>{/if}
      <strong>{title}</strong>
      {#if aside}<span class="uin-status-row-chev">{@render aside()}</span>{:else}<span class="uin-status-row-chev">›</span>{/if}
    </div>
    {#if children}<div class="uin-status-row-line">{@render children()}</div>{/if}
    {#if bar}{@render bar()}{/if}
    {#if meta}<div class="uin-status-row-meta">{@render meta()}</div>{/if}
  </button>
{:else}
  <div class={cn('uin-status-row', className)} aria-label={ariaLabel}>
    <div class="uin-status-row-head">
      {#if leading}<span>{@render leading()}</span>{:else}<span></span>{/if}
      <strong>{title}</strong>
      {#if aside}<span class="uin-status-row-chev">{@render aside()}</span>{/if}
    </div>
    {#if children}<div class="uin-status-row-line">{@render children()}</div>{/if}
    {#if bar}{@render bar()}{/if}
    {#if meta}<div class="uin-status-row-meta">{@render meta()}</div>{/if}
  </div>
{/if}
