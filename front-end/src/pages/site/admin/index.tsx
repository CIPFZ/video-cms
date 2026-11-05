import React, { useEffect } from 'react';
import { Spin } from 'antd';
import { history } from '@umijs/max';
import { getCurrentUserInfo } from '@/services/system/user';

// Entry from the public site: staff go to their role's start page, everyone else to login.
export default function AdminEntry() {
  useEffect(() => {
    const login = () => history.replace('/user/login');
    if (!localStorage.getItem('token')) {
      login();
      return;
    }
    getCurrentUserInfo({ skipErrorHandler: true }).then(r => {
      if (r.code !== 0 || !r.data) {
        localStorage.removeItem('token');
        login();
        return;
      }
      const start = r.data.authority?.defaultRouter || 'dashboard/workplace';
      history.replace(start.startsWith('/') ? start : `/${start}`);
    }).catch(login);
  }, []);
  return <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center' }}><Spin size="large" /></div>;
}
