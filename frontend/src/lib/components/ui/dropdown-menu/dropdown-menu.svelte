<!--
@component DropdownMenu — pop-up menu of actions.

Built on bits-ui's DropdownMenu primitive — keyboard navigation, focus
management, ESC, click-outside, and ARIA roles all handled. Items are
declared as data (a `MenuItem[]`) rather than as subcomponents — more
compact for registry distribution while covering 90% of menu shapes.

A `MenuItem` is one of:
- `{ label, shortcut?, disabled?, tone?, onSelect? }` — default item
- `{ type: 'separator' }` — horizontal rule
- `{ type: 'label', label }` — non-interactive group heading

For sub-menus, checkbox items, or radio items, fork this file and use
bits-ui's `DropdownMenu.Sub`, `DropdownMenu.CheckboxItem`, etc. directly.

Requires `bits-ui` (`npm install bits-ui`).

CSS lives in `./dropdown-menu.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import {DropdownMenu as DM} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  export type MenuItem = {
    type?: 'item' | 'separator' | 'label';
    label?: string;
    shortcut?: string;
    disabled?: boolean;
    tone?: 'default' | 'danger';
    onSelect?: () => void;
  };

  type Side = 'top' | 'right' | 'bottom' | 'left';
  type Align = 'start' | 'center' | 'end';

  type TriggerProps = Record<string, unknown>;

  type Props = {
    items: MenuItem[];
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
    side?: Side;
    align?: Align;
    sideOffset?: number;
    class?: string;
    /**
     * Trigger snippet — receives bits-ui's wiring props you must spread
     * onto your own button element.
     *
     *   {#snippet trigger(props)}
     *     <Button {...props}>Actions</Button>
     *   {/snippet}
     */
    trigger: Snippet<[TriggerProps]>;
  };

  let {
    items,
    open = $bindable(false),
    onOpenChange,
    side = 'bottom',
    align = 'start',
    sideOffset = 6,
    class: className,
    trigger,
  }: Props = $props();
</script>

<DM.Root bind:open onOpenChange={(v: boolean) => onOpenChange?.(v)}>
  <DM.Trigger>
    {#snippet child({props})}
      {@render trigger(props as TriggerProps)}
    {/snippet}
  </DM.Trigger>
  <DM.Portal>
    <DM.Content class={cn('uin-menu', className)} {side} {align} {sideOffset}>
      {#each items as item, i (i)}
        {#if item.type === 'separator'}
          <DM.Separator class="uin-menu-sep" />
        {:else if item.type === 'label'}
          <div class="uin-menu-grouplabel">{item.label}</div>
        {:else}
          <DM.Item
            class={cn('uin-menu-item', item.tone === 'danger' && 'uin-menu-item-danger')}
            disabled={item.disabled}
            onSelect={() => item.onSelect?.()}
          >
            <span class="uin-menu-item-label">{item.label}</span>
            {#if item.shortcut}
              <span class="uin-menu-item-kbd">{item.shortcut}</span>
            {/if}
          </DM.Item>
        {/if}
      {/each}
    </DM.Content>
  </DM.Portal>
</DM.Root>
