import type {Book, ImportSummary, ImporterStatus, Passage, Stats, TOCEntry} from './types';

export type Snapshot = {
  books: Book[];
  stats: Stats;
  dbPath: string;
  hydrating: boolean;
  mcp: {running: boolean; url: string; port: number};
};

const cover = (seed: string) =>
  `https://picsum.photos/seed/${seed}/300/440`;

const sampleBooks: Book[] = [
  {
    id: '1', title: 'The Annotated Turing',
    authors: 'Charles Petzold', description: "A guided tour through Turing's 1936 paper on computable numbers, with historical and mathematical commentary woven through every line. Petzold reproduces the original paper line by line and unpacks each step for a modern reader, building up the formal machinery without losing sight of the human story behind it.\n\nSubjects: Computability, Turing machines, History of computing, Mathematical logic, Halting problem\nGenre: Computer science\nEra: 20th century\nAudience: Developers, computer science students, history of science readers",
    publisher: 'Wiley', publishedDate: '2008-06-16',
    isbn10: '0470229055', isbn13: '9780470229057', language: 'en',
    coverUrl: cover('turing'),
    filePath: '/Users/you/Books/petzold-annotated-turing.epub',
    format: 'epub', metadataSource: 'openlibrary',
    indexStatus: 'indexed', textStatus: 'available', passageCount: 412,
  },
  {
    id: '2', title: 'A Pattern Language',
    authors: 'Christopher Alexander, Sara Ishikawa, Murray Silverstein',
    description: 'Two hundred fifty-three patterns that, taken together, form a language of design from regions and towns down to door-handles.',
    publisher: 'Oxford University Press', publishedDate: '1977',
    isbn10: '0195019199', isbn13: '9780195019193', language: 'en',
    coverUrl: cover('pattern'),
    filePath: '/Users/you/Books/pattern-language.pdf',
    format: 'pdf', metadataSource: 'googlebooks',
    indexStatus: 'indexed', textStatus: 'available', passageCount: 1203,
  },
  {
    id: '3', title: 'Sapiens',
    authors: 'Yuval Noah Harari', description: 'A brief history of humankind.',
    publisher: 'Harper', publishedDate: '2015',
    isbn10: '', isbn13: '9780062316097', language: 'en',
    coverUrl: cover('sapiens'),
    filePath: '/Users/you/Books/sapiens.epub',
    format: 'epub', metadataSource: 'openlibrary',
    indexStatus: 'indexed', textStatus: 'available', passageCount: 318,
  },
  {
    id: '4', title: 'On Writing Well',
    authors: 'William Zinsser', description: '',
    publisher: '', publishedDate: '', isbn10: '', isbn13: '', language: '',
    coverUrl: '',
    filePath: '/Users/you/Books/on-writing-well.epub',
    format: 'epub', metadataSource: '',
    indexStatus: 'indexed', textStatus: 'available', passageCount: 245,
  },
  {
    id: '5', title: 'Calvin and Hobbes — Tenth Anniversary',
    authors: 'Bill Watterson', description: '',
    publisher: '', publishedDate: '', isbn10: '', isbn13: '', language: '',
    coverUrl: '',
    filePath: '/Users/you/Books/calvin-and-hobbes.pdf',
    format: 'pdf', metadataSource: '',
    indexStatus: 'indexed', textStatus: 'text_unavailable', passageCount: 0,
  },
];

const samplePassages: Passage[] = [
  {
    id: '1-000007', bookId: '1', bookTitle: 'The Annotated Turing', authors: 'Charles Petzold',
    label: 'Chapter 2', chunkIndex: 6,
    text: 'It is possible to invent a single machine which can be used to compute any computable sequence...',
    snippet: 'a single machine which can be used to compute any <b>computable</b> sequence...',
  },
  {
    id: '1-000019', bookId: '1', bookTitle: 'The Annotated Turing', authors: 'Charles Petzold',
    label: 'Chapter 3', chunkIndex: 18,
    text: 'The behaviour of the computer at any moment is determined by the symbols which he is observing, and his "state of mind" at that moment.',
    snippet: 'determined by the symbols which he is <b>observing</b>...',
  },
];

export const mockSnapshot = (): Snapshot => ({
  books: sampleBooks,
  stats: {books: sampleBooks.length, passages: 2178, needsMetadata: 2},
  dbPath: '/Users/you/Library/Application Support/A3T Library/library.db',
  hydrating: false,
  mcp: {running: false, url: '', port: 8765},
});

export const mockPassages = (): Passage[] => samplePassages;

export const mockTOC = (): TOCEntry[] => [
  {label: 'Foreword', chunkIndex: 0, pages: 4},
  {label: 'Chapter 1 — On Computable Numbers', chunkIndex: 4, pages: 28},
  {label: 'Chapter 2 — The Universal Machine', chunkIndex: 32, pages: 41},
  {label: 'Chapter 3 — The Halting Problem', chunkIndex: 73, pages: 36},
  {label: 'Chapter 4 — Recursive Functions', chunkIndex: 109, pages: 52},
  {label: 'Chapter 5 — Beyond Computability', chunkIndex: 161, pages: 44},
  {label: 'Notes & Bibliography', chunkIndex: 205, pages: 12},
];

export const mockReaderPassages = (): Passage[] => {
  const out: Passage[] = [];
  const labels = mockTOC();
  for (const l of labels) {
    for (let k = 0; k < l.pages; k++) {
      out.push({
        id: `1-${String(l.chunkIndex + k + 1).padStart(6, '0')}`,
        bookId: '1',
        bookTitle: 'The Annotated Turing',
        authors: 'Charles Petzold',
        label: l.label,
        chunkIndex: l.chunkIndex + k,
        text:
          'It is possible to invent a single machine which can be used to compute any computable sequence. ' +
          'If this machine U is supplied with a tape on the beginning of which is written the standard description (S.D) of some computing machine M, ' +
          'then U will compute the same sequence as M. The behaviour of the computer at any moment is determined by the symbols which it is observing, ' +
          'and its "state of mind" at that moment. We may suppose that there is a bound B to the number of symbols or squares which the computer can observe at one moment. ' +
          'If it wishes to observe more, it must use successive observations. We will also suppose that the number of states of mind which need be taken into account is finite.',
        snippet: '',
      });
    }
  }
  return out;
};

export const mockImportSummary = (): ImportSummary => ({
  imported: 3, updated: 1, skipped: 0, failed: 0,
  messages: ['Imported 3 books from /Users/you/Downloads/library'],
});

export const mockRunningImport = (): ImporterStatus => ({
  running: true,
  discovering: false,
  path: '/Users/you/Documents/Library Imports/sample-books',
  startedAt: new Date(Date.now() - 142_000).toISOString(),
  total: 196608,
  processed: 12345,
  imported: 12300,
  updated: 30,
  skipped: 0,
  failed: 15,
  current: '/Users/.../sample-books/A/Annotated Example.epub.txt',
  recentErrors: [
    '/Users/.../sample-books/0_Other/.DS_Store: unsupported file type: ',
    '/Users/.../sample-books/B/Broken file.epub.txt: read error',
  ],
  done: false,
  cancelled: false,
  durationMs: 142_000,
  enricherQueueDepth: 47,
  indexer: {
    running: true,
    current: '/Users/.../sample-books/A/Architecture Notes.epub.txt',
    pending: 8420,
    indexed: 3925,
    failed: 4,
  },
});

export const mockDiscoveringImport = (): ImporterStatus => ({
  running: true,
  discovering: true,
  path: '/Users/you/Documents/Library Imports/sample-books',
  startedAt: new Date(Date.now() - 4_000).toISOString(),
  total: 18432,
  processed: 0,
  imported: 0, updated: 0, skipped: 0, failed: 0,
  current: '', recentErrors: [],
  done: false, cancelled: false,
  durationMs: 4_000,
  enricherQueueDepth: 0,
  indexer: {running: false, current: '', pending: 0, indexed: 0, failed: 0},
});

export const mockIdleImport = (): ImporterStatus => ({
  running: false,
  discovering: false,
  path: '',
  total: 0, processed: 0,
  imported: 0, updated: 0, skipped: 0, failed: 0,
  current: '', recentErrors: [],
  done: false, cancelled: false,
  durationMs: 0,
  enricherQueueDepth: 0,
  indexer: {running: false, current: '', pending: 0, indexed: 0, failed: 0},
});

export const isWailsAvailable = (): boolean => {
  return typeof window !== 'undefined'
    && typeof (window as any)._wails !== 'undefined';
};
