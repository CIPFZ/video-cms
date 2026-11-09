// Playback decisions of PlayerHost, kept apart from the Otter player so they test without a browser.

export type PlayerSource = { src: string; fallbackSrc?: string; type?: string };
export type PlayerFailure = 'blocked' | 'unsupported';

/** CDNs usually allow the browser to fetch directly; only a failure before the
 *  first frame (CORS, hotlink rules) moves playback to the site's proxy. */
export function shouldUseFallback(attempt: { usingFallback: boolean; started: boolean; hasFallback: boolean }): boolean {
  return attempt.hasFallback && !attempt.usingFallback && !attempt.started;
}

export const isHlsSource = (source: PlayerSource) => {
  const type = (source.type || '').toLowerCase();
  return type === 'm3u8' || type === 'hls' || /\.m3u8(?:$|[?#])/i.test(source.src);
};

/** The MIME hint Otter routes by; without one it sniffs the address. */
export const mimeType = (source: PlayerSource) => {
  if (isHlsSource(source)) return 'application/vnd.apple.mpegurl';
  return (source.type || '').toLowerCase() === 'mp4' ? 'video/mp4' : undefined;
};

export function resumePoint(at: number | undefined, duration: number): number {
  if (!at || at <= 5) return 0;
  return Number.isFinite(duration) && at >= duration - 10 ? 0 : at;
}

// Otter error codes meaning this browser has no way to play the stream; anything else
// (playlist, segment, decode) may be the address and is worth the proxy.
const UNSUPPORTED = new Set(['noRoute', 'noFormat', 'engineUnsupported']);

export function failureKind(error: unknown): PlayerFailure {
  const details = (error as { details?: unknown } | null)?.details;
  return typeof details === 'string' && UNSUPPORTED.has(details) ? 'unsupported' : 'blocked';
}

/** A load replaced by a newer one or by teardown rejects with AbortError; not a failure. */
export const isAbort = (error: unknown) => error instanceof Error && error.name === 'AbortError';
