<!--
@component SplitView — multi-pane container with macOS-flavored chrome. 🍎

Inspired by HIG SplitView. Renders 2 or 3 panes with thin
hairline dividers between them; each non-first pane is preceded by a
drag handle that resizes the previous one.

Pass `panes` as `[{snippet, defaultSize?, min?, max?}]` (sizes in
percent). `direction` controls layout.

For just two panes, `Resizable` is simpler. SplitView is the
multi-pane variant.

CSS lives in `./split-view.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Pane = {
    content: Snippet;
    defaultSize?: number;
    min?: number;
    max?: number;
  };

  type Direction = 'horizontal' | 'vertical';

  type Props = HTMLAttributes<HTMLDivElement> & {
    panes: Pane[];
    direction?: Direction;
    class?: string;
  };

  let {panes, direction = 'horizontal', class: className, ...rest}: Props = $props();

  // Compute initial sizes — equal-share if defaults aren't given
  let sizes = $state<number[]>(
    (() => {
      const total = panes.reduce((s, p) => s + (p.defaultSize ?? 0), 0);
      if (total > 0) {
        const remaining = 100 - total;
        const unsized = panes.filter((p) => p.defaultSize === undefined).length;
        const each = unsized > 0 ? remaining / unsized : 0;
        return panes.map((p) => p.defaultSize ?? each);
      }
      return panes.map(() => 100 / panes.length);
    })()
  );

  let containerEl: HTMLDivElement | undefined = $state();

  function onPointerDown(idx: number) {
    return (e: PointerEvent) => {
      e.preventDefault();
      const target = e.currentTarget as HTMLElement;
      target.setPointerCapture(e.pointerId);

      function onMove(ev: PointerEvent) {
        if (!containerEl) return;
        const rect = containerEl.getBoundingClientRect();
        const isH = direction === 'horizontal';
        const offset = isH ? ev.clientX - rect.left : ev.clientY - rect.top;
        const total = isH ? rect.width : rect.height;
        const pctTotal = (offset / total) * 100;

        // Sum of sizes before idx
        const before = sizes.slice(0, idx).reduce((s, n) => s + n, 0);
        const next = panes[idx];
        const min = next?.min ?? 5;
        const max = next?.max ?? 95;

        const newSize = Math.max(min, Math.min(max, pctTotal - before));
        const oldSize = sizes[idx];
        const delta = newSize - oldSize;
        sizes[idx] = newSize;
        if (sizes[idx + 1] !== undefined) {
          sizes[idx + 1] = Math.max(5, sizes[idx + 1] - delta);
        }
      }

      function onUp(ev: PointerEvent) {
        target.releasePointerCapture(ev.pointerId);
        target.removeEventListener('pointermove', onMove);
        target.removeEventListener('pointerup', onUp);
      }

      target.addEventListener('pointermove', onMove);
      target.addEventListener('pointerup', onUp);
    };
  }
</script>

<div
  bind:this={containerEl}
  class={cn('uin-split', `uin-split-${direction}`, className)}
  {...rest}
>
  {#each panes as pane, i (i)}
    {#if i > 0}
      <div
        class="uin-split-handle"
        role="separator"
        aria-orientation={direction === 'horizontal' ? 'vertical' : 'horizontal'}
        onpointerdown={onPointerDown(i)}
      ></div>
    {/if}
    <div class="uin-split-pane" style="flex-basis: {sizes[i]}%;">
      {@render pane.content()}
    </div>
  {/each}
</div>
