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
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  copy,
  showError,
  showSuccess,
  encodeToBase64,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const useTokensData = (openFluentNotification) => {
  const { t } = useTranslation();

  // Basic state
  const [tokens, setTokens] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [tokenCount, setTokenCount] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [searchMode, setSearchMode] = useState(false); // 是否处于搜索结果视图
  const [groupOptions, setGroupOptions] = useState([]);

  // Selection state
  const [selectedKeys, setSelectedKeys] = useState([]);

  // Edit state
  const [showEdit, setShowEdit] = useState(false);
  const [editingToken, setEditingToken] = useState({
    id: undefined,
  });

  // UI state
  const [compactMode, setCompactMode] = useTableCompactMode('tokens');
  const [showKeys, setShowKeys] = useState({});

  // Form state
  const [formApi, setFormApi] = useState(null);
  const formInitValues = {
    searchKeyword: '',
    searchToken: '',
    searchGroup: '',
    searchBalanceMin: '',
    searchBalanceMax: '',
    searchUsedBalanceMin: '',
    searchUsedBalanceMax: '',
    searchAmountSort: '',
  };

  const normalizeTokenSortKey = (sortKey) => {
    if (sortKey === 'quota_asc' || sortKey === 'quota_desc') {
      return sortKey;
    }
    return '';
  };

  const resolveTokenSort = (sortKey) => {
    // 令牌管理仅开放“金额排序”，不再暴露 ID 排序选项。
    const normalized = normalizeTokenSortKey(sortKey);
    if (normalized === 'quota_asc') {
      return { sort_by: 'remain_quota', sort_order: 'asc' };
    }
    if (normalized === 'quota_desc') {
      return { sort_by: 'remain_quota', sort_order: 'desc' };
    }
    return {};
  };

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      searchToken: formValues.searchToken || '',
      searchGroup: formValues.searchGroup || '',
      searchBalanceMin:
        formValues.searchBalanceMin === null ||
        formValues.searchBalanceMin === undefined ||
        formValues.searchBalanceMin === ''
          ? ''
          : formValues.searchBalanceMin,
      searchBalanceMax:
        formValues.searchBalanceMax === null ||
        formValues.searchBalanceMax === undefined ||
        formValues.searchBalanceMax === ''
          ? ''
          : formValues.searchBalanceMax,
      searchUsedBalanceMin:
        formValues.searchUsedBalanceMin === null ||
        formValues.searchUsedBalanceMin === undefined ||
        formValues.searchUsedBalanceMin === ''
          ? ''
          : formValues.searchUsedBalanceMin,
      searchUsedBalanceMax:
        formValues.searchUsedBalanceMax === null ||
        formValues.searchUsedBalanceMax === undefined ||
        formValues.searchUsedBalanceMax === ''
          ? ''
          : formValues.searchUsedBalanceMax,
      searchAmountSort: normalizeTokenSortKey(formValues.searchAmountSort),
    };
  };

  const hasTokenFilters = (values) => {
    // searchMode 依赖该判断，确保翻页/刷新时能带上当前筛选条件。
    return (
      values.searchKeyword !== '' ||
      values.searchToken !== '' ||
      values.searchGroup !== '' ||
      values.searchBalanceMin !== '' ||
      values.searchBalanceMax !== '' ||
      values.searchUsedBalanceMin !== '' ||
      values.searchUsedBalanceMax !== '' ||
      values.searchAmountSort !== ''
    );
  };

  // Close edit modal
  const closeEdit = () => {
    setShowEdit(false);
    setTimeout(() => {
      setEditingToken({
        id: undefined,
      });
    }, 500);
  };

  // Sync page data from API response
  const syncPageData = (payload) => {
    setTokens(payload.items || []);
    setTokenCount(payload.total || 0);
    setActivePage(payload.page || 1);
    setPageSize(payload.page_size || pageSize);
  };

  // Load tokens function
  const loadTokens = async (page = 1, size = pageSize) => {
    setLoading(true);
    setSearchMode(false);
    const res = await API.get('/api/token/', {
      params: {
        p: page,
        size,
      },
    });
    const { success, message, data } = res.data;
    if (success) {
      syncPageData(data);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Refresh function
  const refresh = async (page = activePage) => {
    if (searchMode) {
      await searchTokens(page, pageSize);
    } else {
      await loadTokens(page);
    }
    setSelectedKeys([]);
  };

  // Copy text function
  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制到剪贴板！'));
    } else {
      Modal.error({
        title: t('无法复制到剪贴板，请手动复制'),
        content: text,
        size: 'large',
      });
    }
  };

  // Open link function for chat integrations
  const onOpenLink = async (type, url, record) => {
    if (url && url.startsWith('fluent')) {
      openFluentNotification(record.key);
      return;
    }
    let status = localStorage.getItem('status');
    let serverAddress = '';
    if (status) {
      status = JSON.parse(status);
      serverAddress = status.server_address;
    }
    if (serverAddress === '') {
      serverAddress = window.location.origin;
    }
    if (url.includes('{cherryConfig}') === true) {
      let cherryConfig = {
        id: 'new-api',
        baseUrl: serverAddress,
        apiKey: 'sk-' + record.key,
      };
      let encodedConfig = encodeURIComponent(
        encodeToBase64(JSON.stringify(cherryConfig)),
      );
      url = url.replaceAll('{cherryConfig}', encodedConfig);
    } else {
      let encodedServerAddress = encodeURIComponent(serverAddress);
      url = url.replaceAll('{address}', encodedServerAddress);
      url = url.replaceAll('{key}', 'sk-' + record.key);
    }

    window.open(url, '_blank');
  };

  // Manage token function (delete, enable, disable)
  const manageToken = async (id, action, record) => {
    setLoading(true);
    let data = { id };
    let res;
    switch (action) {
      case 'delete':
        res = await API.delete(`/api/token/${id}/`);
        break;
      case 'enable':
        data.status = 1;
        res = await API.put('/api/token/?status_only=true', data);
        break;
      case 'disable':
        data.status = 2;
        res = await API.put('/api/token/?status_only=true', data);
        break;
    }
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作成功完成！'));
      let token = res.data.data;
      let newTokens = [...tokens];
      if (action !== 'delete') {
        record.status = token.status;
      }
      setTokens(newTokens);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Search tokens function
  const searchTokens = async (page = 1, size = pageSize) => {
    const normalizedPage = Number.isInteger(page) && page > 0 ? page : 1;
    const normalizedSize =
      Number.isInteger(size) && size > 0 ? size : pageSize;

    const {
      searchKeyword,
      searchToken,
      searchGroup,
      searchBalanceMin,
      searchBalanceMax,
      searchUsedBalanceMin,
      searchUsedBalanceMax,
      searchAmountSort,
    } = getFormValues();
    if (!hasTokenFilters(getFormValues())) {
      setSearchMode(false);
      await loadTokens(normalizedPage, normalizedSize);
      return;
    }
    setSearching(true);
    // 统一通过 params 组装查询，避免手写 URL 拼接遗漏转义。
    const params = {
      keyword: searchKeyword,
      token: searchToken,
      group: searchGroup,
      p: normalizedPage,
      size: normalizedSize,
      ...resolveTokenSort(searchAmountSort),
    };
    if (
      searchBalanceMin !== '' &&
      searchBalanceMin !== null &&
      searchBalanceMin !== undefined
    ) {
      params.balance_min = searchBalanceMin;
    }
    if (
      searchBalanceMax !== '' &&
      searchBalanceMax !== null &&
      searchBalanceMax !== undefined
    ) {
      params.balance_max = searchBalanceMax;
    }
    if (
      searchUsedBalanceMin !== '' &&
      searchUsedBalanceMin !== null &&
      searchUsedBalanceMin !== undefined
    ) {
      params.used_balance_min = searchUsedBalanceMin;
    }
    if (
      searchUsedBalanceMax !== '' &&
      searchUsedBalanceMax !== null &&
      searchUsedBalanceMax !== undefined
    ) {
      params.used_balance_max = searchUsedBalanceMax;
    }
    const res = await API.get('/api/token/search', { params });
    const { success, message, data } = res.data;
    if (success) {
      setSearchMode(true);
      syncPageData(data);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  // Page handlers
  const handlePageChange = (page) => {
    if (searchMode) {
      searchTokens(page, pageSize).then();
    } else {
      loadTokens(page, pageSize).then();
    }
  };

  const handlePageSizeChange = async (size) => {
    setPageSize(size);
    if (searchMode) {
      await searchTokens(1, size);
    } else {
      await loadTokens(1, size);
    }
  };

  // Row selection handlers
  const rowSelection = {
    onSelect: (record, selected) => {},
    onSelectAll: (selected, selectedRows) => {},
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedKeys(selectedRows);
    },
  };

  // Handle row styling
  const handleRow = (record, index) => {
    if (record.status !== 1) {
      return {
        style: {
          background: 'var(--semi-color-disabled-border)',
        },
      };
    } else {
      return {};
    }
  };

  // Batch delete tokens
  const batchDeleteTokens = async () => {
    if (selectedKeys.length === 0) {
      showError(t('请先选择要删除的令牌！'));
      return;
    }
    setLoading(true);
    try {
      const ids = selectedKeys.map((token) => token.id);
      const res = await API.post('/api/token/batch', { ids });
      if (res?.data?.success) {
        const count = res.data.data || 0;
        showSuccess(t('已删除 {{count}} 个令牌！', { count }));
        await refresh();
        setTimeout(() => {
          if (tokens.length === 0 && activePage > 1) {
            refresh(activePage - 1);
          }
        }, 100);
      } else {
        showError(res?.data?.message || t('删除失败'));
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  };

  // Batch copy tokens
  const batchCopyTokens = (copyType) => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个令牌！'));
      return;
    }

    Modal.info({
      title: t('复制令牌'),
      icon: null,
      content: t('请选择你的复制方式'),
      footer: (
        <div className='flex gap-2'>
          <button
            className='px-3 py-1 bg-gray-200 rounded'
            onClick={async () => {
              let content = '';
              for (let i = 0; i < selectedKeys.length; i++) {
                content +=
                  selectedKeys[i].name + '    sk-' + selectedKeys[i].key + '\n';
              }
              await copyText(content);
              Modal.destroyAll();
            }}
          >
            {t('名称+密钥')}
          </button>
          <button
            className='px-3 py-1 bg-blue-500 text-white rounded'
            onClick={async () => {
              let content = '';
              for (let i = 0; i < selectedKeys.length; i++) {
                content += 'sk-' + selectedKeys[i].key + '\n';
              }
              await copyText(content);
              Modal.destroyAll();
            }}
          >
            {t('仅密钥')}
          </button>
        </div>
      ),
    });
  };

  // Initialize data
  useEffect(() => {
    const fetchGroups = async () => {
      try {
        const res = await API.get('/api/group/');
        if (!res?.data?.data) {
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

    loadTokens(1)
      .then()
      .catch((reason) => {
        showError(reason);
      });
    fetchGroups().then();
  }, []);

  return {
    // Basic state
    tokens,
    loading,
    activePage,
    tokenCount,
    pageSize,
    searching,
    groupOptions,

    // Selection state
    selectedKeys,
    setSelectedKeys,

    // Edit state
    showEdit,
    setShowEdit,
    editingToken,
    setEditingToken,
    closeEdit,

    // UI state
    compactMode,
    setCompactMode,
    showKeys,
    setShowKeys,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,

    // Functions
    loadTokens,
    refresh,
    copyText,
    onOpenLink,
    manageToken,
    searchTokens,
    handlePageChange,
    handlePageSizeChange,
    rowSelection,
    handleRow,
    batchDeleteTokens,
    batchCopyTokens,
    syncPageData,

    // Translation
    t,
  };
};
