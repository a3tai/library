import type {ImportLamp, ImporterStatus} from '../../types';

export function emptyImporterStatus(): ImporterStatus {
  return {
    running: false,
    discovering: false,
    path: '',
    total: 0,
    processed: 0,
    imported: 0,
    updated: 0,
    skipped: 0,
    failed: 0,
    current: '',
    recentErrors: [],
    done: false,
    cancelled: false,
    durationMs: 0,
    enricherQueueDepth: 0,
    indexer: {running: false, current: '', pending: 0, indexed: 0, failed: 0},
  };
}

export function importLampFor(importer: ImporterStatus): ImportLamp {
  if (importer.running) return 'running';
  if (importer.error || importer.cancelled) return 'fail';
  if (importer.done && (importer.summary?.failed ?? importer.failed) > 0) return 'warn';
  return 'idle';
}

export function importLabelFor(importer: ImporterStatus): string {
  if (importer.discovering) {
    return `Discovering · ${importer.total.toLocaleString()} files`;
  }
  if (importer.running) {
    return importer.total > 0 ? `Importing · ${Math.round((importer.processed / importer.total) * 100)}%` : 'Importing';
  }
  if (importer.error) return 'Error';
  if (importer.cancelled) return 'Cancelled';
  if (!importer.done) return 'Idle';

  if ((importer.summary?.failed ?? 0) > 0) {
    return `${importer.summary?.imported ?? 0} imported · ${importer.summary?.failed} failed`;
  }
  return `${importer.summary?.imported ?? 0} imported`;
}
