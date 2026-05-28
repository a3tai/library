<script lang="ts">
  import {fmtElapsed, renderMarkdown} from '../../lib/chat/format';
  import ChatToolCard from './ChatToolCard.svelte';
  import type {Book} from '../../types';
  import type {VisibleChatItem} from '../../lib/chat/history';

  type Props = {
    book: Book;
    historyLength: number;
    items: VisibleChatItem[];
    busy: boolean;
    streamingContent: string;
    toolBeats: number;
    elapsedMs: number;
    error: string;
    bindScroller: (element: HTMLDivElement | undefined) => void;
  };

  let {
    book,
    historyLength,
    items,
    busy,
    streamingContent,
    toolBeats,
    elapsedMs,
    error,
    bindScroller,
  }: Props = $props();

  let scrollerEl: HTMLDivElement | undefined = $state();

  $effect(() => {
    bindScroller(scrollerEl);
  });
</script>

<div class="scroller" bind:this={scrollerEl}>
  <div class="thread">
    {#if historyLength === 0}
      <div class="hint">
        <p>Ask anything about <strong>{book.title}</strong>. I'll search its passages and quote what I find.</p>
        <ul>
          <li>What is the central argument of chapter 2?</li>
          <li>Find every passage that mentions the halting problem.</li>
          <li>Summarize the foreword in three sentences.</li>
        </ul>
      </div>
    {/if}

    {#each items as item, index (index + ':' + item.message.role)}
      {@const message = item.message}
      {#if message.role === 'user'}
        <article class="msg user">
          <div class="bubble">{message.content}</div>
        </article>
      {:else if message.role === 'assistant'}
        <article class="msg assistant">
          {#if item.calls.length > 0}
            <div class="toolstrip">
              {#each item.calls as {call, result} (call.id)}
                <ChatToolCard {call} {result} />
              {/each}
            </div>
          {/if}
          {#if message.content && message.content.trim()}
            <div class="bubble md">
              {@html renderMarkdown(message.content)}
            </div>
          {/if}
        </article>
      {/if}
    {/each}

    {#if busy && streamingContent}
      <article class="msg assistant">
        <div class="bubble md streaming">
          {@html renderMarkdown(streamingContent)}
          <span class="caret" aria-hidden="true"></span>
        </div>
      </article>
    {:else if busy}
      <article class="msg assistant">
        <div class="thinking" aria-live="polite">
          <span class="dots"><span></span><span></span><span></span></span>
          <span>
            {#if toolBeats > 0}
              Running tools · round {toolBeats + 1}
            {:else}
              Thinking
            {/if}
            {#if fmtElapsed(elapsedMs)}
              <span class="elapsed">· {fmtElapsed(elapsedMs)}</span>
            {/if}
          </span>
        </div>
      </article>
    {/if}

    {#if error}
      <div class="error" role="alert">{error}</div>
    {/if}
  </div>
</div>

<style>
  .scroller {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .thread {
    max-width: 720px;
    margin: 0 auto;
    padding: var(--s-5) var(--s-4) var(--s-6);
    display: flex;
    flex-direction: column;
    gap: var(--s-4);
  }
  .hint {
    color: var(--fg-mute);
    border: 1px dashed var(--line);
    border-radius: var(--r-md);
    padding: var(--s-4);
    font-size: 13px;
  }
  .hint p {
    margin: 0 0 var(--s-2);
  }
  .hint ul {
    margin: 0;
    padding-left: var(--s-4);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .hint li {
    font-size: 12.5px;
  }
  .msg {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .msg.user {
    align-items: flex-end;
  }
  .msg.user .bubble {
    background: var(--accent);
    color: var(--accent-fg, white);
    border-radius: 14px 14px 4px 14px;
    max-width: 78%;
    padding: 8px 12px;
    line-height: 1.5;
    font-size: 13.5px;
  }
  .msg.assistant .bubble {
    border-radius: 14px 14px 14px 4px;
    background: var(--mat-row);
    border: 1px solid var(--line);
    padding: 10px 14px;
    line-height: 1.55;
    font-size: 13.5px;
    max-width: 78%;
  }
  .toolstrip {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-width: 78%;
  }
  .bubble.md :global(p) {
    margin: 0 0 0.6em;
  }
  .bubble.md :global(p:last-child) {
    margin-bottom: 0;
  }
  .bubble.md :global(ul),
  .bubble.md :global(ol) {
    margin: 0.2em 0 0.6em;
    padding-left: 1.4em;
  }
  .bubble.md :global(li) {
    margin: 0.1em 0;
  }
  .bubble.md :global(li:last-child) {
    margin-bottom: 0;
  }
  .bubble.md :global(h1),
  .bubble.md :global(h2),
  .bubble.md :global(h3) {
    margin: 0.6em 0 0.3em;
    font-weight: 600;
    line-height: 1.3;
  }
  .bubble.md :global(h1) {
    font-size: 16px;
  }
  .bubble.md :global(h2) {
    font-size: 14.5px;
  }
  .bubble.md :global(h3) {
    font-size: 13.5px;
  }
  .bubble.md :global(strong) {
    font-weight: 600;
  }
  .bubble.md :global(em) {
    font-style: italic;
  }
  .bubble.md :global(code) {
    font-family: var(--font-mono);
    font-size: 12px;
    background: color-mix(in srgb, var(--bg-panel) 70%, transparent);
    border: 1px solid var(--line);
    padding: 1px 5px;
    border-radius: 4px;
  }
  .bubble.md :global(pre) {
    margin: 0.4em 0;
    padding: 10px 12px;
    background: color-mix(in srgb, var(--bg-panel) 70%, transparent);
    border: 1px solid var(--line);
    border-radius: 8px;
    overflow-x: auto;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 1.45;
  }
  .bubble.md :global(pre code) {
    background: transparent;
    border: 0;
    padding: 0;
    border-radius: 0;
  }
  .bubble.md :global(blockquote) {
    margin: 0.4em 0;
    padding: 4px 12px;
    border-left: 3px solid var(--accent);
    color: var(--fg-mute);
  }
  .bubble.md :global(a) {
    color: var(--accent);
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .bubble.md :global(hr) {
    border: 0;
    border-top: 1px solid var(--line);
    margin: 0.8em 0;
  }
  .thinking {
    display: inline-flex;
    align-items: center;
    gap: var(--s-2);
    color: var(--fg-mute);
    font-size: 12px;
    padding: 4px 0;
  }
  .thinking .elapsed {
    margin-left: 4px;
    font-variant-numeric: tabular-nums;
    color: var(--fg-dim);
  }
  .dots {
    display: inline-flex;
    gap: 4px;
  }
  .dots span {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--fg-dim);
    animation: bounce 1s ease-in-out infinite both;
  }
  .dots span:nth-child(2) {
    animation-delay: 0.15s;
  }
  .dots span:nth-child(3) {
    animation-delay: 0.3s;
  }
  @keyframes bounce {
    0%,
    80%,
    100% {
      transform: scale(0.5);
      opacity: 0.4;
    }
    40% {
      transform: scale(1);
      opacity: 1;
    }
  }
  .bubble.streaming {
    position: relative;
  }
  .caret {
    display: inline-block;
    width: 2px;
    height: 1em;
    margin-left: 2px;
    vertical-align: -0.15em;
    background: var(--accent);
    animation: caret 1s steps(2) infinite;
  }
  @keyframes caret {
    50% {
      opacity: 0;
    }
  }
  .error {
    padding: var(--s-2) var(--s-3);
    border-radius: var(--r-sm);
    background: color-mix(in srgb, var(--warn) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--warn) 35%, transparent);
    color: var(--warn);
    font-size: 12px;
  }
</style>
