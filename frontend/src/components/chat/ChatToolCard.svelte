<script lang="ts">
  import {prettyJSON, summarizeArgs, summarizeResult} from '../../lib/chat/format';
  import type {VisibleToolCall} from '../../lib/chat/history';

  type Props = VisibleToolCall;

  let {call, result}: Props = $props();
</script>

<details class="tool-card">
  <summary>
    <span class="tool-glyph" aria-hidden="true">
      <svg viewBox="0 0 16 16" width="12" height="12">
        <circle cx="7" cy="7" r="4.5" fill="none" stroke="currentColor" stroke-width="1.4" />
        <path d="M10.5 10.5L13 13" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
      </svg>
    </span>
    <code class="tool-name">{call.name}</code>
    {#if summarizeArgs(call.arguments)}
      <span class="tool-args">{summarizeArgs(call.arguments)}</span>
    {/if}
    <span class="tool-summary">{summarizeResult(result)}</span>
  </summary>
  <div class="tool-detail">
    {#if call.arguments}
      <div class="tool-section">
        <div class="tool-section-label">Arguments</div>
        <pre>{prettyJSON(call.arguments)}</pre>
      </div>
    {/if}
    <div class="tool-section">
      <div class="tool-section-label">Result</div>
      <pre>{prettyJSON(result) || 'no result'}</pre>
    </div>
  </div>
</details>

<style>
  .tool-card {
    border: 1px solid var(--line);
    border-radius: 10px;
    background: color-mix(in srgb, var(--bg-panel) 60%, transparent);
    font-size: 12px;
    overflow: hidden;
  }
  .tool-card[open] {
    background: var(--mat-row);
  }
  .tool-card summary {
    display: flex;
    align-items: center;
    gap: var(--s-2);
    padding: 6px 10px;
    cursor: default;
    list-style: none;
    color: var(--fg-mute);
    -webkit-app-region: no-drag;
    user-select: none;
  }
  .tool-card summary::-webkit-details-marker {
    display: none;
  }
  .tool-card summary:hover {
    background: var(--mat-hover);
  }
  .tool-card[open] summary {
    border-bottom: 1px solid var(--line);
  }
  .tool-glyph {
    display: inline-flex;
    color: var(--accent);
  }
  .tool-name {
    font-family: var(--font-mono);
    font-size: 11.5px;
    color: var(--fg);
    font-weight: 500;
  }
  .tool-args {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg-dim);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1 1 auto;
    min-width: 0;
  }
  .tool-summary {
    margin-left: auto;
    font-size: 11px;
    color: var(--fg-mute);
    padding: 1px 8px;
    border-radius: 999px;
    border: 1px solid var(--line);
    background: var(--bg-base);
    flex-shrink: 0;
    font-variant-numeric: tabular-nums;
  }
  .tool-detail {
    padding: 8px 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .tool-section-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--fg-dim);
    margin-bottom: 4px;
  }
  .tool-detail pre {
    margin: 0;
    padding: 8px 10px;
    background: var(--bg-base);
    border: 1px solid var(--line);
    border-radius: 6px;
    font-family: var(--font-mono);
    font-size: 11px;
    line-height: 1.45;
    color: var(--fg);
    max-height: 240px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
