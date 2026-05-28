<!--
@component ProgressRing — circular determinate progress.

A ring that fills clockwise from 0 → 100%. Use for compact KPIs,
upload-completion icons, sync-status indicators where a horizontal
bar would be too noisy.

Sizes: `sm` (16), `md` (24, default), `lg` (40), `xl` (64).
Pass `value` (0..max). Optional `label` snippet renders centered
inside the ring (typically a percentage or an icon).

CSS lives in `./progress-ring.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {cn} from '../../../utils/cn';

  type Size = 'sm' | 'md' | 'lg' | 'xl';

  type Props = {
    value?: number;
    max?: number;
    size?: Size;
    thickness?: number;
    class?: string;
    label?: Snippet;
    ariaLabel?: string;
  };

  let {
    value = 0,
    max = 1,
    size = 'md',
    thickness,
    class: className,
    label,
    ariaLabel,
  }: Props = $props();

  const dim: Record<Size, number> = {sm: 16, md: 24, lg: 40, xl: 64};
  const defaultThick: Record<Size, number> = {sm: 2, md: 2.5, lg: 3.5, xl: 5};

  const px = $derived(dim[size]);
  const tk = $derived(thickness ?? defaultThick[size]);
  const r = $derived((px - tk) / 2);
  const c = $derived(2 * Math.PI * r);
  const pct = $derived(Math.max(0, Math.min(1, value / Math.max(0.0001, max))));
  const offset = $derived(c * (1 - pct));
</script>

<span
  class={cn('uin-ring', `uin-ring-${size}`, className)}
  role="progressbar"
  aria-valuemin={0}
  aria-valuemax={max}
  aria-valuenow={value}
  aria-label={ariaLabel}
>
  <svg viewBox="0 0 {px} {px}" width={px} height={px}>
    <circle
      class="uin-ring-track"
      cx={px / 2}
      cy={px / 2}
      r={r}
      fill="none"
      stroke-width={tk}
    />
    <circle
      class="uin-ring-fill"
      cx={px / 2}
      cy={px / 2}
      r={r}
      fill="none"
      stroke-width={tk}
      stroke-dasharray={c}
      stroke-dashoffset={offset}
      stroke-linecap="round"
      transform="rotate(-90 {px / 2} {px / 2})"
    />
  </svg>
  {#if label}
    <span class="uin-ring-label">{@render label()}</span>
  {/if}
</span>
