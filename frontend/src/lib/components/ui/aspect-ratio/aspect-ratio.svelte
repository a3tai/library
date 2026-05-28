<!--
@component AspectRatio — keeps a child at a fixed width:height ratio.

A thin wrapper around the CSS `aspect-ratio` property. Wrap an image,
embed, or any block child to lock its proportions regardless of the
parent's width.

`ratio` accepts any number; common picks are `16/9`, `4/3`, `1`, `3/2`.
Default is `16/9`.

CSS lives in `./aspect-ratio.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    ratio?: number;
    class?: string;
    children?: Snippet;
  };

  let {ratio = 16 / 9, class: className, children, ...rest}: Props = $props();
</script>

<div
  class={cn('uin-aspect', className)}
  style="aspect-ratio: {ratio};"
  {...rest}
>
  {#if children}{@render children()}{/if}
</div>
