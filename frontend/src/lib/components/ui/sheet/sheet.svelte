<!--
@component Sheet — slide-in panel from a screen edge. 🍎 macOS-flavored.

Built on bits-ui's `Dialog` primitive (same focus trap, ESC, portal),
but the content slides in from one of four sides and fills that edge.
Use for inspector panels, side navigation, settings pages on mobile.

`side` picks the edge: `right` (default), `left`, `top`, or `bottom`.
`size` picks the cross-axis dimension: `sm` (320), `md` (400, default),
`lg` (520), or `full` (100%).

Requires `bits-ui` (`npm install bits-ui`).

CSS lives in `./sheet.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {Dialog as BD} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  type Side = 'right' | 'left' | 'top' | 'bottom';
  type Size = 'sm' | 'md' | 'lg' | 'full';

  type Props = {
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
    title?: string;
    description?: string;
    side?: Side;
    size?: Size;
    closeLabel?: string;
    class?: string;
    children: Snippet;
    footer?: Snippet;
  };

  let {
    open = $bindable(false),
    onOpenChange,
    title,
    description,
    side = 'right',
    size = 'md',
    closeLabel = 'Close',
    class: className,
    children,
    footer,
  }: Props = $props();
</script>

<BD.Root bind:open onOpenChange={(v: boolean) => onOpenChange?.(v)}>
  <BD.Portal>
    <BD.Overlay class="uin-sheet-overlay" />
    <BD.Content
      class={cn('uin-sheet', `uin-sheet-${side}`, `uin-sheet-${size}`, className)}
    >
      {#if title}
        <header class="uin-sheet-head">
          <BD.Title class="uin-sheet-title">{title}</BD.Title>
          {#if description}
            <BD.Description class="uin-sheet-desc">{description}</BD.Description>
          {/if}
        </header>
      {/if}
      <div class="uin-sheet-body">{@render children()}</div>
      {#if footer}
        <footer class="uin-sheet-foot">{@render footer()}</footer>
      {/if}
      <BD.Close class="uin-sheet-close" aria-label={closeLabel}>
        <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
          <line x1="3.5" y1="3.5" x2="12.5" y2="12.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
          <line x1="12.5" y1="3.5" x2="3.5" y2="12.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
        </svg>
      </BD.Close>
    </BD.Content>
  </BD.Portal>
</BD.Root>
