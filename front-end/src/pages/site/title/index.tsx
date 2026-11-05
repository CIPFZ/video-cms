import React from 'react';
import { useParams } from '@umijs/max';
import { t, useI18n } from '@/i18n';
import SiteLayout from '../SiteLayout';
import TitleDetail from '../TitleDetail';
import { Nothing } from '../components';

export default function SiteTitle() {
  useI18n();
  const id = Number(useParams().id) || 0;
  return <SiteLayout>{id ? <TitleDetail id={id} /> : <Nothing text={t('cms.site.empty.title')} />}</SiteLayout>;
}
