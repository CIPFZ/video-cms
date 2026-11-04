import { t, useI18n } from '@/i18n';
import { PageContainer } from '@ant-design/pro-components';
import { Button, Space } from 'antd';
import {
  ApiOutlined,
  AppstoreOutlined,
  ArrowRightOutlined,
  ArrowUpOutlined,
  BellOutlined,
  BookOutlined,
  BranchesOutlined,
  CloudServerOutlined,
  CodeOutlined,
  DatabaseOutlined,
  DeploymentUnitOutlined,
  GithubOutlined,
  HistoryOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { useStyles } from './styles';

const repository = 'https://github.com/CIPFZ/video-cms/tree/main';

export default function AboutPage() {
  useI18n();
  const { styles } = useStyles();
  const capabilities = [
    {
      icon: <SafetyCertificateOutlined />,
      title: t('cms.about.accounts'),
      description: t('cms.about.accountsDescription'),
    },
    {
      icon: <AppstoreOutlined />,
      title: t('cms.about.navigation'),
      description: t('cms.about.navigationDescription'),
    },
    {
      icon: <ApiOutlined />,
      title: t('cms.about.integration'),
      description: t('cms.about.integrationDescription'),
    },
    {
      icon: <BellOutlined />,
      title: t('cms.about.notices'),
      description: t('cms.about.noticesDescription'),
    },
    {
      icon: <HistoryOutlined />,
      title: t('cms.about.audit'),
      description: t('cms.about.auditDescription'),
    },
    {
      icon: <CloudServerOutlined />,
      title: t('cms.about.operations'),
      description: t('cms.about.operationsDescription'),
    },
  ];
  const technologies = [
    {
      icon: <CodeOutlined />,
      name: t('cms.frontend'),
      description: t('cms.about.frontendDescription'),
      items: ['React', 'UmiJS', 'TypeScript', 'Ant Design'],
    },
    {
      icon: <ApiOutlined />,
      name: t('cms.backend'),
      description: t('cms.about.backendDescription'),
      items: ['Go', 'Gin', 'GORM', 'Casbin'],
    },
    {
      icon: <DatabaseOutlined />,
      name: t('cms.about.data'),
      description: t('cms.about.dataDescription'),
      items: ['MySQL / PostgreSQL / SQLite', 'Redis'],
    },
    {
      icon: <DeploymentUnitOutlined />,
      name: t('cms.about.delivery'),
      description: t('cms.about.deliveryDescription'),
      items: ['Docker Compose', 'OpenTelemetry'],
    },
  ];
  const resources = [
    {
      icon: <CloudServerOutlined />,
      title: t('cms.about.deploymentGuide'),
      description: t('cms.about.deploymentGuideDescription'),
      href: `${repository}/deploy/local/README.md`,
    },
    {
      icon: <ApiOutlined />,
      title: t('cms.about.apiReference'),
      description: t('cms.about.apiReferenceDescription'),
      href: '/api/v1/swagger/index.html',
    },
    {
      icon: <BookOutlined />,
      title: t('cms.about.maintenanceGuide'),
      description: t('cms.about.maintenanceGuideDescription'),
      href: `${repository}/docs/REMEDIATION.md`,
    },
  ];
  return (
    <PageContainer title={false}>
      <div className={styles.page}>
        <section className={styles.hero} aria-labelledby="about-title">
          <div className={styles.intro}>
            <div className={styles.eyebrow}>
              <span className={styles.brandMark} aria-hidden>
                <DeploymentUnitOutlined />
              </span>
              {t('cms.about.foundation')}
            </div>
            <h1 id="about-title">{t('cms.brandName')}</h1>
            <p className={styles.subtitle}>{t('cms.about.tagline')}</p>
            <p className={styles.description}>{t('cms.about.description')}</p>
            <Space size={12} wrap className={styles.actions}>
              <Button
                type="primary"
                icon={<BookOutlined />}
                href={`${repository}/docs`}
                target="_blank"
                rel="noopener noreferrer"
              >
                {t('cms.viewDocumentation')}
              </Button>
              <Button
                icon={<GithubOutlined />}
                href={repository}
                target="_blank"
                rel="noopener noreferrer"
              >
                {t('cms.sourceRepository')}
              </Button>
            </Space>
            <div className={styles.branch}>
              <BranchesOutlined aria-hidden />
              <span>{t('cms.about.baseBranch')}</span>
              <code>main</code>
            </div>
          </div>
          <div
            className={styles.architecture}
            role="group"
            aria-label={t('cms.about.architecture')}
          >
            <div className={styles.diagramHeading}>
              {t('cms.about.architecture')}
            </div>
            <div className={styles.extension}>
              <div className={styles.layerHeading}>
                <AppstoreOutlined aria-hidden />
                <strong>{t('cms.about.businessLayer')}</strong>
                <span>{t('cms.about.extensible')}</span>
              </div>
              <p>{t('cms.about.businessLayerDescription')}</p>
            </div>
            <div className={styles.connector} aria-hidden>
              <ArrowUpOutlined />
            </div>
            <div className={styles.coreLayer}>
              <div className={styles.layerHeading}>
                <SafetyCertificateOutlined aria-hidden />
                <strong>{t('cms.about.sharedLayer')}</strong>
              </div>
              <div className={styles.moduleList}>
                <span>{t('cms.about.accounts')}</span>
                <span>{t('cms.about.navigation')}</span>
                <span>{t('cms.about.notices')}</span>
              </div>
            </div>
            <div className={styles.foundationLayer}>
              <DatabaseOutlined aria-hidden />
              <span>{t('cms.about.foundationLayer')}</span>
            </div>
            <p className={styles.diagramCaption}>
              {t('cms.about.architectureCaption')}
            </p>
          </div>
        </section>

        <section aria-labelledby="about-capabilities">
          <div className={styles.sectionHeading}>
            <div>
              <h2 id="about-capabilities">{t('cms.about.capabilities')}</h2>
              <p>{t('cms.about.capabilitiesDescription')}</p>
            </div>
            <span className={styles.sectionNote}>
              {t('cms.about.capabilitiesNote')}
            </span>
          </div>
          <div className={styles.capabilityGrid}>
            {capabilities.map((item) => (
              <article className={styles.capability} key={item.title}>
                <span className={styles.icon} aria-hidden>
                  {item.icon}
                </span>
                <div>
                  <h3>{item.title}</h3>
                  <p>{item.description}</p>
                </div>
              </article>
            ))}
          </div>
        </section>

        <div className={styles.details}>
          <section className={styles.panel} aria-labelledby="about-tech">
            <div className={styles.sectionHeading}>
              <div>
                <h2 id="about-tech">{t('cms.technologyStack')}</h2>
                <p>{t('cms.about.technologyDescription')}</p>
              </div>
            </div>
            <div className={styles.techList}>
              {technologies.map((item) => (
                <div className={styles.techRow} key={item.name}>
                  <div className={styles.techLabel}>
                    <span aria-hidden>{item.icon}</span>
                    <div>
                      <h3>{item.name}</h3>
                      <p>{item.description}</p>
                    </div>
                  </div>
                  <ul className={styles.tags} aria-label={item.name}>
                    {item.items.map((name) => (
                      <li key={name}>{name}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </section>
          <section className={styles.panel} aria-labelledby="about-resources">
            <div className={styles.sectionHeading}>
              <div>
                <h2 id="about-resources">{t('cms.about.resources')}</h2>
                <p>{t('cms.about.resourcesDescription')}</p>
              </div>
            </div>
            <div className={styles.resourceList}>
              {resources.map((item) => (
                <a
                  className={styles.resource}
                  key={item.href}
                  href={item.href}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <span className={styles.resourceIcon} aria-hidden>
                    {item.icon}
                  </span>
                  <span>
                    <strong>{item.title}</strong>
                    <small>{item.description}</small>
                  </span>
                  <ArrowRightOutlined aria-hidden />
                </a>
              ))}
            </div>
            <div className={styles.resourceNote}>
              <GithubOutlined aria-hidden />
              <span>{t('cms.about.repositoryNote')}</span>
            </div>
          </section>
        </div>
      </div>
    </PageContainer>
  );
}
