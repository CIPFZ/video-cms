// Browse filters live in the URL so a filtered list can be shared and Back restores it.
export type SiteFilters = { type: string; years: string[]; area: string; genre: string; q: string; sort: 'updated' | 'year'; page: number };

export const MEDIA_TYPES = ['movie', 'series', 'anime', 'variety'] as const;
const EMPTY: SiteFilters = { type: '', years: [], area: '', genre: '', q: '', sort: 'updated', page: 1 };

export function parseFilters(params: URLSearchParams): SiteFilters {
  const type = params.get('type') || '';
  const page = Number(params.get('page'));
  return {
    type: (MEDIA_TYPES as readonly string[]).includes(type) ? type : '',
    years: Array.from(new Set((params.get('year') || '').split(',').map(v => v.trim()).filter(v => v === 'none' || /^\d{4}$/.test(v)))),
    area: params.get('area')?.trim() || '',
    genre: params.get('genre')?.trim() || '',
    q: params.get('q')?.trim() || '',
    sort: params.get('sort') === 'year' ? 'year' : 'updated',
    page: Number.isInteger(page) && page > 1 ? page : 1,
  };
}

/** Query string for the filters, defaults omitted; `title` (an open detail) is not a filter. */
export function filtersToParams(filters: Partial<SiteFilters>): URLSearchParams {
  const f = { ...EMPTY, ...filters };
  const out = new URLSearchParams();
  if (f.type) out.set('type', f.type);
  if (f.years.length) out.set('year', f.years.join(','));
  if (f.area) out.set('area', f.area);
  if (f.genre) out.set('genre', f.genre);
  if (f.q) out.set('q', f.q);
  if (f.sort !== 'updated') out.set('sort', f.sort);
  if (f.page > 1) out.set('page', String(f.page));
  return out;
}

export const browseURL = (filters: Partial<SiteFilters> = {}) => {
  const query = filtersToParams(filters).toString();
  return `/browse${query ? `?${query}` : ''}`;
};

/** A change of any filter returns to the first page. */
export const withFilter = (filters: SiteFilters, change: Partial<SiteFilters>): SiteFilters => ({ ...filters, ...change, page: change.page ?? 1 });

export const toggleYear = (years: string[], year: string) => (years.includes(year) ? years.filter(y => y !== year) : [...years, year]);

export const hasFilters = (f: SiteFilters) => !!(f.type || f.years.length || f.area || f.genre || f.q);
