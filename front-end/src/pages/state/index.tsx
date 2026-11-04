import { t, useI18n } from '@/i18n';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Alert, Button, Skeleton, Tag, Typography } from 'antd';
import {
  ApiOutlined,
  ArrowUpOutlined,
  CheckCircleFilled,
  ClockCircleOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  DeploymentUnitOutlined,
  LineChartOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  WarningFilled,
} from '@ant-design/icons';
import { createStyles } from 'antd-style';
import { getServerState, type SystemStatus } from '@/services/system/state';
const useStyles = createStyles(({ token }) => ({
  stack: {
    display: 'grid',
    gap: 24,
  },
  hero: {
    position: 'relative',
    overflow: 'hidden',
    border: `1px solid ${token.colorBorderSecondary}`,
    borderRadius: 16,
    padding: '32px 36px',
    background: `linear-gradient(115deg, ${token.colorBgContainer} 30%, ${token.colorPrimaryBg} 100%)`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 24,
    '@media (max-width: 700px)': {
      padding: 24,
      alignItems: 'flex-start',
      flexDirection: 'column',
    },
  },
  eyebrow: {
    color: token.colorTextSecondary,
    fontSize: 12,
    letterSpacing: 2,
    marginBottom: 16,
  },
  headline: {
    display: 'flex',
    alignItems: 'center',
    gap: 14,
    fontSize: 28,
    fontWeight: 600,
    lineHeight: 1.35,
    marginBottom: 10,
    '@media (max-width: 700px)': {
      fontSize: 23,
    },
  },
  heroMeta: {
    display: 'flex',
    gap: 20,
    flexWrap: 'wrap',
    color: token.colorTextSecondary,
    fontSize: 13,
  },
  heroIcon: {
    fontSize: 86,
    color: token.colorPrimary,
    opacity: 0.15,
    transform: 'rotate(-12deg)',
    marginRight: 12,
    '@media (max-width: 700px)': {
      display: 'none',
    },
  },
  sectionTitle: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'baseline',
    marginBottom: 14,
    gap: 12,
    '& h3': {
      fontSize: 16,
      fontWeight: 600,
      margin: 0,
    },
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, minmax(0, 1fr))',
    gap: 16,
    '@media (max-width: 1100px)': {
      gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
    },
    '@media (max-width: 550px)': {
      gridTemplateColumns: '1fr',
    },
  },
  service: {
    border: `1px solid ${token.colorBorderSecondary}`,
    background: token.colorBgContainer,
    borderRadius: 12,
    padding: 22,
    minWidth: 0,
  },
  serviceTop: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 24,
  },
  serviceIcon: {
    height: 40,
    width: 40,
    borderRadius: 10,
    background: token.colorFillAlter,
    display: 'grid',
    placeItems: 'center',
    fontSize: 20,
    color: token.colorPrimary,
  },
  serviceName: {
    fontSize: 16,
    fontWeight: 600,
    marginBottom: 6,
  },
  muted: {
    color: token.colorTextSecondary,
    fontSize: 13,
    lineHeight: 1.7,
  },
  lower: {
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1.3fr) minmax(0, 1fr)',
    gap: 20,
    '@media (max-width: 900px)': {
      gridTemplateColumns: '1fr',
    },
  },
  panel: {
    border: `1px solid ${token.colorBorderSecondary}`,
    background: token.colorBgContainer,
    borderRadius: 12,
    padding: 26,
    minWidth: 0,
  },
  facts: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    columnGap: 24,
    rowGap: 26,
    marginTop: 26,
    '& dt': {
      color: token.colorTextSecondary,
      fontSize: 12,
      marginBottom: 8,
    },
    '& dd': {
      margin: 0,
      fontSize: 16,
      fontWeight: 500,
      overflowWrap: 'anywhere',
    },
    '@media (max-width: 550px)': {
      gridTemplateColumns: '1fr',
    },
  },
  observability: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    gap: 18,
  },
  features: {
    display: 'flex',
    gap: 8,
    flexWrap: 'wrap',
    '& span': {
      padding: '5px 10px',
      background: token.colorFillAlter,
      color: token.colorTextSecondary,
      borderRadius: 6,
      fontSize: 12,
    },
  },
  footer: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    fontSize: 12,
    color: token.colorTextTertiary,
  },
}));
function uptime(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  if (minutes < 1) return t('cms.lessThanAMinute');
  if (minutes < 60)
    return t('cms.duration.minutes', {
      value0: minutes,
    });
  if (minutes < 1440)
    return t('cms.duration.hoursMinutes', {
      value0: Math.floor(minutes / 60),
      value1: minutes % 60,
    });
  return t('cms.duration.daysHours', {
    value0: Math.floor(minutes / 1440),
    value1: Math.floor((minutes % 1440) / 60),
  });
}
export default function SystemStatusPage() {
  const locale = useI18n();
  const services = [
    {
      key: 'backend',
      name: t('cms.backendService'),
      description: t('cms.handlesRequestsAndAuthorization'),
      icon: <CloudServerOutlined />,
    },
    {
      key: 'database',
      name: t('cms.database'),
      description: t('cms.storesAccountsMenusAndPlatformData'),
      icon: <DatabaseOutlined />,
    },
    {
      key: 'redis',
      name: 'Redis',
      description: t('cms.providesCachingAndSharedRateLimits'),
      icon: <DeploymentUnitOutlined />,
    },
    {
      key: 'mongo',
      name: 'MongoDB',
      description: t('cms.optionalDocumentStorage'),
      icon: <ApiOutlined />,
    },
  ];
  const { styles, theme } = useStyles();
  const [status, setStatus] = useState<SystemStatus>();
  const [error, setError] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [checkedAt, setCheckedAt] = useState<Date>();
  const mounted = useRef(false);
  const pending = useRef(false);
  const refresh = useCallback(async () => {
    if (pending.current) return;
    pending.current = true;
    setRefreshing(true);
    try {
      const result = await getServerState();
      if (mounted.current) {
        setError(result.code !== 0);
        if (result.code === 0) {
          setStatus(result.data.server);
          setCheckedAt(new Date());
        }
      }
    } catch {
      if (mounted.current) setError(true);
    } finally {
      pending.current = false;
      if (mounted.current) setRefreshing(false);
    }
  }, []);
  useEffect(() => {
    mounted.current = true;
    void refresh();
    const timer = setInterval(() => {
      if (!document.hidden) void refresh();
    }, 15000);
    return () => {
      mounted.current = false;
      clearInterval(timer);
    };
  }, [refresh]);
  const unavailable =
    status &&
    Object.values(status.checks).some((value) => value === 'unavailable');
  const active = status
    ? Object.values(status.checks).filter((value) => value !== 'disabled')
        .length
    : 0;
  const good = status
    ? Object.values(status.checks).filter((value) => value === 'ok').length
    : 0;
  const healthy = !!status && !unavailable && !error;
  const title = error
    ? t('cms.platformStatusIsUnconfirmed')
    : unavailable
      ? t('cms.someServicesNeedAttention')
      : status
        ? t('cms.platformIsHealthy')
        : t('cms.checkingPlatformStatus');
  return (
    <PageContainer
      title={t('cms.systemStatus')}
      subTitle={t('cms.serviceAvailabilityAndRuntimeOverview')}
      extra={
        <Button
          icon={<ReloadOutlined spin={refreshing} />}
          loading={refreshing}
          onClick={() => void refresh()}
        >
          {t('cms.refreshStatus')}
        </Button>
      }
    >
      <div className={styles.stack}>
        <section
          className={styles.hero}
          aria-label={t('cms.overallPlatformStatus')}
        >
          <div>
            <div className={styles.eyebrow}>
              {t('cms.serviceOverviewBrand')}
            </div>
            <div className={styles.headline} role="status">
              {status || error ? (
                healthy ? (
                  <CheckCircleFilled
                    style={{
                      color: theme.colorSuccess,
                    }}
                  />
                ) : (
                  <WarningFilled
                    style={{
                      color: theme.colorWarning,
                    }}
                  />
                )
              ) : (
                <ClockCircleOutlined />
              )}
              {title}
            </div>
            <div className={styles.heroMeta}>
              <span>
                {status
                  ? t('cms.enabledServiceChecksAreHealthy', {
                      value0: good,
                      value1: active,
                    })
                  : t('cms.connectingToTheBackend')}
              </span>
              <span>
                <ClockCircleOutlined />{' '}
                {checkedAt
                  ? t('cms.lastUpdatedAt', {
                      value0: checkedAt.toLocaleTimeString(locale, {
                        hour12: false,
                      }),
                    })
                  : t('cms.waitingForTheFirstCheck')}
              </span>
            </div>
          </div>
          <SafetyCertificateOutlined className={styles.heroIcon} />
        </section>
        {error && (
          <Alert
            type="warning"
            showIcon
            message={t('cms.statusUpdateFailed')}
            description={t('cms.showingTheLastSuccessfulCheckRefresh')}
          />
        )}
        {!status ? (
          !error && (
            <Skeleton
              active
              paragraph={{
                rows: 5,
              }}
            />
          )
        ) : (
          <>
            <section aria-label={t('cms.serviceAvailability')}>
              <div className={styles.sectionTitle}>
                <h3>{t('cms.serviceAvailability')}</h3>
                <span className={styles.muted}>
                  {t('cms.checkedAutomaticallyEvery15Seconds')}
                </span>
              </div>
              <div className={styles.grid}>
                {services.map((service) => {
                  const value = status.checks[service.key];
                  const label = error
                    ? t('cms.confirmationNeeded')
                    : value === 'ok'
                      ? t('cms.healthy')
                      : value === 'disabled'
                        ? t('cms.notEnabled')
                        : t('cms.unavailable');
                  return (
                    <article
                      className={styles.service}
                      key={service.key}
                      aria-label={`${service.name}：${label}`}
                    >
                      <div className={styles.serviceTop}>
                        <div className={styles.serviceIcon}>{service.icon}</div>
                        <Tag
                          bordered={false}
                          color={
                            error
                              ? 'warning'
                              : value === 'ok'
                                ? 'success'
                                : value === 'disabled'
                                  ? 'default'
                                  : 'error'
                          }
                          style={{
                            margin: 0,
                          }}
                        >
                          {label}
                        </Tag>
                      </div>
                      <div className={styles.serviceName}>{service.name}</div>
                      <div className={styles.muted}>{service.description}</div>
                    </article>
                  );
                })}
              </div>
            </section>
            <div className={styles.lower}>
              <section
                className={styles.panel}
                aria-label={t('cms.platformInformation')}
              >
                <div className={styles.sectionTitle}>
                  <h3>{t('cms.platformInformation')}</h3>
                  <Tag bordered={false}>{t('cms.currentInstance')}</Tag>
                </div>
                <dl className={styles.facts}>
                  <div>
                    <dt>{t('cms.platformVersion')}</dt>
                    <dd>{status.version}</dd>
                  </div>
                  <div>
                    <dt>{t('cms.instanceUptime')}</dt>
                    <dd>{uptime(status.uptimeSeconds)}</dd>
                  </div>
                  <div>
                    <dt>{t('cms.goRuntime')}</dt>
                    <dd>{status.goVersion}</dd>
                  </div>
                  <div>
                    <dt>{t('cms.databaseSchema')}</dt>
                    <dd
                      style={{
                        fontSize: 12,
                        fontFamily: 'monospace',
                      }}
                    >
                      {status.schemaVersion}
                    </dd>
                  </div>
                </dl>
              </section>
              <section
                className={`${styles.panel} ${styles.observability}`}
                aria-label={t('cms.observability')}
              >
                <div
                  className={styles.sectionTitle}
                  style={{
                    marginBottom: 0,
                    width: '100%',
                  }}
                >
                  <h3>
                    <LineChartOutlined
                      style={{
                        marginRight: 8,
                        color: theme.colorPrimary,
                      }}
                    />
                    {t('cms.explorePlatformActivity')}
                  </h3>
                </div>
                <Typography.Paragraph
                  className={styles.muted}
                  style={{
                    margin: 0,
                  }}
                >
                  {t('cms.exploreHistoricalTrendsApplicationLogsAnd')}
                </Typography.Paragraph>
                <div className={styles.features}>
                  <span>{t('cms.requestTrends')}</span>
                  <span>{t('cms.applicationLogs')}</span>
                  <span>{t('cms.traces')}</span>
                  <span>{t('cms.alerts')}</span>
                </div>
                {status.observabilityEnabled ? (
                  <Button
                    type="primary"
                    href="/observability/d/base-frame/base-frame"
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{
                      marginTop: 'auto',
                    }}
                  >
                    {t('cms.openObservability')}
                    <ArrowUpOutlined rotate={45} />
                  </Button>
                ) : (
                  <Tag
                    style={{
                      marginTop: 'auto',
                    }}
                    bordered={false}
                  >
                    {t('cms.observabilityIsNotEnabled')}
                  </Tag>
                )}
              </section>
            </div>
          </>
        )}
        <div className={styles.footer}>
          <SafetyCertificateOutlined />
          {t('cms.thisPageShowsCurrentServiceStatus')}
        </div>
      </div>
    </PageContainer>
  );
}
