<!--
@component ContextMenu — right-click pop-up menu.

Built on bits-ui's ContextMenu primitive. Wraps a target region; right-
clicking anywhere inside the region opens the menu at the cursor.

Items follow the same `MenuItem` shape as `DropdownMenu`:
- `{ label, shortcut?, disabled?, tone?, onSelect? }`
- `{ type: 'separator' }`
- `{ type: 'label', label }`

Reuses the `.uin-menu-*` classes from DropdownMenu's CSS.

Requires `bits-ui` and `dropdown-menu` (for shared chrome).

CSS lives in `./context-menu.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {ContextMenu as CM} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  type MenuItem = {
    type?: 'item' | 'separator' | 'label';
    label?: string;
    shortcut?: string;
    disabled?: boolean;
    tone?: 'default' | 'danger';
    onSelect?: () => void;
  };

  type Props = {
    items: MenuItem[];
    class?: string;
    children: Snippet;
  };

  let {items, class: className, children}: Props = $props();
</script>

<CM.Root>
  <CM.Trigger class="uin-ctxmenu-trigger">{@render children()}</CM.Trigger>
  <CM.Portal>
    <CM.Content class={cn('uin-menu', className)}>
      {#each items as item, i (i)}
        {#if item.type === 'separator'}
          <CM.Separator class="uin-menu-sep" />
        {:else if item.type === 'label'}
          <div class="uin-menu-grouplabel">{item.label}</div>
        {:else}
          <CM.Item
            class={cn('uin-menu-item', item.tone === 'danger' && 'uin-menu-item-danger')}
            disabled={item.disabled}
            onSelect={() => item.onSelect?.()}
          >
            <span class="uin-menu-item-label">{item.label}</span>
            {#if item.shortcut}
              <span class="uin-menu-item-kbd">{item.shortcut}</span>
            {/if}
          </CM.Item>
        {/if}
      {/each}
    </CM.Content>
  </CM.Portal>
</CM.Root>
