<!--
@component OutlineView — disclosure-tree (TreeView). 🍎

A recursive list with collapsible rows and a chevron disclosure
glyph — the macOS Finder / sidebar tree pattern.

`nodes: TreeNode[]` is recursive: `{id, label, icon?, children?}`.
Selection is single-value via `bind:selectedId`. Per-node expand state
is managed internally; controlled mode would require forking.

For deep trees (>1000 nodes) reach for a virtualised tree later.

CSS lives in `./outline-view.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {untrack} from 'svelte';
  import {cn} from '../../../utils/cn';

  type TreeNode = {
    id: string;
    label: string;
    icon?: Snippet;
    children?: TreeNode[];
  };

  type Props = {
    nodes: TreeNode[];
    selectedId?: string;
    defaultExpanded?: string[];
    onSelect?: (id: string) => void;
    class?: string;
    ariaLabel?: string;
  };

  let {
    nodes,
    selectedId = $bindable<string>(''),
    defaultExpanded = [],
    onSelect,
    class: className,
    ariaLabel = 'Outline',
  }: Props = $props();

  // Snapshot defaultExpanded once via untrack(); the prop is treated as one-shot init.
  let expanded = $state<Set<string>>(untrack(() => new Set(defaultExpanded)));

  function toggle(id: string) {
    const next = new Set(expanded);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    expanded = next;
  }

  function pick(id: string) {
    selectedId = id;
    onSelect?.(id);
  }
</script>

{#snippet row(node: TreeNode, depth: number)}
  {@const open = expanded.has(node.id)}
  {@const hasChildren = !!node.children?.length}
  <li>
    <div
      class={cn('uin-tree-row', selectedId === node.id && 'uin-tree-row-active')}
      style="padding-left: calc({depth} * var(--uin-s-3) + var(--uin-s-2));"
    >
      <button
        type="button"
        class="uin-tree-disclose"
        aria-label={hasChildren ? (open ? 'Collapse' : 'Expand') : ''}
        aria-expanded={hasChildren ? open : undefined}
        disabled={!hasChildren}
        onclick={(e) => {
          e.stopPropagation();
          if (hasChildren) toggle(node.id);
        }}
      >
        {#if hasChildren}
          <svg class="uin-tree-chev" viewBox="0 0 12 12" width="10" height="10" aria-hidden="true">
            <polyline points="4,3 8,6 4,9" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        {/if}
      </button>
      <button
        type="button"
        class="uin-tree-label"
        aria-current={selectedId === node.id ? 'true' : undefined}
        onclick={() => pick(node.id)}
      >
        {#if node.icon}<span class="uin-tree-icon" aria-hidden="true">{@render node.icon()}</span>{/if}
        <span>{node.label}</span>
      </button>
    </div>
    {#if hasChildren && open}
      <ul class="uin-tree-children" role="group">
        {#each node.children! as child (child.id)}
          {@render row(child, depth + 1)}
        {/each}
      </ul>
    {/if}
  </li>
{/snippet}

<nav class={cn('uin-tree', className)} aria-label={ariaLabel}>
  <ul role="tree">
    {#each nodes as n (n.id)}
      {@render row(n, 0)}
    {/each}
  </ul>
</nav>
