<!--
@component NavItem — sidebar navigation row.

Three slots arranged in a 16px / 1fr / auto grid:
- `dot` (left) — colored marker; pass a `<Lamp />` or any inline element
- default (middle) — the label
- `aside` (right) — count, badge, etc.

`active` highlights the row. Renders a `<button>` so it's keyboard-
focusable and screen-reader-friendly.

CSS lives in `./nav-item.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLButtonAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLButtonAttributes & {
    active?: boolean;
    dot?: Snippet;
    aside?: Snippet;
    class?: string;
    children: Snippet;
  };

  let {
    active = false,
    dot,
    aside,
    class: className,
    children,
    type = 'button',
    ...rest
  }: Props = $props();
</script>

<button
  {type}
  class={cn('uin-nav-item', active && 'uin-nav-item-active', className)}
  aria-current={active ? 'page' : undefined}
  {...rest}
>
  <span class="uin-nav-item-dot">{#if dot}{@render dot()}{/if}</span>
  <span class="uin-nav-item-label">{@render children()}</span>
  {#if aside}<span class="uin-nav-item-aside">{@render aside()}</span>{/if}
</button>
