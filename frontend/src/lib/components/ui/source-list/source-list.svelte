<!--
@component SourceList — sidebar list with icon / label / count. 🍎

The macOS Finder-style sidebar list pattern. Each row carries an
optional icon glyph, a label, and an optional aside count. Items can
be grouped under section headings.

`items: SourceItem[]` is one of:
- `{value, label, icon?, count?, disabled?}` — selectable row
- `{type: 'heading', label}` — non-interactive group title

Bind `value` to track selection (single-select). Renders as a list of
`<NavItem>`-shaped buttons for keyboard accessibility.

CSS lives in `./source-list.css`.
-->
<script lang="ts" generics="T extends string">
  import type {Snippet} from 'svelte';
  import {cn} from '../../../utils/cn';

  type Item =
    | {
        type?: 'item';
        value: T;
        label: string;
        icon?: Snippet;
        count?: number;
        disabled?: boolean;
      }
    | {type: 'heading'; label: string};

  type Props = {
    items: Item[];
    value?: T;
    onSelect?: (next: T) => void;
    class?: string;
    ariaLabel?: string;
  };

  let {
    items,
    value = $bindable<T>('' as T),
    onSelect,
    class: className,
    ariaLabel = 'Sources',
  }: Props = $props();

  function pick(v: T) {
    value = v;
    onSelect?.(v);
  }
</script>

<nav class={cn('uin-srclist', className)} aria-label={ariaLabel}>
  <ul>
    {#each items as item, i (i)}
      {#if item.type === 'heading'}
        <li class="uin-srclist-heading">{item.label}</li>
      {:else}
        <li>
          <button
            type="button"
            class={cn('uin-srclist-row', value === item.value && 'uin-srclist-row-active')}
            disabled={item.disabled}
            aria-current={value === item.value ? 'page' : undefined}
            onclick={() => pick(item.value)}
          >
            <span class="uin-srclist-icon" aria-hidden="true">
              {#if item.icon}{@render item.icon()}{/if}
            </span>
            <span class="uin-srclist-label">{item.label}</span>
            {#if item.count !== undefined}
              <span class="uin-srclist-count">{item.count.toLocaleString()}</span>
            {/if}
          </button>
        </li>
      {/if}
    {/each}
  </ul>
</nav>
