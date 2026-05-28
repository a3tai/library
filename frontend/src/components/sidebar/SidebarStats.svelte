<script lang="ts">
  import type {Stats} from '../../types';

  type Props = {
    stats: Stats;
  };

  let {stats}: Props = $props();
</script>

<div class="stats">
  <div class="stat">
    <span class="stat-label">Books</span>
    <span class="stat-value">{stats.books.toLocaleString()}</span>
  </div>
  <div class="stat-divider" aria-hidden="true"></div>
  <div class="stat">
    <span class="stat-label">Passages</span>
    <span class="stat-value">{stats.passages.toLocaleString()}</span>
  </div>
</div>

<style>
  .stats {
    display: grid;
    grid-template-columns: 1fr 1px 1fr;
    align-items: center;
    padding: var(--uin-s-2);
    border: 1px solid var(--uin-line);
    background: var(--uin-mat-row);
    border-radius: var(--uin-r-md);
  }
  .stat {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .stat-label {
    font-size: 10.5px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--uin-fg-dim);
  }
  .stat-value {
    font-family: var(--uin-font-display);
    font-weight: 500;
    font-size: 17px;
    font-variant-numeric: tabular-nums;
  }
  .stat-divider {
    height: 70%;
    background: var(--uin-line);
  }
</style>
