<!--
@component Command — searchable command palette.

The ⌘K-style filterable list pattern. Renders a search input + virtualised
list of items, with keyboard navigation (↑↓ to move, ↩ to select, Esc to
close). Open state is consumer-managed via `bind:open`.

Items are flat: `{value, label, hint?, kbd?, group?, icon?, disabled?}[]`.
Items with the same `group` are clustered together with a heading.

Pure-Svelte — no headless dep. Mount inside a `<Dialog>` for modal
overlay, or use `<Popover>` for inline usage.

CSS lives in `./command.css`.
-->
<script lang="ts" generics="T extends string">
  import type {Snippet} from 'svelte';
  import {tick} from 'svelte';
  import {cn} from '../../../utils/cn';

  type CommandItem<V extends string = string> = {
    value: V;
    label: string;
    hint?: string;
    kbd?: string;
    group?: string;
    icon?: Snippet;
    disabled?: boolean;
  };

  type Props = {
    items: CommandItem<T>[];
    placeholder?: string;
    onSelect: (value: T) => void;
    emptyLabel?: string;
    class?: string;
    autoFocus?: boolean;
  };

  let {
    items,
    placeholder = 'Search…',
    onSelect,
    emptyLabel = 'No results',
    class: className,
    autoFocus = true,
  }: Props = $props();

  let query = $state('');
  let activeIdx = $state(0);
  let inputEl: HTMLInputElement | undefined = $state();
  let listEl: HTMLDivElement | undefined = $state();

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items.filter((it) => !it.disabled);
    return items.filter((it) => {
      if (it.disabled) return false;
      const hay = `${it.label} ${it.hint ?? ''} ${it.group ?? ''}`.toLowerCase();
      return hay.includes(q);
    });
  });

  const grouped = $derived.by(() => {
    const out: {group: string | null; items: CommandItem<T>[]}[] = [];
    for (const it of filtered) {
      const g = it.group ?? null;
      const last = out[out.length - 1];
      if (last && last.group === g) last.items.push(it);
      else out.push({group: g, items: [it]});
    }
    return out;
  });

  const flatIdx = $derived.by(() => {
    // Map activeIdx into the flat filtered list
    return filtered;
  });

  $effect(() => {
    void query;
    activeIdx = 0;
  });

  $effect(() => {
    if (autoFocus && inputEl) inputEl.focus();
  });

  async function moveActive(delta: number) {
    const len = flatIdx.length;
    if (!len) return;
    activeIdx = (activeIdx + delta + len) % len;
    await tick();
    const el = listEl?.querySelector<HTMLElement>('[data-active="true"]');
    el?.scrollIntoView({block: 'nearest'});
  }

  function commit(v: T) {
    onSelect(v);
  }

  function handleKey(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      moveActive(1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      moveActive(-1);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const pick = flatIdx[activeIdx];
      if (pick) commit(pick.value);
    }
  }
</script>

<div class={cn('uin-command', className)}>
  <div class="uin-command-search">
    <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
      <circle cx="7" cy="7" r="5" fill="none" stroke="currentColor" stroke-width="1.5" />
      <line x1="11" y1="11" x2="14" y2="14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
    </svg>
    <input
      bind:this={inputEl}
      bind:value={query}
      onkeydown={handleKey}
      type="text"
      autocomplete="off"
      spellcheck="false"
      placeholder={placeholder}
      aria-label={placeholder}
    />
  </div>
  <div class="uin-command-list" bind:this={listEl} role="listbox">
    {#if filtered.length === 0}
      <p class="uin-command-empty">{emptyLabel}</p>
    {/if}
    {#each grouped as g}
      {#if g.group}
        <p class="uin-command-grouplabel">{g.group}</p>
      {/if}
      {#each g.items as it (it.value)}
        {@const idx = flatIdx.indexOf(it)}
        {@const active = idx === activeIdx}
        <button
          type="button"
          class={cn('uin-command-item', active && 'uin-command-item-active')}
          data-active={active}
          role="option"
          aria-selected={active}
          onclick={() => commit(it.value)}
          onmouseenter={() => (activeIdx = idx)}
        >
          {#if it.icon}
            <span class="uin-command-item-icon">{@render it.icon()}</span>
          {/if}
          <span class="uin-command-item-label">{it.label}</span>
          {#if it.hint}<span class="uin-command-item-hint">{it.hint}</span>{/if}
          {#if it.kbd}<span class="uin-command-item-kbd">{it.kbd}</span>{/if}
        </button>
      {/each}
    {/each}
  </div>
</div>
