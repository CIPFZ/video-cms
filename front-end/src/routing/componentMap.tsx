import React from 'react';

export const componentMap: Record<string, React.LazyExoticComponent<any>> = {
  'media/schedules': React.lazy(() => import('@/pages/media/schedules')),
  'media/runs': React.lazy(() => import('@/pages/media/runs')),
  'media/health': React.lazy(() => import('@/pages/media/health')),
  'media/sources': React.lazy(() => import('@/pages/media/sources')),
  'media/player': React.lazy(() => import('@/pages/site/legacy-player')),
  'media/library': React.lazy(() => import('@/pages/media/library')),
  'state': React.lazy(() => import('@/pages/state')),
  'about': React.lazy(() => import('@/pages/about')),
  'dashboard/workplace': React.lazy(() => import('@/pages/dashboard/workplace')),
  'components/RouterLayout': React.lazy(() => import('@/components/RouterLayout')),

  'sys/user': React.lazy(() => import('@/pages/sys/user')),
  'sys/authority': React.lazy(() => import('@/pages/sys/authority')),
  'sys/menu': React.lazy(() => import('@/pages/sys/menu')),
  'sys/api': React.lazy(() => import('@/pages/sys/api')),
  'sys/api-token': React.lazy(() => import('@/pages/sys/api-token')),
  'sys/operation': React.lazy(() => import('@/pages/sys/operation')),
  'sys/notice': React.lazy(() => import('@/pages/sys/notice')),


  'user/info': React.lazy(() => import('@/pages/account/settings')),

};

export const getComponent = (name?: string) => {
  if (!name) return undefined;
  return componentMap[name];
};
