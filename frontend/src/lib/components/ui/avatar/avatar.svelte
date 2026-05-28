<!--
@component Avatar — circular user image with initials fallback.

Renders an `<img>` when `src` is supplied; falls back to a circle
with `initials` (or the first letter of `name`) when the image fails
to load or no `src` is given.

Sizes: `xs` (20), `sm` (24), `md` (32, default), `lg` (40), `xl` (56).

CSS lives in `./avatar.css`.
-->
<script lang="ts">
  import type {HTMLAttributes} from 'svelte/elements';
  import {cn} from '../../../utils/cn';

  type Size = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

  type Props = HTMLAttributes<HTMLSpanElement> & {
    src?: string;
    alt?: string;
    name?: string;
    initials?: string;
    size?: Size;
    class?: string;
  };

  let {
    src,
    alt,
    name,
    initials,
    size = 'md',
    class: className,
    ...rest
  }: Props = $props();

  let imgFailed = $state(false);

  const fallback = $derived(
    initials ??
      (name ? name.trim().charAt(0).toUpperCase() : '')
  );
  const showImage = $derived(!!src && !imgFailed);
  const ariaLabel = $derived(alt ?? name ?? undefined);
</script>

<span
  class={cn('uin-avatar', `uin-avatar-${size}`, className)}
  role="img"
  aria-label={ariaLabel}
  {...rest}
>
  {#if showImage}
    <img
      class="uin-avatar-img"
      src={src}
      alt={alt ?? name ?? ''}
      onerror={() => (imgFailed = true)}
    />
  {:else}
    <span class="uin-avatar-fallback" aria-hidden="true">{fallback}</span>
  {/if}
</span>
