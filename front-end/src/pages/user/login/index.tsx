import { useFormLocale } from '@/i18n/useFormLocale';
import { t, useI18n } from '@/i18n';
import type { API } from '@/services/system/types';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import {
  LoginForm,
  ProFormCheckbox,
  ProFormText,
} from '@ant-design/pro-components';
import { FormattedMessage, Helmet, useIntl, useModel } from '@umijs/max';
import { Alert, App, Tabs } from 'antd';
import { createStyles } from 'antd-style';
import React, { useState, useEffect } from 'react';
import { flushSync } from 'react-dom';
import { Footer, SelectLang } from '@/components';
import { login, getPublicConfig } from '@/services/system/user';
const useStyles = createStyles(({ token }) => {
  return {
    action: {
      marginLeft: '8px',
      color: 'rgba(0, 0, 0, 0.2)',
      fontSize: '24px',
      verticalAlign: 'middle',
      cursor: 'pointer',
      transition: 'color 0.3s',
      '&:hover': {
        color: token.colorPrimaryActive,
      },
    },
    lang: {
      width: 42,
      height: 42,
      lineHeight: '42px',
      position: 'fixed',
      right: 16,
      borderRadius: token.borderRadius,
      ':hover': {
        backgroundColor: token.colorBgTextHover,
      },
    },
    container: {
      display: 'flex',
      flexDirection: 'column',
      height: '100vh',
      overflow: 'auto',
      backgroundImage: "url('/V-_oS6r-i7wAAAAAAAAAAAAAFl94AQBr.png')",
      backgroundSize: '100% 100%',
    },
  };
});

// 选择语言
const Lang = () => {
  useI18n();
  const { styles } = useStyles();
  return (
    <div className={styles.lang} data-lang>
      {SelectLang && <SelectLang />}
    </div>
  );
};
const LoginMessage: React.FC<{
  content: string;
}> = ({ content }) => {
  useI18n();
  return (
    <Alert
      style={{
        marginBottom: 24,
      }}
      message={content}
      type="error"
      showIcon
    />
  );
};
const Login: React.FC = () => {
  const localeFormRef1 = useFormLocale();
  useI18n();
  const [registrationEnabled, setRegistrationEnabled] = useState(false);
  useEffect(() => {
    getPublicConfig()
      .then((r) => setRegistrationEnabled(r.data.registrationEnabled))
      .catch(() => setRegistrationEnabled(false));
  }, []);
  const [userLoginState, setUserLoginState] = useState<API.LoginResult>({
    msg: '',
    code: -1,
  });
  const [type, setType] = useState<string>('account');
  const { initialState, setInitialState } = useModel('@@initialState');
  const { styles } = useStyles();
  const { message } = App.useApp();
  const intl = useIntl();

  // 获取用户信息
  const fetchUserInfo = async () => {
    const userInfo = await initialState?.fetchUserInfo?.();
    if (userInfo) {
      flushSync(() => {
        setInitialState((s) => ({
          ...s,
          currentUser: userInfo,
        }));
      });
    }
  };

  // 处理表单提交 - 登录
  const handleSubmit = async (values: API.LoginParams) => {
    try {
      // 登录
      const response = await login({
        ...values,
        type,
      });
      // 判断是否登录成功
      if (response.code === 0) {
        const defaultLoginSuccessMessage = intl.formatMessage({
          id: 'pages.login.success',
          defaultMessage: '登录成功！',
        });
        message.success(defaultLoginSuccessMessage);
        // ✨✨✨ 关键修改：保存 Token ✨✨✨
        // 假设后端返回结构是 response.data.token
        const token = response.data?.token;
        if (token) {
          localStorage.setItem('token', token);
        }
        // 获取用户具体信息
        await fetchUserInfo();
        // 路由跳转
        const urlParams = new URL(window.location.href).searchParams;
        const redirect = urlParams.get('redirect');
        window.location.href =
          redirect?.startsWith('/') &&
          !redirect.startsWith('//') &&
          !redirect.includes('\\')
            ? redirect
            : '/';
        return;
      }
      console.log(response);
      // 如果失败去设置登录失败错误信息
      setUserLoginState({
        code: response.code,
        msg: response.msg,
      });
    } catch (error) {
      const defaultLoginFailureMessage = intl.formatMessage({
        id: 'pages.login.failure',
        defaultMessage: '登录失败，请重试！',
      });
      console.log(error);
      message.error(defaultLoginFailureMessage);
    }
  };
  return (
    <div className={styles.container}>
      <Helmet>
        <title>
          {intl.formatMessage({
            id: 'menu.login',
            defaultMessage: '登录页',
          })}
          {` - ${intl.formatMessage({ id: 'cms.brandName' })}`}
        </title>
      </Helmet>
      <Lang />
      <div
        style={{
          flex: '1',
          padding: '32px 0',
        }}
      >
        <LoginForm
          contentStyle={{
            minWidth: 280,
            maxWidth: '75vw',
          }}
          logo={<img alt="logo" src="/logo.svg" />}
          title={intl.formatMessage({ id: 'cms.brandName' })}
          subTitle={intl.formatMessage({
            id: 'pages.layouts.userLayout.title',
          })}
          initialValues={{
            autoLogin: true,
          }}
          onFinish={async (values) => {
            await handleSubmit(values as API.LoginParams);
          }}
          formRef={localeFormRef1}
        >
          <Tabs
            activeKey={type}
            onChange={setType}
            centered
            items={[
              {
                key: 'account',
                label: intl.formatMessage({
                  id: 'pages.login.accountLogin.tab',
                  defaultMessage: '账户密码登录',
                }),
              },
            ]}
          />
          {/*如果登录失败显示*/}
          {userLoginState.code > 0 && (
            <LoginMessage
              content={intl.formatMessage({
                id: 'pages.login.accountLogin.errorMessage',
                defaultMessage: '账户或密码错误',
              })}
            />
          )}
          <>
            <ProFormText
              name="username"
              fieldProps={{
                size: 'large',
                prefix: <UserOutlined />,
              }}
              placeholder={intl.formatMessage({
                id: 'pages.login.username.placeholder',
                defaultMessage: '请输入用户名',
              })}
              rules={[
                {
                  required: true,
                  message: (
                    <FormattedMessage
                      id="pages.login.username.required"
                      defaultMessage="请输入用户名!"
                    />
                  ),
                },
              ]}
            />
            <ProFormText.Password
              name="password"
              fieldProps={{
                size: 'large',
                prefix: <LockOutlined />,
              }}
              placeholder={intl.formatMessage({
                id: 'pages.login.password.placeholder',
                defaultMessage: '请输入密码',
              })}
              rules={[
                {
                  required: true,
                  message: (
                    <FormattedMessage
                      id="pages.login.password.required"
                      defaultMessage="请输入密码！"
                    />
                  ),
                },
              ]}
            />
          </>
          <div
            style={{
              marginBottom: 24,
            }}
          >
            <ProFormCheckbox noStyle name="autoLogin">
              <FormattedMessage
                id="pages.login.rememberMe"
                defaultMessage="记住我"
              />
            </ProFormCheckbox>
            {registrationEnabled && (
              <a
                href="#/user/register"
                style={{
                  float: 'right',
                }}
              >
                {t('cms.createAnAccount')}
              </a>
            )}
          </div>
        </LoginForm>
      </div>
      <Footer />
    </div>
  );
};
export default Login;
