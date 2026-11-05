import React from 'react';
import { Tag, message } from 'antd';
import { t } from '@/i18n';
import type { MediaItem, PlayEpisode } from '@/services/media';
import './style.less';
export const timeText = (v?: string) => v && !v.startsWith('0001') ? new Date(v).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : '—';
export const modeNames: Record<string, string> = { health: '连通性检查', sample: '试运行', incremental: '增量采集', full: '全量采集' };
export const statusNames: Record<string, string> = { queued: '排队中', running: '执行中', success: '已完成', partial: '部分完成', failed: '失败', cancelled: '已取消', skipped: '已跳过', ok: '正常', error: '异常' };
const colors: Record<string, string> = { queued: 'blue', running: 'processing', success: 'green', partial: 'orange', failed: 'red', ok: 'green', error: 'red' };
export const StatusTag = ({ value }: { value?: string }) => <Tag color={colors[value || '']}>{statusNames[value || ''] || value || '未执行'}</Tag>;
export function Cover({ url, title, small = false, className }: { url?: string; title: string; small?: boolean; className?: string }) {
 const [failed, setFailed] = React.useState(false);
 React.useEffect(() => setFailed(false), [url]);
 // Resource sites commonly reject hotlinked images that carry a foreign Referer.
 return <div className={['media-cover', small && 'media-cover-small', className].filter(Boolean).join(' ')}>{url && !failed ? <img src={url} alt={title} loading="lazy" referrerPolicy="no-referrer" onError={() => setFailed(true)} /> : <div className="media-cover-placeholder"><span>{t('cms.media.cover.mark')}</span><small>{t('cms.media.cover.none')}</small></div>}</div>;
}

// ---- Library / player helpers ----
// Labels are read at render time so they follow the UI language; older rows stored the Chinese names as keys.
const typeMeta = (kind: 'movie' | 'series' | 'anime' | 'variety', color: string) => ({ get label() { return t(`cms.media.type.${kind}`); }, color });
export const mediaTypeMeta: Record<string, { label: string; color: string }> = {
 movie: typeMeta('movie', 'blue'), series: typeMeta('series', 'purple'),
 anime: typeMeta('anime', 'cyan'), variety: typeMeta('variety', 'gold'),
 电影: typeMeta('movie', 'blue'), 电视剧: typeMeta('series', 'purple'),
 动漫: typeMeta('anime', 'cyan'), 综艺: typeMeta('variety', 'gold'),
};
export const mediaTypeOptions = ['movie', 'series', 'anime', 'variety'];
export const splitList = (value?: string) => Array.from(new Set((value || '').split(/[,，/|、;；]+/).map(v => v.trim()).filter(Boolean)));
export const factsOf = (item: MediaItem) => [item.year, item.area, item.language].map(v => v?.trim()).filter((v): v is string => !!v);
// Upstream sites fill empty synopses with boilerplate and occasionally raw HTML.
export const describe = (value?: string) => {
 const plain = (value || '').replace(/<[^>]+>/g, '').replace(/&nbsp;/g, ' ').trim();
 return /^暂无(简介|介绍)/.test(plain) ? '' : plain;
};
export const TypeTags = ({ value }: { value?: string }) => {
 const tags = splitList((value || '').replace(/\s+/g, ',')).map(raw => ({ raw, ...(mediaTypeMeta[raw.toLowerCase()] || { label: raw, color: 'default' }) }));
 return <>{(tags.length ? tags : [{ raw: '', label: t('cms.media.type.none'), color: 'default' }]).map(tag => <Tag key={tag.raw || tag.label} bordered={false} color={tag.color} className="media-type-tag">{tag.label}</Tag>)}</>;
};
export const GenreTags = ({ value }: { value?: string }) => <>{splitList(value).map(tag => <Tag key={tag} bordered={false} className="media-genre-tag">{tag}</Tag>)}</>;
export const Facts = ({ item }: { item: MediaItem }) => factsOf(item).length ? <span className="media-facts">{factsOf(item).map(fact => <span key={fact}>{fact}</span>)}</span> : null;
export function People({ item, limit = 8 }: { item: MediaItem; limit?: number }) {
 const director = splitList(item.director), actors = splitList(item.actors);
 if (!director.length && !actors.length) return null;
 return <dl className="media-people">{director.length > 0 && <><dt>{t('cms.media.people.director')}</dt><dd>{director.join(' / ')}</dd></>}{actors.length > 0 && <><dt>{t('cms.media.people.cast')}</dt><dd>{actors.slice(0, limit).join(' / ')}{actors.length > limit ? t('cms.media.people.more') : ''}</dd></>}</dl>;
}

export const episodeTitle = (episode: Pick<PlayEpisode, 'episodeNo' | 'episodeName'>) =>
 episode.episodeName && !/^(m3u8|mp4|hls|播放)$/i.test(episode.episodeName.trim()) ? episode.episodeName : t('cms.media.episodeNo', { n: episode.episodeNo });
// The back office plays through the public player.
export const playerURL = (itemId: number, lineId?: number, episodeId?: number, restart = false) => {
 const params = new URLSearchParams();
 if (lineId) params.set('line', String(lineId));
 if (episodeId) params.set('ep', String(episodeId));
 if (restart) params.set('restart', '1');
 const query = params.toString();
 return '/#/watch/' + itemId + (query ? '?' + query : '');
};
export const openPlayer = (url: string) => window.open(url, '_blank', 'noopener,noreferrer');

// The CMS is often served over plain HTTP where navigator.clipboard is unavailable.
export const copyText = (value: string, success = '已复制') => {
 const fallback = () => {
  const area = document.createElement('textarea');
  area.value = value;
  area.setAttribute('readonly', '');
  area.style.position = 'fixed';
  area.style.opacity = '0';
  document.body.appendChild(area);
  area.select();
  const ok = document.execCommand('copy');
  document.body.removeChild(area);
  if (ok) message.success(success); else message.error('复制失败，请手动复制');
 };
 if (navigator.clipboard && window.isSecureContext) void navigator.clipboard.writeText(value).then(() => message.success(success)).catch(fallback);
 else fallback();
};

export type WatchProgress = { lineId: number; episodeId: number; episodeTitle: string; time: number; updatedAt: number };
const progressKey = (itemId: number) => 'media-progress:' + itemId;
export const readProgress = (itemId: number): WatchProgress | undefined => {
 try {
  const value = JSON.parse(localStorage.getItem(progressKey(itemId)) || 'null');
  return value?.episodeId ? value : undefined;
 } catch { return undefined; }
};
export const writeProgress = (itemId: number, value: WatchProgress) => {
 try { localStorage.setItem(progressKey(itemId), JSON.stringify(value)); } catch { /* storage may be disabled */ }
};
export const clockText = (seconds: number) => {
 const s = Math.max(0, Math.floor(seconds)), h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), r = s % 60;
 return (h ? h + ':' + String(m).padStart(2, '0') : String(m)) + ':' + String(r).padStart(2, '0');
};

// Range buttons for long episode lists; labels follow the displayed order.
export function EpisodeRanges({ total, size, reverse, value, onChange }: { total: number; size: number; reverse: boolean; value: number; onChange: (index: number) => void }) {
 const count = Math.ceil(total / size);
 if (count <= 1) return null;
 return <div className="media-ranges">{Array.from({ length: count }, (_, index) => {
  const from = index * size + 1, to = Math.min((index + 1) * size, total);
  const label = reverse ? `${total - from + 1}-${total - to + 1}` : `${from}-${to}`;
  return <button type="button" key={label} className={'media-range' + (index === value ? ' is-active' : '')} onClick={() => onChange(index)}>{label}</button>;
 })}</div>;
}
