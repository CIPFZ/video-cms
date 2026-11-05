import { isPublicPath } from './publicPaths';

describe('isPublicPath', () => {
  it('covers the public site and the admin entry', () => {
    for (const path of ['/', '/browse', '/title/12', '/watch/12', '/admin', '/media/player']) {
      expect(isPublicPath(path)).toBe(true);
    }
  });

  it('leaves the back office alone', () => {
    for (const path of ['/dashboard/workplace', '/media/library', '/sys/user', '/user/login', '/browser', '/watchlist']) {
      expect(isPublicPath(path)).toBe(false);
    }
  });
});
