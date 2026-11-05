// Pages of the public site: visitors are never sent to the login page from these.
const PUBLIC = /^\/(browse|title|watch|admin)(\/|$)/;

export const isPublicPath = (path: string): boolean =>
  path === '/' || path === '' || path === '/media/player' || PUBLIC.test(path);

/** Path of the current hash route, without its query. */
export const currentHashPath = (): string =>
  window.location.hash.replace(/^#/, '').split('?')[0] || '/';
