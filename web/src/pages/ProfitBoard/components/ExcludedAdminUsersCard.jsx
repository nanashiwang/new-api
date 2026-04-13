/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import React, { useCallback, useMemo, useRef, useState } from 'react';
import { Card, Select, Typography } from '@douyinfe/semi-ui';
import { API } from '../../../helpers';

const { Text } = Typography;

const formatUserLabel = (user) =>
  `${user.display_name || user.username || `#${user.id}`} · ${user.username}`;

const ExcludedAdminUsersCard = ({
  adminUsers,
  excludedUserIDs,
  onChange,
  t,
}) => {
  const [searchResults, setSearchResults] = useState([]);
  const [searching, setSearching] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const debounceRef = useRef(null);
  // 缓存搜索选中的普通用户信息，避免选中后只显示 ID
  const [selectedUserCache, setSelectedUserCache] = useState(new Map());

  const doSearch = useCallback(async (keyword) => {
    if (!keyword.trim()) {
      setSearchResults([]);
      setSearching(false);
      return;
    }
    setSearching(true);
    try {
      const res = await API.get(
        `/api/user/search?keyword=${encodeURIComponent(keyword.trim())}`,
      );
      if (res.data?.success && Array.isArray(res.data.data?.data)) {
        setSearchResults(res.data.data.data);
      } else {
        setSearchResults([]);
      }
    } catch {
      setSearchResults([]);
    } finally {
      setSearching(false);
    }
  }, []);

  const handleSearch = useCallback(
    (keyword) => {
      setSearchKeyword(keyword);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      if (!keyword.trim()) {
        setSearchResults([]);
        setSearching(false);
        return;
      }
      setSearching(true);
      debounceRef.current = setTimeout(() => doSearch(keyword), 300);
    },
    [doSearch],
  );

  const optionList = useMemo(() => {
    const hasSearch = searchKeyword.trim().length > 0;

    if (hasSearch) {
      // 搜索模式：只展示搜索结果
      return searchResults.map((user) => ({
        label: formatUserLabel(user),
        value: String(user.id),
      }));
    }

    // 默认模式：展示管理员 + 已选非管理员用户
    const adminIdSet = new Set(
      (adminUsers || []).map((u) => String(u.id)),
    );
    const options = (adminUsers || []).map((user) => ({
      label: formatUserLabel(user),
      value: String(user.id),
    }));

    // 已选但不在管理员列表中的用户
    (excludedUserIDs || []).forEach((userID) => {
      const value = String(userID);
      if (!adminIdSet.has(value)) {
        const cached = selectedUserCache.get(value);
        options.push({
          label: cached || t('用户 #{{id}}', { id: userID }),
          value,
        });
      }
    });

    return options;
  }, [adminUsers, excludedUserIDs, searchKeyword, searchResults, selectedUserCache, t]);

  const handleChange = useCallback(
    (value) => {
      // 缓存新选中用户的 label 信息
      const newCache = new Map(selectedUserCache);
      (value || []).forEach((v) => {
        if (!newCache.has(v)) {
          const opt = optionList.find((o) => o.value === v);
          if (opt) newCache.set(v, opt.label);
        }
      });
      setSelectedUserCache(newCache);
      setSearchKeyword('');
      setSearchResults([]);

      onChange(
        (value || [])
          .map((item) => Number(item))
          .filter((item) => Number.isInteger(item) && item > 0),
      );
    },
    [onChange, optionList, selectedUserCache],
  );

  return (
    <Card
      bordered={false}
      title={t('收入排除')}
      className='rounded-xl'
    >
      <div className='space-y-2'>
        <Text type='tertiary' size='small'>
          {t('选中的用户请求不计入本站配置收入，但上游费用和利润仍继续统计')}
        </Text>
        <Select
          multiple
          remote
          filter={false}
          searchPosition='dropdown'
          maxTagCount={3}
          loading={searching}
          value={(excludedUserIDs || []).map((item) => String(item))}
          optionList={optionList}
          placeholder={t('选择要排除收入的用户')}
          style={{ width: '100%' }}
          onSearch={handleSearch}
          onChange={handleChange}
          emptyContent={
            searchKeyword.trim()
              ? searching
                ? t('搜索中…')
                : t('未找到匹配用户')
              : t('输入 ID 或用户名搜索更多用户')
          }
        />
      </div>
    </Card>
  );
};

export default ExcludedAdminUsersCard;
