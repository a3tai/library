<!--
@component PageBody — main content region for page-level layouts.

A token-padded `<main>` block to pair with `PageHeader` and
`PageFooter`. Set `padded={false}` if your inner content already
manages its own padding (tables, full-bleed media).

CSS lives in `./page-body.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLElement> & {
    padded?: boolean;
    class?: string;
    children?: Snippet;
  };

  let {padded = true, class: className, children, ...rest}: Props = $props();
</script>

<main class={cn('uin-pg-body', padded && 'uin-pg-body-padded', className)} {...rest}>
  {#if children}{@render children()}{/if}
</main>
