import React, { useEffect, useState } from 'react';
import { ConfigProvider, Input, theme } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { Link, history, useLocation, useSearchParams } from '@umijs/max';
import { SelectLang } from '@/components';
import { t, useI18n } from '@/i18n';
import { MEDIA_TYPES, browseURL } from './filters';
import './site.less';

export const siteTheme = { algorithm: theme.darkAlgorithm, token: { colorPrimary: '#3b82f6', borderRadius: 8 } };
export const typeLabel = (kind: string) => t(`cms.media.type.${kind as (typeof MEDIA_TYPES)[number]}`);

export default function SiteLayout({ children, wide = false }: { children: React.ReactNode; wide?: boolean }) {
  useI18n();
  const location = useLocation();
  const [params] = useSearchParams();
  const current = location.pathname === '/browse' ? params.get('type') || 'all' : location.pathname === '/' ? 'home' : '';
  const [query, setQuery] = useState(params.get('q') || '');
  useEffect(() => setQuery(params.get('q') || ''), [params.get('q')]);
  const nav = [{ key: 'home', to: '/', label: t('cms.site.nav.home') }, ...MEDIA_TYPES.map(kind => ({ key: kind, to: browseURL({ type: kind }), label: typeLabel(kind) }))];
  return (
    <ConfigProvider theme={siteTheme}>
      <div className="site-page">
        <header className="site-header">
          <div className={`site-header-inner${wide ? ' is-wide' : ''}`}>
            <Link to="/" className="site-brand"><img src="/logo.svg" alt="" />{t('cms.brandName')}</Link>
            <nav className="site-nav" aria-label={t('cms.site.nav.label')}>
              {nav.map(item => <Link key={item.key} to={item.to} className={current === item.key ? 'is-active' : undefined} aria-current={current === item.key ? 'page' : undefined}>{item.label}</Link>)}
            </nav>
            <Input.Search className="site-search" allowClear value={query} onChange={event => setQuery(event.target.value)} placeholder={t('cms.site.search.placeholder')} aria-label={t('cms.site.search.placeholder')}
              onSearch={value => history.push(browseURL({ q: value.trim() }))} />
            <div className="site-actions">
              <SelectLang />
              <a className="site-admin" href="/#/admin"><SettingOutlined />{t('cms.site.admin')}</a>
            </div>
          </div>
        </header>
        <main className={`site-main${wide ? ' is-wide' : ''}`}>{children}</main>
        <footer className="site-footer">{t('cms.site.footer')}</footer>
      </div>
    </ConfigProvider>
  );
}
