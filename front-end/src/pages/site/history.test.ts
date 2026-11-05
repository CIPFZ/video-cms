import { forgetRecent, readRecent, recordRecent } from './history';

describe('recently watched', () => {
  beforeEach(() => localStorage.clear());

  it('keeps one entry per title, newest first, at most 20', () => {
    for (let id = 1; id <= 25; id++) recordRecent({ id, title: `T${id}`, episodeTitle: '第 1 集', updatedAt: id });
    recordRecent({ id: 10, title: 'T10', episodeTitle: '第 2 集', updatedAt: 99 });
    const list = readRecent();
    expect(list).toHaveLength(20);
    expect(list[0]).toMatchObject({ id: 10, episodeTitle: '第 2 集' });
    expect(list.filter(v => v.id === 10)).toHaveLength(1);
    forgetRecent(10);
    expect(readRecent().some(v => v.id === 10)).toBe(false);
  });

  it('ignores damaged storage', () => {
    localStorage.setItem('site-recent-watch', '{oops');
    expect(readRecent()).toEqual([]);
    localStorage.setItem('site-recent-watch', JSON.stringify([{ id: 'x' }, { id: 3, title: 'ok' }]));
    expect(readRecent()).toEqual([{ id: 3, title: 'ok' }]);
  });
});
