import type {Book, SearchFilters, SearchResults, SortKey} from '../../types';

export function defaultSearchFilters(): SearchFilters {
  return {
    type: 'books',
    sort: 'relevance',
    formats: new Set<string>(),
    languages: new Set<string>(),
  };
}

export function setStringFilter(
  filters: SearchFilters,
  key: 'formats' | 'languages',
  value: string
): SearchFilters {
  const next = new Set(filters[key]);
  if (next.has(value)) next.delete(value);
  else next.add(value);
  return {...filters, [key]: next};
}

export function setSort(filters: SearchFilters, sort: SortKey): SearchFilters {
  return {...filters, sort};
}

export function filterBooks(books: Book[], filters: SearchFilters): Book[] {
  let out = [...books];
  if (filters.formats.size > 0) {
    out = out.filter((b) => filters.formats.has((b.format || '').toLowerCase()));
  }
  if (filters.languages.size > 0) {
    out = out.filter((b) => filters.languages.has((b.language || '').toLowerCase()));
  }
  if (filters.sort === 'title') {
    out.sort((a, b) => a.title.localeCompare(b.title));
  } else if (filters.sort === 'author') {
    out.sort((a, b) => (a.authors || '').localeCompare(b.authors || ''));
  }
  return out;
}

export function bookFormatBuckets(books: Book[]): [string, number][] {
  return bucketBooks(books, (book) => (book.format || '').toLowerCase());
}

export function bookLanguageBuckets(books: Book[]): [string, number][] {
  return bucketBooks(books, (book) => (book.language || '').toLowerCase());
}

export function hasAnyResults(results: SearchResults): boolean {
  return (
    results.books.length +
      results.passages.length +
      results.authors.length +
      results.subjects.length >
    0
  );
}

function bucketBooks(books: Book[], select: (book: Book) => string): [string, number][] {
  const buckets = new Map<string, number>();
  for (const book of books) {
    const value = select(book);
    if (!value) continue;
    buckets.set(value, (buckets.get(value) ?? 0) + 1);
  }
  return Array.from(buckets.entries()).sort((a, b) => b[1] - a[1]);
}
