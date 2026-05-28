<script lang="ts">
  import type {SettingsDTO, StatsDTO} from '../../lib/api/library';

  type Props = {
    current: SettingsDTO | null;
    stats: StatsDTO | null;
  };

  let {current, stats}: Props = $props();
</script>

<div class="panel">
  <div class="readonly">
    <div class="readonly-row">
      <span class="readonly-label">Database path</span>
      <code class="readonly-value">{current?.dbPath ?? ''}</code>
    </div>
    <div class="readonly-row">
      <span class="readonly-label">Books</span>
      <code class="readonly-value">{stats?.books?.toLocaleString() ?? '-'}</code>
    </div>
    <div class="readonly-row">
      <span class="readonly-label">Passages</span>
      <code class="readonly-value">{stats?.passages?.toLocaleString() ?? '-'}</code>
    </div>
    <div class="readonly-row">
      <span class="readonly-label">Awaiting metadata</span>
      <code class="readonly-value">{stats?.needsMetadata?.toLocaleString() ?? '-'}</code>
    </div>
  </div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-5);
    max-width: 640px;
  }
  .readonly {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-2);
  }
  .readonly-row {
    display: flex;
    align-items: baseline;
    gap: var(--uin-s-3);
    padding: var(--uin-s-2) 0;
    border-bottom: 1px solid var(--uin-line);
  }
  .readonly-label {
    flex: 0 0 160px;
    font-size: 12px;
    color: var(--uin-fg-mute);
  }
  .readonly-value {
    font-family: var(--uin-font-mono);
    font-size: 12px;
    color: var(--uin-fg);
    word-break: break-all;
  }
</style>
