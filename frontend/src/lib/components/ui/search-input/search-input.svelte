<!--
@component SearchInput — pill-shaped search field built on `Input`.

Composes `Input` with a magnifier glyph in the leading slot and either a
× clear button (when value is non-empty) or a ⌘K keyboard hint (when
empty) in the trailing slot.

CSS lives in `./search-input.css`. Depends on `./input.css` and
`./kbd.css` being loaded too.
-->
<script lang="ts">
  import type {HTMLInputAttributes} from 'svelte/elements';
  import Input from '../input/input.svelte';
  import Kbd from '../kbd/kbd.svelte';
  import {cn} from '../../../utils/cn';

  type Props = Omit<HTMLInputAttributes, 'type' | 'class' | 'value' | 'size'> & {
    value?: string;
    showShortcut?: boolean;
    shortcutLabel?: string;
    onclear?: () => void;
    class?: string;
    ref?: HTMLInputElement | null;
  };

  let {
    value = $bindable(''),
    showShortcut = true,
    shortcutLabel = '⌘K',
    onclear,
    class: className,
    placeholder = 'Search',
    ref = $bindable<HTMLInputElement | null>(null),
    ...rest
  }: Props = $props();

  function handleClear() {
    value = '';
    onclear?.();
  }
</script>

<Input
  type="search"
  shape="pill"
  bind:value
  bind:ref
  {placeholder}
  class={cn('uin-search', className)}
  {...rest}
>
  {#snippet leading()}
    <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
      <circle cx="7" cy="7" r="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
      <line x1="11" y1="11" x2="14" y2="14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
    </svg>
  {/snippet}
  {#snippet trailing()}
    {#if value}
      <button
        class="uin-search-clear"
        type="button"
        aria-label="Clear"
        onclick={handleClear}
      >×</button>
    {:else if showShortcut}
      <span class="uin-search-kbd-slot" aria-hidden="true">
        <Kbd>{shortcutLabel}</Kbd>
      </span>
    {/if}
  {/snippet}
</Input>
