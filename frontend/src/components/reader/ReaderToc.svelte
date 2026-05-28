<script lang="ts">
  import type {TOCEntry} from '../../types';

  type Props = {
    toc: TOCEntry[];
    currentIndex: number;
    onJump: (chunkIndex: number) => void;
  };

  let {toc, currentIndex, onJump}: Props = $props();
</script>

<aside class="toc" aria-label="Contents">
  <div class="toc-head">Contents</div>
  <ol>
    {#each toc as entry, i}
      <li>
        <button
          type="button"
          class:current={i === currentIndex}
          onclick={() => onJump(entry.chunkIndex)}
        >
          <span class="num">{i + 1}</span>
          <span class="label">{entry.label}</span>
          <span class="pages">{entry.pages}</span>
        </button>
      </li>
    {/each}
  </ol>
</aside>

<style>
  .toc {
    width: 280px;
    border-right: 1px solid var(--reader-line);
    background: color-mix(in srgb, var(--reader-bg) 92%, transparent);
    overflow: auto;
    padding: var(--s-4) 0;
    display: flex;
    flex-direction: column;
    -webkit-app-region: no-drag;
  }
  .toc-head {
    padding: 0 var(--s-4) var(--s-2);
    font-size: 10.5px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--reader-mute);
    font-weight: 500;
  }
  .toc ol {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
  }
  .toc li button {
    width: 100%;
    display: grid;
    grid-template-columns: 28px 1fr auto;
    gap: var(--s-2);
    align-items: baseline;
    padding: 6px var(--s-4);
    border: 0;
    background: transparent;
    color: var(--reader-fg);
    text-align: left;
    cursor: default;
    border-left: 2px solid transparent;
    transition: background-color var(--dur-1) var(--ease-standard);
  }
  .toc li button:hover {
    background: color-mix(in srgb, var(--reader-fg) 6%, transparent);
  }
  .toc li button.current {
    border-left-color: var(--reader-accent);
    background: color-mix(in srgb, var(--reader-accent) 10%, transparent);
  }
  .toc .num,
  .toc .pages {
    color: var(--reader-mute);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
  }
  .toc .label {
    font-size: 12.5px;
    line-height: 1.3;
  }

  @media (max-width: 760px) {
    .toc {
      width: 100%;
      position: absolute;
      inset: 49px 0 0 0;
      z-index: 5;
    }
  }
</style>
