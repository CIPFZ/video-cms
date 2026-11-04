import { render } from '../../../tests/render';
import React from 'react';
import fs from 'node:fs';
import path from 'node:path';
import { fireEvent, screen } from '@testing-library/react';

import AvatarPicker, { AVATAR_PRESETS, avatarUrl } from '.';

describe('AvatarPicker', () => {
  it('ships an SVG file for every preset and nothing else', () => {
    const dir = path.resolve(__dirname, '../../../public/avatars');
    expect(fs.readdirSync(dir).sort()).toEqual(
      AVATAR_PRESETS.map((name) => `${name}.svg`).sort(),
    );
  });

  it('marks the current avatar and reports a newly chosen one', () => {
    const onChange = jest.fn();
    render(<AvatarPicker value={avatarUrl('cat')} onChange={onChange} />);

    expect(screen.getAllByRole('button')).toHaveLength(AVATAR_PRESETS.length);
    expect(
      screen.getByRole('button', { name: '小猫' }).getAttribute('aria-pressed'),
    ).toBe('true');

    fireEvent.click(screen.getByRole('button', { name: '小猫' }));
    expect(onChange).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '水獭' }));
    expect(onChange).toHaveBeenCalledWith('/avatars/otter.svg');
  });
});
