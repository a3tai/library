<script lang="ts">
  import Button from '../../lib/components/ui/button/button.svelte';
  import Lamp from '../../lib/components/ui/lamp/lamp.svelte';
  import ProgressBar from '../../lib/components/ui/progress-bar/progress-bar.svelte';
  import Spinner from '../../lib/components/ui/spinner/spinner.svelte';
  import StatusRow from '../../lib/components/ui/status-row/status-row.svelte';
  import type {ImportLamp, ImporterStatus} from '../../types';

  type Props = {
    importer: ImporterStatus;
    importLamp: ImportLamp;
    importLabel: string;
    mcpRunning: boolean;
    mcpURL: string;
    mcpPort: number;
    version: string;
    onOpenImporter: () => void;
    onToggleMCP: () => void;
    onMCPPortChange: (port: number) => void;
  };

  let {
    importer,
    importLamp,
    importLabel,
    mcpRunning,
    mcpURL,
    mcpPort,
    version,
    onOpenImporter,
    onToggleMCP,
    onMCPPortChange,
  }: Props = $props();
</script>

<div class="foot">
  {#if importer.running}
    <div class="foot-status">
      <StatusRow
        title={importer.discovering ? 'Discovering' : 'Importing'}
        onClick={onOpenImporter}
        ariaLabel="Show importer details"
      >
        {#snippet leading()}<Spinner size="sm" />{/snippet}
        {#if importer.discovering}
          Walking · {importer.total.toLocaleString()} files
        {:else}
          {importer.processed.toLocaleString()} of {importer.total.toLocaleString()}
          {#if importer.total > 0}
            · {Math.round((importer.processed / importer.total) * 100)}%
          {/if}
        {/if}
        {#snippet bar()}
          {#if importer.discovering}
            <ProgressBar size="sm" indeterminate />
          {:else if importer.total > 0}
            <ProgressBar size="sm" value={importer.processed} max={importer.total} />
          {/if}
        {/snippet}
        {#snippet meta()}
          {#if importer.enricherQueueDepth > 0}
            Hydrating {importer.enricherQueueDepth.toLocaleString()} in queue
          {/if}
        {/snippet}
      </StatusRow>
    </div>
  {/if}

  <details class="mcp">
    <summary>
      <span>MCP</span>
      <Lamp state={mcpRunning ? 'running' : 'idle'} />
    </summary>
    <div class="mcp-body">
      <label class="mcp-row">
        <span>Local port</span>
        <input
          type="number"
          min="1"
          max="65535"
          value={mcpPort}
          disabled={mcpRunning}
          oninput={(event) => onMCPPortChange(Number((event.target as HTMLInputElement).value) || 0)}
        />
      </label>
      <Button
        size="sm"
        variant={mcpRunning ? 'outline' : 'primary'}
        onclick={onToggleMCP}
        class="mcp-toggle"
      >
        {mcpRunning ? 'Stop service' : 'Start service'}
      </Button>
      <p class="mcp-meta">{mcpURL || 'Off - start when an MCP client needs HTTP access.'}</p>
    </div>
  </details>

  <div class="status-foot" title={importLabel}>
    <span class="status-foot-label">Status</span>
    <span class="status-foot-value">{importLabel}</span>
    <Lamp state={importLamp} />
  </div>

  <div class="version-foot" aria-label="A3T: Library version">
    Library {version || '-'}
  </div>
</div>

<style>
  .foot {
    display: flex;
    flex-direction: column;
    gap: 0;
  }
  .foot-status {
    padding: var(--uin-s-2) var(--uin-s-3);
  }
  .mcp {
    border-top: 1px solid var(--uin-line);
    border-bottom: 1px solid var(--uin-line);
  }
  .mcp summary {
    list-style: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px var(--uin-s-3);
    cursor: default;
    color: var(--uin-fg-mute);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 500;
    -webkit-app-region: no-drag;
  }
  .mcp summary::-webkit-details-marker {
    display: none;
  }
  .mcp summary:hover {
    background: var(--uin-mat-hover);
  }
  .mcp-body {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
    padding: var(--uin-s-2) var(--uin-s-3) var(--uin-s-3);
  }
  .mcp-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--uin-s-2);
    font-size: 11px;
    color: var(--uin-fg-mute);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .mcp-row input {
    width: 80px;
    text-align: right;
    font-variant-numeric: tabular-nums;
    -webkit-app-region: no-drag;
    border: 1px solid var(--uin-line);
    background: var(--uin-mat-panel);
    color: var(--uin-fg);
    border-radius: var(--uin-r-sm);
    padding: 3px 6px;
    font: inherit;
  }
  :global(.library-sidebar .mcp-toggle) {
    width: 100%;
    justify-content: center;
  }
  .mcp-meta {
    margin: 0;
    font-size: 10.5px;
    color: var(--uin-fg-mute);
    overflow-wrap: anywhere;
  }
  .status-foot {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--uin-s-2);
    padding: 6px var(--uin-s-3);
    color: var(--uin-fg-mute);
    font-size: 11px;
    border-bottom: 1px solid var(--uin-line);
  }
  .status-foot-label {
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 500;
  }
  .status-foot-value {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--uin-fg);
    font-variant-numeric: tabular-nums;
  }
  .version-foot {
    padding: 6px var(--uin-s-3);
    text-align: center;
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.02em;
  }
</style>
