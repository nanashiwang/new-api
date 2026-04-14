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
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Card, TreeSelect, Typography } from '@douyinfe/semi-ui';
import { API } from '../../../helpers';

const { Text } = Typography;

const GROUP_ADMIN = 'profit-board-admin-users';
const GROUP_COMMON = 'profit-board-common-users';
const ADMIN_ROLE_THRESHOLD = 10;
const DEFAULT_EXPANDED_KEYS = [GROUP_ADMIN];

const isAdminUser = (user) => Number(user?.role || 0) >= ADMIN_ROLE_THRESHOLD;

const formatUserLabel = (user) => {
  const id = Number(user?.id || 0);
  const username = String(user?.username || '').trim();
  const displayName = String(user?.display_name || '').trim();
  const primary = displayName || username || `#${id}`;

  if (displayName && username && displayName !== username) {
    return `${displayName} · ${username} (#${id})`;
  }
  if (username) {
    return `${primary} · #${id}`;
  }
  return primary;
};

const mergeUsers = (...lists) => {
  const map = new Map();
  lists.flat().forEach((user) => {
    const id = Number(user?.id || 0);
    if (!id || map.has(id)) return;
    map.set(id, user);
  });
  return Array.from(map.values());
};

const groupUsersByRole = (users = []) => {
  const admin = [];
  const common = [];
  users.forEach((user) => {
    if (isAdminUser(user)) {
      admin.push(user);
      return;
    }
    common.push(user);
  });
  return { admin, common };
};

const buildUserNode = (user) => ({
  key: `user-${user.id}`,
  value: String(user.id),
  label: formatUserLabel(user),
  isLeaf: true,
});

const buildGroupNode = (key, label, users = [], disabled = false) => ({
  key,
  value: key,
  label,
  disabled,
  isLeaf: false,
  children: users.map(buildUserNode),
});

const ExcludedAdminUsersCard = ({
  adminUsers,
  excludedUserIDs,
  onChange,
  t,
}) => {
  const [commonUsers, setCommonUsers] = useState([]);
  const [commonUsersLoaded, setCommonUsersLoaded] = useState(false);
  const [commonUsersLoading, setCommonUsersLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchResults, setSearchResults] = useState({ admin: [], common: [] });
  const [expandedKeys, setExpandedKeys] = useState(DEFAULT_EXPANDED_KEYS);
  const [userCache, setUserCache] = useState({});
  const searchTimerRef = useRef(null);
  const searchRequestRef = useRef(0);

  const cacheUsers = useCallback((users = []) => {
    if (!Array.isArray(users) || users.length === 0) return;
    setUserCache((prev) => {
      const next = { ...prev };
      users.forEach((user) => {
        const id = Number(user?.id || 0);
        if (!id) return;
        next[id] = user;
      });
      return next;
    });
  }, []);

  const fetchUsers = useCallback(async (params) => {
    const res = await API.get('/api/profit_board/user_options', { params });
    if (!res.data?.success) {
      throw new Error(res.data?.message || '加载用户选项失败');
    }
    return res.data?.data?.items || [];
  }, []);

  const loadCommonUsers = useCallback(async () => {
    if (commonUsersLoaded || commonUsersLoading) return;
    setCommonUsersLoading(true);
    try {
      const users = await fetchUsers({
        role_group: 'common',
        page_size: 100,
      });
      cacheUsers(users);
      setCommonUsers(users);
      setCommonUsersLoaded(true);
    } catch (error) {
      setCommonUsers([]);
      throw error;
    } finally {
      setCommonUsersLoading(false);
    }
  }, [cacheUsers, commonUsersLoaded, commonUsersLoading, fetchUsers]);

  useEffect(() => {
    const ids = (excludedUserIDs || [])
      .map((item) => Number(item))
      .filter((item) => Number.isInteger(item) && item > 0)
      .filter((item) => !userCache[item]);
    if (!ids.length) return;

    let cancelled = false;
    (async () => {
      try {
        const users = await fetchUsers({
          ids: ids.join(','),
          page_size: Math.min(Math.max(ids.length, 1), 100),
        });
        if (cancelled) return;
        cacheUsers(users);
      } catch {
        // 保持静默，未回填时仍允许显示占位 ID
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [cacheUsers, excludedUserIDs, fetchUsers, userCache]);

  useEffect(() => {
    return () => {
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }
    };
  }, []);

  const selectedUsers = useMemo(
    () =>
      (excludedUserIDs || [])
        .map((id) => userCache[Number(id)])
        .filter(Boolean),
    [excludedUserIDs, userCache],
  );

  const selectedCommonUsers = useMemo(
    () => selectedUsers.filter((user) => !isAdminUser(user)),
    [selectedUsers],
  );

  const defaultCommonUsers = useMemo(
    () => mergeUsers(selectedCommonUsers, commonUsers),
    [commonUsers, selectedCommonUsers],
  );

  const treeData = useMemo(() => {
    const keyword = searchKeyword.trim();
    if (keyword) {
      const matchedGroups = [];
      if (searchResults.admin.length) {
        matchedGroups.push(
          buildGroupNode(
            GROUP_ADMIN,
            t('管理员'),
            searchResults.admin,
            true,
          ),
        );
      }
      if (searchResults.common.length) {
        matchedGroups.push(
          buildGroupNode(
            GROUP_COMMON,
            t('普通用户'),
            searchResults.common,
            true,
          ),
        );
      }
      return matchedGroups;
    }

    return [
      buildGroupNode(
        GROUP_ADMIN,
        t('管理员'),
        adminUsers || [],
        true,
      ),
      buildGroupNode(
        GROUP_COMMON,
        commonUsersLoaded ? t('普通用户') : t('普通用户（展开加载）'),
        defaultCommonUsers,
        true,
      ),
    ];
  }, [
    adminUsers,
    commonUsersLoaded,
    defaultCommonUsers,
    searchKeyword,
    searchResults.admin,
    searchResults.common,
    t,
  ]);

  const handleLoadData = useCallback(
    async (treeNode) => {
      if (searchKeyword.trim()) return;
      if (treeNode?.key !== GROUP_COMMON) return;
      await loadCommonUsers();
    },
    [loadCommonUsers, searchKeyword],
  );

  const handleExpand = useCallback(
    (nextExpandedKeys) => {
      setExpandedKeys(nextExpandedKeys);
    },
    [],
  );

  const handleSearch = useCallback(
    (keyword) => {
      setSearchKeyword(keyword);
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }

      const trimmedKeyword = keyword.trim();
      if (!trimmedKeyword) {
        searchRequestRef.current += 1;
        setSearching(false);
        setSearchResults({ admin: [], common: [] });
        setExpandedKeys(DEFAULT_EXPANDED_KEYS);
        return;
      }

      setSearching(true);
      const requestID = searchRequestRef.current + 1;
      searchRequestRef.current = requestID;
      searchTimerRef.current = setTimeout(async () => {
        try {
          const users = await fetchUsers({
            role_group: 'all',
            keyword: trimmedKeyword,
            page_size: 100,
          });
          if (searchRequestRef.current !== requestID) return;
          cacheUsers(users);
          const grouped = groupUsersByRole(users);
          setSearchResults(grouped);
          const nextExpandedKeys = [];
          if (grouped.admin.length) nextExpandedKeys.push(GROUP_ADMIN);
          if (grouped.common.length) nextExpandedKeys.push(GROUP_COMMON);
          setExpandedKeys(
            nextExpandedKeys.length
              ? nextExpandedKeys
              : [GROUP_ADMIN, GROUP_COMMON],
          );
        } catch {
          if (searchRequestRef.current !== requestID) return;
          setSearchResults({ admin: [], common: [] });
          setExpandedKeys([GROUP_ADMIN, GROUP_COMMON]);
        } finally {
          if (searchRequestRef.current === requestID) {
            setSearching(false);
          }
        }
      }, 300);
    },
    [cacheUsers, fetchUsers],
  );

  const handleChange = useCallback(
    (value) => {
      const nextIDs = (Array.isArray(value) ? value : [value])
        .map((item) => Number(item))
        .filter((item) => Number.isInteger(item) && item > 0);
      onChange(nextIDs);
    },
    [onChange],
  );

  const value = useMemo(
    () => (excludedUserIDs || []).map((item) => String(item)),
    [excludedUserIDs],
  );

  const emptyContent = searchKeyword.trim()
    ? searching
      ? t('搜索中…')
      : t('未找到匹配用户')
    : commonUsersLoading
      ? t('普通用户加载中…')
      : t('展开普通用户，或输入 ID / 用户名 / 昵称搜索');

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
        <Text type='tertiary' size='small'>
          {t('支持按 ID / 用户名 / 昵称搜索管理员和普通用户；普通用户默认收拢')}
        </Text>
        <TreeSelect
          multiple
          leafOnly
          filterTreeNode={() => true}
          searchPosition='dropdown'
          maxTagCount={3}
          value={value}
          treeData={treeData}
          expandedKeys={expandedKeys}
          placeholder={t('选择要排除收入的用户')}
          searchPlaceholder={t('搜 ID / 用户名 / 昵称')}
          emptyContent={emptyContent}
          style={{ width: '100%' }}
          onChange={handleChange}
          onExpand={handleExpand}
          onSearch={handleSearch}
          loadData={handleLoadData}
        />
      </div>
    </Card>
  );
};

export default ExcludedAdminUsersCard;
