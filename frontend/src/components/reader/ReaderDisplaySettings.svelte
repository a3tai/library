<script lang="ts">
  import type {ReaderFamily, ReaderTheme} from './types';

  type Props = {
    theme: ReaderTheme;
    family: ReaderFamily;
    fontPx: number;
    widthCh: number;
    onClose: () => void;
    onFontDelta: (delta: number) => void;
    onFontReset: () => void;
    onFamilyChange: (family: ReaderFamily) => void;
    onThemeChange: (theme: ReaderTheme) => void;
    onWidthChange: (width: number) => void;
  };

  let {
    theme,
    family,
    fontPx,
    widthCh,
    onClose,
    onFontDelta,
    onFontReset,
    onFamilyChange,
    onThemeChange,
    onWidthChange,
  }: Props = $props();
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="popover-anchor" onclick={onClose} role="presentation"></div>
<div class="popover settings-popover" role="dialog" aria-label="Display settings">
  <div class="row">
    <span class="lbl">Font size</span>
    <div class="seg">
      <button type="button" onclick={() => onFontDelta(-1)} aria-label="Smaller">A-</button>
      <button type="button" onclick={onFontReset} aria-label="Reset">{fontPx}</button>
      <button type="button" onclick={() => onFontDelta(1)} aria-label="Larger">A+</button>
    </div>
  </div>
  <div class="row">
    <span class="lbl">Family</span>
    <div class="seg">
      <button type="button" class:on={family === 'serif'} onclick={() => onFamilyChange('serif')}>Serif</button>
      <button type="button" class:on={family === 'sans'} onclick={() => onFamilyChange('sans')}>Sans</button>
    </div>
  </div>
  <div class="row">
    <span class="lbl">Theme</span>
    <div class="seg theme">
      <button type="button" class:on={theme === 'auto'} onclick={() => onThemeChange('auto')} aria-label="Auto" title="Auto (matches system)">A</button>
      <button type="button" class:on={theme === 'light'} onclick={() => onThemeChange('light')} aria-label="Light"><span class="sw light"></span></button>
      <button type="button" class:on={theme === 'sepia'} onclick={() => onThemeChange('sepia')} aria-label="Sepia"><span class="sw sepia"></span></button>
      <button type="button" class:on={theme === 'dark'} onclick={() => onThemeChange('dark')} aria-label="Dark"><span class="sw dark"></span></button>
    </div>
  </div>
  <div class="row">
    <span class="lbl">Width</span>
    <input
      type="range"
      min="48"
      max="92"
      step="2"
      value={widthCh}
      oninput={(event) => onWidthChange(Number((event.target as HTMLInputElement).value))}
      aria-label="Reading width"
    />
  </div>
</div>

<style>
  .popover-anchor {
    position: fixed;
    inset: 0;
    z-index: 1;
  }
  .popover {
    position: fixed;
    top: 56px;
    right: var(--s-4);
    z-index: 2;
    width: 280px;
    padding: var(--s-3);
    border-radius: var(--r-md);
    background: var(--reader-bg);
    color: var(--reader-fg);
    border: 1px solid var(--reader-line);
    box-shadow: 0 18px 48px rgba(0, 0, 0, 0.18);
    display: flex;
    flex-direction: column;
    gap: var(--s-3);
  }
  .row {
    display: grid;
    grid-template-columns: 70px 1fr;
    gap: var(--s-3);
    align-items: center;
  }
  .row .lbl {
    font-size: 10.5px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--reader-mute);
    font-weight: 500;
  }
  .seg {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: 1fr;
    border: 1px solid var(--reader-line);
    border-radius: var(--r-sm);
    overflow: hidden;
  }
  .seg button {
    border: 0;
    background: transparent;
    color: var(--reader-fg);
    padding: 5px 8px;
    cursor: default;
    font-size: 12px;
    border-right: 1px solid var(--reader-line);
  }
  .seg button:last-child {
    border-right: 0;
  }
  .seg button:hover {
    background: color-mix(in srgb, var(--reader-fg) 8%, transparent);
  }
  .seg button.on {
    background: color-mix(in srgb, var(--reader-accent) 18%, transparent);
    color: var(--reader-accent);
  }
  .seg.theme button {
    display: grid;
    place-items: center;
    padding: 4px;
  }
  .sw {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    border: 1px solid var(--reader-line);
    display: block;
  }
  .sw.light {
    background: #fafaf7;
  }
  .sw.sepia {
    background: #f4ecd8;
  }
  .sw.dark {
    background: #1c1c1e;
  }
  input[type='range'] {
    -webkit-appearance: none;
    appearance: none;
    background: transparent;
    width: 100%;
  }
  input[type='range']::-webkit-slider-runnable-track {
    height: 3px;
    background: color-mix(in srgb, var(--reader-fg) 14%, transparent);
    border-radius: 999px;
  }
  input[type='range']::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 14px;
    height: 14px;
    margin-top: -5.5px;
    background: var(--reader-accent);
    border-radius: 50%;
    border: 2px solid var(--reader-bg);
  }
</style>
