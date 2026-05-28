<script lang="ts">
  import {
    AlertTriangle,
    BookOpen,
    Clock,
    FolderTree,
    Settings,
    Tag,
    UserRound,
  } from '@lucide/svelte';
  import NavItem from '../../lib/components/ui/nav-item/nav-item.svelte';
  import type {LibraryView, Stats} from '../../types';

  type Props = {
    view: LibraryView;
    stats: Stats;
    onSwitchView: (next: LibraryView) => void;
  };

  let {view, stats, onSwitchView}: Props = $props();

  type NavRow =
    | {type: 'item'; value: LibraryView; label: string; icon: typeof BookOpen; count?: number}
    | {type: 'heading'; label: string};

  const navRows = $derived<NavRow[]>([
    {type: 'item', value: 'library', label: 'Library', icon: BookOpen, count: stats.books},
    {type: 'item', value: 'recent', label: 'Recently added', icon: Clock},
    {type: 'heading', label: 'Browse'},
    {type: 'item', value: 'categories', label: 'Categories', icon: FolderTree},
    {type: 'item', value: 'byauthor', label: 'By author', icon: UserRound},
    {type: 'item', value: 'bysubject', label: 'By subject', icon: Tag},
    {type: 'heading', label: 'Pending'},
    {
      type: 'item',
      value: 'unprocessed',
      label: 'Unprocessed',
      icon: AlertTriangle,
      count: stats.needsMetadata > 0 ? stats.needsMetadata : undefined,
    },
    {type: 'heading', label: 'App'},
    {type: 'item', value: 'settings', label: 'Settings', icon: Settings},
  ]);
</script>

<nav class="views" aria-label="Library views">
  {#each navRows as row, i (i)}
    {#if row.type === 'heading'}
      <p class="views-heading">{row.label}</p>
    {:else}
      {@const Icon = row.icon}
      <NavItem active={view === row.value} onclick={() => onSwitchView(row.value)}>
        {#snippet dot()}<Icon size={14} strokeWidth={1.6} />{/snippet}
        {row.label}
        {#snippet aside()}
          {#if row.count !== undefined}
            <span class="views-count">{row.count.toLocaleString()}</span>
          {/if}
        {/snippet}
      </NavItem>
    {/if}
  {/each}
</nav>

<style>
  .views {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .views-heading {
    margin: var(--uin-s-5) 0 var(--uin-s-1);
    padding: 0 var(--uin-s-2);
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.16em;
    color: var(--uin-fg-dim);
    font-weight: 600;
  }
  .views-heading:first-child {
    margin-top: 0;
  }
  .views-count {
    font-family: var(--uin-font-mono);
    font-size: 10.5px;
    color: var(--uin-fg-dim);
    font-variant-numeric: tabular-nums;
  }
</style>
