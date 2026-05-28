<script lang="ts">
  import {LibraryService} from '../lib/api/library';
  import {Events} from '@wailsio/runtime';
  import ChatHeader from './chat/ChatHeader.svelte';
  import ChatThread from './chat/ChatThread.svelte';
  import {visibleChatItems} from '../lib/chat/history';
  import type {Book, ChatMessage, ChatResponse, ChatToolCall} from '../types';

  type Props = {
    book: Book;
    demoMode?: boolean;
    onclose: () => void;
  };

  let {book, demoMode = false, onclose}: Props = $props();

  let history = $state<ChatMessage[]>([]);
  let toolTrace = $state<ChatToolCall[]>([]);
  let draft = $state('');
  let busy = $state(false);
  let model = $state('');
  let available = $state(true);
  let error = $state('');
  // streamingContent is the in-flight assistant response while a turn is
  // streaming. Cleared between rounds and on turn end (the synchronous
  // ChatTurn return is the source of truth for the final history).
  let streamingContent = $state('');
  let activeTurnId: string | null = null;

  // Elapsed-time tracker so a slow agent loop doesn't feel like a frozen
  // chat. Updates every ~500ms while busy; cleared on turn end.
  let turnStartedAt = $state<number | null>(null);
  let elapsedMs = $state(0);
  // Latest tool that fired this turn (rough — backend doesn't yet emit a
  // per-tool event, but chat:turn:round implies "a tool batch just ran").
  let toolBeats = $state(0);
  let scrollerEl: HTMLDivElement | undefined = $state();
  let composerEl: HTMLTextAreaElement | undefined = $state();
  let rootEl: HTMLDivElement | undefined = $state();

  // Debug logging — controlled by LIBRARY_CHAT_DEBUG on the backend.
  // console to silence. Default on so the user gets useful trace data
  // while we stabilise the chat pipeline.
  const CHAT_DEBUG = true;
  function dbg(...args: unknown[]) {
    if (!CHAT_DEBUG) return;
    // eslint-disable-next-line no-console
    console.debug('[chat]', ...args);
  }

  const visibleItems = $derived(visibleChatItems(history));

  // Elapsed-time ticker: while a turn is in flight, refresh elapsedMs
  // every 500ms so the UI shows a live counter alongside "Thinking…".
  $effect(() => {
    if (!busy || turnStartedAt === null) return;
    const interval = window.setInterval(() => {
      elapsedMs = Date.now() - (turnStartedAt as number);
    }, 500);
    return () => window.clearInterval(interval);
  });

  // Trace history + visibleHistory mutations so the dev console shows
  // exactly when a turn lands and what shape it has.
  $effect(() => {
    dbg('history mutated', {
      total: history.length,
      shape: history.map((m) => ({
        role: m.role,
        contentLen: (m.content ?? '').length,
        calls: m.toolCalls?.length ?? 0,
      })),
    });
  });
  $effect(() => {
    dbg('visibleItems recomputed', {
      count: visibleItems.length,
      roles: visibleItems.map((item) => item.message.role),
    });
  });

  $effect(() => {
    composerEl?.focus();
    void probe();
    dbg('effect: subscribing to chat events');
    // Subscribe once for this Chat instance; tear down on unmount so a
    // re-opened chat doesn't leak handlers.
    const offDelta = Events.On('chat:turn:delta', (e) => {
      const data = e.data as {turnId: string; content: string} | undefined;
      if (!data) {
        dbg('delta: ignored (no data)');
        return;
      }
      if (data.turnId !== activeTurnId) {
        dbg('delta: turnId mismatch', {got: data.turnId, active: activeTurnId});
        return;
      }
      streamingContent += data.content;
      dbg('delta', {len: data.content.length, total: streamingContent.length});
      queueMicrotask(scrollToBottom);
    });
    const offRound = Events.On('chat:turn:round', (e) => {
      const data = e.data as {turnId: string} | undefined;
      if (!data || data.turnId !== activeTurnId) {
        dbg('round: ignored', {data, active: activeTurnId});
        return;
      }
      dbg('round: clearing streaming buffer (had len)', streamingContent.length);
      // Round closed (model emitted tool_calls). Clear the in-flight bubble;
      // the next round's content will start fresh, and the synchronous
      // return will bring the canonical history with tool messages slotted.
      streamingContent = '';
      toolBeats++;
    });
    const offEnd = Events.On('chat:turn:end', (e) => {
      const data = e.data as {turnId: string} | undefined;
      if (!data || data.turnId !== activeTurnId) {
        dbg('end: ignored', {data, active: activeTurnId});
        return;
      }
      dbg('end: turn complete', data.turnId);
      streamingContent = '';
      activeTurnId = null;
    });
    return () => {
      dbg('effect: cleanup, unsubscribing');
      offDelta();
      offRound();
      offEnd();
    };
  });

  async function probe() {
    if (demoMode) {
      available = true;
      model = 'demo-model (preview)';
      return;
    }
    try {
      available = (await LibraryService.ChatAvailable()) as boolean;
      if (!available) error = `LM Studio isn't reachable. Make sure the local server is running.`;
    } catch (e) {
      available = false;
      error = e instanceof Error ? e.message : String(e);
    }
  }

  async function send() {
    const message = draft.trim();
    if (!message || busy) return;
    draft = '';
    busy = true;
    error = '';
    history = [...history, {role: 'user', content: message}];
    queueMicrotask(scrollToBottom);

    if (demoMode) {
      await new Promise((r) => setTimeout(r, 300));
      const trace: ChatToolCall = {
        id: 'demo-1',
        name: 'search_passages',
        arguments: JSON.stringify({query: message, book_id: book.id, limit: 3}),
        result: 'Sample tool result: 3 matching passages found.',
      };
      toolTrace = [...toolTrace, trace];
      history = [
        ...history,
        {
          role: 'assistant',
          content: `(preview reply) Drawing on three passages from **"${book.title}"**, here's what stands out:\n\n- Point one\n- Point two\n- Point three`,
          toolCalls: [{id: trace.id, name: trace.name, arguments: trace.arguments}],
        },
        {role: 'tool', toolCallId: trace.id, name: trace.name, content: trace.result},
      ];
      busy = false;
      queueMicrotask(scrollToBottom);
      return;
    }

    const turnId =
      typeof crypto !== 'undefined' && 'randomUUID' in crypto
        ? crypto.randomUUID()
        : `t-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    activeTurnId = turnId;
    streamingContent = '';
    turnStartedAt = Date.now();
    elapsedMs = 0;
    toolBeats = 0;
    dbg('send: starting turn', {turnId, message: message.slice(0, 80), historyLen: history.length});

    try {
      const resp = (await LibraryService.ChatTurn({
        bookId: book.id,
        message,
        history: history.slice(0, -1), // user already appended locally; backend adds it
        turnId,
      })) as ChatResponse;
      dbg('send: ChatTurn resolved', {
        replyRole: resp.reply?.role,
        replyContentLen: resp.reply?.content?.length ?? 0,
        historyLen: resp.history?.length ?? 0,
        toolCallsCount: resp.toolCalls?.length ?? 0,
        model: resp.model,
        error: resp.error,
      });
      if (resp.error) {
        error = resp.error;
      }
      if (resp.history) history = resp.history;
      if (resp.toolCalls?.length) toolTrace = [...toolTrace, ...resp.toolCalls];
      if (resp.model) model = resp.model;
    } catch (e) {
      dbg('send: ChatTurn threw', e);
      error = e instanceof Error ? e.message : String(e);
    } finally {
      dbg('send: finally — clearing buffers');
      activeTurnId = null;
      streamingContent = '';
      busy = false;
      turnStartedAt = null;
      queueMicrotask(scrollToBottom);
    }
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onclose();
      return;
    }
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      send();
    }
  }

  function scrollToBottom() {
    if (!scrollerEl) return;
    scrollerEl.scrollTop = scrollerEl.scrollHeight;
  }

  function toggleMaximise() {
    LibraryService.ToggleMaximise().catch(() => {});
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="chat" bind:this={rootEl}>
  <ChatHeader {book} {available} {model} {onclose} onDoubleClick={toggleMaximise} />

  <ChatThread
    {book}
    historyLength={history.length}
    items={visibleItems}
    {busy}
    {streamingContent}
    {toolBeats}
    {elapsedMs}
    {error}
    bindScroller={(element) => (scrollerEl = element)}
  />

  <footer class="composer">
    <textarea
      bind:this={composerEl}
      bind:value={draft}
      placeholder={available ? 'Ask the book…  (⌘↵ to send)' : 'Start LM Studio to begin chatting'}
      rows="2"
      disabled={busy || !available}
    ></textarea>
    <button class="send" type="button" onclick={send} disabled={busy || !available || !draft.trim()}>
      {busy ? 'Thinking…' : 'Send'}
    </button>
  </footer>
</div>

<style>
  .chat {
    position: fixed;
    inset: 0;
    z-index: 200;
    display: flex;
    flex-direction: column;
    background: var(--bg-base);
    color: var(--fg);
  }

  .composer {
    flex-shrink: 0;
    border-top: 1px solid var(--line);
    background: var(--bg-base);
    padding: var(--s-3) var(--s-4);
    display: grid;
    grid-template-columns: 1fr auto;
    gap: var(--s-2);
    align-items: end;
    max-width: 820px;
    width: 100%;
    margin: 0 auto;
    box-sizing: border-box;
  }
  textarea {
    width: 100%;
    resize: none;
    background: var(--mat-panel);
    border: 1px solid var(--line);
    border-radius: var(--r-md);
    color: var(--fg);
    font: inherit;
    padding: 8px 10px;
    line-height: 1.4;
    font-size: 13.5px;
  }
  textarea:focus-visible { outline: none; box-shadow: var(--focus-ring); border-color: transparent; }
  .send {
    border: 0;
    border-radius: var(--r-md);
    background: var(--accent);
    color: var(--accent-fg, white);
    padding: 9px 16px;
    font-weight: 500;
    cursor: default;
  }
  .send:disabled { opacity: 0.4; }
  .send:hover:not(:disabled) { filter: brightness(1.06); }
  .send:focus-visible { outline: none; box-shadow: var(--focus-ring); }
</style>
