import type { API } from '@/services/system/types';
import { request } from '@umijs/max';

/** 登录接口 POST /api/v1/user/login */
export async function login(body: API.LoginParams, options?: { [key: string]: any }) {
  return request<API.CommonResponse>('/api/v1/user/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data: body,
    ...(options || {}),
  });
}

/** 退出登录接口 POST /api/v1/user/logout */
export async function outLogin(options?: { [key: string]: any }) {
  return request<Record<string, any>>('/api/v1/sys/user/logout', {
    method: 'POST',
    ...(options || {}),
  });
}

/** 获取当前的用户 GET /api/v1/user/getSelfInfo */
export async function getCurrentUserInfo(options?: { [key: string]: any }) {
  return request<{
    code: number;
    msg: string;
    data: API.UserInfo;
  }>('/api/v1/sys/user/getSelfInfo', {
    method: 'GET',
    ...(options || {}),
  });
}

/** 获取用户列表接口 GET /api/v1/user/getUserList */
export async function getUserList(body: any, options?: { [key: string]: any }) {
  return request<API.CommonResponse>('/api/v1/sys/user/getUserList', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    data: body,
    ...(options || {}),
  });
}

// 新增用户
export async function addUser(body: any) {
  return request('/api/v1/sys/user/addUser', { method: 'POST', data: body });
}

// 更新用户
export async function updateUser(body: any) {
  return request('/api/v1/sys/user/updateUser', { method: 'PUT', data: body });
}

// 删除用户
export async function deleteUser(body: { id: number }) {
  return request('/api/v1/sys/user/deleteUser', { method: 'DELETE', data: body });
}

// 重置密码
export async function resetPassword(body: { id: number, password: string }) {
  return request('/api/v1/sys/user/resetPassword', { method: 'POST', data: body });
}

// 切换角色
export async function switchAuthority(body: { authorityId: number }) {
  return request<API.CommonResponse>('/api/v1/sys/user/switchAuthority', {
    method: 'POST',
    data: body,
  });
}

/** 更新个人基本信息 (昵称、简介) */
export async function updateSelfInfo(body: API.UpdateSelfInfoParams) {
  return request<API.CommonResponse>('/api/v1/sys/user/info', {
    method: 'PUT',
    data: body,
  });
}

/** 更新 UI 配置 (主题、布局等) */
export async function updateUiConfig(body: API.UpdateUiConfigParams) {
  return request<API.CommonResponse>('/api/v1/sys/user/ui-config', {
    method: 'PUT',
    data: body,
  });
}

/** 从预置头像中选择当前用户头像 */
export async function updateAvatar(avatar: string) {
  return request<API.CommonResponse>('/api/v1/sys/user/avatar', {
    method: 'POST',
    data: { avatar },
  });
}

export function getPublicConfig() {
 return request<{code: number; data: {registrationEnabled: boolean}}>('/api/v1/public/config');
}
