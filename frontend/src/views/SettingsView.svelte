<!--
  SettingsView — center panel for the Settings sidebar entry / ⌘, menu.

  Three horizontal tabs:
    · LLM     — LM Studio URL, API key, embed/chat model dropdowns,
                Save + Test.
    · MCP     — local MCP HTTP server: port, start/stop, URL.
    · Library — diagnostics (DB path, current stats).

  Saving LLM settings rebuilds the embedder/aimeta clients in-process
  — no app restart needed. MCP start/stop is also live.
-->
<script lang="ts">
  import {onMount} from 'svelte';
  import {LibraryService, type MCPStatusDTO, type SettingsDTO, type StatsDTO} from '../lib/api/library';

  import SettingsLibraryDiagnostics from '../components/settings/SettingsLibraryDiagnostics.svelte';
  import SettingsLLMPanel from '../components/settings/SettingsLLMPanel.svelte';
  import SettingsMCPPanel from '../components/settings/SettingsMCPPanel.svelte';
  import PageBody from '../lib/components/ui/page-body/page-body.svelte';
  import PageHeader from '../lib/components/ui/page-header/page-header.svelte';
  import ScrollArea from '../lib/components/ui/scroll-area/scroll-area.svelte';
  import Tabs from '../lib/components/ui/tabs/tabs.svelte';
  import Typography from '../lib/components/ui/typography/typography.svelte';
  import {toast} from '../lib/components/ui/toast/toast.svelte';

  type TabValue = 'llm' | 'mcp' | 'library';
  let tab = $state<TabValue>('llm');
  const tabs = [
    {value: 'llm' as TabValue, label: 'LLM'},
    {value: 'mcp' as TabValue, label: 'MCP'},
    {value: 'library' as TabValue, label: 'Library'},
  ];

  let loaded = $state(false);
  let saving = $state(false);
  let testing = $state(false);
  let mcpBusy = $state(false);

  // Current effective values from the backend (DB → env → defaults).
  let current = $state<SettingsDTO | null>(null);
  let stats = $state<StatsDTO | null>(null);
  let mcp = $state<MCPStatusDTO | null>(null);

  // LLM form state. apiKeyChanged tracks whether the user actually
  // touched the field — only then do we send a value back (otherwise the
  // existing key is preserved server-side).
  let url = $state('');
  let embedModel = $state('');
  let chatModel = $state('');
  let apiKey = $state('');
  let apiKeyChanged = $state(false);

  // MCP form state — separate from `mcp` so the user can type a new port
  // without immediately mutating live status display.
  let mcpPortInput = $state(8765);

  // Models LM Studio currently advertises. Empty until reachable; UI
  // falls back to text inputs in that case.
  let availableModels = $state<string[]>([]);

  onMount(() => {
    void reload();
  });

  async function reload() {
    try {
      const [s, st, m] = await Promise.all([
        LibraryService.Settings() as Promise<SettingsDTO>,
        LibraryService.Stats() as Promise<StatsDTO>,
        LibraryService.MCPStatus() as Promise<MCPStatusDTO>,
      ]);
      current = s;
      stats = st;
      mcp = m;
      url = s.lmstudioURL;
      embedModel = s.lmstudioEmbedModel;
      chatModel = s.lmstudioChatModel;
      apiKey = '';
      apiKeyChanged = false;
      mcpPortInput = m?.port && m.port > 0 ? m.port : s.mcpPort && s.mcpPort > 0 ? s.mcpPort : 8765;
      loaded = true;
      void refreshModels();
    } catch (err) {
      toast.error({title: err instanceof Error ? err.message : String(err)});
    }
  }

  async function refreshModels() {
    try {
      const list = (await LibraryService.ListLMStudioModels()) as string[];
      availableModels = list ?? [];
    } catch {
      availableModels = [];
    }
  }

  async function save() {
    saving = true;
    try {
      const payload: any = {
        lmstudioURL: url,
        lmstudioEmbedModel: embedModel,
        lmstudioChatModel: chatModel,
      };
      if (apiKeyChanged) {
        payload.apiKey = apiKey;
      }
      const next = (await LibraryService.UpdateSettings(payload)) as SettingsDTO;
      current = next;
      url = next.lmstudioURL;
      embedModel = next.lmstudioEmbedModel;
      chatModel = next.lmstudioChatModel;
      apiKey = '';
      apiKeyChanged = false;
      toast.success({title: 'Settings saved.'});
    } catch (err) {
      toast.error({title: err instanceof Error ? err.message : String(err)});
    } finally {
      saving = false;
    }
  }

  async function testConnection() {
    testing = true;
    try {
      const ok = await LibraryService.TestLMStudio();
      if (ok) {
        await refreshModels();
        toast.success({
          title: 'LM Studio reachable.',
          description:
            availableModels.length > 0
              ? `${availableModels.length} model${availableModels.length === 1 ? '' : 's'} available.`
              : 'No models loaded — load one in LM Studio to enable vector search.',
        });
      } else {
        toast.error({title: 'LM Studio did not respond.'});
      }
    } catch (err) {
      toast.error({title: err instanceof Error ? err.message : String(err)});
    } finally {
      testing = false;
    }
  }

  async function toggleMCP() {
    if (mcpBusy) return;
    mcpBusy = true;
    try {
      const next = mcp?.running
        ? ((await LibraryService.StopMCPServer()) as MCPStatusDTO)
        : ((await LibraryService.StartMCPServer(mcpPortInput || 0)) as MCPStatusDTO);
      mcp = next;
      if (next.port > 0) mcpPortInput = next.port;
      toast.success({title: next.running ? 'MCP server started.' : 'MCP server stopped.'});
    } catch (err) {
      toast.error({title: err instanceof Error ? err.message : String(err)});
    } finally {
      mcpBusy = false;
    }
  }

  function markApiKeyChanged() {
    apiKeyChanged = true;
  }
</script>

<div class="settings">
  <ScrollArea class="settings-scroll">
    <PageHeader title="Settings" description="LLM connection, MCP server, and library diagnostics." sticky />

    <PageBody>
      <Tabs
        {tabs}
        bind:value={tab}
        direction="horizontal"
        ariaLabel="Settings sections"
        class="settings-tabs"
      />

      {#if !loaded}
        <Typography variant="body" tone="dim">Loading…</Typography>
      {:else if tab === 'llm'}
        <SettingsLLMPanel
          {current}
          {availableModels}
          bind:url
          bind:embedModel
          bind:chatModel
          bind:apiKey
          {saving}
          {testing}
          onApiKeyChange={markApiKeyChanged}
          onSave={save}
          onTestConnection={testConnection}
        />
      {:else if tab === 'mcp'}
        <SettingsMCPPanel {mcp} bind:mcpPortInput busy={mcpBusy} onToggle={toggleMCP} />
      {:else if tab === 'library'}
        <SettingsLibraryDiagnostics {current} {stats} />
      {/if}
    </PageBody>
  </ScrollArea>
</div>

<style>
  .settings {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: var(--uin-bg-base);
  }
  /* .settings-scroll gutter lives in app.css alongside .main-scroll —
     same reason: Svelte's CSS extractor can drop :global() rules from a
     component style block when the class only appears as a prop. */
  :global(.settings-tabs) {
    margin-bottom: var(--uin-s-5);
  }
</style>
