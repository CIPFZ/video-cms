import routes from '../../config/routes';

const hasRoutePath = (list: any[] = [], path: string): boolean => {
  for (const route of list) {
    if (route?.path === path) {
      return true;
    }
    if (Array.isArray(route?.routes) && hasRoutePath(route.routes, path)) {
      return true;
    }
  }
  return false;
};

describe('routes config', () => {
  it('serves the public site at the root, outside the back-office layout', () => {
    const top = (path: string) => (routes as any[]).find((route) => route.path === path && !route.id);
    for (const path of ['/', '/browse', '/title/:id', '/watch/:id', '/admin', '/media/player']) {
      expect(top(path)).toMatchObject({ layout: false });
    }
    // Back-office routes come from the menus; nothing in the layout claims "/".
    const layoutRoute = (routes as any[]).find((route) => route.id === 'ant-design-pro-layout');
    expect(layoutRoute?.routes).toEqual([]);
  });

  it('leaves business routes to backend menus', () => {
    expect(hasRoutePath(routes as any[], '/sys/api-token')).toBe(false);
    expect(hasRoutePath(routes as any[], '/account/settings')).toBe(false);
  });
});
