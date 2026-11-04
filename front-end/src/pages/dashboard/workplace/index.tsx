import { menuLabel } from '@/i18n/menus';
import { t, useI18n, formatDate } from '@/i18n';
import React, { useMemo, useRef, useState } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Link, useModel, useRequest } from '@umijs/max';
import {
  Alert,
  Avatar,
  Button,
  Modal,
  Pagination,
  Skeleton,
  Space,
  Tag,
  message,
} from 'antd';
import {
  ApiOutlined,
  AppstoreOutlined,
  ArrowRightOutlined,
  BellOutlined,
  CheckCircleFilled,
  CloudServerOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  DeploymentUnitOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
  WarningFilled,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { getComponent } from '@/routing/componentMap';
import { getServerState } from '@/services/system/state';
import { markNoticeRead, type MyNotice } from '@/services/system/notice';
import { noticePageSize, useNotices } from './useNotices';
import { useStyles } from './styles';
interface WorkplaceMenu {
  path: string;
  name?: string;
  nameEn?: string;
  locale?: string;
  component?: string;
  icon?: React.ReactNode;
  hideInMenu?: boolean;
  routes?: WorkplaceMenu[];
}
function pageMenus(
  menus: WorkplaceMenu[],
  includeHidden = false,
): WorkplaceMenu[] {
  return menus.flatMap((menu) => {
    if (
      (!includeHidden && menu.hideInMenu) ||
      !menu.path?.startsWith('/') ||
      menu.path.startsWith('//')
    )
      return [];
    if (menu.routes?.length) return pageMenus(menu.routes, includeHidden);
    return menu.component &&
      getComponent(menu.component) &&
      menu.component !== 'components/RouterLayout'
      ? [menu]
      : [];
  });
}
export default function Workplace() {
  const locale = useI18n();
  const descriptions: Record<string, string> = {
    'sys/user': t('cms.manageAccountsAndMembers'),
    'sys/authority': t('cms.assignRolesAndAccess'),
    'sys/menu': t('cms.configureNavigationAndPages'),
    'sys/api': t('cms.manageApiResources'),
    'sys/api-token': t('cms.manageProgrammaticAccess'),
    'sys/notice': t('cms.publishAndDeliverNotices'),
    'sys/operation': t('cms.reviewActivityAndChanges'),
    state: t('cms.checkServiceHealth'),
    about: t('cms.explorePlatformInformation'),
  };
  const services = [
    {
      key: 'backend',
      name: t('cms.backendService'),
      icon: <CloudServerOutlined />,
    },
    {
      key: 'database',
      name: t('cms.database'),
      icon: <DatabaseOutlined />,
    },
    {
      key: 'redis',
      name: 'Redis',
      icon: <DeploymentUnitOutlined />,
    },
    {
      key: 'mongo',
      name: 'MongoDB',
      icon: <ApiOutlined />,
    },
  ];
  const levels = {
    info: {
      label: t('cms.notice'),
      color: 'blue',
    },
    warning: {
      label: t('cms.reminder'),
      color: 'orange',
    },
    error: {
      label: t('cms.important'),
      color: 'red',
    },
  };
  const { styles, theme } = useStyles();
  const { initialState } = useModel('@@initialState');
  const user = initialState?.currentUser;
  const menus = initialState?.menuData;
  const shortcuts = useMemo(
    () =>
      pageMenus(menus || []).filter(
        (menu) => menu.component !== 'dashboard/workplace',
      ),
    [menus],
  );
  const allPages = useMemo(() => pageMenus(menus || [], true), [menus]);
  const statePage = shortcuts.find((menu) => menu.component === 'state');
  const accountPage = allPages.find((menu) => menu.component === 'user/info');
  const serverQuery = useRequest(
    async () => {
      const result = await getServerState();
      if (result.code !== 0)
        throw new Error(result.msg || t('cms.unableToLoadStatus'));
      return {
        data: {
          ...result.data,
          checkedAt: new Date(),
        },
      };
    },
    {
      ready: !!statePage,
      pollingInterval: 15000,
      pollingWhenHidden: false,
    },
  );
  const server = serverQuery.data?.server;
  const serverError = !!serverQuery.error;
  const unavailable =
    server && Object.values(server.checks).includes('unavailable');
  const healthLabel = serverError
    ? t('cms.statusUnconfirmed')
    : !server
      ? t('cms.checkingStatus')
      : unavailable
        ? t('cms.someServicesAreUnavailable')
        : t('cms.servicesAreHealthy');
  const healthColor =
    serverError || unavailable
      ? theme.colorWarning
      : !server
        ? theme.colorTextSecondary
        : theme.colorSuccess;
  const notices = useNotices();
  const [selected, setSelected] = useState<MyNotice>();
  const [marking, setMarking] = useState(false);
  const markingRef = useRef(false);
  const [messageApi, messageContext] = message.useMessage();
  const now = new Date();
  const hour = now.getHours();
  const greeting =
    hour < 6
      ? t('cms.goodEvening')
      : hour < 12
        ? t('cms.goodMorning')
        : hour < 18
          ? t('cms.goodAfternoon')
          : t('cms.goodEvening.509f35');
  const nickname = user?.nickName || user?.username || t('cms.user');
  const role = user?.authority?.authorityName || t('cms.noRoleAssigned');
  const unread =
    notices.data?.list.filter((notice) => !notice.readAt).length || 0;
  const confirmRead = async () => {
    if (!selected || markingRef.current) return;
    markingRef.current = true;
    setMarking(true);
    try {
      const result = await markNoticeRead({
        noticeId: selected.ID,
      });
      if (result.code !== 0)
        throw new Error(result.msg || t('cms.operationFailed'));
      messageApi.success(t('cms.markedAsRead'));
      setSelected(undefined);
      notices.refresh();
    } catch {
      messageApi.error(t('cms.unableToMarkAsReadPlease'));
    } finally {
      markingRef.current = false;
      setMarking(false);
    }
  };
  return (
    <PageContainer
      title={t('cms.workplace')}
      subTitle={t('cms.startYourDayHere')}
    >
      {messageContext}
      <div className={styles.stack}>
        <section className={styles.hero} aria-label={t('cms.workOverview')}>
          <div className={styles.welcome}>
            <div>
              <div className={styles.eyebrow}>
                {t('cms.workOverviewBrand')}
              </div>
              <h1 className={styles.greeting}>
                {t('cms.welcomeUser', { greeting, name: nickname })}
              </h1>
              <div className={styles.muted}>
                {t('cms.yourToolsNoticesAndPlatformStatus')}
              </div>
            </div>
            <time
              className={styles.calendar}
              dateTime={dayjs(now).format('YYYY-MM-DD')}
            >
              <strong>{String(now.getDate()).padStart(2, '0')}</strong>
              <span>
                {formatDate(now, { year: 'numeric', month: 'long' })}
                <br />
                {now.toLocaleDateString(locale, {
                  weekday: 'long',
                })}
              </span>
            </time>
          </div>
          <div className={styles.summary}>
            <div className={styles.metric}>
              <div className={styles.icon}>
                <AppstoreOutlined />
              </div>
              <div>
                <div className={styles.muted}>{t('cms.availableTools')}</div>
                <strong>{shortcuts.length}</strong>
                <span className={styles.muted}>
                  {' '}
                  {t('cms.toolsUnit', { count: shortcuts.length })}
                </span>
              </div>
            </div>
            <div className={styles.metric}>
              <div
                className={styles.icon}
                style={{
                  color: theme.colorWarning,
                  background: theme.colorWarningBg,
                }}
              >
                <BellOutlined />
              </div>
              <div>
                <div className={styles.muted}>{t('cms.personalNotices')}</div>
                <strong>
                  {notices.loading || notices.error
                    ? '—'
                    : (notices.data?.total ?? 0)}
                </strong>
                <span className={styles.muted}>
                  {' '}
                  {t('cms.noticesUnit', { count: notices.data?.total || 0 })}
                </span>
              </div>
            </div>
            <div className={styles.metric}>
              <div
                className={styles.icon}
                style={{
                  color: theme.colorSuccess,
                  background: theme.colorSuccessBg,
                }}
              >
                <SafetyCertificateOutlined />
              </div>
              <div>
                <div className={styles.muted}>{t('cms.currentRole')}</div>
                <strong
                  style={{
                    fontSize: 17,
                  }}
                >
                  {role}
                </strong>
              </div>
            </div>
          </div>
        </section>
        <div className={styles.columns}>
          <div className={styles.stack}>
            <section className={styles.panel} aria-label={t('cms.quickAccess')}>
              <div className={styles.sectionTitle}>
                <h2>{t('cms.quickAccess')}</h2>
                <span className={styles.muted}>
                  {t('cms.yourToolsOneClickAway')}
                </span>
              </div>
              {shortcuts.length ? (
                <nav
                  className={styles.links}
                  aria-label={t('cms.workplaceShortcuts')}
                >
                  {shortcuts.map((menu) => (
                    <Link
                      className={styles.quickLink}
                      to={menu.path}
                      key={menu.path}
                      aria-label={menuLabel(menu)}
                    >
                      <div className={styles.linkTop}>
                        <span className={styles.linkIcon}>
                          {menu.icon || <AppstoreOutlined />}
                        </span>
                        <ArrowRightOutlined />
                      </div>
                      <div>
                        <strong>{menuLabel(menu)}</strong>
                        <br />
                        <small>
                          {descriptions[menu.component || ''] ||
                            t('cms.openThisPage')}
                        </small>
                      </div>
                    </Link>
                  ))}
                </nav>
              ) : (
                <div className={styles.empty}>
                  <AppstoreOutlined className={styles.emptyIcon} />
                  <h3>{t('cms.noToolsAvailable')}</h3>
                  <span className={styles.muted}>
                    {t('cms.contactAnAdministratorToRequestAccess')}
                  </span>
                </div>
              )}
            </section>
            <section className={styles.panel} aria-label={t('cms.myNotices')}>
              <div className={styles.sectionTitle}>
                <h2>{t('cms.myNotices')}</h2>
                <Button
                  type="text"
                  size="small"
                  icon={<ReloadOutlined />}
                  loading={notices.loading}
                  onClick={notices.refresh}
                >
                  {t('cms.refreshNotices')}
                </Button>
              </div>
              {notices.error ? (
                <Alert
                  type="warning"
                  showIcon
                  message={t('cms.unableToLoadNotices')}
                  description={t(
                    'cms.noticesAreTemporarilyUnavailablePleaseTry',
                  )}
                  action={
                    <Button size="small" onClick={notices.refresh}>
                      {t('cms.retry')}
                    </Button>
                  }
                />
              ) : notices.loading ? (
                <Skeleton
                  active
                  paragraph={{
                    rows: 4,
                  }}
                />
              ) : (
                <>
                  <div className={styles.noticeMeta}>
                    <span>
                      {t('cms.noticeTotal', {
                        count: notices.data?.total || 0,
                      })}
                    </span>
                    <span>{t('cms.noticePageUnread', { count: unread })}</span>
                  </div>
                  {!notices.data?.list.length ? (
                    <div className={styles.empty}>
                      <div className={styles.emptyIcon}>
                        <BellOutlined />
                      </div>
                      <h3>{t('cms.noNoticesYet')}</h3>
                      <span className={styles.muted}>
                        {t('cms.announcementsAndRemindersSentToYou')}
                      </span>
                    </div>
                  ) : (
                    notices.data.list.map((notice) => {
                      const level = levels[notice.level] || levels.info;
                      return (
                        <article
                          key={notice.ID}
                          className={styles.notice}
                          aria-label={notice.title}
                        >
                          <div className={styles.noticeHeading}>
                            {!notice.readAt && (
                              <span
                                className={styles.unreadDot}
                                aria-hidden="true"
                              />
                            )}
                            <Tag
                              color={level.color}
                              bordered={false}
                              style={{
                                margin: 0,
                              }}
                            >
                              {level.label}
                            </Tag>
                            <h3>{notice.title}</h3>
                            {notice.needConfirm && !notice.readAt && (
                              <Tag bordered={false}>
                                {t('cms.confirmationNeeded')}
                              </Tag>
                            )}
                          </div>
                          <p className={styles.noticePreview}>
                            {notice.content}
                          </p>
                          <div className={styles.noticeFooter}>
                            <span>
                              {formatDate(notice.createdAt)} ·{' '}
                              {notice.readAt ? t('cms.read') : t('cms.unread')}
                            </span>
                            <Button
                              type="link"
                              size="small"
                              onClick={() => setSelected(notice)}
                            >
                              {t('cms.viewDetails')}
                              <ArrowRightOutlined />
                            </Button>
                          </div>
                        </article>
                      );
                    })
                  )}
                  {(notices.data?.total || 0) > noticePageSize && (
                    <div className={styles.pagination}>
                      <Pagination
                        size="small"
                        current={notices.page}
                        total={notices.data?.total}
                        pageSize={noticePageSize}
                        showSizeChanger={false}
                        onChange={notices.changePage}
                      />
                    </div>
                  )}
                </>
              )}
            </section>
          </div>
          <aside className={styles.stack}>
            {statePage && (
              <section
                className={styles.panel}
                aria-label={t('cms.serviceOverview')}
              >
                <div className={styles.sectionTitle}>
                  <h2>{t('cms.serviceOverview')}</h2>
                  <Button
                    type="text"
                    size="small"
                    aria-label={t('cms.refreshServiceOverview')}
                    icon={<ReloadOutlined spin={serverQuery.loading} />}
                    loading={serverQuery.loading}
                    onClick={serverQuery.refresh}
                  />
                </div>
                <div
                  className={styles.health}
                  role="status"
                  style={{
                    color: healthColor,
                    background:
                      serverError || unavailable
                        ? theme.colorWarningBg
                        : !server
                          ? theme.colorFillAlter
                          : theme.colorSuccessBg,
                  }}
                >
                  {serverError || unavailable ? (
                    <WarningFilled />
                  ) : !server ? (
                    <ClockCircleOutlined />
                  ) : (
                    <CheckCircleFilled />
                  )}
                  {healthLabel}
                </div>
                {serverError ? (
                  <div
                    className={styles.muted}
                    style={{
                      marginTop: 14,
                    }}
                  >
                    {t('cms.unableToUpdateStatusRefreshTo')}
                  </div>
                ) : !server ? (
                  <Skeleton
                    active
                    title={false}
                    paragraph={{
                      rows: 4,
                    }}
                  />
                ) : (
                  services.map((service) => {
                    const status = server.checks[service.key];
                    return (
                      <div key={service.key} className={styles.service}>
                        <span className={styles.serviceName}>
                          {service.icon}
                          {service.name}
                        </span>
                        <Tag
                          bordered={false}
                          color={
                            status === 'ok'
                              ? 'success'
                              : status === 'disabled'
                                ? 'default'
                                : 'warning'
                          }
                        >
                          {status === 'ok'
                            ? t('cms.healthy')
                            : status === 'disabled'
                              ? t('cms.notEnabled')
                              : t('cms.unavailable')}
                        </Tag>
                      </div>
                    );
                  })
                )}
                <div className={styles.panelFooter}>
                  <span>
                    {serverQuery.data?.checkedAt
                      ? t('cms.updatedAt', {
                          value0: dayjs(serverQuery.data.checkedAt).format(
                            'HH:mm:ss',
                          ),
                        })
                      : t('cms.checkedEvery15Seconds')}
                  </span>
                  <Link to={statePage.path}>
                    {t('cms.viewFullStatus')}
                    <ArrowRightOutlined />
                  </Link>
                </div>
              </section>
            )}
            <section
              className={styles.panel}
              aria-label={t('cms.currentAccount')}
            >
              <div className={styles.sectionTitle}>
                <h2>{t('cms.currentAccount')}</h2>
                <Tag
                  bordered={false}
                  color="blue"
                  style={{
                    margin: 0,
                  }}
                >
                  {t('cms.signedIn')}
                </Tag>
              </div>
              <div className={styles.account}>
                <Avatar
                  size={44}
                  src={user?.avatar}
                  icon={<UserOutlined />}
                  style={{
                    flexShrink: 0,
                    background: theme.colorPrimaryBg,
                    color: theme.colorPrimary,
                  }}
                />
                <div
                  style={{
                    minWidth: 0,
                  }}
                >
                  <strong>{nickname}</strong>
                  <span className={styles.muted}>@{user?.username}</span>
                </div>
              </div>
              <dl className={styles.accountFacts}>
                <div>
                  <dt>{t('cms.currentRole')}</dt>
                  <dd>{role}</dd>
                </div>
                <div>
                  <dt>{t('cms.contactEmail')}</dt>
                  <dd>{user?.email || t('cms.notSet')}</dd>
                </div>
              </dl>
              {accountPage && (
                <div className={styles.panelFooter}>
                  <span>{t('cms.manageYourProfileAndAccountSecurity')}</span>
                  <Link to={accountPage.path}>
                    {t('cms.accountSettings')}
                    <ArrowRightOutlined />
                  </Link>
                </div>
              )}
            </section>
          </aside>
        </div>
      </div>
      <Modal
        open={!!selected}
        title={selected?.title}
        onCancel={() => {
          if (!marking) setSelected(undefined);
        }}
        maskClosable={!marking}
        closable={!marking}
        keyboard={!marking}
        footer={
          <Space>
            <Button disabled={marking} onClick={() => setSelected(undefined)}>
              {t('cms.close')}
            </Button>
            {selected && !selected.readAt && (
              <Button
                type="primary"
                loading={marking}
                onClick={() => void confirmRead()}
              >
                {selected.needConfirm
                  ? t('cms.confirmAsRead')
                  : t('cms.markAsRead')}
              </Button>
            )}
          </Space>
        }
      >
        {selected && (
          <>
            <Space wrap>
              <Tag color={(levels[selected.level] || levels.info).color}>
                {(levels[selected.level] || levels.info).label}
              </Tag>
              <span className={styles.muted}>
                {formatDate(selected.createdAt)}
              </span>
              {selected.readAt && <Tag>{t('cms.read')}</Tag>}
            </Space>
            <div className={styles.noticeBody}>{selected.content}</div>
            {selected.needConfirm && !selected.readAt && (
              <Alert
                type="info"
                showIcon
                message={t('cms.pleaseConfirmThatYouHaveRead')}
              />
            )}
          </>
        )}
      </Modal>
    </PageContainer>
  );
}
