import React from 'react';
import { Button, Empty, Result, Skeleton } from 'antd';
import { ReloadOutlined, RightOutlined } from '@ant-design/icons';
import { Link } from '@umijs/max';
import { t } from '@/i18n';
import type { SiteCard } from '@/services/site';
import { Cover, splitList } from '../media/shared';
import { typeLabel } from './SiteLayout';

export const titleURL = (id: number) => `/title/${id}`;

const badge = (card: SiteCard) => {
  const kinds = splitList(card.mediaType);
  if (!card.episodes || (kinds.includes('movie') && card.episodes <= 1)) return '';
  return t('cms.site.card.episodes', { n: card.episodes });
};

export function PosterCard({ card, note }: { card: SiteCard; note?: string }) {
  const meta = [card.year, card.area, splitList(card.mediaType).map(typeLabel)[0]].filter(Boolean).join(' · ');
  const label = badge(card);
  return (
    <Link to={titleURL(card.id)} className="site-card">
      <div className="site-card-poster"><Cover url={card.coverUrl} title={card.title} />{label && <span className="site-card-badge">{label}</span>}</div>
      <div className="site-card-title" title={card.title}>{card.title}</div>
      <div className="site-card-meta">{note || meta || ' '}</div>
    </Link>
  );
}

export function PosterGrid({ cards }: { cards: SiteCard[] }) {
  return <div className="site-grid">{cards.map(card => <PosterCard key={card.id} card={card} />)}</div>;
}

const SKELETON_KEYS = Array.from({ length: 48 }, (_, i) => `skeleton-${i}`);

export function GridSkeleton({ count = 12 }: { count?: number }) {
  return <div className="site-grid" aria-busy="true">{SKELETON_KEYS.slice(0, count).map(key => <div key={key} className="site-card"><Skeleton.Node active className="site-card-skeleton"><span /></Skeleton.Node><Skeleton active title={false} paragraph={{ rows: 2, width: ['80%', '50%'] }} /></div>)}</div>;
}

export function PosterRow({ title, more, children }: { title: string; more?: string; children: React.ReactNode }) {
  return (
    <section className="site-row">
      <div className="site-row-head"><h2>{title}</h2>{more && <Link to={more} className="site-row-more">{t('cms.site.row.more')}<RightOutlined /></Link>}</div>
      <div className="site-row-track">{children}</div>
    </section>
  );
}

export function LoadFailed({ onRetry }: { onRetry: () => void }) {
  return <Result status="warning" title={t('cms.site.error.title')} subTitle={t('cms.site.error.detail')} extra={<Button type="primary" icon={<ReloadOutlined />} onClick={onRetry}>{t('cms.site.error.retry')}</Button>} />;
}

export function Nothing({ text, action }: { text: string; action?: React.ReactNode }) {
  return <Empty className="site-empty" image={Empty.PRESENTED_IMAGE_SIMPLE} description={text}>{action}</Empty>;
}
