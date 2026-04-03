import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import { createDefaultUpstreamAccountDraft } from '../utils';

export const useUpstreamAccounts = ({
  options,
  loadOptions,
  upstreamConfig,
  setUpstreamConfig,
  runFullRefresh,
}) => {
  const { t } = useTranslation();
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

  // Load trend when editing account changes
  useEffect(() => {
    if (!editingAccountId) {
      setAccountTrend(null);
      return;
    }
    loadAccountTrend(editingAccountId);
  }, [editingAccountId, loadAccountTrend]);

  // Sync draft from editing account
  useEffect(() => {
    if (!editingAccount) return;
    if (Number(accountDraft.id || 0) === Number(editingAccount.id || 0)) return;
    setAccountDraft({
      id: editingAccount.id,
      name: editingAccount.name || '',
      remark: editingAccount.remark || '',
      account_type: editingAccount.account_type || 'newapi',
      base_url: editingAccount.base_url || '',
      user_id: editingAccount.user_id || 0,
      access_token: '',
      access_token_masked: editingAccount.access_token_masked || '',
      low_balance_threshold_usd: editingAccount.low_balance_threshold_usd || 0,
      enabled: editingAccount.enabled !== false,
    });
  }, [accountDraft.id, editingAccount]);

  const resetAccountDraft = useCallback(() => {
    setEditingAccountId(0);
    setAccountDraft(createDefaultUpstreamAccountDraft());
  }, []);

  const openCreateSideSheet = useCallback(() => {
    setAccountDraft(createDefaultUpstreamAccountDraft());
    setSideSheetVisible(true);
  }, []);

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
        low_balance_threshold_usd: account.low_balance_threshold_usd || 0,
        enabled: account.enabled !== false,
      });
      setSideSheetVisible(true);
    },
    [accounts],
  );

  const closeSideSheet = useCallback(() => {
    setSideSheetVisible(false);
  }, []);

  const saveAccount = useCallback(async () => {
    setSavingAccount(true);
    try {
      const method = accountDraft.id ? 'put' : 'post';
      const url = accountDraft.id
        ? `/api/profit_board/upstream_accounts/${accountDraft.id}`
        : '/api/profit_board/upstream_accounts';
      const res = await API[method](url, accountDraft);
      if (!res.data.success) return showError(res.data.message);
      showSuccess(accountDraft.id ? t('上游账户已更新') : t('上游账户已创建'));
      await loadOptions();
      if (accountDraft.id) {
        setEditingAccountId(accountDraft.id);
        await loadAccountTrend(accountDraft.id);
      }
      if (!accountDraft.id && res.data.data?.id) {
        setUpstreamConfig((prev) => ({
          ...prev,
          upstream_account_id: res.data.data.id,
        }));
        setEditingAccountId(res.data.data.id);
        await loadAccountTrend(res.data.data.id);
      }
      setSideSheetVisible(false);
      resetAccountDraft();
    } catch (error) {
      showError(error);
    } finally {
      setSavingAccount(false);
    }
  }, [
    accountDraft,
    loadAccountTrend,
    loadOptions,
    resetAccountDraft,
    setUpstreamConfig,
    t,
  ]);

  const syncAccount = useCallback(
    async (accountId) => {
      if (!accountId) return;
      setSyncingAccountId(accountId);
      try {
        const res = await API.post(
          `/api/profit_board/upstream_accounts/${accountId}/sync`,
        );
        if (!res.data.success) return showError(res.data.message);
        const syncedStatus = res.data.data?.status;
        if (syncedStatus === 'failed') {
          showError(
            t('同步失败') +
              (res.data.data?.error_message
                ? `：${res.data.data.error_message}`
                : ''),
          );
        } else if (syncedStatus === 'needs_baseline') {
          showSuccess(t('首次同步完成，下次开始统计近 7 天已用'));
        } else {
          showSuccess(t('账户数据已刷新'));
        }
        await loadOptions();
        await loadAccountTrend(accountId);
        if (
          Number(upstreamConfig.upstream_account_id || 0) === Number(accountId)
        ) {
          await runFullRefresh();
        }
      } catch (error) {
        showError(error);
      } finally {
        setSyncingAccountId(0);
      }
    },
    [
      loadAccountTrend,
      loadOptions,
      runFullRefresh,
      t,
      upstreamConfig.upstream_account_id,
    ],
  );

  const syncAllAccounts = useCallback(async () => {
    setSyncingAllAccounts(true);
    try {
      const res = await API.post(
        '/api/profit_board/upstream_accounts/sync_all',
      );
      if (!res.data.success) return showError(res.data.message);
      showSuccess(t('全部账户已刷新'));
      await loadOptions();
      if (editingAccountId) {
        await loadAccountTrend(editingAccountId);
      }
      if (Number(upstreamConfig.upstream_account_id || 0) > 0) {
        await runFullRefresh();
      }
    } catch (error) {
      showError(error);
    } finally {
      setSyncingAllAccounts(false);
    }
  }, [
    editingAccountId,
    loadAccountTrend,
    loadOptions,
    runFullRefresh,
    t,
    upstreamConfig.upstream_account_id,
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
        showSuccess(t('上游账户已删除'));
        await loadOptions();
        if (
          Number(upstreamConfig.upstream_account_id || 0) === Number(accountId)
        ) {
          setUpstreamConfig((prev) => ({
            ...prev,
            upstream_account_id: 0,
          }));
        }
        setSideSheetVisible(false);
        resetAccountDraft();
      } catch (error) {
        showError(error);
      } finally {
        setDeletingAccountId(0);
      }
    },
    [loadOptions, resetAccountDraft, setUpstreamConfig, t, upstreamConfig.upstream_account_id],
  );

  return {
    accounts,
    accountDraft,
    setAccountDraft,
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
    saveAccount,
    syncAccount,
    syncAllAccounts,
    deleteAccount,
    resetAccountDraft,
    loadAccountTrend,
    openCreateSideSheet,
    openEditSideSheet,
    closeSideSheet,
  };
};
