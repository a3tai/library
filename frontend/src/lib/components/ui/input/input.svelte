<!--
@component Input — base text input primitive.

Other inputs (`SearchInput`, `NumberInput`, etc.) compose this one.
- `bind:value` support
- Optional `leading` / `trailing` snippets for inline adornments
- Three shapes: `pill`, `rounded` (default), `square`
- Two sizes: `sm` and `md`

CSS lives in `./input.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLInputAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Shape = 'pill' | 'rounded' | 'square';
  type Size = 'sm' | 'md';

  type Props = Omit<HTMLInputAttributes, 'class' | 'value' | 'size'> & {
    value?: string;
    shape?: Shape;
    size?: Size;
    leading?: Snippet;
    trailing?: Snippet;
    class?: string;
    inputClass?: string;
    /** Exposes the inner <input> element to the consumer. */
    ref?: HTMLInputElement | null;
  };

  let {
    value = $bindable(''),
    shape = 'rounded',
    size = 'md',
    leading,
    trailing,
    class: className,
    inputClass,
    type = 'text',
    ref = $bindable<HTMLInputElement | null>(null),
    ...rest
  }: Props = $props();
</script>

<div
  class={cn(
    'uin-input',
    `uin-input-${shape}`,
    `uin-input-${size}`,
    leading && 'uin-input-has-leading',
    trailing && 'uin-input-has-trailing',
    className,
  )}
>
  {#if leading}
    <span class="uin-input-leading" aria-hidden="true">{@render leading()}</span>
  {/if}
  <input bind:this={ref} class={cn('uin-input-el', inputClass)} bind:value {type} {...rest} />
  {#if trailing}
    <span class="uin-input-trailing">{@render trailing()}</span>
  {/if}
</div>
