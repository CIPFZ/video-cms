import { useFormLocale } from '@/i18n/useFormLocale';
import { t, useI18n } from '@/i18n';
import React, { useState, useEffect, useRef } from 'react'; // ✨ 引入 useEffect
import { GridContent } from '@ant-design/pro-components';
import { Avatar, Menu, Typography, message, List } from 'antd';
import {
  ProCard,
  ProForm,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useModel } from '@umijs/max';
import { createStyles } from 'antd-style';
import { MobileOutlined, MailOutlined, UserOutlined } from '@ant-design/icons';
import AvatarPicker from '@/components/AvatarPicker';
// ✨ 引入 API
import { updateAvatar, updateSelfInfo } from '@/services/system/user';
const { Title } = Typography;
const useStyles = createStyles(({ token }) => {
  return {
    avatarWrapper: {
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      paddingTop: 20,
    },
    preview: {
      marginBottom: 20,
    },
    title: {
      marginBottom: 12,
      color: token.colorTextHeading,
      fontWeight: 500,
    },
    desc: {
      marginTop: 12,
      color: token.colorTextSecondary,
      fontSize: 12,
      textAlign: 'center',
    },
  };
});

// --- 子组件：基本设置 (BaseView) ---
const BaseView: React.FC<{
  currentUser: any;
  refresh: () => void;
}> = ({ currentUser, refresh }) => {
  const localeFormRef1 = useFormLocale();
  useI18n();
  const { styles } = useStyles();
  const [savingAvatar, setSavingAvatar] = useState(false);
  const handleAvatarChange = async (avatar: string) => {
    setSavingAvatar(true);
    try {
      const res = await updateAvatar(avatar);
      if (res.code === 0) {
        message.success(t('cms.avatar.updated'));
        refresh();
      } else {
        message.error(res.msg || t('cms.updateFailed'));
      }
    } catch {
      message.error(t('cms.requestFailedPleaseTryAgainLater'));
    } finally {
      setSavingAvatar(false);
    }
  };
  const handleFinish = async (values: any) => {
    try {
      // ✨ 调用后端接口更新
      const res = await updateSelfInfo({
        nickName: values.nickName,
        bio: values.bio,
      });
      if (res.code === 0) {
        message.success(t('cms.profileUpdated'));
        // ✨ 刷新全局状态，让右上角名字和页面数据同步更新
        refresh();
      } else {
        message.error(res.msg || t('cms.updateFailed'));
      }
    } catch (error) {
      message.error(t('cms.requestFailedPleaseTryAgainLater'));
    }
  };
  return (
    <div
      style={{
        display: 'flex',
        gap: '48px',
        flexDirection: 'row',
      }}
    >
      {/* 左侧表单 */}
      <div
        style={{
          flex: 1,
          maxWidth: '440px',
        }}
      >
        <ProForm
          layout="vertical"
          onFinish={handleFinish}
          submitter={{
            searchConfig: {
              submitText: t('cms.updateProfile'),
            },
            render: (_, dom) => dom[1],
          }}
          // ✨ 使用 request 来回填数据，或者直接用 initialValues (这里用 initialValues 更简单，因为 currentUser 已经传进来了)
          initialValues={{
            nickName: currentUser?.nickName,
            phone: currentUser?.phone,
            email: currentUser?.email,
            bio: currentUser?.bio, // ✨ 绑定后端返回的 bio
          }}
          // 关键：当 currentUser 变化时重置表单，防止刷新后数据没变
          key={currentUser?.id}
          formRef={localeFormRef1}
        >
          <ProFormText
            width="md"
            name="nickName"
            label={t('cms.nickname')}
            rules={[
              {
                required: true,
                message: t('cms.enterYourNickname'),
              },
            ]}
          />

          <ProFormTextArea
            name="bio"
            label={t('cms.bio')}
            placeholder={t('cms.tellUsAboutYourself')}
            fieldProps={{
              rows: 4,
              showCount: true,
              maxLength: 200,
            }}
          />

          <ProFormText
            width="md"
            name="phone"
            label={t('cms.phoneNumber')}
            disabled
            fieldProps={{
              prefix: (
                <MobileOutlined
                  style={{
                    color: '#999',
                  }}
                />
              ),
            }}
            extra={t('cms.toChangeYourPhoneNumberContact')}
          />

          <ProFormText
            width="md"
            name="email"
            label={t('cms.email')}
            disabled
            fieldProps={{
              prefix: (
                <MailOutlined
                  style={{
                    color: '#999',
                  }}
                />
              ),
            }}
            extra={t('cms.toChangeYourEmailContactAn')}
          />
        </ProForm>
      </div>

      {/* 右侧头像 */}
      <div className={styles.avatarWrapper}>
        <div className={styles.title}>{t('cms.avatar')}</div>
        <Avatar
          size={96}
          src={currentUser?.avatar}
          icon={<UserOutlined />}
          className={styles.preview}
        />
        <AvatarPicker
          value={currentUser?.avatar}
          disabled={savingAvatar}
          onChange={handleAvatarChange}
        />
        <div className={styles.desc}>{t('cms.avatar.choose')}</div>
      </div>
    </div>
  );
};

// --- 子组件：安全设置 (SecurityView) ---
const SecurityView: React.FC<{
  currentUser: any;
}> = ({ currentUser }) => {
  useI18n();
  const handleModify = (type: string) => {
    message.info(
      t('cms.contactAnAdministratorToChange', {
        value0: type,
      }),
    );
  };
  const data = [
    {
      title: t('cms.accountPassword'),
      description: t('cms.contactAnAdministratorToResetYour'),
      actions: [
        <a key="Modify" onClick={() => handleModify(t('cms.password'))}>
          {t('cms.change')}
        </a>,
      ],
    },
    {
      title: t('cms.recoveryPhone'),
      description: t('cms.phone', {
        value0: currentUser?.phone || t('cms.notLinked'),
      }),
      actions: [
        <a key="Modify" onClick={() => handleModify(t('cms.phone.9c01ad'))}>
          {t('cms.change')}
        </a>,
      ],
    },
    {
      title: t('cms.recoveryEmail'),
      description: t('cms.email.5e91be', {
        value0: currentUser?.email || t('cms.notLinked'),
      }),
      actions: [
        <a key="Modify" onClick={() => handleModify(t('cms.email'))}>
          {t('cms.change')}
        </a>,
      ],
    },
    {
      title: t('cms.mfaDevice'),
      description: t('cms.mfaEnrollmentIsNotAvailableYet'),
      actions: [
        <a key="bind" onClick={() => handleModify('MFA')}>
          {t('cms.link')}
        </a>,
      ],
    },
  ];
  return (
    <List
      itemLayout="horizontal"
      dataSource={data}
      renderItem={(item) => (
        <List.Item actions={item.actions}>
          <List.Item.Meta title={item.title} description={item.description} />
        </List.Item>
      )}
    />
  );
};

// --- 主页面 ---
const Settings: React.FC = () => {
  useI18n();
  const { initialState, setInitialState, refresh } = useModel('@@initialState'); // ✨ 获取 refresh 方法
  const currentUser = initialState?.currentUser;
  const [initConfig, setInitConfig] = useState<'base' | 'security'>('base');
  const menuMap: Record<'base' | 'security', string> = {
    base: t('cms.profile'),
    security: t('cms.accountSecurity'),
  };
  const renderChildren = () => {
    switch (initConfig) {
      case 'base':
        // ✨ 将 refresh 传给子组件，以便修改完资料后刷新全局状态
        return <BaseView currentUser={currentUser} refresh={refresh} />;
      case 'security':
        return <SecurityView currentUser={currentUser} />;
      default:
        return null;
    }
  };
  return (
    <GridContent>
      <ProCard
        style={{
          height: '100%',
          minHeight: 600,
        }}
        bodyStyle={{
          padding: 0,
          display: 'flex',
          flexDirection: 'row',
        }}
        bordered
      >
        <div
          style={{
            width: 224,
            borderRight: '1px solid #f0f0f0',
            padding: '16px 0',
          }}
        >
          <Menu
            mode="inline"
            selectedKeys={[initConfig]}
            onClick={({ key }) => setInitConfig(key as 'base' | 'security')}
            style={{
              border: 'none',
            }}
          >
            {(Object.keys(menuMap) as Array<keyof typeof menuMap>).map(
              (item) => (
                <Menu.Item key={item}>{menuMap[item]}</Menu.Item>
              ),
            )}
          </Menu>
        </div>

        <div
          style={{
            flex: 1,
            padding: '24px 40px',
          }}
        >
          <Title
            level={4}
            style={{
              marginBottom: 24,
            }}
          >
            {menuMap[initConfig]}
          </Title>
          {renderChildren()}
        </div>
      </ProCard>
    </GridContent>
  );
};
export default Settings;
