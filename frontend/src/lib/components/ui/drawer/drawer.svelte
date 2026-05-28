<!--
@component Drawer — bottom-anchored sheet with a drag handle.

Mobile-first slide-up panel with a small grab handle at the top, the
shape iOS apps use for share sheets and settings sub-pages. Built on
bits-ui's `Dialog` primitive — same focus trap, ESC, portal.

For non-bottom edges or non-handle chrome, reach for `Sheet` instead.
This component is opinionated for the bottom-up modal shape.

`size` picks the height: `sm` (40vh), `md` (60vh, default), `lg`
(80vh), `full` (100vh).

Requires `bits-ui` (`npm install bits-ui`).

CSS lives in `./drawer.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {Dialog as BD} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  type Size = 'sm' | 'md' | 'lg' | 'full';

  type Props = {
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
    title?: string;
    description?: string;
    size?: Size;
    class?: string;
    children: Snippet;
    footer?: Snippet;
  };

  let {
    open = $bindable(false),
    onOpenChange,
    title,
    description,
    size = 'md',
    class: className,
    children,
    footer,
  }: Props = $props();
</script>

<BD.Root bind:open onOpenChange={(v: boolean) => onOpenChange?.(v)}>
  <BD.Portal>
    <BD.Overlay class="uin-drawer-overlay" />
    <BD.Content class={cn('uin-drawer', `uin-drawer-${size}`, className)}>
      <div class="uin-drawer-handle" aria-hidden="true"></div>
      {#if title}
        <header class="uin-drawer-head">
          <BD.Title class="uin-drawer-title">{title}</BD.Title>
          {#if description}
            <BD.Description class="uin-drawer-desc">{description}</BD.Description>
          {/if}
        </header>
      {/if}
      <div class="uin-drawer-body">{@render children()}</div>
      {#if footer}
        <footer class="uin-drawer-foot">{@render footer()}</footer>
      {/if}
    </BD.Content>
  </BD.Portal>
</BD.Root>
