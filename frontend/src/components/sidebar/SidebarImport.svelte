<script lang="ts">
  import {Plus} from '@lucide/svelte';
  import Badge from '../../lib/components/ui/badge/badge.svelte';
  import Button from '../../lib/components/ui/button/button.svelte';
  import Typography from '../../lib/components/ui/typography/typography.svelte';
  import type {ImporterStatus} from '../../types';

  type Props = {
    importer: ImporterStatus;
    onAddBooks: () => void;
  };

  let {importer, onAddBooks}: Props = $props();

  const queuedCount = $derived(importer.queuedPaths?.length ?? 0);

  function lastSegment(p: string): string {
    if (!p) return '';
    const trimmed = p.replace(/[/\\]+$/, '');
    const parts = trimmed.split(/[/\\]/);
    return parts[parts.length - 1] || trimmed;
  }
</script>

<div class="add-block">
  <Button variant="primary" size="md" onclick={onAddBooks} class="add-btn">
    <Plus size={14} strokeWidth={2} />
    <span>Import</span>
    {#if queuedCount > 0}
      <Badge variant="accent" size="sm">+{queuedCount}</Badge>
    {/if}
  </Button>

  {#if queuedCount > 0}
    <div class="queued">
      <Typography variant="eyebrow" tone="dim">Up next</Typography>
      <ul class="path-list" role="list">
        {#each importer.queuedPaths!.slice(0, 4) as q (q)}
          <li class="path" title={q}>{lastSegment(q)}</li>
        {/each}
        {#if queuedCount > 4}
          <li class="path-more">+{queuedCount - 4} more...</li>
        {/if}
      </ul>
    </div>
  {/if}
</div>

<style>
  .add-block {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
  }
  :global(.library-sidebar .add-btn) {
    width: 100%;
    justify-content: center;
    gap: var(--uin-s-2);
  }
  .queued {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-1);
    padding: 0 var(--uin-s-1);
  }
  .path-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .path {
    font-size: 11px;
    color: var(--uin-fg);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .path-more {
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-style: italic;
  }
</style>
