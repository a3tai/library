<!--
@component Accordion — vertical stack of collapsible items.

Built on bits-ui's Accordion primitive. Pass `items` as
`{value, title, content}[]` — `value` is the per-item key bound to
the open state. `type="single"` allows one item open at a time;
`type="multiple"` allows several.

For per-item rich content (snippets instead of strings), pass items
with a `content` snippet (typed via `ItemSnippet`).

Requires `bits-ui` (`npm install bits-ui`).

CSS lives in `./accordion.css`.
-->
<script lang="ts" generics="T extends string">
  import {Accordion as BA} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  type AccordionItem = {
    value: T;
    title: string;
    content: string;
    disabled?: boolean;
  };

  type AccordionType = 'single' | 'multiple';

  type Props = {
    items: AccordionItem[];
    type?: AccordionType;
    value?: T | T[];
    onValueChange?: (next: T | T[]) => void;
    class?: string;
  };

  let {
    items,
    type = 'single' as AccordionType,
    value = $bindable<T | T[]>(type === 'single' ? ('' as T) : []),
    onValueChange,
    class: className,
  }: Props = $props();
</script>

{#if type === 'single'}
  <BA.Root
    type="single"
    bind:value={value as T}
    onValueChange={(v: string) => onValueChange?.(v as T)}
    class={cn('uin-accordion', className)}
  >
    {#each items as item (item.value)}
      <BA.Item value={item.value} class="uin-accordion-item" disabled={item.disabled}>
        <BA.Header>
          <BA.Trigger class="uin-accordion-trigger">
            <span class="uin-accordion-title">{item.title}</span>
            <svg class="uin-accordion-chev" viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
              <polyline points="3,4.5 6,8 9,4.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </BA.Trigger>
        </BA.Header>
        <BA.Content class="uin-accordion-content">{item.content}</BA.Content>
      </BA.Item>
    {/each}
  </BA.Root>
{:else}
  <BA.Root
    type="multiple"
    bind:value={value as T[]}
    onValueChange={(v: string[]) => onValueChange?.(v as T[])}
    class={cn('uin-accordion', className)}
  >
    {#each items as item (item.value)}
      <BA.Item value={item.value} class="uin-accordion-item" disabled={item.disabled}>
        <BA.Header>
          <BA.Trigger class="uin-accordion-trigger">
            <span class="uin-accordion-title">{item.title}</span>
            <svg class="uin-accordion-chev" viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
              <polyline points="3,4.5 6,8 9,4.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </BA.Trigger>
        </BA.Header>
        <BA.Content class="uin-accordion-content">{item.content}</BA.Content>
      </BA.Item>
    {/each}
  </BA.Root>
{/if}
