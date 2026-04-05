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
import { useCallback, useEffect, useMemo, useState } from 'react';
import { API, showError, showSuccess } from '../../../helpers';
import {
  createDefaultUpstreamAccountDraft,
  getUpstreamAccountDraftValidation,
  getUpstreamAccountSuggestedName,
  normalizeUpstreamAccountBaseUrl,
  prepareUpstreamAccountDraftForSave,
} from '../utils';

export const useUpstreamAccounts = ({
  options,
  loadOptions,
  comboConfigs,
  upstreamConfig,
  setUpstreamConfig,
  runFullRefresh,
}) => {
  const [accountDraft, setAccountDraft] = useState(
    createDefaultUpstreamAccountDraft(),
  );
  const [editingAccountId, setEditingAccountId] = useState(0);
  const [savingAccount, setSavingAccount] = useState(false);
  const [syncingAccountId, setSyncingAccountId] = useState(0);
  const [syncingAllAccounts, setSyncingAllAccounts] = useState(false);
  const [deletingAccountId, setDeletingAccountId] = useState(0);
  const [accountTrendLoading, setAccountTrendLoading] = useState(false);
  const [accountTrend, setAccountTrend] = useState(null);
  const [sideSheetVisible, setSideSheetVisible] = useState(false);
  const [detailSideSheetVisible, setDetailSideSheetVisible] = useState(false);
  const [accountDraftTouched, setAccountDraftTouched] = useState({});
  const [accountDraftSubmitted, setAccountDraftSubmitted] = useState(false);
  const [accountNameManuallyEdited, setAccountNameManuallyEdited] =
    useState(false);

  const accounts = useMemo(
    () => options.upstream_accounts || [],
    [options.upstream_accounts],
  );

  const editingAccount = useMemo(
    () =>
      accounts.find((item) => item.id === Number(editingAccountId || 0)) ||
      null,
    [editingAccountId, accounts],
  );

  const activeWalletAccountIds = useMemo(() => {
    const ids = new Set();
    (comboConfigs || []).forEach((item) => {
      if (item?.upstream_mode !== 'wallet_observer') return;
      const accountId = Number(item?.upstream_account_id || 0);
      if (accountId > 0) ids.add(accountId);
    });
    if (upstreamConfig?.upstream_mode === 'wallet_observer') {
      const legacyAccountId = Number(upstreamConfig?.upstream_account_id || 0);
      if (legacyAccountId > 0) ids.add(legacyAccountId);
    }
    return ids;
  }, [
    comboConfigs,
    upstreamConfig?.upstream_account_id,
    upstreamConfig?.upstream_mode,
  ]);

  // Auto-select first account when list changes
  useEffect(() => {
    if (!accounts.length) {
      if (editingAccountId) setEditingAccountId(0);
      setAccountTrend(null);
      return;
    }
    const current = accounts.find(
      (item) => item.id === Number(editingAccountId || 0),
    );
    if (current) return;
    const nextAccount =
      accounts.find((item) => item.enabled !== false) || accounts[0];
    setEditingAccountId(nextAccount?.id || 0);
  }, [editingAccountId, accounts]);

  const loadAccountTrend = useCallback(async (accountId) => {
    if (!accountId) {
      setAccountTrend(null);
      return;
    }
    setAccountTrend(null);
    setAccountTrendLoading(true);
    try {
      const end = Math.floor(Date.now() / 1000);
      const start = end - 7 * 24 * 60 * 60;
      const res = await API.get(
        `/api/profit_board/upstream_accounts/${accountId}/trend`,
        {
          params: {
            start_timestamp: start,
            end_timestamp: end,
            granularity: 'day',
          },
        },
      );
      if (!res.data.success) return showError(res.data.message);
      setAccountTrend(res.data.data || null);
    } catch (error) {
      showError(error);
    } finally {
      setAccountTrendLoading(false);
    }
  }, []);

  // Load trend when detail side sheet opens or account changes
  useEffect(() => {
    if (!detailSideSheetVisible || !editingAccountId) {
      setAccountTrend(null);
      return;
    }
    const currentAccount = accounts.find(
      (item) => item.id === Number(editingAccountId || 0),
    );
    if (currentAccount?.resource_display_mode === 'wallet') {
      setAccountTrend(null);
      return;
    }
    loadAccountTrend(editingAccountId);
  }, [accounts, detailSideSheetVisible, editingAccountId, loadAccountTrend]);

  const resetAccountDraftUiState = useCallback(() => {
    setAccountDraftTouched({});
    setAccountDraftSubmitted(false);
    setAccountNameManuallyEdited(false);
  }, []);

  const touchAccountDraftField = useCallback((field) => {
    if (!field) return;
    setAccountDraftTouched((prev) =>
      prev[field] ? prev : { ...prev, [field]: true },
    );
  }, []);

  const updateAccountDraftField = useCallback(
    (field, value) => {
      if (!field) return;
      touchAccountDraftField(field);
      if (field === 'name') {
        setAccountNameManuallyEdited(true);
      }
      setAccountDraft((prev) => {
        const next = {
          ...prev,
          [field]: field === 'user_id' ? Number(value || 0) : value,
        };
        if (field === 'base_url' && !accountNameManuallyEdited && !prev.id) {
          const suggestedName = getUpstreamAccountSuggestedName(value);
          if (suggestedName) {
            next.name = suggestedName;
          }
        }
        return next;
      });
    },
    [accountNameManuallyEdited, touchAccountDraftField],
  );

  const normalizeAccountDraftBaseUrl = useCallback(() => {
    touchAccountDraftField('base_url');
    setAccountDraft((prev) => {
      const normalizedBaseUrl = normalizeUpstreamAccountBaseUrl(prev.base_url);
      const next = {
        ...prev,
        base_url: normalizedBaseUrl,
      };
      if (!accountNameManuallyEdited && !prev.id) {
        const suggestedName = getUpstreamAccountSuggestedName(normalizedBaseUrl);
        if (suggestedName) {
          next.name = suggestedName;
        }
      }
      return next;
    });
  }, [accountNameManuallyEdited, touchAccountDraftField]);

  const accountDraftValidation = useMemo(
    () =>
      getUpstreamAccountDraftValidation(accountDraft, {
        allowSuggestedName: !accountNameManuallyEdited,
      }),
    [accountDraft, accountNameManuallyEdited],
  );

  const accountDraftErrors = useMemo(() => {
    const visibleErrors = {};
    Object.entries(accountDraftValidation.errors).forEach(([field, message]) => {
      if (accountDraftSubmitted || accountDraftTouched[field]) {
        visibleErrors[field] = message;
      }
    });
    return visibleErrors;
  }, [
    accountDraftSubmitted,
    accountDraftTouched,
    accountDraftValidation.errors,
  ]);

  const accountDraftCanSave = accountDraftValidation.isValid;

  const resetAccountDraft = useCallback(() => {
    setAccountDraft(createDefaultUpstreamAccountDraft());
    resetAccountDraftUiState();
  }, [resetAccountDraftUiState]);

  const openCreateSideSheet = useCallback(() => {
    setAccountDraft(createDefaultUpstreamAccountDraft());
    resetAccountDraftUiState();
    setSideSheetVisible(true);
  }, [resetAccountDraftUiState]);

  const openEditSideSheet = useCallback(
    (accountId) => {
      const account = accounts.find((item) => item.id === accountId);
      if (!account) return;
      setAccountDraft({
        id: account.id,
        name: account.name || '',
        remark: account.remark || '',
        account_type: account.account_type || 'newapi',
        base_url: account.base_url || '',
        user_id: account.user_id || 0,
        access_token: '',
        access_token_masked: account.access_token_masked || '',
        resource_display_mode: account.resource_display_mode || 'both',
        low_balance_threshold_usd: account.low_balance_threshold_usd || 0,
        enabled: account.enabled !== false,
      });
      setAccountDraftTouched({});
      setAccountDraftSubmitted(false);
      setAccountNameManuallyEdited(true);
      setSideSheetVisible(true);
    },
    [accounts],
  );

  const closeSideSheet = useCallback(() => {
    setSideSheetVisible(false);
  }, []);

  const openDetailSideSheet = useCallback(
    (accountId) => {
      if (!accountId) return;
      setEditingAccountId(Number(accountId));
      setDetailSideSheetVisible(true);
    },
    [],
  );

  const closeDetailSideSheet = useCallback(() => {
    setDetailSideSheetVisible(false);
  }, []);

  const syncAccountInternal = useCallback(
    async (accountId) => {
      if (!accountId) return false;
      setSyncingAccountId(accountId);
      try {
        const res = await API.post(
          `/api/profit_board/upstream_accounts/${accountId}/sync`,
        );
        if (!res.data.success) {
          showError(res.data.message);
          return false;
        }
        const syncedStatus = res.data.data?.status;
        if (syncedStatus === 'failed') {
          showError(
            '同步失败' +
              (res.data.data?.error_message
                ? `：${res.data.data.error_message}`
                : ''),
          );
        } else if (syncedStatus === 'needs_baseline') {
          showSuccess('首次同步完成，下次开始统计近 7 天已用');
        } else {
          showSuccess('账户数据已刷新');
        }
        await loadOptions();
        await loadAccountTrend(accountId);
        if (activeWalletAccountIds.has(Number(accountId))) {
          await runFullRefresh();
        }
        return syncedStatus !== 'failed';
      } catch (error) {
        showError(error);
        return false;
      } finally {
        setSyncingAccountId(0);
      }
    },
    [
      activeWalletAccountIds,
      loadAccountTrend,
      loadOptions,
      runFullRefresh,
    ],
  );

  const saveAccount = useCallback(async () => {
    setAccountDraftSubmitted(true);
    const validation = getUpstreamAccountDraftValidation(accountDraft, {
      allowSuggestedName: !accountNameManuallyEdited,
    });
    if (!validation.isValid) {
      showError(validation.firstError);
      return;
    }
    setSavingAccount(true);
    try {
      const preparedDraft = prepareUpstreamAccountDraftForSave(accountDraft, {
        allowSuggestedName: !accountNameManuallyEdited,
      });
      const isEditing = !!preparedDraft.id;
      const method = isEditing ? 'put' : 'post';
      const url = isEditing
        ? `/api/profit_board/upstream_accounts/${preparedDraft.id}`
        : '/api/profit_board/upstream_accounts';
      const res = await API[method](url, preparedDraft);
      if (!res.data.success) return showError(res.data.message);
      if (isEditing) {
        showSuccess('上游账户已更新');
        await loadOptions();
        setEditingAccountId(preparedDraft.id);
        await loadAccountTrend(preparedDraft.id);
      } else {
        const createdAccountId = res.data.data?.id || 0;
        showSuccess('上游账户已创建，正在自动同步');
        setUpstreamConfig((prev) => ({
          ...prev,
          upstream_account_id: createdAccountId || prev.upstream_account_id,
        }));
        setEditingAccountId(createdAccountId);
        if (createdAccountId > 0) {
          await syncAccountInternal(createdAccountId);
        } else {
          await loadOptions();
        }
      }
      setSideSheetVisible(false);
      resetAccountDraft();
    } catch (error) {
      showError(error);
    } finally {
      setSavingAccount(false);
    }
  }, [
    accountNameManuallyEdited,
    accountDraft,
    loadAccountTrend,
    loadOptions,
    resetAccountDraft,
    setUpstreamConfig,
    syncAccountInternal,
  ]);

  const syncAccount = useCallback(
    async (accountId) => {
      await syncAccountInternal(accountId);
    },
    [syncAccountInternal],
  );

  const syncAllAccounts = useCallback(async () => {
    setSyncingAllAccounts(true);
    try {
      const res = await API.post(
        '/api/profit_board/upstream_accounts/sync_all',
      );
      if (!res.data.success) return showError(res.data.message);
      showSuccess('全部账户已刷新');
      await loadOptions();
      if (editingAccountId) {
        await loadAccountTrend(editingAccountId);
      }
      if (activeWalletAccountIds.size > 0) {
        await runFullRefresh();
      }
    } catch (error) {
      showError(error);
    } finally {
      setSyncingAllAccounts(false);
    }
  }, [
    activeWalletAccountIds,
    editingAccountId,
    loadAccountTrend,
    loadOptions,
    runFullRefresh,
  ]);

  const deleteAccount = useCallback(
    async (accountId) => {
      if (!accountId) return;
      setDeletingAccountId(accountId);
      try {
        const res = await API.delete(
          `/api/profit_board/upstream_accounts/${accountId}`,
        );
        if (!res.data.success) return showError(res.data.message);
        showSuccess('上游账户已删除');
        await loadOptions();
        if (
          Number(upstreamConfig.upstream_account_id || 0) === Number(accountId)
        ) {
          setUpstreamConfig((prev) => ({
            ...prev,
            upstream_account_id: 0,
          }));
        }
        if (Number(editingAccountId || 0) === Number(accountId)) {
          setDetailSideSheetVisible(false);
          setAccountTrend(null);
        }
        setSideSheetVisible(false);
        resetAccountDraft();
      } catch (error) {
        showError(error);
      } finally {
        setDeletingAccountId(0);
      }
    },
    [
      loadOptions,
      editingAccountId,
      resetAccountDraft,
      setUpstreamConfig,
      upstreamConfig.upstream_account_id,
    ],
  );

  return {
    accounts,
    accountDraft,
    setAccountDraft,
    updateAccountDraftField,
    normalizeAccountDraftBaseUrl,
    touchAccountDraftField,
    accountDraftErrors,
    accountDraftCanSave,
    accountDraftValidation,
    editingAccountId,
    setEditingAccountId,
    editingAccount,
    accountTrend,
    accountTrendLoading,
    savingAccount,
    syncingAccountId,
    syncingAllAccounts,
    deletingAccountId,
    sideSheetVisible,
    detailSideSheetVisible,
    saveAccount,
    syncAccount,
    syncAllAccounts,
    deleteAccount,
    resetAccountDraft,
    loadAccountTrend,
    openCreateSideSheet,
    openEditSideSheet,
    closeSideSheet,
    openDetailSideSheet,
    closeDetailSideSheet,
  };
};
