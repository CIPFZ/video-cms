import { failureKind, isAbort, isHlsSource, mimeType, resumePoint, shouldUseFallback } from './playback';

describe('playback decisions', () => {
  it('moves to the proxy only once, and only before the first frame', () => {
    expect(shouldUseFallback({ usingFallback: false, started: false, hasFallback: true })).toBe(true);
    expect(shouldUseFallback({ usingFallback: true, started: false, hasFallback: true })).toBe(false);
    expect(shouldUseFallback({ usingFallback: false, started: true, hasFallback: true })).toBe(false);
    expect(shouldUseFallback({ usingFallback: false, started: false, hasFallback: false })).toBe(false);
  });

  it('recognises HLS by type or address', () => {
    expect(isHlsSource({ src: 'https://cdn.example/a.mp4', type: 'm3u8' })).toBe(true);
    expect(isHlsSource({ src: 'https://cdn.example/index.m3u8?t=1' })).toBe(true);
    expect(isHlsSource({ src: 'https://cdn.example/a.mp4', type: 'mp4' })).toBe(false);
  });

  it('gives Otter a MIME hint only when the format is known', () => {
    expect(mimeType({ src: 'https://cdn.example/a', type: 'm3u8' })).toBe('application/vnd.apple.mpegurl');
    expect(mimeType({ src: 'https://cdn.example/a', type: 'mp4' })).toBe('video/mp4');
    expect(mimeType({ src: 'https://cdn.example/a' })).toBeUndefined();
  });

  it('tells a browser without a route from an address that failed', () => {
    expect(failureKind({ details: 'noRoute' })).toBe('unsupported');
    expect(failureKind({ details: 'engineUnsupported' })).toBe('unsupported');
    expect(failureKind({ details: 'manifestLoadError' })).toBe('blocked');
    expect(failureKind(new Error('boom'))).toBe('blocked');
    expect(isAbort(Object.assign(new Error('replaced'), { name: 'AbortError' }))).toBe(true);
    expect(isAbort({ details: 'loadAborted' })).toBe(false);
  });

  it('resumes only away from the very start and end', () => {
    expect(resumePoint(3, 600)).toBe(0);
    expect(resumePoint(120, 600)).toBe(120);
    expect(resumePoint(595, 600)).toBe(0);
    expect(resumePoint(120, NaN)).toBe(120);
  });
});
