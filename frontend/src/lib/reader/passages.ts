import type {Passage, TOCEntry} from '../../types';

export type ReaderPassageGroup = {
  label: string;
  entry: TOCEntry;
  items: Passage[];
};

export function readerChapterId(chunkIndex: number): string {
  return `chap-${chunkIndex}`;
}

export function groupPassagesByLabel(passages: Passage[], toc: TOCEntry[]): ReaderPassageGroup[] {
  const out: ReaderPassageGroup[] = [];

  for (const passage of passages) {
    const label = passage.label || '';
    const last = out[out.length - 1];
    if (last && last.label === label) {
      last.items.push(passage);
      continue;
    }

    const tocMatch = toc.find((entry) => entry.label === label) ?? {
      label,
      chunkIndex: passage.chunkIndex,
      pages: 1,
    };
    out.push({label, entry: tocMatch, items: [passage]});
  }

  return out;
}
