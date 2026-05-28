<script lang="ts">
  import type {Book} from '../../types';

  type Props = {
    book: Book;
    available: boolean;
    model: string;
    onclose: () => void;
    onDoubleClick: (event: MouseEvent) => void;
  };

  let {book, available, model, onclose, onDoubleClick}: Props = $props();
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<header class="bar" ondblclick={onDoubleClick}>
  <button class="icon" type="button" onclick={onclose} title="Back (Esc)" aria-label="Close chat">
    <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
      <path d="M10 3L5 8l5 5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </button>
  <div class="title-block">
    <strong>{book.title}</strong>
    <span class="sub">
      {#if available}
        Chat · {model || 'LM Studio'}
      {:else}
        Chat · LM Studio not reachable
      {/if}
    </span>
  </div>
  <div class="meta">
    {#if !available}
      <span class="dot offline" aria-hidden="true"></span>
    {:else}
      <span class="dot online" aria-hidden="true"></span>
    {/if}
  </div>
</header>

<style>
  .bar {
    flex-shrink: 0;
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--s-3);
    padding: 6px var(--s-4) 6px 80px;
    border-bottom: 1px solid var(--line);
    background: color-mix(in srgb, var(--bg-base) 85%, transparent);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    -webkit-app-region: drag;
  }
  .icon {
    -webkit-app-region: no-drag;
    width: 28px;
    height: 28px;
    display: grid;
    place-items: center;
    border: 0;
    background: transparent;
    color: var(--fg);
    border-radius: var(--r-sm);
    cursor: default;
  }
  .icon:hover {
    background: var(--mat-hover);
  }
  .icon:focus-visible {
    outline: none;
    box-shadow: var(--focus-ring);
  }
  .title-block {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 0;
    line-height: 1.2;
  }
  .title-block strong {
    font-size: 12.5px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 60vw;
  }
  .title-block .sub {
    font-size: 11px;
    color: var(--fg-mute);
  }
  .meta {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .dot.online {
    background: #34c759;
  }
  .dot.offline {
    background: var(--warn);
  }
</style>
