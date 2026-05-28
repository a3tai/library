<!--
@component Breadcrumb — hierarchical navigation trail.

Renders a horizontal list of links with separators between them. The
last item is treated as the current page (no link, muted). Pure
markup — no headless dep.

`items: {label, href?}[]`. Items without an `href` are non-interactive
labels. The component decides separator style; pass a `separator`
snippet to override (default is a slim "/" glyph).

CSS lives in `./breadcrumb.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {cn} from '../../../utils/cn';

  type Item = {
    label: string;
    href?: string;
  };

  type Props = {
    items: Item[];
    ariaLabel?: string;
    class?: string;
    separator?: Snippet;
  };

  let {
    items,
    ariaLabel = 'Breadcrumb',
    class: className,
    separator,
  }: Props = $props();
</script>

<nav class={cn('uin-crumb', className)} aria-label={ariaLabel}>
  <ol class="uin-crumb-list">
    {#each items as item, i (i)}
      <li class="uin-crumb-item">
        {#if i > 0}
          <span class="uin-crumb-sep" aria-hidden="true">
            {#if separator}{@render separator()}{:else}/{/if}
          </span>
        {/if}
        {#if item.href && i < items.length - 1}
          <a class="uin-crumb-link" href={item.href}>{item.label}</a>
        {:else}
          <span
            class={cn('uin-crumb-current', i < items.length - 1 && 'uin-crumb-static')}
            aria-current={i === items.length - 1 ? 'page' : undefined}
          >
            {item.label}
          </span>
        {/if}
      </li>
    {/each}
  </ol>
</nav>
