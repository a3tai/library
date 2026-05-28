<!--
@component Resizable — two-pane container with a drag handle between them.

A flex container with `primary` and `secondary` snippet slots and a
draggable hairline between them. Pointer-based — no library dep,
works with mouse / touch / pen.

`primarySize` is bound as a percentage (0..100). `direction` switches
between horizontal split (default, left/right panes) and vertical
split (top/bottom).

CSS lives in `./resizable.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Direction = 'horizontal' | 'vertical';

  type Props = HTMLAttributes<HTMLDivElement> & {
    direction?: Direction;
    primarySize?: number;
    minPrimary?: number;
    maxPrimary?: number;
    class?: string;
    primary: Snippet;
    secondary: Snippet;
  };

  let {
    direction = 'horizontal',
    primarySize = $bindable(50),
    minPrimary = 10,
    maxPrimary = 90,
    class: className,
    primary,
    secondary,
    ...rest
  }: Props = $props();

  let containerEl: HTMLDivElement | undefined = $state();

  function onPointerDown(e: PointerEvent) {
    e.preventDefault();
    const target = e.currentTarget as HTMLElement;
    target.setPointerCapture(e.pointerId);

    function onMove(ev: PointerEvent) {
      if (!containerEl) return;
      const rect = containerEl.getBoundingClientRect();
      const isH = direction === 'horizontal';
      const offset = isH ? ev.clientX - rect.left : ev.clientY - rect.top;
      const total = isH ? rect.width : rect.height;
      const pct = Math.max(minPrimary, Math.min(maxPrimary, (offset / total) * 100));
      primarySize = pct;
    }

    function onUp(ev: PointerEvent) {
      target.releasePointerCapture(ev.pointerId);
      target.removeEventListener('pointermove', onMove);
      target.removeEventListener('pointerup', onUp);
    }

    target.addEventListener('pointermove', onMove);
    target.addEventListener('pointerup', onUp);
  }
</script>

<div
  bind:this={containerEl}
  class={cn('uin-resize', `uin-resize-${direction}`, className)}
  {...rest}
>
  <div class="uin-resize-pane uin-resize-primary" style="flex-basis: {primarySize}%;">
    {@render primary()}
  </div>
  <div
    class="uin-resize-handle"
    role="separator"
    aria-orientation={direction === 'horizontal' ? 'vertical' : 'horizontal'}
    aria-valuenow={primarySize}
    aria-valuemin={minPrimary}
    aria-valuemax={maxPrimary}
    onpointerdown={onPointerDown}
  >
    <span class="uin-resize-grip" aria-hidden="true"></span>
  </div>
  <div class="uin-resize-pane uin-resize-secondary" style="flex-basis: {100 - primarySize}%;">
    {@render secondary()}
  </div>
</div>
