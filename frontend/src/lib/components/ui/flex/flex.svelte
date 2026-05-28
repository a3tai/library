<!--
@component Flex — generic flex container with all axes exposed.

When `Stack` / `HStack` / `VStack` aren't expressive enough — usually
because you need a non-default `direction`, `wrap`, or `align` —
reach for `Flex`. Same gap/align/justify props as the others, plus
explicit `direction` and `wrap`.

Default `direction='row'` (Stack and VStack already cover column).
Default `gap=2`, `align='start'`, `justify='start'`, `wrap='nowrap'`.

CSS lives in `./flex.css`.
-->
<script lang="ts">
  import type {Snippet} from 'svelte';
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Direction = 'row' | 'column' | 'row-reverse' | 'column-reverse';
  type Gap = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;
  type Align = 'start' | 'center' | 'end' | 'stretch' | 'baseline';
  type Justify = 'start' | 'center' | 'end' | 'between' | 'around';
  type Wrap = 'nowrap' | 'wrap' | 'wrap-reverse';

  type Props = HTMLAttributes<HTMLDivElement> & {
    direction?: Direction;
    gap?: Gap;
    align?: Align;
    justify?: Justify;
    wrap?: Wrap;
    class?: string;
    children?: Snippet;
  };

  let {
    direction = 'row',
    gap = 2,
    align = 'start',
    justify = 'start',
    wrap = 'nowrap',
    class: className,
    children,
    ...rest
  }: Props = $props();
</script>

<div
  class={cn(
    'uin-flex',
    `uin-flex-dir-${direction}`,
    `uin-flex-gap-${gap}`,
    `uin-flex-align-${align}`,
    `uin-flex-justify-${justify}`,
    `uin-flex-wrap-${wrap}`,
    className
  )}
  {...rest}
>
  {#if children}{@render children()}{/if}
</div>
