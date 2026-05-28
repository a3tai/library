<!--
  WindowChrome — drag-region title bar with centered search.

  Wails draws OS traffic lights on top of the window's left edge
  (MacTitleBarHiddenInset), so we leave a 76px-wide invisible spacer
  in the leading slot rather than rendering Mittsu's TrafficLights.
-->
<script lang="ts">
  import {RefreshCw} from '@lucide/svelte';
  import Button from '../lib/components/ui/button/button.svelte';
  import SearchInput from '../lib/components/ui/search-input/search-input.svelte';
  import Spinner from '../lib/components/ui/spinner/spinner.svelte';
  import TitleBar from '../lib/components/ui/title-bar/title-bar.svelte';

  type Props = {
    query: string;
    hydrating: boolean;
    onQueryChange: (next: string) => void;
    onQueryInput: () => void;
    onSearchFocus: () => void;
    onSearchEnter: () => void;
    onClear: () => void;
    onHydrate: () => void;
    onDoubleClick: (e: MouseEvent) => void;
    bindEl?: (el: HTMLInputElement | null) => void;
  };

  let {
    query,
    hydrating,
    onQueryChange,
    onQueryInput,
    onSearchFocus,
    onSearchEnter,
    onClear,
    onHydrate,
    onDoubleClick,
    bindEl,
  }: Props = $props();

  let searchValue = $state('');
  $effect(() => {
    searchValue = query;
  });

  let searchEl: HTMLInputElement | null = $state(null);
  $effect(() => {
    bindEl?.(searchEl);
  });
</script>

<TitleBar height="46px" class="chrome" ondblclick={onDoubleClick}>
  {#snippet leading()}
    <span class="chrome-traffic" aria-hidden="true"></span>
  {/snippet}
  <div class="chrome-search">
    <SearchInput
      bind:value={searchValue}
      bind:ref={searchEl}
      placeholder="Search · author:turing or subject:halting"
      aria-label="Search library"
      shortcutLabel="⌘K"
      class="chrome-search-input"
      oninput={() => {
        onQueryChange(searchValue);
        onQueryInput();
      }}
      onfocus={onSearchFocus}
      onkeydown={(e: KeyboardEvent) => {
        if (e.key === 'Enter') onSearchEnter();
      }}
      onclear={() => {
        searchValue = '';
        onQueryChange('');
        onClear();
      }}
    />
  </div>
  {#snippet trailing()}
    <Button
      variant="ghost"
      size="sm"
      icon
      onclick={onHydrate}
      disabled={hydrating}
      aria-label="Refresh metadata"
      title="Refresh metadata"
    >
      {#if hydrating}
        <Spinner size="sm" />
      {:else}
        <RefreshCw size={14} strokeWidth={1.6} />
      {/if}
    </Button>
  {/snippet}
</TitleBar>

<style>
  .chrome-traffic {
    display: inline-block;
    width: 76px;
    height: 28px;
  }
  .chrome-search {
    -webkit-app-region: no-drag;
    width: 100%;
    max-width: 720px;
    margin: 0 auto;
  }
  :global(.chrome-search .chrome-search-input) {
    height: 30px;
  }
</style>
