import { browseURL, filtersToParams, parseFilters, toggleYear, withFilter } from './filters';

describe('site filters', () => {
  it('round-trips every filter through the URL', () => {
    const filters = parseFilters(new URLSearchParams('type=series&year=2024,none&area=中国&genre=悬疑&q=长季&sort=year&page=3'));
    expect(filters).toEqual({ type: 'series', years: ['2024', 'none'], area: '中国', genre: '悬疑', q: '长季', sort: 'year', page: 3 });
    expect(parseFilters(filtersToParams(filters))).toEqual(filters);
  });

  it('drops unknown or malformed values', () => {
    expect(parseFilters(new URLSearchParams('type=adult&year=20x4,,2024,2024&sort=random&page=-2'))).toEqual({
      type: '', years: ['2024'], area: '', genre: '', q: '', sort: 'updated', page: 1,
    });
  });

  it('omits defaults from shared links', () => {
    expect(browseURL()).toBe('/browse');
    expect(browseURL({ type: 'movie' })).toBe('/browse?type=movie');
  });

  it('returns to the first page when a filter changes', () => {
    const filters = parseFilters(new URLSearchParams('page=4'));
    expect(withFilter(filters, { area: '日本' }).page).toBe(1);
    expect(withFilter(filters, { page: 5 }).page).toBe(5);
    expect(toggleYear(toggleYear([], '2024'), '2024')).toEqual([]);
  });
});
