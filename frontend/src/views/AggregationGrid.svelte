<!--
  AggregationGrid — flat list of {name, count} groups (authors / subjects
  / categories). Drilling in re-runs search with a `field:"name"` query.
-->
<script lang="ts">
  import Empty from '../lib/components/ui/empty/empty.svelte';
  import Spinner from '../lib/components/ui/spinner/spinner.svelte';
  import Typography from '../lib/components/ui/typography/typography.svelte';
  import type {AggregateGroup} from '../types';

  type Props = {
    groups: AggregateGroup[];
    loading?: boolean;
    eyebrow: string;
    onSelect: (name: string) => void;
  };

  let {groups, loading = false, eyebrow, onSelect}: Props = $props();
</script>

<div class="agg">
  <Typography variant="eyebrow" tone="dim" class="agg-eyebrow">{eyebrow}</Typography>
  {#if loading}
    <div class="loading"><Spinner /></div>
  {:else if groups.length === 0}
    <Empty title="Nothing here yet" description="Run the enricher on more books to populate this view." />
  {:else}
    <ul class="rows" role="list">
      {#each groups as g (g.name)}
        <li>
          <button type="button" class="row" onclick={() => onSelect(g.name)}>
            <span class="row-name">{g.name}</span>
            <span class="row-count">{g.count.toLocaleString()}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .agg {
    padding: var(--uin-s-4) 0;
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-3);
  }
  .agg :global(.agg-eyebrow) {
    margin: 0;
  }
  .loading {
    display: grid;
    place-items: center;
    padding: var(--uin-s-7) 0;
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: var(--uin-s-1);
  }
  .row {
    -webkit-app-region: no-drag;
    width: 100%;
    display: flex;
    justify-content: space-between;
    gap: var(--uin-s-3);
    padding: var(--uin-s-2) var(--uin-s-3);
    border: 1px solid var(--uin-line);
    background: var(--uin-mat-row);
    color: var(--uin-fg);
    text-align: left;
    border-radius: var(--uin-r-sm);
    font-size: 12.5px;
    cursor: default;
    transition:
      background-color var(--uin-dur-1) var(--uin-ease-standard),
      border-color var(--uin-dur-1) var(--uin-ease-standard);
  }
  .row:hover {
    background: var(--uin-mat-hover);
    border-color: var(--uin-line-strong);
  }
  .row:focus-visible {
    outline: none;
    box-shadow: var(--uin-focus-ring);
  }
  .row-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row-count {
    font-family: var(--uin-font-mono);
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-variant-numeric: tabular-nums;
  }
</style>
