<!--
@component Sidebar — collapsible side panel.

A semantic `<aside>` with header / body / footer slots. The body
typically holds `<NavItem />` rows. `collapsed` shrinks the panel to
an icon rail (48px); `side` flips it to the right edge of the layout.

Wire `collapsed` from your shell state — Sidebar is presentational
and doesn't manage its own collapse logic.

CSS lives in `./sidebar.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLElement> & {
    collapsed?: boolean;
    side?: 'left' | 'right';
    width?: string;
    collapsedWidth?: string;
    class?: string;
    header?: Snippet;
    footer?: Snippet;
    children?: Snippet;
  };

  let {
    collapsed = false,
    side = 'left',
    width = '240px',
    collapsedWidth = '48px',
    class: className,
    header,
    footer,
    children,
    ...rest
  }: Props = $props();
</script>

<aside
  class={cn('uin-sidebar', `uin-sidebar-${side}`, collapsed && 'uin-sidebar-collapsed', className)}
  style="--uin-sidebar-w: {width}; --uin-sidebar-cw: {collapsedWidth};"
  data-collapsed={collapsed}
  {...rest}
>
  {#if header}
    <header class="uin-sidebar-head">{@render header()}</header>
  {/if}
  <div class="uin-sidebar-body">
    {#if children}{@render children()}{/if}
  </div>
  {#if footer}
    <footer class="uin-sidebar-foot">{@render footer()}</footer>
  {/if}
</aside>
