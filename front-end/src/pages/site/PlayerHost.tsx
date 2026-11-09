import React, { useRef } from 'react';
import { OtterPlayer } from '@otter-player/react';
import { createVodPlayer } from '@otter-player/presets/vod';
import type { Player } from '@otter-player/presets';
import '@otter-player/ui-skin-default';
import { currentLocale } from '@/i18n';
import { failureKind, isAbort, isHlsSource, mimeType, resumePoint, shouldUseFallback, type PlayerFailure, type PlayerSource } from './playback';

// The site's only contact with the video engine: Otter's VOD preset with its default skin.

export type { PlayerFailure, PlayerSource } from './playback';
export type PlayerHostProps = {
  source?: PlayerSource;
  /** Shown in the player's top bar and the Media Session. */
  title?: string;
  /** Seconds to resume from; ignored near the start or the end. */
  startTime?: number;
  /** Changing it reloads the same source from the direct address. */
  reloadKey?: number;
  onReady?: (resumedAt: number) => void;
  onFailure?: (failure: PlayerFailure) => void;
  onTime?: (seconds: number, duration: number) => void;
  onPlayingChange?: (playing: boolean) => void;
  onEnded?: () => void;
};
type Handlers = Omit<PlayerHostProps, 'source' | 'reloadKey'>;

/** Loads the source into the player, moves to the proxy on an early failure and
 *  reports progress; returns the teardown of what it started. */
function play(player: Player, source: PlayerSource, handlers: { current: Handlers }): () => void {
  let disposed = false;
  let started = false;
  let usingFallback = false;
  let attempt = 0;
  let failedAttempt = -1;
  const wanted = resumePoint(handlers.current.startTime, NaN);

  const failed = (id: number, error: unknown) => {
    // The load's rejection and the store's error report the same failure once.
    if (disposed || id !== attempt || failedAttempt === id || isAbort(error)) return;
    failedAttempt = id;
    const kind = failureKind(error);
    if (kind === 'blocked' && source.fallbackSrc && shouldUseFallback({ usingFallback, started, hasFallback: true })) {
      usingFallback = true;
      open(source.fallbackSrc);
      return;
    }
    handlers.current.onFailure?.(kind);
  };
  // Otter retries a blocked playlist quietly for about 20 s before it gives up; one
  // fetch of the playlist tells a CORS or hotlink refusal in a single request.
  const preflight = new AbortController();
  const openDirect = () => {
    if (!source.fallbackSrc || !isHlsSource(source)) { open(source.src); return; }
    const id = attempt;
    fetch(source.src, { signal: preflight.signal }).then(response => {
      if (!response.ok) throw Error(`playlist ${response.status}`);
      if (!disposed) open(source.src);
    }).catch(error => {
      if (!isAbort(error)) failed(id, error);
    });
  };
  const open = (src: string) => {
    const id = ++attempt;
    player.load({ src, type: mimeType(source) }, { startTime: wanted || undefined, autoplay: true }).then(() => {
      if (disposed || id !== attempt) return;
      // The duration is known only now: a position near the end starts over.
      const at = resumePoint(wanted, player.store.getState().duration);
      if (wanted && !at) player.store.dispatch('seek', { time: 0 });
      handlers.current.onReady?.(at);
    }, error => failed(id, error));
  };

  const unsubscribe = player.store.subscribe((state, prev) => {
    if (disposed) return;
    if (state.readyState >= 2) started = true;
    if (state.currentTime !== prev.currentTime) handlers.current.onTime?.(state.currentTime, state.duration);
    if (state.paused !== prev.paused) handlers.current.onPlayingChange?.(!state.paused);
    if (state.ended && !prev.ended) handlers.current.onEnded?.();
    if (state.status === 'error' && state.error?.fatal && state.error !== prev.error) failed(attempt, state.error);
  });
  openDirect();
  return () => { disposed = true; preflight.abort(); unsubscribe(); };
}

export default function PlayerHost({ source, title, reloadKey, ...rest }: PlayerHostProps) {
  // Callbacks change every render; the player reads the latest through a ref.
  const handlers = useRef<Handlers>({ title, ...rest });
  handlers.current = { title, ...rest };

  const create = (container: HTMLElement) => {
    const player = createVodPlayer({ container, title: handlers.current.title, lang: currentLocale() });
    const stop = source?.src ? play(player, source, handlers) : undefined;
    return { destroy: () => { stop?.(); return player.destroy(); } };
  };
  return <OtterPlayer className="mp-player" create={create} deps={[source?.src, source?.fallbackSrc, source?.type, reloadKey]} />;
}
