import NoticePopup from '@/components/NoticePopup';
import { t } from '@/i18n';
import { localizeMenus } from '@/i18n/menus';
import type { API } from '@/services/system/types';
import { LinkOutlined } from '@ant-design/icons';
import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import { SettingDrawer } from '@ant-design/pro-components';
import type { RequestConfig, RunTimeLayoutConfig } from '@umijs/max';
import { history, Link } from '@umijs/max';
import React from 'react';
import {
  AvatarDropdown,
  AvatarName,
  Footer,
  Question,
  SelectLang,
  SiteLink,
} from '@/components';
import { getCurrentUserInfo } from '@/services/system/user';
import defaultSettings from '../config/defaultSettings';
import { errorConfig } from './requestErrorConfig';
import '@ant-design/v5-patch-for-react-19';
import { processMenuData, buildRoutes } from '@/routing/menuHelpers';
import { fetchMenuData } from '@/routing/menuDataStore';
import { withMenuIcon } from '@/routing/menuIcon';
import { updateUiConfig } from '@/services/system/user';
import { isPublicPath } from '@/routing/publicPaths';

const isDev = process.env.NODE_ENV === 'development' || process.env.CI;
const loginPath = '/user/login';

// ✨ 定义关键业务状态码 (与后端 pkg/errcode 保持一致)
const Code = {
  SUCCESS: 0,
  UNAUTHORIZED: 1003, // 未登录/Token过期
};

/**
 * @see https://umijs.org/docs/api/runtime-config#getinitialstate
 * */
export async function getInitialState(): Promise<{
  settings?: Partial<LayoutSettings>;
  currentUser?: API.UserInfo;
  loading?: boolean;
  fetchUserInfo?: (() => Promise<API.UserInfo | undefined>) | undefined;
  menuData?: any[];
}> {
  // 1. 定义获取用户信息的内部函数
  const fetchUserInfo = async () => {
    try {
      const msg = await getCurrentUserInfo({
        skipErrorHandler: true, // 跳过默认拦截器，我们需要自己处理 code
      });

      // ✅ 场景 1：成功
      if (msg.code === Code.SUCCESS && msg.data) {
        return msg.data;
      }

      // 🚨 场景 2：明确的未登录 (1003)
      if (msg.code === Code.UNAUTHORIZED) {
        // 只有这里才执行登出逻辑；前台访客不跳登录页
        localStorage.removeItem('token');
        if (!isPublicPath(history.location.pathname)) history.push(loginPath);
        return undefined;
      }

      // ⚠️ 场景 3：其他业务错误 (如 1000 系统内部错误)
      // 抛出错误，触发 Ant Design Pro 的 ErrorPage (显示“加载失败，点击重试”)
      // 而不是把用户踢回登录页
      throw new Error(msg.msg || t('cms.unableToLoadUser'));
    } catch (error) {
      // 这里的 error 可能是网络错误 (fetch failed) 或上面抛出的业务错误
      // 我们不在这里做跳转，而是返回 undefined，让 Layout 决定如何展示
      console.error('fetchUserInfo error:', error);
    }
    return undefined;
  };

  // 2. 判断逻辑
  const { location } = history;
  const token = localStorage.getItem('token');

  // 只有 (非登录页) 且 (有Token) 时才请求
  if (
    ![loginPath, '/user/register', '/user/register-result'].includes(
      location.pathname,
    ) &&
    token
  ) {
    try {
      // 并行请求
      const [currentUser, rawMenuData] = await Promise.all([
        fetchUserInfo(),
        fetchMenuData().catch(() => []),
      ]);

      // 如果没拿到用户 (可能是 1003 跳走了，也可能是 500 报错了)
      if (!currentUser) {
        // 返回空状态，如果是因为报错导致的 undefined，ProLayout 会展示错误页
        return {
          fetchUserInfo,
          settings: defaultSettings as Partial<LayoutSettings>,
        };
      }

      const menuData = processMenuData(rawMenuData);
      const settings =
        (currentUser?.settings as Partial<LayoutSettings>) ||
        (defaultSettings as Partial<LayoutSettings>);

      return {
        fetchUserInfo,
        currentUser,
        settings,
        menuData,
      };
    } catch (e) {
      console.error('getInitialState system error:', e);
    }
  }

  return {
    fetchUserInfo,
    settings: defaultSettings as Partial<LayoutSettings>,
  };
}

// ProLayout 支持的api https://procomponents.ant.design/components/layout
export const layout: RunTimeLayoutConfig = ({
  initialState,
  setInitialState,
}) => {
  const handleSettingChange = (settings: any) => {
    setInitialState((pre) => ({ ...pre, settings }));
    updateUiConfig({ settings }).catch((err) =>
      console.error('配置同步失败', err),
    );
  };

  return {
    actionsRender: () => [
      <SiteLink key="site" />,
      <Question key="doc" />,
      <SelectLang key="SelectLang" />,
    ],
    avatarProps: {
      src: initialState?.currentUser?.avatar,
      title: <AvatarName />,
      render: (_, avatarChildren) => {
        return <AvatarDropdown>{avatarChildren}</AvatarDropdown>;
      },
    },
    waterMarkProps: {
      content: initialState?.currentUser?.nickName,
    },
    footerRender: () => <Footer />,

    // ✨ 页面变更 - 路由跳转逻辑优化
    onPageChange: () => {
      const { location } = history;
      const token = localStorage.getItem('token');

      // 1. 未登录检查：
      // 如果没有 currentUser，且连 token 都没有，那必须去登录
      // (注意：如果 token 存在但 currentUser 为空，可能是接口 500 了，此时不跳登录，而是停留在当前页显示错误)
      if (
        !initialState?.currentUser &&
        !token &&
        location.pathname !== loginPath
      ) {
        history.push(loginPath);
        return;
      }

      // 2. 已登录检查：如果已登录且在根路径，跳转默认路由
      if (initialState?.currentUser && location.pathname === '/') {
        const defaultRouter = initialState.currentUser.authority?.defaultRouter;
        if (defaultRouter) {
          const targetPath = defaultRouter.startsWith('/')
            ? defaultRouter
            : `/${defaultRouter}`;
          history.replace(targetPath);
        } else {
          history.replace('/dashboard/workplace');
        }
      }
    },
    bgLayoutImgList: [
      {
        src: 'https://mdn.alipayobjects.com/yuyan_qk0oxh/afts/img/D2LWSqNny4sAAAAAAAAAAAAAFl94AQBr',
        left: 85,
        bottom: 100,
        height: '303px',
      },
      {
        src: 'https://mdn.alipayobjects.com/yuyan_qk0oxh/afts/img/C2TWRpJpiC0AAAAAAAAAAAAAFl94AQBr',
        bottom: -68,
        right: -45,
        height: '303px',
      },
      {
        src: 'https://mdn.alipayobjects.com/yuyan_qk0oxh/afts/img/F6vSTbj8KpYAAAAAAAAAAAAAFl94AQBr',
        bottom: 0,
        left: 0,
        width: '331px',
      },
    ],
    links: isDev
      ? [
          <Link key="openapi" to="/umi/plugin/openapi" target="_blank">
            <LinkOutlined />
            <span>{t('cms.openApiDocs')}</span>
          </Link>,
        ]
      : [],
    menuHeaderRender: undefined,
    menuDataRender: () => localizeMenus(initialState?.menuData || []),
    subMenuItemRender: (item, defaultDom) =>
      withMenuIcon(item.icon, defaultDom),
    menuItemRender: (item, defaultDom) => {
      const label = withMenuIcon(item.icon, defaultDom);
      if (item.isUrl || !item.path || history.location.pathname === item.path)
        return label;
      return (
        <Link to={item.path.replace('/*', '')} target={item.target}>
          {label}
        </Link>
      );
    },
    childrenRender: (children) => {
      return (
        <>
          {children}
          <NoticePopup userId={initialState?.currentUser?.ID} />
          <SettingDrawer
            disableUrlParams
            enableDarkTheme
            settings={initialState?.settings}
            onSettingChange={handleSettingChange}
            hideCopyButton={true}
            hideHintAlert={true}
          />
        </>
      );
    },
    ...initialState?.settings,
    // Settings saved per user may carry an older title, so the brand wins.
    title: t('cms.brandName'),
  };
};

// --- 动态路由 ---
export async function patchClientRoutes({ routes }: { routes: any[] }) {
  const token = localStorage.getItem('token');
  if (!token) return;

  try {
    const rawMenuData = await fetchMenuData().catch(() => []);
    if (!rawMenuData || rawMenuData.length === 0) return;

    const dynamicRoutes = buildRoutes(rawMenuData);
    // Matched by id: the public home also uses path "/".
    const layoutRoute = routes.find(
      (route) => route.id === 'ant-design-pro-layout',
    );

    if (layoutRoute) {
      if (!layoutRoute.routes) {
        layoutRoute.routes = [];
      }
      // "/" belongs to the public site; /admin sends staff to their default page.
      layoutRoute.routes.unshift(...dynamicRoutes);
    }
  } catch (e) {
    console.error('patchClientRoutes error:', e);
  }
}

export const request: RequestConfig = {
  ...errorConfig,
};
