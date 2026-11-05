import { useEffect, useLayoutEffect } from 'react';
import { history, useLocation } from '@umijs/max';

const positions = new Map<string, number>();

/** Site pages swap in place, so the window would keep its scroll: a new page starts at
 *  the top, and Back returns to where it was once `ready` says the rows are there again. */
export function useScrollMemory(ready: boolean) {
  const { key } = useLocation();
  // Layout phase: stop listening before the next page's shorter content clamps the scroll.
  useLayoutEffect(() => {
    const save = () => positions.set(key, window.scrollY);
    window.addEventListener('scroll', save, { passive: true });
    return () => window.removeEventListener('scroll', save);
  }, [key]);
  useEffect(() => {
    if (ready) window.scrollTo(0, history.action === 'POP' ? positions.get(key) || 0 : 0);
  }, [key, ready]);
}
