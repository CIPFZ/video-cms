import { DesktopOutlined, GlobalOutlined, QuestionCircleOutlined } from '@ant-design/icons';
import { setLocale } from '@umijs/max';
import { Button, Dropdown } from 'antd';
import { t, useI18n } from '@/i18n';

export type SiderTheme = 'light' | 'dark';

export const SelectLang: React.FC = () => {
  const locale = useI18n();
  return (
    <Dropdown
      trigger={['click']}
      menu={{
        selectedKeys: [locale],
        items: [
          { key: 'zh-CN', label: '简体中文' },
          { key: 'en-US', label: 'English' },
        ],
        onClick: ({ key }) => {
          setLocale(key, false);
        },
      }}
    >
      <Button
        type="text"
        aria-label={t('cms.language.change')}
        title={t('cms.language.change')}
        icon={<GlobalOutlined />}
      />
    </Dropdown>
  );
};

export const Question: React.FC = () => {
  const locale = useI18n();
  return (
    <a
      href="https://github.com/CIPFZ/video-cms"
      target="_blank"
      rel="noreferrer"
      aria-label={t('cms.projectDocs')}
      style={{
        display: 'inline-flex',
        padding: 4,
        fontSize: 18,
        color: 'inherit',
      }}
    >
      <QuestionCircleOutlined />
    </a>
  );
};

// Back office to the public site: same tab, the site needs no login.
export const SiteLink: React.FC = () => {
  useI18n();
  return (
    <a
      href="/#/"
      aria-label={t('cms.site.visit')}
      title={t('cms.site.visit')}
      style={{
        display: 'inline-flex',
        padding: 4,
        fontSize: 18,
        color: 'inherit',
      }}
    >
      <DesktopOutlined />
    </a>
  );
};
