import { useCallback, useEffect, useRef, useState } from 'react';
import { API } from '../../../helpers/api';
import { showError, showSuccess } from '../../../helpers';

export function useCRSData() {
  const [sites, setSites] = useState([]);
  const [aggregate, setAggregate] = useState(null);
  const [observer, setObserver] = useState(null);
  const [accounts, setAccounts] = useState([]);
  const [accountsTotal, setAccountsTotal] = useState(0);
  const [loadingOverview, setLoadingOverview] = useState(false);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [refreshingAll, setRefreshingAll] = useState(false);
  const [refreshingSiteId, setRefreshingSiteId] = useState(null);
  const [savingSite, setSavingSite] = useState(false);
  const [deletingSiteId, setDeletingSiteId] = useState(null);
  const [siteDetail, setSiteDetail] = useState(null);
  const [loadingSiteDetail, setLoadingSiteDetail] = useState(false);
  const mountedRef = useRef(true);
  const lastAccountsQueryRef = useRef({
    page: 1,
    page_size: 50,
    site_id: 0,
    platform: '',
    status: '',
    quota_state: '',
    keyword: '',
  });

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const safeSet = useCallback((setter, value) => {
    if (mountedRef.current) setter(value);
  }, []);

  const loadOverview = useCallback(async () => {
    safeSet(setLoadingOverview, true);
    try {
      const res = await API.get('/api/crs/overview');
      if (res.data?.success) {
        safeSet(setSites, res.data.sites ?? []);
        safeSet(setAggregate, res.data.aggregate ?? null);
        safeSet(setObserver, res.data.observer ?? null);
      }
    } catch (err) {
      // showError is already handled by axios interceptor
    } finally {
      safeSet(setLoadingOverview, false);
    }
  }, [safeSet]);

  const loadAccounts = useCallback(
    async (query = {}) => {
      const nextQuery = {
        ...lastAccountsQueryRef.current,
        ...query,
      };
      lastAccountsQueryRef.current = nextQuery;
      safeSet(setLoadingAccounts, true);
      try {
        const params = new URLSearchParams();
        Object.entries(nextQuery).forEach(([key, value]) => {
          if (value === '' || value == null) return;
          params.set(key, String(value));
        });
        const res = await API.get(`/api/crs/accounts?${params.toString()}`);
        if (res.data?.success) {
          safeSet(setAccounts, res.data.data ?? []);
          safeSet(setAccountsTotal, res.data.total ?? 0);
          return res.data;
        }
      } catch (err) {
        // handled by interceptor
      } finally {
        safeSet(setLoadingAccounts, false);
      }
      return null;
    },
    [safeSet],
  );

  const loadSiteAccounts = useCallback(
    async (id) => {
      if (!id) return null;
      safeSet(setLoadingSiteDetail, true);
      try {
        const res = await API.get(`/api/crs/sites/${id}/accounts`);
        if (res.data?.success) {
          safeSet(setSiteDetail, res.data);
          return res.data;
        }
      } catch (err) {
        // handled by interceptor
      } finally {
        safeSet(setLoadingSiteDetail, false);
      }
      return null;
    },
    [safeSet],
  );

  useEffect(() => {
    loadOverview();
  }, [loadOverview]);

  const refreshSite = useCallback(
    async (id) => {
      safeSet(setRefreshingSiteId, id);
      try {
        const res = await API.post(`/api/crs/sites/${id}/refresh`, {});
        if (res.data?.success) {
          showSuccess('刷新成功');
          await Promise.all([loadOverview(), loadAccounts()]);
          if (siteDetail?.site?.id === id) {
            await loadSiteAccounts(id);
          }
          return true;
        }
        showError(res.data?.message ?? '刷新失败');
      } catch (err) {
        // handled by interceptor
      } finally {
        safeSet(setRefreshingSiteId, null);
      }
      return false;
    },
    [
      loadAccounts,
      loadOverview,
      loadSiteAccounts,
      safeSet,
      siteDetail?.site?.id,
    ],
  );

  const refreshAll = useCallback(async () => {
    safeSet(setRefreshingAll, true);
    try {
      const res = await API.post('/api/crs/refresh_all', {});
      if (res.data?.success) {
        const failedItems = (res.data.data ?? []).filter(
          (item) => !item.success,
        );
        if (failedItems.length > 0) {
          showError(`有 ${failedItems.length} 个站点刷新失败`);
        } else {
          showSuccess('全部刷新完成');
        }
        await Promise.all([loadOverview(), loadAccounts()]);
        if (siteDetail?.site?.id) {
          await loadSiteAccounts(siteDetail.site.id);
        }
        return failedItems.length === 0;
      }
      showError(res.data?.message ?? '批量刷新失败');
    } catch (err) {
      // handled by interceptor
    } finally {
      safeSet(setRefreshingAll, false);
    }
    return false;
  }, [loadAccounts, loadOverview, loadSiteAccounts, safeSet, siteDetail]);

  const createSite = useCallback(
    async (payload) => {
      safeSet(setSavingSite, true);
      try {
        const res = await API.post('/api/crs/sites', payload);
        if (res.data?.success) {
          showSuccess('站点已创建');
          await Promise.all([loadOverview(), loadAccounts({ page: 1 })]);
          return true;
        }
        showError(res.data?.message ?? '创建失败');
        return false;
      } catch (err) {
        return false;
      } finally {
        safeSet(setSavingSite, false);
      }
    },
    [loadAccounts, loadOverview, safeSet],
  );

  const updateSite = useCallback(
    async (id, payload) => {
      safeSet(setSavingSite, true);
      try {
        const res = await API.put(`/api/crs/sites/${id}`, payload);
        if (res.data?.success) {
          showSuccess('站点已更新');
          await Promise.all([loadOverview(), loadAccounts()]);
          if (siteDetail?.site?.id === id) {
            await loadSiteAccounts(id);
          }
          return true;
        }
        showError(res.data?.message ?? '更新失败');
        return false;
      } catch (err) {
        return false;
      } finally {
        safeSet(setSavingSite, false);
      }
    },
    [
      loadAccounts,
      loadOverview,
      loadSiteAccounts,
      safeSet,
      siteDetail?.site?.id,
    ],
  );

  const deleteSite = useCallback(
    async (id) => {
      safeSet(setDeletingSiteId, id);
      try {
        const res = await API.delete(`/api/crs/sites/${id}`);
        if (res.data?.success) {
          showSuccess('站点已删除');
          await Promise.all([loadOverview(), loadAccounts({ page: 1 })]);
          if (siteDetail?.site?.id === id) {
            safeSet(setSiteDetail, null);
          }
          return true;
        }
        showError(res.data?.message ?? '删除失败');
        return false;
      } catch (err) {
        return false;
      } finally {
        safeSet(setDeletingSiteId, null);
      }
    },
    [loadAccounts, loadOverview, safeSet, siteDetail?.site?.id],
  );

  return {
    sites,
    aggregate,
    observer,
    accounts,
    accountsTotal,
    loadingOverview,
    loadingAccounts,
    refreshingAll,
    refreshingSiteId,
    savingSite,
    deletingSiteId,
    siteDetail,
    loadingSiteDetail,
    loadOverview,
    loadAccounts,
    loadSiteAccounts,
    setSiteDetail,
    refreshSite,
    refreshAll,
    createSite,
    updateSite,
    deleteSite,
  };
}
