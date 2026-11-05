import React, { useEffect, useMemo, useState } from 'react';
import { Button, Skeleton, Space, Tabs, Typography } from 'antd';
import { CaretRightFilled, HistoryOutlined } from '@ant-design/icons';
import { Link } from '@umijs/max';
import { t, useI18n } from '@/i18n';
import type { MediaItem } from '@/services/media';
import { getSiteItem, type SiteDetail, type SiteLine } from '@/services/site';
import { Cover, EpisodeRanges, Facts, GenreTags, People, TypeTags, describe, episodeTitle, readProgress } from '../media/shared';
import { LoadFailed, Nothing } from './components';
import { useScrollMemory } from './useScrollMemory';

const RANGE = 60;

export const lineLabel = (line: SiteLine, index: number) => `${t('cms.site.line', { n: index + 1 })}${line.name ? ` · ${line.name}` : ''}`;

export const watchURL = (id: number, lineId?: number, episodeId?: number, restart = false) => {
  const params = new URLSearchParams();
  if (lineId) params.set('line', String(lineId));
  if (episodeId) params.set('ep', String(episodeId));
  if (restart) params.set('restart', '1');
  const query = params.toString();
  return `/watch/${id}${query ? `?${query}` : ''}`;
};

function Episodes({ id, line, watchedId }: { id: number; line: SiteLine; watchedId?: number }) {
  const [reverse, setReverse] = useState(false);
  const [range, setRange] = useState(0);
  const ordered = reverse ? [...line.episodes].reverse() : line.episodes;
  return (
    <div className="site-episodes">
      {line.episodes.length > 1 && (
        <div className="site-episodes-head">
          <Typography.Text type="secondary">{t('cms.site.detail.episodeCount', { n: line.episodes.length })}</Typography.Text>
          <Button size="small" type="text" onClick={() => { setReverse(!reverse); setRange(0); }}>{reverse ? t('cms.site.detail.descending') : t('cms.site.detail.ascending')}</Button>
        </div>
      )}
      <EpisodeRanges total={ordered.length} size={RANGE} reverse={reverse} value={range} onChange={setRange} />
      <div className="site-episode-grid">
        {ordered.slice(range * RANGE, (range + 1) * RANGE).map(episode => (
          <Link key={episode.id} to={watchURL(id, line.id, episode.id)} className={episode.id === watchedId ? 'is-watched' : undefined} title={episodeTitle({ episodeNo: episode.episodeNo, episodeName: episode.name })}>
            {episode.id === watchedId && <HistoryOutlined />}{episodeTitle({ episodeNo: episode.episodeNo, episodeName: episode.name })}
          </Link>
        ))}
      </div>
    </div>
  );
}

export default function TitleDetail({ id }: { id: number }) {
  useI18n();
  const [detail, setDetail] = useState<SiteDetail>();
  const [state, setState] = useState<'loading' | 'ready' | 'error'>('loading');
  const [attempt, setAttempt] = useState(0);
  const [lineId, setLineId] = useState<number>();
  useEffect(() => {
    let live = true;
    setState('loading');
    getSiteItem(id).then(r => {
      if (!live) return;
      if (r.code !== 0) throw Error(r.msg);
      setDetail(r.data);
      setState('ready');
    }).catch(() => live && setState('error'));
    return () => { live = false; };
  }, [id, attempt]);
  const progress = useMemo(() => readProgress(id), [id, detail]);
  useEffect(() => {
    if (!detail) return;
    setLineId(detail.lines.find(line => line.id === progress?.lineId)?.id || detail.lines[0]?.id);
    document.title = [detail.title, t('cms.brandName')].join(' · ');
  }, [detail]);
  useScrollMemory(state !== 'loading');

  if (state === 'error') return <LoadFailed onRetry={() => setAttempt(n => n + 1)} />;
  if (state === 'loading' || !detail) return <div className="site-detail"><Skeleton.Node active className="site-detail-cover-skeleton"><span /></Skeleton.Node><Skeleton active paragraph={{ rows: 6 }} /></div>;

  const first = detail.lines[0];
  const description = describe(detail.description);
  const item: MediaItem = { ...detail, status: 'active' };
  // A single line with a single episode has nothing to choose: the play button covers it.
  const choices = detail.lines.length > 1 || (first?.episodes.length || 0) > 1;
  return (
    <div className="site-title">
      {detail.coverUrl && <div className="site-title-backdrop" aria-hidden><img src={detail.coverUrl} alt="" referrerPolicy="no-referrer" onError={event => { event.currentTarget.style.display = 'none'; }} /></div>}
      <div className="site-detail">
        <Cover url={detail.coverUrl} title={detail.title} className="site-detail-cover" />
        <div className="site-detail-body">
          <h1>{detail.title}</h1>
          {detail.originalTitle && detail.originalTitle !== detail.title && <div className="site-detail-original">{detail.originalTitle}</div>}
          <div className="site-detail-facts"><TypeTags value={detail.mediaType} /><Facts item={item} /></div>
          <GenreTags value={detail.genres} />
          <People item={item} />
          <Typography.Paragraph className="site-detail-description" ellipsis={description ? { rows: 4, expandable: 'collapsible', symbol: expanded => (expanded ? t('cms.site.detail.collapse') : t('cms.site.detail.expand')) } : false}>
            {description || t('cms.site.detail.noDescription')}
          </Typography.Paragraph>
          {first ? (
            <Space wrap className="site-detail-actions">
              {progress
                ? <Link to={watchURL(detail.id, progress.lineId, progress.episodeId)}><Button type="primary" size="large" icon={<HistoryOutlined />}>{t('cms.site.detail.continue', { episode: progress.episodeTitle })}</Button></Link>
                : <Link to={watchURL(detail.id, first.id, first.episodes[0]?.id)}><Button type="primary" size="large" icon={<CaretRightFilled />}>{t('cms.site.detail.play')}</Button></Link>}
              {progress && <Link to={watchURL(detail.id, first.id, first.episodes[0]?.id, true)}><Button size="large">{t('cms.site.detail.fromStart')}</Button></Link>}
            </Space>
          ) : <Nothing text={t('cms.site.detail.noLines')} />}
        </div>
      </div>
      {choices && (
        <section className="site-detail-lines">
          <h2>{t('cms.site.watch.episodes')}</h2>
          <Tabs activeKey={String(lineId)} onChange={key => setLineId(Number(key))} items={detail.lines.map((line, index) => ({
            key: String(line.id), label: lineLabel(line, index),
            children: <Episodes id={detail.id} line={line} watchedId={progress?.lineId === line.id ? progress.episodeId : undefined} />,
          }))} />
        </section>
      )}
    </div>
  );
}
