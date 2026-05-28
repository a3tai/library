import {marked} from 'marked';
import DOMPurify from 'dompurify';

marked.setOptions({gfm: true, breaks: true});

export function renderMarkdown(src: string): string {
  if (!src) return '';
  try {
    const raw = marked.parse(src, {async: false}) as string;
    if (typeof raw !== 'string') {
      return DOMPurify.sanitize(src);
    }
    return DOMPurify.sanitize(raw, {USE_PROFILES: {html: true}});
  } catch {
    return DOMPurify.sanitize(src);
  }
}

export function summarizeResult(text: string): string {
  if (!text) return 'no result';
  const t = text.trim();
  if (!t) return 'no result';
  if (t.startsWith('[')) {
    try {
      const arr = JSON.parse(t);
      if (Array.isArray(arr)) {
        return arr.length === 0 ? 'no matches' : `${arr.length} result${arr.length === 1 ? '' : 's'}`;
      }
    } catch {
      // fall through to head preview
    }
  }
  if (t.startsWith('{')) {
    try {
      JSON.parse(t);
      return 'object';
    } catch {
      // fall through
    }
  }
  return t.length > 80 ? t.slice(0, 80) + '…' : t;
}

export function prettyJSON(text: string): string {
  const t = (text ?? '').trim();
  if (!t) return '';
  if (t.startsWith('[') || t.startsWith('{')) {
    try {
      return JSON.stringify(JSON.parse(t), null, 2);
    } catch {
      // fall through
    }
  }
  return t;
}

export function summarizeArgs(args: string): string {
  const t = (args ?? '').trim();
  if (!t) return '';
  if (t.startsWith('{')) {
    try {
      const obj = JSON.parse(t);
      const parts: string[] = [];
      for (const [k, v] of Object.entries(obj)) {
        let s = typeof v === 'string' ? v : JSON.stringify(v);
        if (s.length > 32) s = s.slice(0, 32) + '…';
        parts.push(`${k}=${s}`);
        if (parts.length >= 3) break;
      }
      return parts.join(' · ');
    } catch {
      // fall through
    }
  }
  return t.length > 60 ? t.slice(0, 60) + '…' : t;
}

export function fmtElapsed(ms: number): string {
  if (ms < 1000) return '';
  const totalSec = Math.floor(ms / 1000);
  if (totalSec < 60) return `${totalSec}s`;
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}m ${s.toString().padStart(2, '0')}s`;
}
