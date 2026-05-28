<!--
@component AlertDialog — confirmation modal for destructive actions.

Like `Dialog`, but built on bits-ui's `AlertDialog` primitive — same
visual chrome but `role="alertdialog"` and the dismiss-on-overlay-click
behavior is disabled (consumers must explicitly choose Cancel).

API: `title`, `description`, `confirmLabel` (default "Confirm"),
`cancelLabel` (default "Cancel"), `tone` (`'default'` or `'danger'` —
the latter colors the confirm button red). `onConfirm` and `onCancel`
callbacks fire with no args.

Requires `bits-ui` (`npm install bits-ui`).

CSS lives in `./alert-dialog.css` plus the shared `./dialog.css` if
co-located; this component reuses `.uin-dialog-*` classes for chrome.
-->
<script lang="ts">
  import {AlertDialog as AD} from 'bits-ui';
  import {cn} from '../../../utils/cn';

  type Tone = 'default' | 'danger';

  type Props = {
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
    title: string;
    description?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    tone?: Tone;
    onConfirm?: () => void;
    onCancel?: () => void;
    class?: string;
  };

  let {
    open = $bindable(false),
    onOpenChange,
    title,
    description,
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    tone = 'default',
    onConfirm,
    onCancel,
    class: className,
  }: Props = $props();
</script>

<AD.Root bind:open onOpenChange={(v: boolean) => onOpenChange?.(v)}>
  <AD.Portal>
    <AD.Overlay class="uin-dialog-overlay" />
    <AD.Content class={cn('uin-dialog', 'uin-dialog-sm', 'uin-alert-dialog', className)}>
      <header class="uin-dialog-head">
        <AD.Title class="uin-dialog-title">{title}</AD.Title>
        {#if description}
          <AD.Description class="uin-dialog-desc">{description}</AD.Description>
        {/if}
      </header>
      <footer class="uin-dialog-foot uin-alert-dialog-foot">
        <AD.Cancel
          class="uin-btn uin-btn-ghost uin-btn-md"
          onclick={() => onCancel?.()}
        >
          {cancelLabel}
        </AD.Cancel>
        <AD.Action
          class={cn(
            'uin-btn',
            tone === 'danger' ? 'uin-alert-dialog-confirm-danger' : 'uin-btn-primary',
            'uin-btn-md'
          )}
          onclick={() => onConfirm?.()}
        >
          {confirmLabel}
        </AD.Action>
      </footer>
    </AD.Content>
  </AD.Portal>
</AD.Root>
