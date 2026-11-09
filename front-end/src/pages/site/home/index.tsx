import React, { useEffect, useState } from 'react';
import { Button } from 'antd';
import { CaretRightFilled, CloseOutlined } from '@ant-design/icons';
import { Link } from '@umijs/max';
import { t, useI18n } from '@/i18n';
import { getSiteHome, type SiteRow } from '@/services/site';
import { Cover, clockText, readProgress } from '../../media/shared';
import SiteLayout, { typeLabel } from '../SiteLayout';
import { watchURL } from '../TitleDetail';
import { GridSkeleton, LoadFailed, Nothing, PosterCard, PosterRow } from '../components';
import { browseURL } from '../filters';
import { forgetRecent, readRecent, type RecentWatch } from '../history';
import { useScrollMemory } from '../useScrollMemory';

const rowTitle = (key: string) => (key === 'latest' ? t('cms.site.row.latest') : typeLabel(key));

// A compact tile that goes straight back to the player, which resumes the saved episode and time.
function ResumeCard({ entry, onForget }: { entry: RecentWatch; onForget: () => void }) {
  const progress = readProgress(entry.id);
  const percent = progress?.duration ? Math.min(100, (progress.time / progress.duration) * 100) : 0;
  const meta = [entry.episodeTitle, progress?.time ? t('cms.site.row.stoppedAt', { time: clockText(progress.time) }) : ''].filter(Boolean).join(' · ');
  const forget = t('cms.site.row.forget', { title: entry.title });
  return (
    <div className="site-resume">
      <Link to={watchURL(entry.id)} className="site-resume-link" aria-label={t('cms.site.row.resume', { title: entry.title })}>
        <div className="site-resume-cover"><Cover url={entry.coverUrl} title={entry.title} /><span className="site-resume-play"><CaretRightFilled /></span></div>
        <div className="site-resume-body">
          <div className="site-resume-title" title={entry.title}>{entry.title}</div>
          <div className="site-resume-meta">{meta}</div>
          {percent > 0 && <div className="site-resume-bar"><i style={{ width: `${percent}%` }} /></div>}
        </div>
      </Link>
      <Button className="site-resume-remove" type="text" size="small" icon={<CloseOutlined />} aria-label={forget} title={forget} onClick={onForget} />
    </div>
  );
}

export default function SiteHome() {
  useI18n();
  const [rows, setRows] = useState<SiteRow[]>([]);
  const [state, setState] = useState<'loading' | 'ready' | 'error'>('loading');
  const [attempt, setAttempt] = useState(0);
  const [recent, setRecent] = useState<RecentWatch[]>(readRecent);
  useScrollMemory(state !== 'loading');
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
        <PosterRow title={t('cms.site.row.continue')} className="site-row-resume">
          {recent.map(entry => <ResumeCard key={entry.id} entry={entry} onForget={() => { forgetRecent(entry.id); setRecent(readRecent()); }} />)}
        </PosterRow>
      )}
      {state === 'error' && <LoadFailed onRetry={() => setAttempt(n => n + 1)} />}
      {state === 'loading' && <GridSkeleton />}
      {state === 'ready' && !rows.length && <Nothing text={t('cms.site.empty.catalogue')} />}
      {rows.map(row => (
        <PosterRow key={row.key} title={rowTitle(row.key)} more={browseURL(row.key === 'latest' ? {} : { type: row.key })}>
          {row.items.map(card => <PosterCard key={card.id} card={card} />)}
        </PosterRow>
      ))}
    </SiteLayout>
  );
}
