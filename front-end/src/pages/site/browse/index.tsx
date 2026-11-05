import React, { useEffect, useMemo, useState } from 'react';
import { Button, Pagination, Segmented, Tag, Typography } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import { history, useSearchParams } from '@umijs/max';
import { t, useI18n } from '@/i18n';
import { getSiteFacets, listSiteItems, type SiteCard, type SiteFacet, type SiteFacets } from '@/services/site';
import SiteLayout, { typeLabel } from '../SiteLayout';
import { GridSkeleton, LoadFailed, Nothing, PosterGrid } from '../components';
import { MEDIA_TYPES, browseURL, filtersToParams, hasFilters, parseFilters, toggleYear, withFilter, type SiteFilters } from '../filters';
import { useScrollMemory } from '../useScrollMemory';

const PAGE_SIZE = 24;

function FilterRow({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="site-filter-row"><span className="site-filter-label">{label}</span><div className="site-filter-options">{children}</div></div>;
}

function Chip({ active, onClick, children, count }: { active: boolean; onClick: () => void; children: React.ReactNode; count?: number }) {
  return <button type="button" className={`site-chip${active ? ' is-active' : ''}`} aria-pressed={active} onClick={onClick}>{children}{count !== undefined && <span className="site-chip-count">{count}</span>}</button>;
}

export default function SiteBrowse() {
  useI18n();
  const [params] = useSearchParams();
  const filterKey = filtersToParams(parseFilters(params)).toString();
  const filters = useMemo(() => parseFilters(new URLSearchParams(filterKey)), [filterKey]);
  const [cards, setCards] = useState<SiteCard[]>([]);
  const [total, setTotal] = useState(0);
  const [state, setState] = useState<'loading' | 'ready' | 'error'>('loading');
  const [attempt, setAttempt] = useState(0);
  const [facets, setFacets] = useState<SiteFacets>();
  useScrollMemory(state !== 'loading');

  useEffect(() => { getSiteFacets().then(r => { if (r.code === 0) setFacets(r.data); }).catch(() => undefined); }, []);
  useEffect(() => {
    document.title = [filters.q || (filters.type ? typeLabel(filters.type) : t('cms.site.browse.title')), t('cms.brandName')].join(' · ');
    let live = true;
    setState('loading');
    listSiteItems({ type: filters.type || undefined, year: filters.years.join(',') || undefined, area: filters.area || undefined, genre: filters.genre || undefined, q: filters.q || undefined, sort: filters.sort, page: filters.page, pageSize: PAGE_SIZE })
      .then(r => {
        if (!live) return;
        if (r.code !== 0) throw Error(r.msg);
        setCards(r.data.list || []);
        setTotal(r.data.total || 0);
        setState('ready');
      })
      .catch(() => live && setState('error'));
    return () => { live = false; };
  }, [filterKey, attempt]);

  const go = (next: SiteFilters) => history.push(browseURL(next));
  const set = (change: Partial<SiteFilters>) => go(withFilter(filters, change));
  const yearLabel = (facet: SiteFacet) => (facet.value === 'none' ? t('cms.site.browse.unknownYear') : facet.value);

  return (
    <SiteLayout>
      <div className="site-filters">
        {filters.q && <div className="site-filter-query"><Typography.Text>{t('cms.site.browse.searching', { q: filters.q })}</Typography.Text><Tag closable onClose={() => set({ q: '' })}>{filters.q}</Tag></div>}
        <FilterRow label={t('cms.site.browse.type')}>
          <Chip active={!filters.type} onClick={() => set({ type: '' })}>{t('cms.site.browse.all')}</Chip>
          {MEDIA_TYPES.map(kind => <Chip key={kind} active={filters.type === kind} onClick={() => set({ type: kind })}>{typeLabel(kind)}</Chip>)}
        </FilterRow>
        {!!facets?.years.length && (
          <FilterRow label={t('cms.site.browse.year')}>
            <Chip active={!filters.years.length} onClick={() => set({ years: [] })}>{t('cms.site.browse.all')}</Chip>
            {facets.years.map(facet => <Chip key={facet.value} count={facet.count} active={filters.years.includes(facet.value)} onClick={() => set({ years: toggleYear(filters.years, facet.value) })}>{yearLabel(facet)}</Chip>)}
          </FilterRow>
        )}
        {!!facets?.areas.length && (
          <FilterRow label={t('cms.site.browse.area')}>
            <Chip active={!filters.area} onClick={() => set({ area: '' })}>{t('cms.site.browse.all')}</Chip>
            {facets.areas.map(facet => <Chip key={facet.value} count={facet.count} active={filters.area === facet.value} onClick={() => set({ area: filters.area === facet.value ? '' : facet.value })}>{facet.value}</Chip>)}
          </FilterRow>
        )}
        {!!facets?.genres.length && (
          <FilterRow label={t('cms.site.browse.genre')}>
            <Chip active={!filters.genre} onClick={() => set({ genre: '' })}>{t('cms.site.browse.all')}</Chip>
            {facets.genres.map(facet => <Chip key={facet.value} count={facet.count} active={filters.genre === facet.value} onClick={() => set({ genre: filters.genre === facet.value ? '' : facet.value })}>{facet.value}</Chip>)}
          </FilterRow>
        )}
        <div className="site-filter-foot">
          <Typography.Text type="secondary">{t('cms.site.browse.total', { n: total })}</Typography.Text>
          <div className="site-filter-foot-actions">
            {hasFilters(filters) && <Button type="text" size="small" icon={<CloseOutlined />} onClick={() => go(withFilter(parseFilters(new URLSearchParams()), {}))}>{t('cms.site.browse.clear')}</Button>}
            <Segmented size="small" value={filters.sort} onChange={value => set({ sort: value as SiteFilters['sort'] })}
              options={[{ value: 'updated', label: t('cms.site.browse.sortUpdated') }, { value: 'year', label: t('cms.site.browse.sortYear') }]} />
          </div>
        </div>
      </div>
      {state === 'error' && <LoadFailed onRetry={() => setAttempt(n => n + 1)} />}
      {state === 'loading' && <GridSkeleton count={PAGE_SIZE} />}
      {state === 'ready' && (cards.length
        ? <PosterGrid cards={cards} />
        : <Nothing text={hasFilters(filters) ? t('cms.site.empty.filtered') : t('cms.site.empty.catalogue')} action={hasFilters(filters) && <Button onClick={() => go(withFilter(parseFilters(new URLSearchParams()), {}))}>{t('cms.site.browse.clear')}</Button>} />)}
      {state === 'ready' && total > PAGE_SIZE && (
        <div className="site-pagination"><Pagination current={filters.page} total={total} pageSize={PAGE_SIZE} showSizeChanger={false} onChange={page => set({ page })} /></div>
      )}
    </SiteLayout>
  );
}
