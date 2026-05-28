<!--
@component Collapsible — toggle visibility of a content region.

Built on bits-ui's Collapsible primitive (handles ARIA + state).
Trigger is rendered as a button-shaped row with a chevron that rotates
when open. The default snippet renders the body that shows / hides.

Bind `open` to control state externally; otherwise it manages its own.

Requires `bits-ui` (`npm install bits-ui`).

CSS lives in `./collapsible.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {Collapsible as BC} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  type Props = {
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
    title: string;
    disabled?: boolean;
    class?: string;
    children: Snippet;
  };

  let {
    open = $bindable(false),
    onOpenChange,
    title,
    disabled = false,
    class: className,
    children,
  }: Props = $props();
</script>

<BC.Root bind:open onOpenChange={(v: boolean) => onOpenChange?.(v)} {disabled}>
  <BC.Trigger class={cn('uin-collapsible-trigger', className)}>
    <span class="uin-collapsible-label">{title}</span>
    <svg class="uin-collapsible-chev" viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
      <polyline points="4,3 8,6 4,9" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </BC.Trigger>
  <BC.Content class="uin-collapsible-content">
    {@render children()}
  </BC.Content>
</BC.Root>
