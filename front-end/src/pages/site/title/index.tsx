import React from 'react';
import { useParams } from '@umijs/max';
import { t, useI18n } from '@/i18n';
import SiteLayout from '../SiteLayout';
import TitleDetail from '../TitleDetail';
import { Nothing } from '../components';

// Shared links land here; inside the site, details open as an overlay instead.
export default function SiteTitle() {
  useI18n();
  const id = Number(useParams().id) || 0;
  return <SiteLayout>{id ? <div className="site-title-page"><TitleDetail id={id} /></div> : <Nothing text={t('cms.site.empty.title')} />}</SiteLayout>;
}
