import { isHlsSource, resumePoint, shouldUseFallback } from './PlayerHost';

describe('PlayerHost decisions', () => {
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

  it('resumes only away from the very start and end', () => {
    expect(resumePoint(3, 600)).toBe(0);
    expect(resumePoint(120, 600)).toBe(120);
    expect(resumePoint(595, 600)).toBe(0);
    expect(resumePoint(120, NaN)).toBe(120);
  });
});
