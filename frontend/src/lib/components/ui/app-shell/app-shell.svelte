<!--
@component AppShell — full-window header / sidebar / main / footer scaffold.

The application chassis: a CSS grid that pins a top header, a side
sidebar, a footer, and lets the main region scroll. Slots are all
optional — pass only the regions you need.

Composition: pass a `<Toolbar>` into `header`, a `<Sidebar>` into
`sidebar`, a `<StatusBar>` (or anything) into `footer`, and your
routed content as the default snippet.

`sidebarSide` flips the layout (default left). `sidebarWidth` and
`headerHeight` are CSS lengths; tweak them at the call site or via
the `--uin-shell-*` variables.

CSS lives in `./app-shell.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    sidebarSide?: 'left' | 'right';
    sidebarWidth?: string;
    headerHeight?: string;
    footerHeight?: string;
    class?: string;
    header?: Snippet;
    sidebar?: Snippet;
    footer?: Snippet;
    children?: Snippet;
  };

  let {
    sidebarSide = 'left',
    sidebarWidth = '240px',
    headerHeight = '40px',
    footerHeight = 'auto',
    class: className,
    header,
    sidebar,
    footer,
    children,
    ...rest
  }: Props = $props();
</script>

<div
  class={cn(
    'uin-shell',
    `uin-shell-${sidebarSide}`,
    !!header && 'uin-shell-has-header',
    !!sidebar && 'uin-shell-has-sidebar',
    !!footer && 'uin-shell-has-footer',
    className
  )}
  style="--uin-shell-sb: {sidebarWidth}; --uin-shell-h: {headerHeight}; --uin-shell-f: {footerHeight};"
  {...rest}
>
  {#if header}
    <div class="uin-shell-head">{@render header()}</div>
  {/if}
  {#if sidebar}
    <div class="uin-shell-side">{@render sidebar()}</div>
  {/if}
  <main class="uin-shell-main">
    {#if children}{@render children()}{/if}
  </main>
  {#if footer}
    <div class="uin-shell-foot">{@render footer()}</div>
  {/if}
</div>
