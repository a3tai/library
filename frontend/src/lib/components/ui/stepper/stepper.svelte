<!--
@component Stepper — ▲▼ connected button pair. 🍎 macOS extra.

The increment/decrement control you'd see on a macOS NumberInput or in
a "Quantity: 4" picker. Two pressable arrow buttons stacked vertically
in one rounded chassis.

Used standalone for any "tick up / tick down" affordance, or composed
inside `NumberInput` as the trailing adornment.

CSS lives in `./stepper.css`.
-->
<script lang="ts">
  import {cn} from '../../../utils/cn';

  type Props = {
    onIncrement?: () => void;
    onDecrement?: () => void;
    disabled?: boolean;
    incrementDisabled?: boolean;
    decrementDisabled?: boolean;
    size?: 'sm' | 'md';
    ariaIncrementLabel?: string;
    ariaDecrementLabel?: string;
    class?: string;
  };

  let {
    onIncrement,
    onDecrement,
    disabled = false,
    incrementDisabled = false,
    decrementDisabled = false,
    size = 'md',
    ariaIncrementLabel = 'Increase',
    ariaDecrementLabel = 'Decrease',
    class: className,
  }: Props = $props();
</script>

<div class={cn('uin-stepper', `uin-stepper-${size}`, className)}>
  <button
    type="button"
    class="uin-stepper-btn uin-stepper-up"
    aria-label={ariaIncrementLabel}
    disabled={disabled || incrementDisabled}
    onclick={onIncrement}
  >
    <svg viewBox="0 0 8 4" width="8" height="4" aria-hidden="true">
      <polyline points="0,4 4,0 8,4" fill="currentColor" />
    </svg>
  </button>
  <button
    type="button"
    class="uin-stepper-btn uin-stepper-down"
    aria-label={ariaDecrementLabel}
    disabled={disabled || decrementDisabled}
    onclick={onDecrement}
  >
    <svg viewBox="0 0 8 4" width="8" height="4" aria-hidden="true">
      <polyline points="0,0 4,4 8,0" fill="currentColor" />
    </svg>
  </button>
</div>
