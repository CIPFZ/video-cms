import React, { useEffect, useRef } from 'react';
import Hls from 'hls.js';

// The site's only contact with the video engine. Phase 1 plays through hls.js;
// once @otter-player is on npm this file becomes <OtterPlayer create={createVodPlayer}>
// with load(src, { startTime, autoplay }) and the store's time / ended state.

export type PlayerSource = { src: string; fallbackSrc?: string; type?: string };
export type PlayerFailure = 'blocked' | 'unsupported';
export type PlayerHostProps = {
  source?: PlayerSource;
  /** Seconds to resume from; ignored near the start or the end. */
  startTime?: number;
  /** Changing it reloads the same source from the direct address. */
  reloadKey?: number;
  onReady?: (resumedAt: number) => void;
  onFailure?: (failure: PlayerFailure) => void;
  onTime?: (seconds: number) => void;
  onPlayingChange?: (playing: boolean) => void;
  onEnded?: () => void;
};

/** CDNs usually allow the browser to fetch directly; only a failure before the
 *  first frame (CORS, hotlink rules) moves playback to the site's proxy. */
export function shouldUseFallback(attempt: { usingFallback: boolean; started: boolean; hasFallback: boolean }): boolean {
  return attempt.hasFallback && !attempt.usingFallback && !attempt.started;
}

export const isHlsSource = (source: PlayerSource) => {
  const type = (source.type || '').toLowerCase();
  return type === 'm3u8' || type === 'hls' || /\.m3u8(?:$|[?#])/i.test(source.src);
};

export function resumePoint(at: number | undefined, duration: number): number {
  if (!at || at <= 5) return 0;
  return Number.isFinite(duration) && at >= duration - 10 ? 0 : at;
}

export default function PlayerHost({ source, startTime, reloadKey, onReady, onFailure, onTime, onPlayingChange, onEnded }: PlayerHostProps) {
  const video = useRef<HTMLVideoElement>(null);
  // Callbacks change every render; the effect reads the latest through a ref.
  const handlers = useRef({ onReady, onFailure, onTime, onPlayingChange, onEnded, startTime });
  handlers.current = { onReady, onFailure, onTime, onPlayingChange, onEnded, startTime };

  useEffect(() => {
    const element = video.current;
    if (!element || !source?.src) return undefined;
    let disposed = false;
    let hls: Hls | undefined;
    let started = false;
    let usingFallback = false;
    const hlsSource = isHlsSource(source);

    const detach = () => {
      hls?.destroy();
      hls = undefined;
      element.removeAttribute('src');
      element.load();
    };
    const attach = (url: string) => {
      detach();
      if (hlsSource && Hls.isSupported()) {
        hls = new Hls({ enableWorker: true, lowLatencyMode: false });
        hls.on(Hls.Events.ERROR, (_event, data) => { if (data.fatal) failed(); });
        hls.loadSource(url);
        hls.attachMedia(element);
      } else if (!hlsSource || element.canPlayType('application/vnd.apple.mpegurl')) {
        element.src = url;
        element.load();
      } else {
        handlers.current.onFailure?.('unsupported');
      }
    };
    const { fallbackSrc } = source;
    const failed = () => {
      if (disposed) return;
      if (fallbackSrc && shouldUseFallback({ usingFallback, started, hasFallback: true })) {
        usingFallback = true;
        attach(fallbackSrc);
        return;
      }
      handlers.current.onFailure?.('blocked');
    };
    const onMeta = () => {
      if (disposed) return;
      const at = resumePoint(handlers.current.startTime, element.duration);
      if (at) element.currentTime = at;
      handlers.current.onReady?.(at);
      void element.play().catch(() => undefined);
    };
    const onData = () => { started = true; };
    const onTimeUpdate = () => handlers.current.onTime?.(element.currentTime);
    const onPlay = () => handlers.current.onPlayingChange?.(true);
    const onPause = () => handlers.current.onPlayingChange?.(false);
    const onEnd = () => { handlers.current.onPlayingChange?.(false); handlers.current.onEnded?.(); };
    // hls.js reports its own errors; the element's error covers native playback.
    const onError = () => { if (!hls) failed(); };
    const events: [string, () => void][] = [['loadedmetadata', onMeta], ['loadeddata', onData], ['timeupdate', onTimeUpdate], ['play', onPlay], ['pause', onPause], ['ended', onEnd], ['error', onError]];
    for (const [name, fn] of events) element.addEventListener(name, fn);
    attach(source.src);
    return () => {
      disposed = true;
      for (const [name, fn] of events) element.removeEventListener(name, fn);
      element.pause();
      detach();
    };
  }, [source?.src, source?.fallbackSrc, source?.type, reloadKey]);

  // biome-ignore lint/a11y/useMediaCaption: upstream streams carry no caption tracks to attach.
  return <video ref={video} controls playsInline preload="metadata" />;
}
