// Titles watched in this browser, newest first, for the "continue watching" row.
// Per-title positions stay in shared.tsx (readProgress / writeProgress).
export type RecentWatch = { id: number; title: string; coverUrl?: string; episodeTitle: string; updatedAt: number };

const KEY = 'site-recent-watch';
const LIMIT = 20;

export function readRecent(): RecentWatch[] {
  try {
    const value = JSON.parse(localStorage.getItem(KEY) || '[]');
    return Array.isArray(value) ? value.filter((v): v is RecentWatch => Number.isInteger(v?.id) && typeof v?.title === 'string') : [];
  } catch {
    return [];
  }
}

export function recordRecent(entry: RecentWatch) {
  const list = [entry, ...readRecent().filter(v => v.id !== entry.id)].slice(0, LIMIT);
  try { localStorage.setItem(KEY, JSON.stringify(list)); } catch { /* storage may be disabled */ }
}

export function forgetRecent(id: number) {
  try { localStorage.setItem(KEY, JSON.stringify(readRecent().filter(v => v.id !== id))); } catch { /* storage may be disabled */ }
}
