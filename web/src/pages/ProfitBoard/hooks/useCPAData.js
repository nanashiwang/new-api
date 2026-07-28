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
import { useCallback, useEffect, useRef, useState } from 'react';
import { API } from '../../../helpers/api';
import { showError, showSuccess, showWarning } from '../../../helpers';
import { useLatestRequestGuard } from '../../../hooks/common/useLatestRequestGuard';

export function useCPAData() {
  const [sites, setSites] = useState([]);
  const [accounts, setAccounts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [refreshingSiteId, setRefreshingSiteId] = useState(null);
  const [deletingSiteId, setDeletingSiteId] = useState(null);
  const mountedRef = useRef(true);
  const overviewRequestGuard = useLatestRequestGuard();

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
    const requestId = overviewRequestGuard.createRequestId();
    safeSet(setLoading, true);
    try {
      const res = await API.get('/api/cpa/overview');
      if (
        overviewRequestGuard.isLatestRequest(requestId) &&
        res.data?.success
      ) {
        safeSet(setSites, res.data.sites ?? []);
        safeSet(setAccounts, res.data.accounts ?? []);
        return res.data;
      }
    } catch {
      // handled by the shared interceptor
    } finally {
      if (overviewRequestGuard.isLatestRequest(requestId)) {
        safeSet(setLoading, false);
      }
    }
    return null;
  }, [overviewRequestGuard, safeSet]);

  useEffect(() => {
    loadOverview();
  }, [loadOverview]);

  const testConnection = useCallback(
    async (payload) => {
      safeSet(setTesting, true);
      try {
        const res = await API.post('/api/cpa/test_connection', payload);
        if (res.data?.success) {
          showSuccess(`连接成功，发现 ${res.data.account_count ?? 0} 个账号`);
          return true;
        }
        showError(res.data?.message ?? '连接失败');
      } catch {
        // handled by the shared interceptor
      } finally {
        safeSet(setTesting, false);
      }
      return false;
    },
    [safeSet],
  );

  const saveSite = useCallback(
    async (site, payload) => {
      safeSet(setSaving, true);
      try {
        const res = site
          ? await API.put(`/api/cpa/sites/${site.id}`, payload)
          : await API.post('/api/cpa/sites', payload);
        if (res.data?.success) {
          if (res.data.sync_success) {
            showSuccess(site ? 'CPA 服务已更新' : '服务已创建并同步');
          } else {
            showWarning(res.data?.message ?? '服务已保存，但首次同步失败');
          }
          await loadOverview();
          return true;
        }
        showError(res.data?.message ?? '保存失败');
      } catch {
        // handled by the shared interceptor
      } finally {
        safeSet(setSaving, false);
      }
      return false;
    },
    [loadOverview, safeSet],
  );

  const refreshSite = useCallback(
    async (id) => {
      safeSet(setRefreshingSiteId, id);
      try {
        const res = await API.post(`/api/cpa/sites/${id}/refresh`, {});
        if (res.data?.success) {
          showSuccess('同步成功');
          await loadOverview();
          return true;
        }
        showError(res.data?.message ?? '同步失败');
      } catch {
        // handled by the shared interceptor
      } finally {
        safeSet(setRefreshingSiteId, null);
      }
      return false;
    },
    [loadOverview, safeSet],
  );

  const deleteSite = useCallback(
    async (id) => {
      safeSet(setDeletingSiteId, id);
      try {
        const res = await API.delete(`/api/cpa/sites/${id}`);
        if (res.data?.success) {
          showSuccess('CPA 服务已删除');
          await loadOverview();
          return true;
        }
        showError(res.data?.message ?? '删除失败');
      } catch {
        // handled by the shared interceptor
      } finally {
        safeSet(setDeletingSiteId, null);
      }
      return false;
    },
    [loadOverview, safeSet],
  );

  return {
    sites,
    accounts,
    loading,
    saving,
    testing,
    refreshingSiteId,
    deletingSiteId,
    loadOverview,
    testConnection,
    saveSite,
    refreshSite,
    deleteSite,
  };
}
