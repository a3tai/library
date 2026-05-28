<!--
@component ScrollArea — bordered, overflow-aware container with styled scrollbars.

Pure-CSS approach: native browser scrollbars styled via
`-webkit-scrollbar` (WebKit / Chromium / Edge) and the
`scrollbar-color` / `scrollbar-width` properties (Firefox 64+).

We deliberately avoid the Radix-style JS-rendered scrollbars to keep
the bundle headless-dep-free; the trade-off is that the chrome on
WebKit and Firefox is similar but not pixel-identical. Both feel
native, which is the goal.

`maxHeight` caps the content area; `axis` picks vertical (default),
horizontal, or both.

CSS lives in `./scroll-area.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Axis = 'vertical' | 'horizontal' | 'both';

  type Props = HTMLAttributes<HTMLDivElement> & {
    maxHeight?: string;
    axis?: Axis;
    class?: string;
    children?: Snippet;
  };

  let {
    maxHeight,
    axis = 'vertical',
    class: className,
    children,
    ...rest
  }: Props = $props();

  const styleStr = $derived(maxHeight ? `max-height: ${maxHeight};` : undefined);
</script>

<div
  class={cn('uin-scroll', `uin-scroll-${axis}`, className)}
  style={styleStr}
  {...rest}
>
  {#if children}{@render children()}{/if}
</div>
