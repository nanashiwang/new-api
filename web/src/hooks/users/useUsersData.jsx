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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

const DEFAULT_ADVANCED_FILTERS = {
  searchRole: '',
  searchStatus: '',
  searchInviterId: '',
  searchInviteeUserId: '',
  searchHasInviter: '',
  searchHasInvitees: '',
  // 剩余额度范围已迁移到高级筛选，与“已使用余额”统一管理。
  searchBalanceMin: '',
  searchBalanceMax: '',
  // 已使用余额筛选放在高级筛选里，保持主工具条紧凑。
  searchUsedBalanceMin: '',
  searchUsedBalanceMax: '',
  // 组合排序：ID 与余额排序可以同时设置。
  searchIdSortOrder: '',
  searchBalanceSortOrder: '',
};
const USERS_ADVANCED_FILTERS_STORAGE_KEY = 'users-advanced-filters';

const getInitialAdvancedFilters = () => {
  try {
    const raw = localStorage.getItem(USERS_ADVANCED_FILTERS_STORAGE_KEY);
    if (!raw) {
      return DEFAULT_ADVANCED_FILTERS;
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') {
      return DEFAULT_ADVANCED_FILTERS;
    }
    return { ...DEFAULT_ADVANCED_FILTERS, ...parsed };
  } catch (error) {
    return DEFAULT_ADVANCED_FILTERS;
  }
};

export const useUsersData = () => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('users');

  // State management
  const [users, setUsers] = useState([]);
  const [selectedKeys, setSelectedKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [groupOptions, setGroupOptions] = useState([]);
  const [userCount, setUserCount] = useState(0);
  const [advancedFilters, setAdvancedFilters] = useState(
    getInitialAdvancedFilters,
  );

  // Modal states
  const [showAddUser, setShowAddUser] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState({
    id: undefined,
  });

  // Form initial values
  const formInitValues = {
    searchKeyword: '',
    searchGroup: '',
  };

  // Form API reference
  const [formApi, setFormApi] = useState(null);

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      searchGroup: formValues.searchGroup || '',
      ...normalizeAdvancedFilters(advancedFilters),
    };
  };

  const normalizeAdvancedFilters = (filters) => {
    const next = { ...DEFAULT_ADVANCED_FILTERS, ...(filters || {}) };
    return {
      searchRole: next.searchRole ?? '',
      searchStatus: next.searchStatus ?? '',
      searchInviterId: next.searchInviterId ?? '',
      searchInviteeUserId: next.searchInviteeUserId ?? '',
      searchHasInviter: next.searchHasInviter ?? '',
      searchHasInvitees: next.searchHasInvitees ?? '',
      searchBalanceMin: next.searchBalanceMin ?? '',
      searchBalanceMax: next.searchBalanceMax ?? '',
      searchUsedBalanceMin: next.searchUsedBalanceMin ?? '',
      searchUsedBalanceMax: next.searchUsedBalanceMax ?? '',
      searchIdSortOrder: next.searchIdSortOrder ?? '',
      searchBalanceSortOrder: next.searchBalanceSortOrder ?? '',
    };
  };

  const hasAdvancedFilters = (filters) => {
    const normalized = normalizeAdvancedFilters(filters);
    return Object.values(normalized).some(
      (value) => value !== '' && value !== null && value !== undefined,
    );
  };

  const hasBalanceFilters = (balanceMin, balanceMax) => {
    return (
      (balanceMin !== '' && balanceMin !== null && balanceMin !== undefined) ||
      (balanceMax !== '' && balanceMax !== null && balanceMax !== undefined)
    );
  };

  // Set user format with key field
  const setUserFormat = (users) => {
    // 每次重载列表都重置选择状态，避免“跨页残留选择”导致批量误操作。
    setSelectedKeys([]);
    for (let i = 0; i < users.length; i++) {
      users[i].key = users[i].id;
    }
    setUsers(users);
  };

  // 表格行选择配置（用于批量操作）
  const rowSelection = {
    selectedRowKeys: selectedKeys.map((user) => user.id),
    // Semi Table 会同时给 selectedRowKeys 与 selectedRows，
    // 这里直接保留 selectedRows，后续批量操作可直接取 id/状态/角色等字段。
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedKeys(selectedRows || []);
    },
  };

  // Load users data
  const loadUsers = async (
    startIdx,
    pageSize,
    idSortOrder = '',
    balanceSortOrder = '',
  ) => {
    setLoading(true);
    try {
      const params = {
        p: startIdx,
        page_size: pageSize,
      };
      if (idSortOrder === 'asc' || idSortOrder === 'desc') {
        params.id_sort_order = idSortOrder;
      }
      if (balanceSortOrder === 'asc' || balanceSortOrder === 'desc') {
        params.balance_sort_order = balanceSortOrder;
      }
      const res = await API.get('/api/user/', { params });
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page);
        setUserCount(data.total);
        setUserFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      // 网络异常/后端异常时也要退出 loading，避免列表一直转圈。
      showError(error?.message || t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  // Search users with keyword and group
  const searchUsers = async (
    startIdx,
    pageSize,
    searchKeyword = null,
    searchGroup = null,
    advanced = null,
    searchBalanceMin = null,
    searchBalanceMax = null,
  ) => {
    // If no parameters passed, get values from form
    let resolvedAdvanced = normalizeAdvancedFilters(
      advanced === null ? advancedFilters : advanced,
    );
    if (searchKeyword === null || searchGroup === null || advanced === null) {
      const formValues = getFormValues();
      if (searchKeyword === null) {
        searchKeyword = formValues.searchKeyword;
      }
      if (searchGroup === null) {
        searchGroup = formValues.searchGroup;
      }
      if (searchBalanceMin === null) {
        searchBalanceMin = formValues.searchBalanceMin;
      }
      if (searchBalanceMax === null) {
        searchBalanceMax = formValues.searchBalanceMax;
      }
      if (advanced === null) {
        resolvedAdvanced = normalizeAdvancedFilters(formValues);
      }
    }
    if (searchBalanceMin === null) {
      searchBalanceMin = resolvedAdvanced.searchBalanceMin;
    }
    if (searchBalanceMax === null) {
      searchBalanceMax = resolvedAdvanced.searchBalanceMax;
    }

    const keyword = (searchKeyword || '').trim();
    const group = (searchGroup || '').trim();
    const balanceMin = searchBalanceMin;
    const balanceMax = searchBalanceMax;
    if (
      keyword === '' &&
      group === '' &&
      !hasAdvancedFilters(resolvedAdvanced) &&
      !hasBalanceFilters(balanceMin, balanceMax)
    ) {
      // If keyword is blank, load files instead
      await loadUsers(
        startIdx,
        pageSize,
        resolvedAdvanced.searchIdSortOrder,
        resolvedAdvanced.searchBalanceSortOrder,
      );
      return;
    }
    // 搜索分支也需要接管 loading 状态：
    // 首屏若命中“已保存高级筛选”会直接走这里，若不置回 loading=false，表格会一直转圈。
    setLoading(true);
    setSearching(true);
    try {
      // 前端统一通过 axios params 传参，避免手写拼接 URL 导致转义遗漏。
      // 后端再做类型解析与参数化查询，形成“双保险”。
      const params = {
        keyword,
        group,
        p: startIdx,
        page_size: pageSize,
      };
      if (
        resolvedAdvanced.searchIdSortOrder === 'asc' ||
        resolvedAdvanced.searchIdSortOrder === 'desc'
      ) {
        params.id_sort_order = resolvedAdvanced.searchIdSortOrder;
      }
      if (
        resolvedAdvanced.searchBalanceSortOrder === 'asc' ||
        resolvedAdvanced.searchBalanceSortOrder === 'desc'
      ) {
        params.balance_sort_order = resolvedAdvanced.searchBalanceSortOrder;
      }
      if (resolvedAdvanced.searchRole !== '') {
        params.role = resolvedAdvanced.searchRole;
      }
      if (resolvedAdvanced.searchStatus !== '') {
        params.status = resolvedAdvanced.searchStatus;
      }
      if (resolvedAdvanced.searchInviterId !== '') {
        params.inviter_id = resolvedAdvanced.searchInviterId;
      }
      if (resolvedAdvanced.searchInviteeUserId !== '') {
        params.invitee_user_id = resolvedAdvanced.searchInviteeUserId;
      }
      if (resolvedAdvanced.searchHasInviter !== '') {
        params.has_inviter = resolvedAdvanced.searchHasInviter;
      }
      if (resolvedAdvanced.searchHasInvitees !== '') {
        params.has_invitees = resolvedAdvanced.searchHasInvitees;
      }
      // 金额筛选参数统一由 axios params 透传，后端进行范围校验与参数化查询。
      if (balanceMin !== '' && balanceMin !== null && balanceMin !== undefined) {
        params.balance_min = balanceMin;
      }
      if (balanceMax !== '' && balanceMax !== null && balanceMax !== undefined) {
        params.balance_max = balanceMax;
      }
      if (
        resolvedAdvanced.searchUsedBalanceMin !== '' &&
        resolvedAdvanced.searchUsedBalanceMin !== null &&
        resolvedAdvanced.searchUsedBalanceMin !== undefined
      ) {
        params.used_balance_min = resolvedAdvanced.searchUsedBalanceMin;
      }
      if (
        resolvedAdvanced.searchUsedBalanceMax !== '' &&
        resolvedAdvanced.searchUsedBalanceMax !== null &&
        resolvedAdvanced.searchUsedBalanceMax !== undefined
      ) {
        params.used_balance_max = resolvedAdvanced.searchUsedBalanceMax;
      }
      const res = await API.get('/api/user/search', { params });
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page);
        setUserCount(data.total);
        setUserFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.message || t('请求失败'));
    } finally {
      setSearching(false);
      setLoading(false);
    }
  };

  const applyAdvancedFilters = async (nextFilters) => {
    const normalized = normalizeAdvancedFilters(nextFilters);
    setAdvancedFilters(normalized);
    await searchUsers(1, pageSize, null, null, normalized);
  };

  const resetAdvancedFilters = async () => {
    setAdvancedFilters(DEFAULT_ADVANCED_FILTERS);
    await searchUsers(1, pageSize, null, null, DEFAULT_ADVANCED_FILTERS);
  };

  // Manage user operations (promote, demote, enable, disable, delete)
  const manageUser = async (userId, action, record) => {
    // Trigger loading state to force table re-render
    setLoading(true);

    const res = await API.post('/api/user/manage', {
      id: userId,
      action,
    });

    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作成功完成！'));
      const user = res.data.data;

      // Create a new array and new object to ensure React detects changes
      const newUsers = users.map((u) => {
        if (u.id === userId) {
          if (action === 'delete') {
            return { ...u, DeletedAt: new Date() };
          }
          return { ...u, status: user.status, role: user.role };
        }
        return u;
      });

      setUsers(newUsers);
    } else {
      showError(message);
    }

    setLoading(false);
  };

  // 用户批量管理（启用/禁用/删除）
  const batchManageUsers = async (action) => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个用户！'));
      return;
    }
    setLoading(true);
    try {
      const ids = selectedKeys.map((user) => user.id);
      const res = await API.post('/api/user/manage/batch', { ids, action });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('批量操作失败'));
        return;
      }

      const successCount = Number(data?.success_count || 0);
      const failedCount = Number(data?.failed_count || 0);
      if (failedCount > 0) {
        // 失败明细由后端返回 failed 列表，这里先给汇总提示，避免提示过长影响阅读。
        showSuccess(
          t('批量操作完成: {{success}}个成功, {{failed}}个失败', {
            success: successCount,
            failed: failedCount,
          }),
        );
      } else if (action === 'enable') {
        showSuccess(t('已批量启用 {{count}} 个用户', { count: successCount }));
      } else if (action === 'disable') {
        showSuccess(t('已批量禁用 {{count}} 个用户', { count: successCount }));
      } else if (action === 'delete') {
        showSuccess(t('已批量删除 {{count}} 个用户', { count: successCount }));
      } else {
        showSuccess(t('操作成功完成！'));
      }

      await refresh();
    } catch (error) {
      showError(error?.message || t('批量操作失败'));
    } finally {
      setLoading(false);
    }
  };

  const resetUserPasskey = async (user) => {
    if (!user) {
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/reset_passkey`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('Passkey 已重置'));
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(t('操作失败，请重试'));
    }
  };

  const resetUserTwoFA = async (user) => {
    if (!user) {
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/2fa`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('二步验证已重置'));
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(t('操作失败，请重试'));
    }
  };

  // Handle page change
  const handlePageChange = (page) => {
    setActivePage(page);
    const {
      searchKeyword,
      searchGroup,
      searchBalanceMin,
      searchBalanceMax,
    } = getFormValues();
    if (
      searchKeyword === '' &&
      searchGroup === '' &&
      !hasAdvancedFilters(advancedFilters) &&
      !hasBalanceFilters(searchBalanceMin, searchBalanceMax)
    ) {
      loadUsers(
        page,
        pageSize,
        advancedFilters.searchIdSortOrder,
        advancedFilters.searchBalanceSortOrder,
      ).then();
    } else {
      searchUsers(
        page,
        pageSize,
        searchKeyword,
        searchGroup,
        null,
        searchBalanceMin,
        searchBalanceMax,
      ).then();
    }
  };

  // Handle page size change
  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    const {
      searchKeyword,
      searchGroup,
      searchBalanceMin,
      searchBalanceMax,
    } = getFormValues();
    if (
      searchKeyword === '' &&
      searchGroup === '' &&
      !hasAdvancedFilters(advancedFilters) &&
      !hasBalanceFilters(searchBalanceMin, searchBalanceMax)
    ) {
      loadUsers(
        1,
        size,
        advancedFilters.searchIdSortOrder,
        advancedFilters.searchBalanceSortOrder,
      )
        .then()
        .catch((reason) => {
          showError(reason);
        });
      return;
    }
    searchUsers(
      1,
      size,
      null,
      null,
      null,
      searchBalanceMin,
      searchBalanceMax,
    )
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Handle table row styling for disabled/deleted users
  const handleRow = (record, index) => {
    if (record.DeletedAt !== null || record.status !== 1) {
      return {
        style: {
          background: 'var(--semi-color-disabled-border)',
        },
      };
    } else {
      return {};
    }
  };

  // Refresh data
  const refresh = async (page = activePage) => {
    const {
      searchKeyword,
      searchGroup,
      searchBalanceMin,
      searchBalanceMax,
    } = getFormValues();
    if (
      searchKeyword === '' &&
      searchGroup === '' &&
      !hasAdvancedFilters(advancedFilters) &&
      !hasBalanceFilters(searchBalanceMin, searchBalanceMax)
    ) {
      await loadUsers(
        page,
        pageSize,
        advancedFilters.searchIdSortOrder,
        advancedFilters.searchBalanceSortOrder,
      );
    } else {
      await searchUsers(
        page,
        pageSize,
        searchKeyword,
        searchGroup,
        null,
        searchBalanceMin,
        searchBalanceMax,
      );
    }
  };

  // Fetch groups data
  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      if (res === undefined) {
        return;
      }
      setGroupOptions(
        res.data.data.map((group) => ({
          label: group,
          value: group,
        })),
      );
    } catch (error) {
      showError(error.message);
    }
  };

  // Modal control functions
  const closeAddUser = () => {
    setShowAddUser(false);
  };

  const closeEditUser = () => {
    setShowEditUser(false);
    setEditingUser({
      id: undefined,
    });
  };

  // Initialize data on component mount
  useEffect(() => {
    // 页面首次加载时，如果本地存在高级筛选条件，则直接恢复该条件并执行查询。
    // 这样管理员刷新页面后不需要重复手动设置筛选项。
    if (hasAdvancedFilters(advancedFilters)) {
      searchUsers(1, pageSize, '', '', advancedFilters)
        .then()
        .catch((reason) => {
          showError(reason);
        });
    } else {
      loadUsers(0, pageSize)
        .then()
        .catch((reason) => {
          showError(reason);
        });
    }
    fetchGroups().then();
  }, []);

  useEffect(() => {
    try {
      localStorage.setItem(
        USERS_ADVANCED_FILTERS_STORAGE_KEY,
        JSON.stringify(advancedFilters),
      );
    } catch (error) {}
  }, [advancedFilters]);

  return {
    // Data state
    users,
    selectedKeys,
    loading,
    activePage,
    pageSize,
    userCount,
    searching,
    groupOptions,

    // Modal state
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    setShowEditUser,
    setEditingUser,

    // Form state
    formInitValues,
    formApi,
    setFormApi,
    advancedFilters,
    setAdvancedFilters,
    defaultAdvancedFilters: DEFAULT_ADVANCED_FILTERS,

    // UI state
    compactMode,
    setCompactMode,

    // Actions
    rowSelection,
    loadUsers,
    searchUsers,
    manageUser,
    batchManageUsers,
    resetUserPasskey,
    resetUserTwoFA,
    handlePageChange,
    handlePageSizeChange,
    handleRow,
    refresh,
    closeAddUser,
    closeEditUser,
    getFormValues,
    applyAdvancedFilters,
    resetAdvancedFilters,
    hasAdvancedFilters,

    // Translation
    t,
  };
};
