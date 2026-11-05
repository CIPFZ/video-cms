import React, { useEffect, useState } from 'react';
import { Button } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import { t, useI18n } from '@/i18n';
import { getSiteHome, type SiteRow } from '@/services/site';
import SiteLayout, { typeLabel } from '../SiteLayout';
import TitleOverlay from '../TitleOverlay';
import { GridSkeleton, LoadFailed, Nothing, PosterCard, PosterRow, useTitleOverlay } from '../components';
import { browseURL } from '../filters';
import { forgetRecent, readRecent, type RecentWatch } from '../history';

const rowTitle = (key: string) => (key === 'latest' ? t('cms.site.row.latest') : typeLabel(key));

export default function SiteHome() {
  useI18n();
  const { open } = useTitleOverlay();
  const [rows, setRows] = useState<SiteRow[]>([]);
  const [state, setState] = useState<'loading' | 'ready' | 'error'>('loading');
  const [attempt, setAttempt] = useState(0);
  const [recent, setRecent] = useState<RecentWatch[]>(readRecent);
  useEffect(() => {
    document.title = t('cms.brandName');
    let live = true;
    setState('loading');
    getSiteHome().then(r => {
      if (!live) return;
      if (r.code !== 0) throw Error(r.msg);
      setRows(r.data || []);
      setState('ready');
    }).catch(() => live && setState('error'));
    return () => { live = false; };
  }, [attempt]);

  return (
    <SiteLayout>
      {recent.length > 0 && (
        <PosterRow title={t('cms.site.row.continue')}>
          {recent.map(entry => (
            <div key={entry.id} className="site-recent">
              <PosterCard card={{ id: entry.id, title: entry.title, coverUrl: entry.coverUrl, mediaType: '', episodes: 0, updatedAt: '' }} note={entry.episodeTitle} onOpen={open} />
              <Button className="site-recent-remove" size="small" shape="circle" icon={<CloseOutlined />} aria-label={t('cms.site.row.forget', { title: entry.title })} title={t('cms.site.row.forget', { title: entry.title })}
                onClick={() => { forgetRecent(entry.id); setRecent(readRecent()); }} />
            </div>
          ))}
        </PosterRow>
      )}
      {state === 'error' && <LoadFailed onRetry={() => setAttempt(n => n + 1)} />}
      {state === 'loading' && <GridSkeleton />}
      {state === 'ready' && !rows.length && <Nothing text={t('cms.site.empty.catalogue')} />}
      {rows.map(row => (
        <PosterRow key={row.key} title={rowTitle(row.key)} more={browseURL(row.key === 'latest' ? {} : { type: row.key })}>
          {row.items.map(card => <PosterCard key={card.id} card={card} onOpen={open} />)}
        </PosterRow>
      ))}
      <TitleOverlay />
    </SiteLayout>
  );
}
