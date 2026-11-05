export default [
  {
    path: '/user',
    layout: false,
    routes: [
      {
        name: 'login',
        path: '/user/login',
        component: './user/login',
      },
      {
        name: 'register',
        path: '/user/register',
        component: './user/register',
      },
      {
        name: 'register-result',
        path: '/user/register-result',
        component: './user/register-result',
      },
      {
        path: '/user',
        redirect: '/user/login',
      },
      {
        component: '404',
        path: '/user/*',
      },
    ],
  },
  // Public site: no login, its own layout.
  { path: '/', layout: false, component: './site/home' },
  { path: '/browse', layout: false, component: './site/browse' },
  { path: '/title/:id', layout: false, component: './site/title' },
  { path: '/watch/:id', layout: false, component: './site/watch' },
  // Entry to the back office: the role's start page, or login.
  { path: '/admin', layout: false, component: './site/admin' },
  // Old back-office player links.
  { path: '/media/player', layout: false, component: './site/legacy-player' },
  {
    path: '/',
    id: 'ant-design-pro-layout',
    routes: [],
  },
  {
    path: '/*',
    component: '404',
  },
];
