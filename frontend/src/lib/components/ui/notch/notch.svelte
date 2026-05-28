<!--
@component Notch — frosted-material section header. 🍎 macOS extra.

The translucent header bar you'd see at the top of a sidebar or panel
in macOS apps: backdrop-blurred fill, hairline bottom border, optional
leading icon on the left and trailing actions on the right.

Notch doesn't position itself; wrap it in `position: sticky` (or place
it inside a fixed container) at the call site if you want stick-on-
scroll behavior. The component itself is just the visual chrome.

CSS lives in `./notch.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    title?: string;
    subtitle?: string;
    leading?: Snippet;
    trailing?: Snippet;
    class?: string;
    children?: Snippet;
  };

  let {
    title,
    subtitle,
    leading,
    trailing,
    class: className,
    children,
    ...rest
  }: Props = $props();
</script>

<div class={cn('uin-notch', className)} {...rest}>
  {#if leading}
    <div class="uin-notch-leading">{@render leading()}</div>
  {/if}
  <div class="uin-notch-text">
    {#if title}<p class="uin-notch-title">{title}</p>{/if}
    {#if subtitle}<p class="uin-notch-sub">{subtitle}</p>{/if}
    {#if children}{@render children()}{/if}
  </div>
  {#if trailing}
    <div class="uin-notch-trailing">{@render trailing()}</div>
  {/if}
</div>
