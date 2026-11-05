import React from 'react';
import { Button, Drawer } from 'antd';
import { CloseOutlined, ExpandOutlined } from '@ant-design/icons';
import { Link } from '@umijs/max';
import { t } from '@/i18n';
import TitleDetail from './TitleDetail';
import { titleURL, useTitleOverlay } from './components';

// The details of `?title=` over the current page; the list underneath keeps its state.
export default function TitleOverlay() {
  const { openId, close } = useTitleOverlay();
  return (
    <Drawer open={!!openId} onClose={close} width="min(980px, 100vw)" destroyOnHidden rootClassName="site-overlay" closable={false} styles={{ header: { display: 'none' } }}>
      <div className="site-overlay-tools">
        {openId && <Link to={titleURL(openId)} title={t('cms.site.detail.openPage')}><Button type="text" icon={<ExpandOutlined />} aria-label={t('cms.site.detail.openPage')} /></Link>}
        <Button type="text" icon={<CloseOutlined />} onClick={close} aria-label={t('cms.site.detail.close')} />
      </div>
      {openId && <TitleDetail id={openId} />}
    </Drawer>
  );
}
