import { t, useI18n } from '@/i18n';
import React from 'react';
import { Avatar } from 'antd';
import { createStyles } from 'antd-style';

// 与后端 model.AvatarPresets 保持一致，文件位于 public/avatars。
export const AVATAR_PRESETS = [
  'capybara',
  'bear',
  'cat',
  'fox',
  'rabbit',
  'panda',
  'dog',
  'koala',
  'penguin',
  'tiger',
  'pig',
  'otter',
] as const;

export const avatarUrl = (name: string) => `/avatars/${name}.svg`;

const useStyles = createStyles(({ token }) => ({
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, 56px)',
    gap: 12,
  },
  option: {
    padding: 2,
    border: '2px solid transparent',
    borderRadius: '50%',
    background: 'none',
    cursor: 'pointer',
    lineHeight: 0,
    '&:hover': { borderColor: token.colorPrimaryBorderHover },
    '&:disabled': { cursor: 'wait' },
  },
  selected: {
    borderColor: `${token.colorPrimary} !important`,
  },
}));

interface AvatarPickerProps {
  value?: string;
  onChange?: (url: string) => void;
  disabled?: boolean;
}

const AvatarPicker: React.FC<AvatarPickerProps> = ({
  value,
  onChange,
  disabled,
}) => {
  useI18n();
  const { styles, cx } = useStyles();
  return (
    <div className={styles.grid}>
      {AVATAR_PRESETS.map((name) => {
        const url = avatarUrl(name);
        const label = t(`cms.avatar.${name}`);
        return (
          <button
            key={name}
            type="button"
            aria-pressed={value === url}
            aria-label={label}
            title={label}
            disabled={disabled}
            className={cx(styles.option, value === url && styles.selected)}
            onClick={() => value !== url && onChange?.(url)}
          >
            <Avatar size={48} src={url} alt={label} />
          </button>
        );
      })}
    </div>
  );
};

export default AvatarPicker;
