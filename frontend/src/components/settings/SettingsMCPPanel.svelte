<script lang="ts">
  import type {MCPStatusDTO} from '../../lib/api/library';
  import Button from '../../lib/components/ui/button/button.svelte';
  import Field from '../../lib/components/ui/field/field.svelte';
  import Input from '../../lib/components/ui/input/input.svelte';
  import Lamp from '../../lib/components/ui/lamp/lamp.svelte';

  type Props = {
    mcp: MCPStatusDTO | null;
    mcpPortInput: number;
    busy: boolean;
    onToggle: () => void;
  };

  let {mcp, mcpPortInput = $bindable(8765), busy, onToggle}: Props = $props();

  function onPortInput(event: Event) {
    const value = Number((event.target as HTMLInputElement).value);
    mcpPortInput = Number.isFinite(value) ? value : 0;
  }
</script>

<div class="panel">
  <div class="status-row">
    <Lamp state={mcp?.running ? 'running' : 'idle'} />
    <span class="status-text">{mcp?.running ? 'Running' : 'Stopped'}</span>
    {#if mcp?.running && mcp.url}
      <code class="status-url">{mcp.url}</code>
    {/if}
  </div>

  <div class="grid">
    <Field
      label="Local port"
      description="Port the MCP HTTP server binds to on 127.0.0.1. The server only listens on loopback."
    >
      {#snippet children({id})}
        <Input
          {id}
          type="number"
          min={1}
          max={65535}
          value={String(mcpPortInput)}
          oninput={onPortInput}
          disabled={mcp?.running}
          placeholder="8765"
        />
      {/snippet}
    </Field>
  </div>

  <div class="actions">
    <Button variant={mcp?.running ? 'outline' : 'primary'} disabled={busy} onclick={onToggle}>
      {busy ? '...' : mcp?.running ? 'Stop server' : 'Start server'}
    </Button>
  </div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-5);
    max-width: 640px;
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--uin-s-4);
  }
  .actions {
    display: flex;
    gap: var(--uin-s-2);
  }
  .status-row {
    display: flex;
    align-items: center;
    gap: var(--uin-s-2);
    padding: var(--uin-s-2) var(--uin-s-3);
    border: 1px solid var(--uin-line);
    border-radius: var(--uin-r-sm);
    background: var(--uin-bg-elev);
  }
  .status-text {
    font-size: 13px;
    color: var(--uin-fg);
  }
  .status-url {
    margin-left: auto;
    font-family: var(--uin-font-mono);
    font-size: 12px;
    color: var(--uin-fg-mute);
  }
</style>
