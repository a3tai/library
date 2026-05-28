<!--
  ImporterSheet — right-side slide-in panel showing importer detail.

  Mirrors the substance of the legacy ImporterDialog: counts,
  progress bar, source path, indexer pass, queued paths, recent
  errors, and Cancel / Done actions. Wraps Mittsu's <Sheet>.
-->
<script lang="ts">
  import Badge from '../lib/components/ui/badge/badge.svelte';
  import Button from '../lib/components/ui/button/button.svelte';
  import Lamp from '../lib/components/ui/lamp/lamp.svelte';
  import ProgressBar from '../lib/components/ui/progress-bar/progress-bar.svelte';
  import Sheet from '../lib/components/ui/sheet/sheet.svelte';
  import Stat from '../lib/components/ui/stat/stat.svelte';
  import Typography from '../lib/components/ui/typography/typography.svelte';
  import type {ImportLamp, ImporterStatus} from '../types';

  type Props = {
    open: boolean;
    importer: ImporterStatus;
    importLamp: ImportLamp;
    onOpenChange: (next: boolean) => void;
    onCancel: () => void;
  };

  let {open, importer, importLamp, onOpenChange, onCancel}: Props = $props();

  const title = $derived(
    importer.discovering
      ? 'Discovering files'
      : importer.running
        ? 'Importing books'
        : importer.cancelled
          ? 'Import cancelled'
          : importer.done
            ? 'Import complete'
            : 'Importer'
  );

  const description = $derived(
    importer.discovering
      ? `${importer.total.toLocaleString()} files found · ${Math.round(importer.durationMs / 1000)}s elapsed`
      : importer.running && importer.total > 0
        ? `${importer.processed.toLocaleString()} of ${importer.total.toLocaleString()} files`
        : importer.done
          ? `${importer.processed.toLocaleString()} processed in ${Math.round(importer.durationMs / 1000)}s`
          : 'No import is currently running.'
  );
</script>

<Sheet bind:open onOpenChange={onOpenChange} side="right" size="lg" {title} {description}>
  <div class="body">
    <div class="head-row">
      <Lamp state={importLamp} size={10} />
      <span class="head-label">{title}</span>
    </div>

    {#if importer.discovering}
      <ProgressBar size="md" indeterminate />
    {:else if importer.total > 0 || importer.running}
      <ProgressBar size="md" value={importer.processed} max={importer.total || 1} />
    {/if}

    <div class="counts">
      <Stat label="Imported" value={importer.imported.toLocaleString()} />
      <Stat label="Updated" value={importer.updated.toLocaleString()} />
      <Stat label="Skipped" value={importer.skipped.toLocaleString()} />
      <Stat label="Failed" value={importer.failed.toLocaleString()} />
    </div>

    <dl class="meta">
      {#if importer.path}
        <div><dt>Source</dt><dd><code title={importer.path}>{importer.path}</code></dd></div>
      {/if}
      {#if importer.running && importer.current}
        <div><dt>Current</dt><dd><code title={importer.current}>{importer.current}</code></dd></div>
      {/if}
      {#if importer.indexer.running || importer.indexer.pending > 0 || importer.indexer.indexed > 0}
        <div>
          <dt>Pass 2 · Indexing</dt>
          <dd>
            {#if importer.indexer.running}
              <span class="pct">{importer.indexer.indexed.toLocaleString()} indexed</span>
              · {importer.indexer.pending.toLocaleString()} pending
              {#if importer.indexer.failed > 0}
                · <span class="fail">{importer.indexer.failed} failed</span>
              {/if}
              {#if importer.indexer.current}
                <code class="dim" title={importer.indexer.current}>{importer.indexer.current}</code>
              {/if}
            {:else if importer.indexer.pending > 0}
              Queued · {importer.indexer.pending.toLocaleString()} books awaiting passage extraction
            {:else}
              Idle · {importer.indexer.indexed.toLocaleString()} indexed
            {/if}
          </dd>
        </div>
      {/if}
      {#if importer.enricherQueueDepth > 0}
        <div><dt>Hydration queue</dt><dd>{importer.enricherQueueDepth.toLocaleString()} pending</dd></div>
      {/if}
      {#if (importer.queuedPaths?.length ?? 0) > 0}
        <div>
          <dt>Up next</dt>
          <dd>
            <ol class="queued-list">
              {#each importer.queuedPaths! as q (q)}
                <li><code title={q}>{q}</code></li>
              {/each}
            </ol>
          </dd>
        </div>
      {/if}
      {#if importer.error}
        <div><dt>Error</dt><dd class="fail">{importer.error}</dd></div>
      {/if}
    </dl>

    {#if importer.recentErrors.length > 0}
      <section class="errors">
        <Typography variant="eyebrow" tone="dim">Recent errors</Typography>
        <Badge variant="warn" size="sm">{importer.recentErrors.length}</Badge>
        <ol>
          {#each importer.recentErrors.slice(-8) as err, i (i)}
            <li><code>{err}</code></li>
          {/each}
        </ol>
      </section>
    {/if}
  </div>

  {#snippet footer()}
    <div class="foot">
      {#if importer.running}
        <Button variant="outline" size="sm" onclick={onCancel}>Cancel import</Button>
      {/if}
      <Button variant="primary" size="sm" onclick={() => onOpenChange(false)}>Done</Button>
    </div>
  {/snippet}
</Sheet>

<style>
  .body {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-3);
  }
  .head-row {
    display: flex;
    align-items: center;
    gap: var(--uin-s-2);
    color: var(--uin-fg);
    font-weight: 500;
  }
  .head-label {
    font-size: 12.5px;
  }
  .counts {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--uin-s-2);
  }
  .meta {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
    margin: 0;
  }
  .meta div {
    display: grid;
    grid-template-columns: 120px 1fr;
    gap: var(--uin-s-3);
    align-items: baseline;
  }
  .meta dt {
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin: 0;
  }
  .meta dd {
    margin: 0;
    font-size: 12px;
    color: var(--uin-fg);
    overflow-wrap: anywhere;
  }
  .meta code {
    font-family: var(--uin-font-mono);
    font-size: 11px;
    color: var(--uin-fg-mute);
  }
  .meta code.dim {
    color: var(--uin-fg-dim);
  }
  .pct {
    color: var(--uin-accent);
    font-weight: 500;
  }
  .fail {
    color: var(--uin-danger);
  }
  .queued-list {
    list-style: decimal inside;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .errors {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-1);
  }
  .errors ol {
    margin: 0;
    padding-left: var(--uin-s-4);
    font-family: var(--uin-font-mono);
    font-size: 11px;
    color: var(--uin-fg-mute);
  }
  .foot {
    display: flex;
    justify-content: flex-end;
    gap: var(--uin-s-2);
  }
</style>
