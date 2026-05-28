<!--
@component TrafficLights — macOS window controls (close / min / max). 🍎

The familiar three-circle cluster: red (close), yellow (minimise),
green (zoom). Renders three buttons. Pass `onClose`, `onMinimise`,
`onZoom` to wire them to your window manager (Tauri / Wails / Electron
window APIs).

By default the dot glyphs (× − +) only show on hover, matching macOS.
Pass `alwaysShowGlyphs` to keep them visible.

CSS lives in `./traffic-lights.css`.
-->
<script lang="ts">
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Props = HTMLAttributes<HTMLDivElement> & {
    onClose?: () => void;
    onMinimise?: () => void;
    onZoom?: () => void;
    disabled?: boolean;
    alwaysShowGlyphs?: boolean;
    class?: string;
  };

  let {
    onClose,
    onMinimise,
    onZoom,
    disabled = false,
    alwaysShowGlyphs = false,
    class: className,
    ...rest
  }: Props = $props();
</script>

<div
  class={cn('uin-tl', alwaysShowGlyphs && 'uin-tl-show', disabled && 'uin-tl-disabled', className)}
  {...rest}
>
  <button class="uin-tl-btn uin-tl-close" type="button" aria-label="Close" {disabled} onclick={onClose}>
    <span aria-hidden="true">×</span>
  </button>
  <button class="uin-tl-btn uin-tl-min" type="button" aria-label="Minimise" {disabled} onclick={onMinimise}>
    <span aria-hidden="true">−</span>
  </button>
  <button class="uin-tl-btn uin-tl-zoom" type="button" aria-label="Zoom" {disabled} onclick={onZoom}>
    <span aria-hidden="true">+</span>
  </button>
</div>
