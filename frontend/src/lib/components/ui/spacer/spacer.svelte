<!--
@component Spacer — flex helper that fills remaining space.

Drop one between two children of a flex container (HStack, Stack, or
any `display: flex`) to push them to opposite ends. Equivalent to
giving one of the siblings `margin-left: auto`, but explicit and
direction-agnostic.

CSS lives in `./spacer.css`.
-->
<script lang="ts">
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    class?: string;
  };

  let {class: className, ...rest}: Props = $props();
</script>

<div class={cn('uin-spacer', className)} aria-hidden="true" {...rest}></div>
