export type Book = {
  id: string;
  title: string;
  authors: string;
  description: string;
  publisher: string;
  publishedDate: string;
  isbn10: string;
  isbn13: string;
  language: string;
  coverUrl: string;
  filePath: string;
  format: string;
  metadataSource: string;
  indexStatus: string;
  textStatus: string;
  passageCount: number;
};

export type Passage = {
  id: string;
  bookId: string;
  bookTitle: string;
  authors: string;
  label: string;
  chunkIndex: number;
  text: string;
  snippet: string;
};

export type TOCEntry = {
  label: string;
  chunkIndex: number;
  pages: number;
};

export type Stats = {
  books: number;
  passages: number;
  needsMetadata: number;
};

export type ImportSummary = {
  imported: number;
  updated: number;
  skipped: number;
  failed: number;
  messages: string[];
};

export type IndexerState = {
  running: boolean;
  current: string;
  pending: number;
  indexed: number;
  failed: number;
};

export type ImporterStatus = {
  running: boolean;
  discovering: boolean;
  path: string;
  startedAt?: string;
  finishedAt?: string;
  total: number;
  processed: number;
  imported: number;
  updated: number;
  skipped: number;
  failed: number;
  current: string;
  recentErrors: string[];
  done: boolean;
  cancelled: boolean;
  error?: string;
  summary?: ImportSummary;
  durationMs: number;
  enricherQueueDepth: number;
  indexer: IndexerState;
  queuedPaths?: string[];
};

export type ChatToolCall = {
  id: string;
  name: string;
  arguments: string;
  result?: string;
  error?: string;
};

export type ChatMessage = {
  role: 'system' | 'user' | 'assistant' | 'tool';
  content?: string;
  name?: string;
  toolCallId?: string;
  toolCalls?: ChatToolCall[];
};

export type ChatResponse = {
  reply: ChatMessage;
  history: ChatMessage[];
  toolCalls: ChatToolCall[];
  model: string;
  available: boolean;
  error?: string;
};

export type AggregateGroup = {
  name: string;
  count: number;
};

export type SearchResults = {
  query: string;
  books: Book[];
  passages: Passage[];
  authors: AggregateGroup[];
  subjects: AggregateGroup[];
};

export type LibraryView =
  | 'library'
  | 'recent'
  | 'categories'
  | 'byauthor'
  | 'bysubject'
  | 'unprocessed'
  | 'settings';

// Search-pane filters live in the App shell but the FilterPanel and
// SearchResults components both need to read/write them, so the shape is
// shared here.
export type ResultType = 'books' | 'passages' | 'authors' | 'subjects';
export type SortKey = 'relevance' | 'newest' | 'title' | 'author';
export type SearchFilters = {
  type: ResultType;
  sort: SortKey;
  formats: Set<string>;
  languages: Set<string>;
};

// Color of the import-status lamp in the sidebar / dialog header.
//   idle    — gray, nothing in flight
//   running — green, an import or indexer pass is active
//   warn    — yellow, finished with non-zero failures
//   fail    — red, errored or cancelled
export type ImportLamp = 'idle' | 'running' | 'warn' | 'fail';
