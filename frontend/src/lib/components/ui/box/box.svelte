<!--
@component Box — generic `<div>` wrapper with token padding + radius shortcuts.

The unstyled foundation. By default Box has no chrome; reach for it
when you need a hook to attach CSS to or a quick padded container
without writing a one-off rule.

`pad` (1–8) maps to `--uin-s-1` … `--uin-s-8`. `radius` (sm/md/lg/xl)
maps to the matching `--uin-r-*` token. Both are optional — omit them
and you get a stylistically empty `<div>`.

CSS lives in `./box.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Pad = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;
  type Radius = 'sm' | 'md' | 'lg' | 'xl';

  type Props = HTMLAttributes<HTMLDivElement> & {
    pad?: Pad;
    radius?: Radius;
    class?: string;
    children?: Snippet;
  };

  let {pad, radius, class: className, children, ...rest}: Props = $props();
</script>

<div
  class={cn(
    'uin-box',
    pad && `uin-box-pad-${pad}`,
    radius && `uin-box-r-${radius}`,
    className
  )}
  {...rest}
>
  {#if children}{@render children()}{/if}
</div>
