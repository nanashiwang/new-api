import React, { useEffect, useState, useId } from 'react';
import { Button, Select } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  availableFavorites,
  favoriteStorageKey,
  readGroupFavorites,
  toggleGroupFavorite,
} from './groupFavoriteStorage';

export default function TokenGroupFavorites({
  userId,
  groups,
  value,
  onSelect,
}) {
  const { t } = useTranslation();
  const labelId = useId();
  const key = favoriteStorageKey(userId);
  const [saved, setSaved] = useState({ key: null, items: [] });
  const [storageError, setStorageError] = useState(false);
  const favorites = saved.key === key ? saved.items : [];
  useEffect(() => {
    setStorageError(false);
    let items = [];
    try {
      items = readGroupFavorites(window.localStorage, key);
    } catch {
      /* Browser storage may be blocked. */
    }
    setSaved({ key, items });
    const sync = (event) => {
      if (event.key === key || event.key === null) {
        try {
          setSaved({
            key,
            items: readGroupFavorites(window.localStorage, key),
          });
        } catch {
          /* Keep in-memory preferences. */
        }
      }
    };
    window.addEventListener('storage', sync);
    return () => window.removeEventListener('storage', sync);
  }, [key]);
  const options = availableFavorites(groups, favorites);
  const canFavorite = Boolean(
    key && value && groups.some((group) => group.value === value),
  );
  const isFavorite = favorites.includes(value);
  const toggle = () => {
    if (!canFavorite) return;
    const next = toggleGroupFavorite(favorites, value);
    setSaved({ key, items: next });
    try {
      window.localStorage.setItem(key, JSON.stringify(next));
      setStorageError(false);
    } catch {
      setStorageError(true);
    }
  };
  if (!key) return null;
  return (
    <div className='mb-2 rounded-lg bg-gray-50 dark:bg-gray-900 p-3'>
      <span id={labelId} className='sr-only'>
        {t('我的收藏分组')}
      </span>
      <div className='flex flex-wrap items-center gap-2'>
        <Select
          aria-labelledby={labelId}
          placeholder={
            options.length ? t('我的收藏分组') : t('选择分组后可收藏')
          }
          value={
            options.some((option) => option.value === value) ? value : undefined
          }
          optionList={options.map((option) => ({
            value: option.value,
            label: option.value,
          }))}
          onChange={(group) => {
            if (options.some((option) => option.value === group))
              onSelect(group);
          }}
          filter
          style={{ flex: '1 1 180px', minWidth: 0 }}
          emptyContent={t('暂无可用收藏，请先选择并收藏分组')}
        />
        <Button
          type='tertiary'
          disabled={!canFavorite}
          onClick={toggle}
          aria-pressed={isFavorite}
        >
          {isFavorite ? '★ ' : '☆ '}
          {isFavorite ? t('取消收藏') : t('收藏当前分组')}
        </Button>
      </div>
      <div
        className='mt-1 text-xs text-gray-500'
        role={storageError ? 'status' : undefined}
      >
        {storageError
          ? t('无法保存收藏，关闭页面后可能丢失')
          : t('收藏仅保存在当前浏览器，按账号隔离')}
      </div>
    </div>
  );
}
