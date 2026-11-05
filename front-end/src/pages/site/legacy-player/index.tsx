import { useEffect } from 'react';
import { history } from '@umijs/max';
import { watchURL } from '../TitleDetail';

// Old back-office player links (/media/player?id=&lineId=&episodeId=) open the public player.
export default function LegacyPlayer() {
  useEffect(() => {
    const hashQuery = window.location.hash.includes('?') ? window.location.hash.split('?')[1] : '';
    const params = new URLSearchParams(hashQuery || window.location.search);
    const id = Number(params.get('id')) || 0;
    history.replace(id ? watchURL(id, Number(params.get('lineId')) || undefined, Number(params.get('episodeId')) || undefined, params.get('restart') === '1') : '/');
  }, []);
  return null;
}
