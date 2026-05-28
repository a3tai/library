<!--
@component Inspector — right-side property panel. 🍎

A macOS-style sidebar with a header and stacked sections of
labeled rows. Each section has a title and a list of `{label, value}`
entries — typical of the "Properties" panel in macOS apps (Books,
Photos, Mail).

`sections` is `[{title, rows: [{label, value}]}]` for the simple shape;
pass `children` instead for fully custom content.

CSS lives in `./inspector.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Row = {label: string; value: string};
  type Section = {title: string; rows: Row[]};

  type Props = HTMLAttributes<HTMLElement> & {
    title?: string;
    sections?: Section[];
    width?: string;
    class?: string;
    children?: Snippet;
  };

  let {
    title,
    sections,
    width = '280px',
    class: className,
    children,
    ...rest
  }: Props = $props();
</script>

<aside
  class={cn('uin-inspector', className)}
  style="--uin-inspector-w: {width};"
  aria-label={title ?? 'Inspector'}
  {...rest}
>
  {#if title}
    <header class="uin-inspector-head">
      <p class="uin-inspector-title">{title}</p>
    </header>
  {/if}
  <div class="uin-inspector-body">
    {#if sections}
      {#each sections as sec}
        <section class="uin-inspector-section">
          <p class="uin-inspector-section-title">{sec.title}</p>
          <dl class="uin-inspector-rows">
            {#each sec.rows as row}
              <div class="uin-inspector-row">
                <dt>{row.label}</dt>
                <dd>{row.value}</dd>
              </div>
            {/each}
          </dl>
        </section>
      {/each}
    {/if}
    {#if children}{@render children()}{/if}
  </div>
</aside>
