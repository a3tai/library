<!--
@component NumberInput — numeric text field with ▲▼ stepper.

Composes `Input` (type=number) with `Stepper` in the trailing slot for
incrementing / decrementing the value. Honors `min`, `max`, `step`.
Native arrow-key support via the underlying `<input type="number">`.

Bind `value` as a number; the component handles the string ↔ number
conversion.

CSS lives in `./number-input.css`.
-->
<script lang="ts">
  import type {HTMLInputAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';
  import Input from '../input/input.svelte';
  import Stepper from '../stepper/stepper.svelte';

  type Props = Omit<HTMLInputAttributes, 'type' | 'class' | 'value' | 'size' | 'min' | 'max' | 'step'> & {
    value?: number;
    min?: number;
    max?: number;
    step?: number;
    size?: 'sm' | 'md';
    class?: string;
  };

  let {
    value = $bindable(0),
    min,
    max,
    step = 1,
    size = 'md',
    class: className,
    disabled,
    ...rest
  }: Props = $props();

  function clamp(n: number): number {
    if (typeof min === 'number' && n < min) return min;
    if (typeof max === 'number' && n > max) return max;
    return n;
  }

  function increment() {
    value = clamp(value + step);
  }
  function decrement() {
    value = clamp(value - step);
  }

  const incrementDisabled = $derived(typeof max === 'number' && value >= max);
  const decrementDisabled = $derived(typeof min === 'number' && value <= min);

  // Bridge string<->number through a derived/effect pair on a string-typed input
  let strValue = $state('');
  $effect(() => {
    strValue = String(value ?? '');
  });

  function handleInput(e: Event) {
    const t = e.target as HTMLInputElement;
    const n = Number(t.value);
    if (!Number.isNaN(n) && t.value.trim() !== '') {
      value = clamp(n);
    } else if (t.value.trim() === '') {
      value = 0;
    }
  }
</script>

<Input
  type="number"
  shape="rounded"
  size={size}
  class={cn('uin-number-input', className)}
  inputClass="uin-number-input-el"
  value={strValue}
  oninput={handleInput}
  {min}
  {max}
  {step}
  {disabled}
  {...rest}
>
  {#snippet trailing()}
    <Stepper
      size={size}
      onIncrement={increment}
      onDecrement={decrement}
      {incrementDisabled}
      {decrementDisabled}
      disabled={disabled || false}
    />
  {/snippet}
</Input>
