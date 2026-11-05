import { request } from '@umijs/max';

// Public site API: read-only, needs no login.
export type SiteCard = { id: number; title: string; mediaType: string; year?: string; area?: string; genres?: string; coverUrl?: string; episodes: number; updatedAt: string };
export type SiteEpisode = { id: number; episodeNo: number; name?: string; url: string; type: string };
export type SiteLine = { id: number; name: string; episodes: SiteEpisode[] };
export type SiteDetail = SiteCard & { originalTitle?: string; language?: string; director?: string; actors?: string; description?: string; lines: SiteLine[] };
export type SiteFacet = { value: string; count: number };
export type SiteFacets = { years: SiteFacet[]; areas: SiteFacet[]; genres: SiteFacet[] };
export type SiteRow = { key: string; items: SiteCard[] };
export type SiteQuery = { type?: string; year?: string; area?: string; genre?: string; q?: string; sort?: string; page?: number; pageSize?: number };
type API<T> = { code: number; msg?: string; data: T };

const base = '/api/v1/public/media';
const open = { skipErrorHandler: true };

export const getSiteHome = () => request<API<SiteRow[]>>(base + '/home', open);
export const listSiteItems = (params: SiteQuery) => request<API<{ list: SiteCard[]; total: number; page: number; pageSize: number }>>(base + '/items', { ...open, params });
export const getSiteFacets = () => request<API<SiteFacets>>(base + '/facets', open);
export const getSiteItem = (id: number) => request<API<SiteDetail>>(base + '/items/' + id, open);
export const siteStreamURL = (episodeId: number) => base + '/episodes/' + episodeId + '/stream';
