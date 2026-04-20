import { useCallback, useEffect, useRef, useState } from 'react';
import { API } from '../../../helpers/api';
import { showError, showSuccess } from '../../../helpers';

export function useCRSData() {
  const [sites, setSites] = useState([]);
  const [aggregate, setAggregate] = useState(null);
  const [loadingOverview, setLoadingOverview] = useState(false);
  const [refreshingAll, setRefreshingAll] = useState(false);
  const [refreshingSiteId, setRefreshingSiteId] = useState(null);
  const [savingSite, setSavingSite] = useState(false);
  const [deletingSiteId, setDeletingSiteId] = useState(null);
  const mountedRef = useRef(true);

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
      }
    } catch (err) {
      // showError is already handled by axios interceptor
    } finally {
      safeSet(setLoadingOverview, false);
    }
  }, [safeSet]);

  // 初始加载
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
          await loadOverview();
        }
      } catch (err) {
        // handled by interceptor
      } finally {
        safeSet(setRefreshingSiteId, null);
      }
    },
    [loadOverview, safeSet],
  );

  const refreshAll = useCallback(async () => {
    safeSet(setRefreshingAll, true);
    try {
      await API.post('/api/crs/refresh_all', {});
      showSuccess('全部刷新完成');
      await loadOverview();
    } catch (err) {
      // handled by interceptor
    } finally {
      safeSet(setRefreshingAll, false);
    }
  }, [loadOverview, safeSet]);

  const createSite = useCallback(
    async (payload) => {
      safeSet(setSavingSite, true);
      try {
        const res = await API.post('/api/crs/sites', payload);
        if (res.data?.success) {
          showSuccess('站点已创建');
          await loadOverview();
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
    [loadOverview, safeSet],
  );

  const updateSite = useCallback(
    async (id, payload) => {
      safeSet(setSavingSite, true);
      try {
        const res = await API.put(`/api/crs/sites/${id}`, payload);
        if (res.data?.success) {
          showSuccess('站点已更新');
          await loadOverview();
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
    [loadOverview, safeSet],
  );

  const deleteSite = useCallback(
    async (id) => {
      safeSet(setDeletingSiteId, id);
      try {
        const res = await API.delete(`/api/crs/sites/${id}`);
        if (res.data?.success) {
          showSuccess('站点已删除');
          await loadOverview();
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
    [loadOverview, safeSet],
  );

  return {
    sites,
    aggregate,
    loadingOverview,
    refreshingAll,
    refreshingSiteId,
    savingSite,
    deletingSiteId,
    loadOverview,
    refreshSite,
    refreshAll,
    createSite,
    updateSite,
    deleteSite,
  };
}
