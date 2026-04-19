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
import { Button, Card, Space, Tag, TreeSelect, Typography } from '@douyinfe/semi-ui';
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

const buildUserNode = (user, keyword = '') => ({
  key: `user-${user.id}`,
  value: String(user.id),
  label: renderHighlightedLabel(formatUserLabel(user), keyword),
  isLeaf: true,
});

const buildGroupNode = (key, label, users = [], disabled = false, keyword = '') => ({
  key,
  value: key,
  label,
  disabled,
  isLeaf: false,
  children: users.map((user) => buildUserNode(user, keyword)),
});

const renderHighlightedLabel = (label, keyword) => {
  if (!keyword) return label;
  const text = String(label || '');
  const lower = text.toLowerCase();
  const target = keyword.toLowerCase();
  const idx = lower.indexOf(target);
  if (idx < 0) return text;
  return (
    <span>
      {text.slice(0, idx)}
      <span
        style={{
          background: 'var(--semi-color-warning-light-default)',
          color: 'var(--semi-color-warning-hover)',
          padding: '0 2px',
          borderRadius: 2,
        }}
      >
        {text.slice(idx, idx + keyword.length)}
      </span>
      {text.slice(idx + keyword.length)}
    </span>
  );
};

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

  // 首次挂载主动拉取普通用户，避免用户需要手动展开
  useEffect(() => {
    if (!commonUsersLoaded && !commonUsersLoading) {
      loadCommonUsers().catch(() => {
        // 静默失败，展开时会再次尝试
      });
    }
  }, [commonUsersLoaded, commonUsersLoading, loadCommonUsers]);

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
            keyword,
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
            keyword,
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
        commonUsersLoading
          ? t('普通用户（加载中…）')
          : commonUsersLoaded
            ? t('普通用户')
            : t('普通用户'),
        defaultCommonUsers,
        true,
      ),
    ];
  }, [
    adminUsers,
    commonUsersLoaded,
    commonUsersLoading,
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

  const selectedStats = useMemo(() => {
    const admin = selectedUsers.filter(isAdminUser).length;
    const common = selectedUsers.length - admin;
    const fallback = (excludedUserIDs || []).length - selectedUsers.length;
    return { admin, common, fallback, total: (excludedUserIDs || []).length };
  }, [excludedUserIDs, selectedUsers]);

  const handleClearAll = useCallback(() => {
    onChange([]);
  }, [onChange]);

  const handleSelectAllAdmins = useCallback(() => {
    const currentIds = new Set(
      (excludedUserIDs || []).map((item) => Number(item)).filter(Boolean),
    );
    (adminUsers || []).forEach((user) => {
      const id = Number(user?.id || 0);
      if (id > 0) currentIds.add(id);
    });
    onChange(Array.from(currentIds));
  }, [adminUsers, excludedUserIDs, onChange]);

  const handleRemoveSelected = useCallback(
    (id) => {
      const next = (excludedUserIDs || [])
        .map((item) => Number(item))
        .filter((item) => item !== Number(id) && item > 0);
      onChange(next);
    },
    [excludedUserIDs, onChange],
  );

  const emptyContent = searchKeyword.trim()
    ? searching
      ? t('搜索中…')
      : t('未找到匹配用户')
    : commonUsersLoading
      ? t('普通用户加载中…')
      : t('输入 ID / 用户名 / 昵称搜索');

  return (
    <Card
      bordered={false}
      title={t('收入排除')}
      className='rounded-xl'
    >
      <div className='space-y-3'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <Space spacing={6} wrap>
            <Tag color='blue' shape='circle'>
              {t('已排除')} {selectedStats.total}
            </Tag>
            <Tag color='violet' shape='circle'>
              {t('管理员')} {selectedStats.admin}
            </Tag>
            <Tag color='cyan' shape='circle'>
              {t('普通用户')} {selectedStats.common}
            </Tag>
            {selectedStats.fallback > 0 ? (
              <Tag color='grey' shape='circle'>
                {t('未回填')} {selectedStats.fallback}
              </Tag>
            ) : null}
          </Space>
          <Space spacing={6} wrap>
            <Button
              size='small'
              theme='light'
              type='primary'
              onClick={handleSelectAllAdmins}
              disabled={!adminUsers?.length}
            >
              {t('全选管理员')}
            </Button>
            <Button
              size='small'
              theme='light'
              type='danger'
              onClick={handleClearAll}
              disabled={selectedStats.total === 0}
            >
              {t('清空')}
            </Button>
          </Space>
        </div>
        <Text type='tertiary' size='small' className='block'>
          {t(
            '选中的用户不计入本站配置收入，但上游费用 / 利润仍继续统计（例如内部测试账号可在此排除）',
          )}
        </Text>
        {selectedUsers.length > 0 ? (
          <div
            className='flex flex-wrap gap-1 p-2 rounded-lg'
            style={{ background: 'var(--semi-color-fill-0)' }}
          >
            {selectedUsers.map((user) => (
              <Tag
                key={`selected-${user.id}`}
                closable
                shape='circle'
                color={isAdminUser(user) ? 'violet' : 'cyan'}
                onClose={() => handleRemoveSelected(user.id)}
              >
                {formatUserLabel(user)}
              </Tag>
            ))}
          </div>
        ) : null}
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
