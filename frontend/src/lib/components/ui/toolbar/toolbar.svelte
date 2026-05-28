<!--
@component Toolbar — horizontal action strip. 🍎

A translucent bar with action groups separated by Separator lines.
Used at the top of windows and panels for primary actions.
Accepts any children — typically Button, IconButton, SegmentedControl,
SearchInput.

`density` picks the height (`compact` = 32, `comfortable` = 40, `roomy`
= 48). The toolbar auto-marks itself as `no-drag` for Tauri/Wails so
its buttons stay clickable inside a drag region.

CSS lives in `./toolbar.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Density = 'compact' | 'comfortable' | 'roomy';

  type Props = HTMLAttributes<HTMLDivElement> & {
    density?: Density;
    bordered?: boolean;
    class?: string;
    children?: Snippet;
  };

  let {
    density = 'comfortable',
    bordered = true,
    class: className,
    children,
    ...rest
  }: Props = $props();
</script>

<div
  class={cn('uin-toolbar', `uin-toolbar-${density}`, bordered && 'uin-toolbar-bordered', className)}
  role="toolbar"
  {...rest}
>
  {#if children}{@render children()}{/if}
</div>
