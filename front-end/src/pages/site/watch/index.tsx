import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, ConfigProvider, Empty, Select, Space, Spin, Switch, Tooltip, Typography, message } from 'antd';
import { CopyOutlined, ExportOutlined, LeftOutlined, ReloadOutlined, SortAscendingOutlined, SortDescendingOutlined, StepBackwardOutlined, StepForwardOutlined, SwapOutlined } from '@ant-design/icons';
import { Link, history, useParams, useSearchParams } from '@umijs/max';
import { t, useI18n } from '@/i18n';
import type { MediaItem } from '@/services/media';
import { getSiteItem, siteStreamURL, type SiteDetail, type SiteEpisode, type SiteLine } from '@/services/site';
import { Cover, EpisodeRanges, Facts, GenreTags, People, TypeTags, clockText, copyText, describe, episodeTitle, readProgress, writeProgress } from '../../media/shared';
import PlayerHost, { type PlayerFailure } from '../PlayerHost';
import { siteTheme } from '../SiteLayout';
import { lineLabel, watchURL } from '../TitleDetail';
import { titleURL } from '../components';
import { recordRecent } from '../history';

const RANGE_SIZE = 100;
const AUTO_NEXT_KEY = 'media-auto-next';
const nameOf = (episode: SiteEpisode) => episodeTitle({ episodeNo: episode.episodeNo, episodeName: episode.name });

export default function SiteWatch() {
  useI18n();
  const itemId = Number(useParams().id) || 0;
  const [params] = useSearchParams();
  const requestedLine = Number(params.get('line')) || 0;
  const requestedEpisode = Number(params.get('ep')) || 0;
  const restart = params.get('restart') === '1';
  const [detail, setDetail] = useState<SiteDetail>();
  const [loading, setLoading] = useState(true);
  const [lineId, setLineId] = useState<number>();
  const [episodeId, setEpisodeId] = useState<number>();
  const [startTime, setStartTime] = useState(0);
  const [playerState, setPlayerState] = useState<'loading' | 'ready' | PlayerFailure>('loading');
  const [playing, setPlaying] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const [reverse, setReverse] = useState(false);
  const [range, setRange] = useState(0);
  const [autoNext, setAutoNext] = useState(() => localStorage.getItem(AUTO_NEXT_KEY) !== '0');
  const grid = useRef<HTMLDivElement>(null);
  const position = useRef({ time: 0, saved: 0 });

  const lines = detail?.lines || [];
  const selectedLine = lines.find(line => line.id === lineId) || lines[0];
  const selectedEpisode = selectedLine?.episodes.find(episode => episode.id === episodeId) || selectedLine?.episodes[0];
  const episodeIndex = selectedLine && selectedEpisode ? selectedLine.episodes.indexOf(selectedEpisode) : -1;
  const hasPrev = episodeIndex > 0;
  const hasNext = !!selectedLine && episodeIndex >= 0 && episodeIndex < selectedLine.episodes.length - 1;
  const ordered = useMemo(() => (selectedLine ? (reverse ? [...selectedLine.episodes].reverse() : selectedLine.episodes) : []), [selectedLine, reverse]);
  const nextLine = lines.length > 1 && selectedLine ? lines[(lines.indexOf(selectedLine) + 1) % lines.length] : undefined;
  const source = useMemo(() => (selectedEpisode?.url ? { src: selectedEpisode.url, fallbackSrc: siteStreamURL(selectedEpisode.id), type: selectedEpisode.type } : undefined), [selectedEpisode?.id, selectedEpisode?.url, selectedEpisode?.type]);

  useEffect(() => {
    if (!itemId) { setLoading(false); return; }
    setLoading(true);
    getSiteItem(itemId).then(result => {
      if (result.code !== 0) throw Error(result.msg);
      setDetail(result.data);
      const saved = restart ? undefined : readProgress(itemId);
      // An explicit episode wins; otherwise continue where this browser left off.
      let wantedLine = requestedLine, wantedEpisode = requestedEpisode;
      if (!wantedEpisode && saved && (!wantedLine || wantedLine === saved.lineId)) { wantedLine = saved.lineId; wantedEpisode = saved.episodeId; }
      const all = result.data.lines;
      const line = all.find(entry => entry.id === wantedLine) || all.find(entry => entry.episodes.some(episode => episode.id === wantedEpisode)) || all[0];
      const episode = line?.episodes.find(entry => entry.id === wantedEpisode) || line?.episodes[0];
      setStartTime(saved && episode && saved.episodeId === episode.id ? saved.time : 0);
      setLineId(line?.id);
      setEpisodeId(episode?.id);
    }).catch(() => message.error(t('cms.site.watch.loadFailed'))).finally(() => setLoading(false));
  }, [itemId]);

  const save = (time = position.current.time) => {
    if (!detail || !selectedLine || !selectedEpisode) return;
    writeProgress(itemId, { lineId: selectedLine.id, episodeId: selectedEpisode.id, episodeTitle: nameOf(selectedEpisode), time: Math.floor(time), updatedAt: Date.now() });
    recordRecent({ id: itemId, title: detail.title, coverUrl: detail.coverUrl, episodeTitle: nameOf(selectedEpisode), updatedAt: Date.now() });
  };
  const saveRef = useRef(save);
  saveRef.current = save;
  // Record the episode right away so "continue watching" follows episode switches.
  useEffect(() => {
    if (!selectedEpisode) return undefined;
    position.current = { time: startTime, saved: startTime };
    setPlayerState('loading');
    setPlaying(false);
    saveRef.current(startTime);
    const leave = () => saveRef.current();
    window.addEventListener('pagehide', leave);
    return () => { window.removeEventListener('pagehide', leave); if (position.current.time > 0) saveRef.current(); };
  }, [selectedEpisode?.id, reloadKey]);

  useEffect(() => {
    const index = ordered.findIndex(episode => episode.id === selectedEpisode?.id);
    if (index >= 0) setRange(Math.floor(index / RANGE_SIZE));
  }, [selectedEpisode?.id, ordered]);
  useEffect(() => {
    const box = grid.current;
    const active = box?.querySelector<HTMLElement>('.is-active');
    if (box && active) box.scrollTop = active.offsetTop - box.clientHeight / 2 + active.offsetHeight / 2;
  }, [selectedEpisode?.id, range, reverse, detail]);
  useEffect(() => {
    if (detail) document.title = [detail.title, selectedEpisode && nameOf(selectedEpisode), t('cms.brandName')].filter(Boolean).join(' · ');
  }, [detail, selectedEpisode?.id]);

  const selectEpisode = (line: SiteLine, episode: SiteEpisode) => {
    const saved = readProgress(itemId);
    setStartTime(saved?.episodeId === episode.id ? saved.time : 0);
    setLineId(line.id);
    setEpisodeId(episode.id);
    history.replace(watchURL(itemId, line.id, episode.id));
  };
  // Switching lines keeps the same episode number when the other line has it.
  const switchLine = (line?: SiteLine) => {
    if (line?.episodes.length) selectEpisode(line, line.episodes.find(episode => episode.episodeNo === selectedEpisode?.episodeNo) || line.episodes[0]);
  };
  const moveEpisode = (offset: number) => {
    const next = selectedLine?.episodes[episodeIndex + offset];
    if (selectedLine && next) selectEpisode(selectedLine, next);
  };
  const toggleAutoNext = (checked: boolean) => { setAutoNext(checked); localStorage.setItem(AUTO_NEXT_KEY, checked ? '1' : '0'); };
  const openOriginal = () => { if (selectedEpisode) window.open(selectedEpisode.url, '_blank', 'noopener,noreferrer'); };

  if (loading || !detail) {
    return (
      <ConfigProvider theme={siteTheme}>
        <div className="mp-page mp-center">
          {loading ? <Spin size="large" /> : <Empty description={t('cms.site.watch.notFound')}><Link to="/"><Button>{t('cms.site.watch.home')}</Button></Link></Empty>}
        </div>
      </ConfigProvider>
    );
  }

  const description = describe(detail.description);
  const item: MediaItem = { ...detail, status: 'active' };
  const visible = ordered.slice(range * RANGE_SIZE, (range + 1) * RANGE_SIZE);
  const lineIndex = selectedLine ? lines.indexOf(selectedLine) : -1;
  return (
    <ConfigProvider theme={siteTheme}>
      <div className="mp-page">
        {detail.coverUrl && <img className="mp-backdrop" src={detail.coverUrl} alt="" aria-hidden referrerPolicy="no-referrer" onError={event => { event.currentTarget.style.display = 'none'; }} />}
        <header className="mp-topbar">
          <Link className="mp-back" to="/"><LeftOutlined /> {t('cms.brandName')}</Link>
          <div className="mp-topbar-title"><Link to={titleURL(detail.id)}>{detail.title}</Link>{selectedEpisode && <em>{nameOf(selectedEpisode)}</em>}</div>
        </header>

        <div className="mp-layout">
          <main className="mp-main">
            <div className="mp-stage">
              <PlayerHost source={source} startTime={startTime} reloadKey={reloadKey}
                onReady={at => { setPlayerState('ready'); if (at) message.info(t('cms.site.watch.resumed', { time: clockText(at) })); }}
                onFailure={setPlayerState}
                onTime={time => { position.current.time = time; if (Math.abs(time - position.current.saved) >= 5) { position.current.saved = time; save(time); } }}
                onPlayingChange={value => { setPlaying(value); if (!value) save(); }}
                onEnded={() => { if (autoNext && hasNext) moveEpisode(1); }} />
              {playerState === 'loading' && selectedEpisode && <div className="mp-overlay mp-overlay-passive"><Spin size="large" /><div className="mp-overlay-desc">{t('cms.site.watch.loading')}</div></div>}
              {(playerState === 'blocked' || playerState === 'unsupported') && (
                <div className="mp-overlay">
                  <div className="mp-overlay-title">{t('cms.site.watch.failedTitle')}</div>
                  <div className="mp-overlay-desc">{playerState === 'unsupported' ? t('cms.site.watch.unsupported') : t('cms.site.watch.blocked')}</div>
                  <Space wrap>
                    <Button type="primary" icon={<ReloadOutlined />} onClick={() => setReloadKey(key => key + 1)}>{t('cms.site.watch.retry')}</Button>
                    {nextLine && <Button icon={<SwapOutlined />} onClick={() => switchLine(nextLine)}>{t('cms.site.watch.otherLine')}</Button>}
                    <Button icon={<ExportOutlined />} onClick={openOriginal}>{t('cms.site.watch.original')}</Button>
                  </Space>
                </div>
              )}
              {!selectedEpisode && <div className="mp-overlay"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('cms.site.detail.noLines')} /></div>}
            </div>

            <div className="mp-controls">
              <div className="mp-now">
                <span className="mp-now-label">{t('cms.site.watch.nowPlaying')}</span>
                <b>{selectedEpisode ? nameOf(selectedEpisode) : '—'}</b>
                {selectedLine && <span className="mp-now-line">{lineLabel(selectedLine, lineIndex)}</span>}
              </div>
              <Space wrap size={8}>
                <Button icon={<StepBackwardOutlined />} disabled={!hasPrev} onClick={() => moveEpisode(-1)}>{t('cms.site.watch.previous')}</Button>
                <Button icon={<StepForwardOutlined />} disabled={!hasNext} onClick={() => moveEpisode(1)}>{t('cms.site.watch.next')}</Button>
                <span className="mp-autonext"><Switch id="mp-autonext" size="small" checked={autoNext} onChange={toggleAutoNext} /><label htmlFor="mp-autonext">{t('cms.site.watch.autoNext')}</label></span>
                <Tooltip title={t('cms.site.watch.copy')}><Button type="text" icon={<CopyOutlined />} aria-label={t('cms.site.watch.copy')} disabled={!selectedEpisode} onClick={() => selectedEpisode && copyText(selectedEpisode.url, t('cms.site.watch.copied'))} /></Tooltip>
                <Tooltip title={t('cms.site.watch.original')}><Button type="text" icon={<ExportOutlined />} aria-label={t('cms.site.watch.original')} disabled={!selectedEpisode} onClick={openOriginal} /></Tooltip>
              </Space>
            </div>

            <section className="mp-info">
              <Cover url={detail.coverUrl} title={detail.title} className="mp-cover" />
              <div>
                <h1>{detail.title}</h1>
                {detail.originalTitle && detail.originalTitle !== detail.title && <div className="mp-original">{detail.originalTitle}</div>}
                <div><TypeTags value={detail.mediaType} /><Facts item={item} /></div>
                <GenreTags value={detail.genres} />
                <People item={item} />
                <Typography.Paragraph className="mp-description" ellipsis={description ? { rows: 3, expandable: 'collapsible', symbol: expanded => (expanded ? t('cms.site.detail.collapse') : t('cms.site.detail.expand')) } : false}>
                  {description || t('cms.site.detail.noDescription')}
                </Typography.Paragraph>
              </div>
            </section>
          </main>

          <aside className="mp-sidebar">
            <div className="mp-sidebar-head">
              <b>{t('cms.site.watch.episodes')}</b>
              {selectedLine && <span className="mp-count">{t('cms.site.detail.episodeCount', { n: selectedLine.episodes.length })}</span>}
              {(selectedLine?.episodes.length || 0) > 1 && (
                <Button size="small" type="text" icon={reverse ? <SortDescendingOutlined /> : <SortAscendingOutlined />} onClick={() => setReverse(!reverse)}>{reverse ? t('cms.site.detail.descending') : t('cms.site.detail.ascending')}</Button>
              )}
            </div>
            {lines.length > 1 && (
              <Select className="mp-line-select" value={selectedLine?.id} onChange={id => switchLine(lines.find(line => line.id === id))}
                options={lines.map((line, index) => ({ value: line.id, label: `${lineLabel(line, index)} · ${t('cms.site.detail.episodeCount', { n: line.episodes.length })}` }))} />
            )}
            <EpisodeRanges total={ordered.length} size={RANGE_SIZE} reverse={reverse} value={range} onChange={setRange} />
            <div className="mp-episode-grid" ref={grid}>
              {visible.map(episode => {
                const active = episode.id === selectedEpisode?.id;
                return (
                  <button key={episode.id} type="button" title={nameOf(episode)} aria-pressed={active} className={`mp-episode${active ? ' is-active' : ''}`} onClick={() => selectedLine && selectEpisode(selectedLine, episode)}>
                    {active && <span className={`mp-eq${playing ? ' is-playing' : ''}`}><i /><i /><i /></span>}
                    <span>{nameOf(episode)}</span>
                  </button>
                );
              })}
              {!visible.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('cms.site.detail.noLines')} />}
            </div>
            <p className="mp-note">{t('cms.site.watch.note')}</p>
          </aside>
        </div>
      </div>
    </ConfigProvider>
  );
}
