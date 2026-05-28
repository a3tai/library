<!--
@component TitleBar — drag region with centered title. 🍎

The macOS-style window chrome that sits at the very top of an app
frame. Drag-to-move (`-webkit-app-region: drag`) is wired in by
default, with `no-drag` islands for the leading (TrafficLights) and
trailing (window-control) slots.

Pass the `<TrafficLights />` into `leading` for the close/min/zoom
cluster. `trailing` is for window-level affordances (search, share).

CSS lives in `./title-bar.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    title?: string;
    subtitle?: string;
    height?: string;
    class?: string;
    leading?: Snippet;
    trailing?: Snippet;
    /** Custom center content. Replaces title/subtitle when provided —
        useful for hosting a search field, breadcrumb, or tabs. */
    children?: Snippet;
  };

  let {
    title,
    subtitle,
    height = '38px',
    class: className,
    leading,
    trailing,
    children,
    ...rest
  }: Props = $props();
</script>

<div
  class={cn('uin-titlebar', className)}
  style="--uin-titlebar-h: {height};"
  {...rest}
>
  {#if leading}
    <div class="uin-titlebar-leading">{@render leading()}</div>
  {/if}
  <div class="uin-titlebar-text">
    {#if children}
      {@render children()}
    {:else}
      {#if title}<p class="uin-titlebar-title">{title}</p>{/if}
      {#if subtitle}<p class="uin-titlebar-sub">{subtitle}</p>{/if}
    {/if}
  </div>
  {#if trailing}
    <div class="uin-titlebar-trailing">{@render trailing()}</div>
  {/if}
</div>
