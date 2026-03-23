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

import { useRef, useState, useEffect } from 'react';
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
  // 订阅筛选统一口径：active 且未过期。
  searchHasActiveSubscription: '',
  // 可售令牌筛选：是否有启用中的可售令牌。
  searchHasSellableToken: '',
  // 剩余额度区间移入高级筛选，与已用额度筛选归并。
  searchBalanceMin: '',
  searchBalanceMax: '',
  // 将已用额度筛选放在高级面板，保持工具栏紧凑。
  searchUsedBalanceMin: '',
  searchUsedBalanceMax: '',
  // 复合排序支持 ID 与余额同时生效。
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

  // 状态管理
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
  const requestCounter = useRef(0);

  // 弹窗状态
  const [showAddUser, setShowAddUser] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState({
    id: undefined,
  });

  // 表单初始值
  const formInitValues = {
    searchKeyword: '',
    searchGroup: '',
  };

  // 表单 API 引用
  const [formApi, setFormApi] = useState(null);

  // 获取表单值的辅助函数
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
      searchHasActiveSubscription: next.searchHasActiveSubscription ?? '',
      searchHasSellableToken: next.searchHasSellableToken ?? '',
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

  // 为用户数据设置 key 字段
  const setUserFormat = (users) => {
    // 每次重载时重置选择，避免跨页批量操作命中陈旧数据。
    setSelectedKeys([]);
    for (let i = 0; i < users.length; i++) {
      users[i].key = users[i].id;
    }
    setUsers(users);
  };

  // 批量操作的表格行选择配置。
  const rowSelection = {
    selectedRowKeys: selectedKeys.map((user) => user.id),
    // Semi Table 同时提供 selectedRowKeys 和 selectedRows。
    // 直接保存 selectedRows，批量操作可直接使用 id/status/role 字段，无需二次映射。
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedKeys(selectedRows || []);
    },
  };

  // 加载用户数据
  const loadUsers = async (
    startIdx,
    pageSize,
    idSortOrder = '',
    balanceSortOrder = '',
  ) => {
    const reqId = ++requestCounter.current;
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
      if (reqId !== requestCounter.current) {
        return;
      }
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
      // 网络/后端报错时始终退出 loading，避免转圈不止。
      if (reqId === requestCounter.current) {
        showError(error?.message || t('请求失败'));
      }
    } finally {
      if (reqId === requestCounter.current) {
        setLoading(false);
      }
    }
  };

  // 按关键词和分组搜索用户
  const searchUsers = async (
    startIdx,
    pageSize,
    searchKeyword = null,
    searchGroup = null,
    advanced = null,
    searchBalanceMin = null,
    searchBalanceMax = null,
  ) => {
    // 若未传参数，则从表单读取
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
      // 若关键词为空，则改为加载列表数据
      await loadUsers(
        startIdx,
        pageSize,
        resolvedAdvanced.searchIdSortOrder,
        resolvedAdvanced.searchBalanceSortOrder,
      );
      return;
    }
    // 搜索分支也必须控制 loading 状态：
    // 首次加载可能因已保存的高级筛选进入该分支，loading 必须重置为 false。
    const reqId = ++requestCounter.current;
    setLoading(true);
    setSearching(true);
    try {
      // 前端通过 axios params 传递筛选，避免手写 URL 组装/转义错误。
      // 后端仍会做类型解析和参数化查询，作为第二层安全保障。
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
      if (resolvedAdvanced.searchHasActiveSubscription !== '') {
        params.has_active_subscription =
          resolvedAdvanced.searchHasActiveSubscription;
      }
      if (resolvedAdvanced.searchHasSellableToken !== '') {
        params.has_sellable_token = resolvedAdvanced.searchHasSellableToken;
      }
      // 额度筛选经 axios params 下发；后端负责区间校验与参数化查询。
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
      if (reqId !== requestCounter.current) {
        return;
      }
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
      if (reqId === requestCounter.current) {
        showError(error?.message || t('请求失败'));
      }
    } finally {
      if (reqId === requestCounter.current) {
        setSearching(false);
        setLoading(false);
      }
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

  // 用户管理操作（promote/demote/enable/disable/delete）
  const manageUser = async (userId, action, record) => {
    // 触发 loading，强制表格重渲染
    setLoading(true);

    const res = await API.post('/api/user/manage', {
      id: userId,
      action,
    });

    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作成功完成！'));
      const user = res.data.data;

      // 创建新数组和新对象，确保 React 感知变更
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

  // 用户批量管理（启用/禁用/删除）。
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
        // 后端会返回失败明细；先展示简要摘要，保证通知可读性。
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

  // 处理页码变更
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

  // 处理每页条数变更
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

  // 处理禁用/删除用户的行样式
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

  // 刷新数据
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

  // 拉取分组数据
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

  // 弹窗控制函数
  const closeAddUser = () => {
    setShowAddUser(false);
  };

  const closeEditUser = () => {
    setShowEditUser(false);
    setEditingUser({
      id: undefined,
    });
  };

  // 初始化数据 on component mount
  useEffect(() => {
    // 首次加载时恢复本地高级筛选（若存在）并立即查询。
    // 避免页面刷新后需要重新配置筛选。
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
    // 数据状态
    users,
    selectedKeys,
    loading,
    activePage,
    pageSize,
    userCount,
    searching,
    groupOptions,

    // 弹窗状态
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    setShowEditUser,
    setEditingUser,

    // 表单状态
    formInitValues,
    formApi,
    setFormApi,
    advancedFilters,
    setAdvancedFilters,
    defaultAdvancedFilters: DEFAULT_ADVANCED_FILTERS,

    // UI 状态
    compactMode,
    setCompactMode,

    // 操作函数
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

    // 国际化
    t,
  };
};
