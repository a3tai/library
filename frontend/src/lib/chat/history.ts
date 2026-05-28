import type {ChatMessage, ChatToolCall} from '../../types';

export type VisibleToolCall = {
  call: ChatToolCall;
  result: string;
};

export type VisibleChatItem = {
  message: ChatMessage;
  calls: VisibleToolCall[];
};

export function visibleChatItems(history: ChatMessage[]): VisibleChatItem[] {
  const visible = visibleMessages(history);
  return visible.map((message, index) => ({
    message,
    calls: message.role === 'assistant' ? callsForVisibleIndex(history, visible, index) : [],
  }));
}

function visibleMessages(history: ChatMessage[]): ChatMessage[] {
  return history.filter((message, index) => {
    if (message.role === 'tool') return false;
    if (message.role === 'assistant') {
      const hasContent = !!(message.content && message.content.trim());
      const hasCalls = !!(message.toolCalls && message.toolCalls.length > 0);
      if (!hasContent && !hasCalls) return false;
      if (!hasContent && hasCalls) {
        const laterReply = history
          .slice(index + 1)
          .some((next) => next.role === 'assistant' && next.content && next.content.trim());
        return !laterReply;
      }
    }
    return true;
  });
}

function callsForVisibleIndex(
  history: ChatMessage[],
  visible: ChatMessage[],
  visibleIndex: number
): VisibleToolCall[] {
  const visibleMessage = visible[visibleIndex];
  const rawIndex = history.findIndex((message) => message === visibleMessage);
  if (rawIndex < 0) return [];

  const out: VisibleToolCall[] = [];
  for (let index = 0; index <= rawIndex; index++) {
    const message = history[index];
    if (message.role !== 'assistant' || !message.toolCalls?.length) continue;
    const isCurrent = index === rawIndex;
    const isFoldable = !message.content || !message.content.trim();
    if (!isCurrent && !isFoldable) continue;
    for (const call of message.toolCalls) {
      out.push({
        call: {id: call.id, name: call.name, arguments: call.arguments},
        result: resultFor(history, call.id),
      });
    }
  }

  const seen = new Set<string>();
  return out.filter(({call}) => {
    if (seen.has(call.id)) return false;
    seen.add(call.id);
    return true;
  });
}

function resultFor(history: ChatMessage[], callId: string): string {
  const tool = history.find((message) => message.role === 'tool' && message.toolCallId === callId);
  return tool?.content ?? '';
}
