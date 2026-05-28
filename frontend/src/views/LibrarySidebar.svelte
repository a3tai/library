<!--
  LibrarySidebar — left rail.

  Brand notch · SourceList views · Stats stat-pair · Importer status
  row (when a run is in flight) · Import button · MCP toggle
  in the footer.
-->
<script lang="ts">
  import Notch from '../lib/components/ui/notch/notch.svelte';
  import ScrollArea from '../lib/components/ui/scroll-area/scroll-area.svelte';
  import Sidebar from '../lib/components/ui/sidebar/sidebar.svelte';
  import type {ImportLamp, ImporterStatus, LibraryView, Stats} from '../types';
  import SidebarFooter from '../components/sidebar/SidebarFooter.svelte';
  import SidebarImport from '../components/sidebar/SidebarImport.svelte';
  import SidebarNav from '../components/sidebar/SidebarNav.svelte';
  import SidebarStats from '../components/sidebar/SidebarStats.svelte';

  type Props = {
    view: LibraryView;
    stats: Stats;
    importer: ImporterStatus;
    importLamp: ImportLamp;
    importLabel: string;
    mcpRunning: boolean;
    mcpURL: string;
    mcpPort: number;
    version: string;
    onSwitchView: (next: LibraryView) => void;
    onAddBooks: () => void;
    onOpenImporter: () => void;
    onToggleMCP: () => void;
    onMCPPortChange: (port: number) => void;
  };

  let {
    view,
    stats,
    importer,
    importLamp,
    importLabel,
    mcpRunning,
    mcpURL,
    mcpPort,
    version,
    onSwitchView,
    onAddBooks,
    onOpenImporter,
    onToggleMCP,
    onMCPPortChange,
  }: Props = $props();
</script>

<Sidebar width="244px" class="library-sidebar">
  {#snippet header()}
    <Notch>
      {#snippet leading()}
        <img class="brand-mark" src="/appicon.png" alt="" aria-hidden="true" />
      {/snippet}
      <span class="brand-text">
        <span class="brand-title">A3T: Library</span>
        <span class="brand-sub">Local library</span>
      </span>
    </Notch>
  {/snippet}

  <ScrollArea class="sidebar-scroll">
    <div class="rail">
      <SidebarStats {stats} />
      <SidebarNav {view} {stats} {onSwitchView} />
      <SidebarImport {importer} {onAddBooks} />
    </div>
  </ScrollArea>

  {#snippet footer()}
    <SidebarFooter
      {importer}
      {importLamp}
      {importLabel}
      {mcpRunning}
      {mcpURL}
      {mcpPort}
      {version}
      {onOpenImporter}
      {onToggleMCP}
      {onMCPPortChange}
    />
  {/snippet}
</Sidebar>

<style>
  :global(.library-sidebar) {
    background: var(--uin-mat-sidebar);
    backdrop-filter: blur(40px) saturate(1.6);
    -webkit-backdrop-filter: blur(40px) saturate(1.6);
    border-right: 1px solid var(--uin-line);
  }
  /* Notch already draws its own bottom hairline; Mittsu's sidebar
     header wrapper would draw a second one. Suppress the wrapper's
     border for this sidebar instance. Same for header padding —
     Notch has its own. */
  :global(.library-sidebar .uin-sidebar-head) {
    border-bottom: 0;
    padding: 0;
  }
  /* Sidebar foot strips horizontal padding so child rows can paint
     edge-to-edge dividers, drops its own top border (the status row
     and MCP details supply their own separators), and keeps a bit
     of bottom padding so the status lamp has breathing room. */
  :global(.library-sidebar .uin-sidebar-foot) {
    padding-left: 0;
    padding-right: 0;
    padding-top: 0;
    padding-bottom: var(--uin-s-2);
    border-top: 0;
  }

  .brand-mark {
    width: 26px;
    height: 26px;
    border-radius: var(--uin-r-sm);
    display: block;
    object-fit: contain;
    /* The PNG ships at 1024×1024 — let the browser downscale and rely
       on the rounded mask to match the rest of the chrome's corner
       radius vocabulary. */
  }
  .brand-text {
    display: flex;
    flex-direction: column;
    line-height: 1.15;
  }
  .brand-title {
    font-size: 13px;
    font-weight: 600;
  }
  .brand-sub {
    font-size: 10.5px;
    color: var(--uin-fg-mute);
  }

  :global(.sidebar-scroll) {
    flex: 1;
    min-height: 0;
  }
  .rail {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-3);
    padding: var(--uin-s-2) var(--uin-s-2) var(--uin-s-3);
  }
</style>
