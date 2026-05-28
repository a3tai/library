<!--
@component Center — flex container that centers its children on both axes.

The smallest possible "make this thing live in the middle" wrapper.
Equivalent to `display: flex; align-items: center; justify-content:
center` so you don't have to remember the recipe each time.

CSS lives in `./center.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    class?: string;
    children?: Snippet;
  };

  let {class: className, children, ...rest}: Props = $props();
</script>

<div class={cn('uin-center', className)} {...rest}>
  {#if children}{@render children()}{/if}
</div>
