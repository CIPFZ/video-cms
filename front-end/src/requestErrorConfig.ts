import { currentLocale } from '@/i18n';
import type { RequestOptions } from '@@/plugin-request/request';
import { clearMenuCache } from './routing/menuDataStore';
import type { RequestConfig } from '@umijs/max';

import { currentHashPath, isPublicPath } from './routing/publicPaths';
import { isBackendRequest, renewedToken } from './utils/authHeaders';

const TOKEN_KEY = 'token';
const HEADER_TOKEN_KEY = 'x-token';

export const errorConfig: RequestConfig = {
  timeout: 10000,
  requestInterceptors: [
    (config: RequestOptions) => {
      if (!isBackendRequest(config.url, config.baseURL)) {
        return config;
      }

      if (config.headers instanceof Headers) {
        config.headers.set('Accept-Language', currentLocale());
      } else {
        config.headers = {
          ...config.headers,
          'Accept-Language': currentLocale(),
        };
      }
      if (config.url?.includes('/user/login')) return config;
      const token = localStorage.getItem(TOKEN_KEY);
      if (!token) {
        return config;
      }

      if (config.headers instanceof Headers) {
        config.headers.set(HEADER_TOKEN_KEY, token);
      } else {
        config.headers = {
          ...(config.headers || {}),
          [HEADER_TOKEN_KEY]: token,
        };
      }

      return config;
    },
  ],
  responseInterceptors: [
    (response) => {
      if (
        isBackendRequest(response.config?.url, response.config?.baseURL) &&
        (response.data as { code?: number } | undefined)?.code === 1003
      ) {
        localStorage.removeItem(TOKEN_KEY);
        clearMenuCache();
        // A stale token on the public site is dropped quietly; visitors stay where they are.
        if (!window.location.hash.startsWith('#/user/') && !isPublicPath(currentHashPath())) {
          window.location.hash = '/user/login';
          window.location.reload();
        }
        return response;
      }
      const newToken = isBackendRequest(
        response.config?.url,
        response.config?.baseURL,
      )
        ? renewedToken(response.headers)
        : undefined;
      if (newToken) {
        localStorage.setItem(TOKEN_KEY, newToken);
      }
      return response;
    },
  ],
};
