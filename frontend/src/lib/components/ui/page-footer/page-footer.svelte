<!--
@component PageFooter — footer region for page-level layouts.

Pairs with `PageHeader` and `PageBody`. Bordered top, muted text,
small font — for "© 2026", links, and footnote-style content.

CSS lives in `./page-footer.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLElement> & {
    class?: string;
    children?: Snippet;
  };

  let {class: className, children, ...rest}: Props = $props();
</script>

<footer class={cn('uin-pg-foot', className)} {...rest}>
  {#if children}{@render children()}{/if}
</footer>
