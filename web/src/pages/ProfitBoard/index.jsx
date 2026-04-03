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

import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Empty, Spin, Tabs, Tag, Typography } from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import { BadgeDollarSign, BarChart3, CircleDollarSign } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import ChartAnalysisCard from './components/ChartAnalysisCard';
import ComboManagerCard from './components/ComboManagerCard';
import DetailTableCard from './components/DetailTableCard';
import OverviewPanel from './components/OverviewPanel';
import PricingRulesCard from './components/PricingRulesCard';
import ProfitBoardHeader from './components/ProfitBoardHeader';
import UpstreamWalletCard from './components/UpstreamWalletCard';
import TimeRangePanel from './components/TimeRangePanel';
import {
  DETAIL_LIMIT,
  REPORT_CACHE_KEY,
  STORAGE_KEY,
  aggregateBreakdownRows,
  buildQueryKey,
  clampNumber,
  combineBreakdownMetrics,
  combineTimeseriesMetrics,
  createBarSpec,
  createDefaultComboPricingConfig,
  createDefaultDraft,
  createDefaultPricingRule,
  createDefaultUpstreamAccountDraft,
  createPresetRanges,
  createTrendSpec,
  formatMoney,
  formatRangeDuration,
  formatRangeLabel,
  formatRatio,
  metricOptions,
  normalizeBatchForState,
  normalizeCachedReportBundle,
  normalizeRestoredState,
  safeParse,
  sitePricingSourceLabelMap,
} from './utils';

const { Text } = Typography;
initVChartSemiTheme();

const ProfitBoardPage = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const isMobile = useIsMobile();
  const cachedBundle = useMemo(
    () =>
      normalizeCachedReportBundle(
        safeParse(localStorage.getItem(REPORT_CACHE_KEY), null),
      ),
    [],
  );
  const restoredState = useMemo(
    () =>
      normalizeRestoredState(safeParse(localStorage.getItem(STORAGE_KEY), {})),
    [],
  );

  const [loading, setLoading] = useState(false);
  const [querying, setQuerying] = useState(false);
  const [overviewQuerying, setOverviewQuerying] = useState(false);
  const [saving, setSaving] = useState(false);
  const [savingAccount, setSavingAccount] = useState(false);
  const [syncingAccountId, setSyncingAccountId] = useState(0);
  const [deletingAccountId, setDeletingAccountId] = useState(0);
  const [activityChecking, setActivityChecking] = useState(false);
  const [options, setOptions] = useState({
    channels: [],
    tags: [],
    groups: [],
    local_models: [],
    site_models: [],
    upstream_accounts: [],
  });
  const [batches, setBatches] = useState(restoredState.batches || []);
  const [draft, setDraft] = useState(
    restoredState.draft || createDefaultDraft(),
  );
  const [editingBatchId, setEditingBatchId] = useState(
    restoredState.editingBatchId || '',
  );
  const [dateRange, setDateRange] = useState(restoredState.dateRange);
  const [granularity, setGranularity] = useState(
    restoredState.granularity || 'day',
  );
  const [customIntervalMinutes, setCustomIntervalMinutes] = useState(
    restoredState.customIntervalMinutes || 15,
  );
  const [chartTab, setChartTab] = useState(restoredState.chartTab || 'trend');
  const [metricKey, setMetricKey] = useState(
    restoredState.metricKey || 'configured_profit_usd',
  );
  const [analysisMode, setAnalysisMode] = useState(
    restoredState.analysisMode || 'business_compare',
  );
  const [viewBatchId, setViewBatchId] = useState(
    restoredState.viewBatchId || 'all',
  );
  const [detailFilter, setDetailFilter] = useState(
    restoredState.detailFilter || null,
  );
  const [comboConfigs, setComboConfigs] = useState(
    restoredState.comboConfigs || [],
  );
  const [upstreamConfig, setUpstreamConfig] = useState(
    restoredState.upstreamConfig || {},
  );
  const [siteConfig, setSiteConfig] = useState(restoredState.siteConfig || {});
  const [overviewReport, setOverviewReport] = useState(null);
  const [report, setReport] = useState(cachedBundle?.report || null);
  const [lastQueryKey, setLastQueryKey] = useState(
    cachedBundle?.queryKey || restoredState.lastQueryKey || '',
  );
  const [autoRefreshMode, setAutoRefreshMode] = useState(
    restoredState.autoRefreshMode || false,
  );
  const [hasNewActivity, setHasNewActivity] = useState(false);
  const [detailRows, setDetailRows] = useState([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailPage, setDetailPage] = useState(restoredState.detailPage || 1);
  const [detailPageSize, setDetailPageSize] = useState(
    restoredState.detailPageSize || 12,
  );
  const [detailTotal, setDetailTotal] = useState(0);
  const [accountDraft, setAccountDraft] = useState(
    createDefaultUpstreamAccountDraft(),
  );
  const [editingAccountId, setEditingAccountId] = useState(0);
  const lastActivityWatermarkRef = useRef(
    cachedBundle?.activityWatermark || '',
  );

  const channelOptions = useMemo(
    () =>
      (options.channels || []).map((item) => ({
        label: item.tag ? `${item.name} (${item.tag})` : item.name,
        value: String(item.id),
      })),
    [options.channels],
  );
  const channelMap = useMemo(
    () =>
      new Map((options.channels || []).map((item) => [String(item.id), item])),
    [options.channels],
  );
  const tagChannelMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((item) => {
      const tag = item.tag || '';
      if (!tag) return;
      const current = map.get(tag) || [];
      current.push(String(item.id));
      map.set(tag, current);
    });
    return map;
  }, [options.channels]);
  const localModelMap = useMemo(
    () =>
      new Map(
        (options.local_models || []).map((item) => [item.model_name, item]),
      ),
    [options.local_models],
  );
  const modelNameOptions = useMemo(
    () =>
      (options.site_models || []).map((item) => ({
        label: item,
        value: item,
      })),
    [options.site_models],
  );
  const batchPayload = useMemo(
    () =>
      batches.map((batch) => ({
        id: batch.id,
        name: batch.name?.trim() || t('未命名组合'),
        scope_type: batch.scope_type,
        channel_ids: (batch.channel_ids || [])
          .map((item) => Number(item))
          .filter(Boolean),
        tags: batch.tags || [],
      })),
    [batches, t],
  );
  const configLookupKey = useMemo(
    () => JSON.stringify(batchPayload),
    [batchPayload],
  );
  const currentQueryKey = useMemo(
    () =>
      buildQueryKey({
        batches: batchPayload,
        shared_site: siteConfig,
        combo_configs: comboConfigs,
        upstream: upstreamConfig,
        site: siteConfig,
        start_timestamp: Math.floor(
          new Date(dateRange?.[0] || 0).getTime() / 1000,
        ),
        end_timestamp: Math.floor(
          new Date(dateRange?.[1] || 0).getTime() / 1000,
        ),
        granularity,
        custom_interval_minutes:
          granularity === 'custom' ? customIntervalMinutes : 0,
      }),
    [
      batchPayload,
      comboConfigs,
      customIntervalMinutes,
      dateRange,
      granularity,
      siteConfig,
      upstreamConfig,
    ],
  );
  const reportMatchesCurrentFilters =
    !!report && lastQueryKey === currentQueryKey;
  const autoRefreshEligible = useMemo(
    () =>
      !!dateRange?.[1] &&
      Math.abs(Date.now() - new Date(dateRange[1]).getTime()) <= 15 * 60 * 1000,
    [dateRange],
  );
  const walletModeEnabled =
    upstreamConfig.upstream_mode === 'wallet_observer';
  const selectedAccount = useMemo(
    () =>
      (options.upstream_accounts || []).find(
        (item) => item.id === Number(upstreamConfig.upstream_account_id || 0),
      ) || null,
    [options.upstream_accounts, upstreamConfig.upstream_account_id],
  );

  const duplicateBatchError = useMemo(() => {
    const ownerMap = new Map();
    for (const batch of batches) {
      const channelIds =
        batch.scope_type === 'tag'
          ? (batch.tags || []).flatMap((tag) => tagChannelMap.get(tag) || [])
          : batch.channel_ids || [];
      for (const channelId of Array.from(new Set(channelIds))) {
        const owner = ownerMap.get(channelId);
        if (owner && owner !== batch.name) {
          const channelName =
            channelMap.get(String(channelId))?.name || `#${channelId}`;
          return `${channelName} 同时出现在组合“${owner}”和“${batch.name}”中，请拆开后再统计`;
        }
        ownerMap.set(channelId, batch.name);
      }
    }
    return '';
  }, [batches, channelMap, tagChannelMap]);

  const validationErrors = useMemo(() => {
    const errors = [];
    if (!batches.length) errors.push(t('请至少添加一个组合'));
    if (duplicateBatchError) errors.push(duplicateBatchError);
    if (
      comboConfigs.some((item) => item.site_mode === 'shared_site_model') &&
      !(siteConfig.model_names || []).length
    ) {
      errors.push(t('有组合启用了读取本站模型价格，请至少选择一个本站模型'));
    }
    if (
      upstreamConfig.upstream_mode === 'wallet_observer' &&
      !Number(upstreamConfig.upstream_account_id || 0)
    ) {
      errors.push(t('钱包扣减模式必须绑定一个上游账户'));
    }
    if (!Array.isArray(dateRange) || !dateRange[0] || !dateRange[1])
      errors.push(t('请选择完整的时间范围'));
    return errors;
  }, [
    batches.length,
    comboConfigs,
    dateRange,
    duplicateBatchError,
    siteConfig.model_names,
    upstreamConfig.upstream_account_id,
    upstreamConfig.upstream_mode,
    t,
  ]);

  const configPayload = useMemo(
    () => ({
      batches: batchPayload,
      shared_site: {
        model_names: siteConfig.model_names || [],
        group: siteConfig.group || '',
        use_recharge_price: !!siteConfig.use_recharge_price,
      },
      combo_configs: comboConfigs,
      upstream: { ...upstreamConfig, fixed_amount: 0 },
      site: { ...siteConfig, fixed_amount: 0 },
    }),
    [batchPayload, comboConfigs, siteConfig, upstreamConfig],
  );

  const queryPayload = useMemo(
    () => ({
      ...configPayload,
      start_timestamp: Math.floor(
        new Date(dateRange?.[0] || 0).getTime() / 1000,
      ),
      end_timestamp: Math.floor(new Date(dateRange?.[1] || 0).getTime() / 1000),
      granularity,
      custom_interval_minutes:
        granularity === 'custom' ? customIntervalMinutes : 0,
      include_details: false,
      detail_limit: 0,
    }),
    [configPayload, customIntervalMinutes, dateRange, granularity],
  );

  const persistState = useCallback(() => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        batches,
        draft,
        editingBatchId,
        dateRange,
        granularity,
        customIntervalMinutes,
        chartTab,
        metricKey,
        analysisMode,
        viewBatchId,
        detailFilter,
        comboConfigs,
        upstreamConfig,
        siteConfig,
        lastQueryKey,
        detailPage,
        detailPageSize,
        autoRefreshMode,
      }),
    );
  }, [
    analysisMode,
    autoRefreshMode,
    batches,
    chartTab,
    comboConfigs,
    customIntervalMinutes,
    dateRange,
    detailFilter,
    detailPage,
    detailPageSize,
    draft,
    editingBatchId,
    granularity,
    lastQueryKey,
    metricKey,
    siteConfig,
    upstreamConfig,
    viewBatchId,
  ]);
  useEffect(() => {
    persistState();
  }, [persistState]);

  useEffect(() => {
    setComboConfigs((prev) =>
      batches.map((batch) => {
        const existing = (prev || []).find(
          (item) => item.combo_id === batch.id,
        );
        const fallback = createDefaultComboPricingConfig(
          batch.id,
          siteConfig,
          upstreamConfig,
        );
        return {
          ...fallback,
          ...(existing || {}),
          combo_id: batch.id,
          site_rules: (existing?.site_rules || fallback.site_rules || []).map(
            (rule) => createDefaultPricingRule(rule),
          ),
          upstream_rules: (
            existing?.upstream_rules ||
            fallback.upstream_rules ||
            []
          ).map((rule) => createDefaultPricingRule(rule)),
        };
      }),
    );
  }, [batches, siteConfig, upstreamConfig]);

  const loadOptions = useCallback(async () => {
    const res = await API.get('/api/profit_board/options');
    if (!res.data.success)
      throw new Error(res.data.message || t('加载选项失败'));
    setOptions(
      res.data.data || {
        channels: [],
        tags: [],
        groups: [],
        local_models: [],
        site_models: [],
        upstream_accounts: [],
      },
    );
  }, [t]);

  const loadConfig = useCallback(async () => {
    if (!configLookupKey || configLookupKey === '[]') return;
    const res = await API.get('/api/profit_board/config', {
      params: { batches: configLookupKey },
    });
    if (!res.data.success)
      throw new Error(res.data.message || t('加载配置失败'));
    const config = res.data.data?.config;
    if (!config) return;
    setSiteConfig((prev) => ({
      ...prev,
      ...(config.shared_site || {}),
      model_names: config.shared_site?.model_names || [],
    }));
    setUpstreamConfig((prev) => ({ ...prev, ...(config.upstream || {}) }));
    setComboConfigs(
      (config.combo_configs || []).map((item) => ({
        ...createDefaultComboPricingConfig(
          item.combo_id || '',
          config.site,
          config.upstream,
        ),
        ...item,
        site_rules: (item.site_rules || []).map((rule) =>
          createDefaultPricingRule(rule),
        ),
        upstream_rules: (item.upstream_rules || []).map((rule) =>
          createDefaultPricingRule(rule),
        ),
      })),
    );
  }, [configLookupKey, t]);

  useEffect(() => {
    const bootstrap = async () => {
      setLoading(true);
      try {
        await loadOptions();
      } catch (error) {
        showError(error);
      } finally {
        setLoading(false);
      }
    };
    bootstrap();
  }, [loadOptions]);

  useEffect(() => {
    if (batchPayload.length) loadConfig().catch(showError);
  }, [batchPayload.length, configLookupKey, loadConfig]);

  const runOverviewQuery = useCallback(async () => {
    if (validationErrors.length > 0) return;
    setOverviewQuerying(true);
    try {
      const res = await API.post('/api/profit_board/overview', configPayload);
      if (!res.data.success) return showError(res.data.message);
      setOverviewReport(res.data.data);
    } catch (error) {
      showError(error);
    } finally {
      setOverviewQuerying(false);
    }
  }, [configPayload, validationErrors.length]);

  const runQuery = useCallback(async () => {
    if (validationErrors.length > 0) return showError(validationErrors[0]);
    setQuerying(true);
    setDetailPage(1);
    try {
      const res = await API.post('/api/profit_board/query', queryPayload);
      if (!res.data.success) return showError(res.data.message);
      const nextReport = res.data.data;
      setReport(nextReport);
      setLastQueryKey(currentQueryKey);
      setHasNewActivity(false);
      lastActivityWatermarkRef.current =
        nextReport?.meta?.activity_watermark || '';
      localStorage.setItem(
        REPORT_CACHE_KEY,
        JSON.stringify({
          report: nextReport,
          queryKey: currentQueryKey,
          activityWatermark: nextReport?.meta?.activity_watermark || '',
        }),
      );
    } catch (error) {
      showError(error);
    } finally {
      setQuerying(false);
    }
  }, [currentQueryKey, queryPayload, validationErrors]);

  const runFullRefresh = useCallback(async () => {
    await runOverviewQuery();
    await runQuery();
  }, [runOverviewQuery, runQuery]);

  const checkActivity = useCallback(async () => {
    if (!autoRefreshMode || !autoRefreshEligible || validationErrors.length > 0)
      return;
    setActivityChecking(true);
    try {
      const res = await API.post('/api/profit_board/activity', queryPayload);
      if (!res.data.success) return;
      const nextWatermark = res.data.data?.activity_watermark || '';
      if (!lastActivityWatermarkRef.current)
        lastActivityWatermarkRef.current = nextWatermark;
      else if (
        nextWatermark &&
        nextWatermark !== lastActivityWatermarkRef.current
      )
        setHasNewActivity(true);
    } catch (error) {
    } finally {
      setActivityChecking(false);
    }
  }, [
    autoRefreshEligible,
    autoRefreshMode,
    queryPayload,
    validationErrors.length,
  ]);

  useEffect(() => {
    if (!autoRefreshMode) return undefined;
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') checkActivity();
    }, 90000);
    return () => window.clearInterval(timer);
  }, [autoRefreshMode, checkActivity]);

  const saveConfig = useCallback(async () => {
    if (validationErrors.length > 0) return showError(validationErrors[0]);
    setSaving(true);
    try {
      const res = await API.put('/api/profit_board/config', configPayload);
      if (!res.data.success) return showError(res.data.message);
      const savedConfig = res.data.data?.config;
      if (savedConfig) {
        setSiteConfig((prev) => ({
          ...prev,
          ...(savedConfig.shared_site || {}),
        }));
        setUpstreamConfig((prev) => ({
          ...prev,
          ...(savedConfig.upstream || {}),
        }));
        setComboConfigs(
          (savedConfig.combo_configs || []).map((item) => ({
            ...createDefaultComboPricingConfig(
              item.combo_id || '',
              savedConfig.site,
              savedConfig.upstream,
            ),
            ...item,
            site_rules: (item.site_rules || []).map((rule) =>
              createDefaultPricingRule(rule),
            ),
            upstream_rules: (item.upstream_rules || []).map((rule) =>
              createDefaultPricingRule(rule),
            ),
          })),
        );
      }
      showSuccess(t('收益看板配置已保存'));
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  }, [configPayload, t, validationErrors]);

  const resetAccountDraft = useCallback(() => {
    setEditingAccountId(0);
    setAccountDraft(createDefaultUpstreamAccountDraft());
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
      showSuccess(
        accountDraft.id ? t('上游账户已更新') : t('上游账户已创建'),
      );
      await loadOptions();
      if (!accountDraft.id && res.data.data?.id) {
        setUpstreamConfig((prev) => ({
          ...prev,
          upstream_account_id: res.data.data.id,
        }));
      }
      resetAccountDraft();
    } catch (error) {
      showError(error);
    } finally {
      setSavingAccount(false);
    }
  }, [accountDraft, loadOptions, resetAccountDraft, t]);

  const syncAccount = useCallback(
    async (accountId) => {
      if (!accountId) return;
      setSyncingAccountId(accountId);
      try {
        const res = await API.post(
          `/api/profit_board/upstream_accounts/${accountId}/sync`,
        );
        if (!res.data.success) return showError(res.data.message);
        showSuccess(t('上游钱包已同步'));
        await loadOptions();
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
    [loadOptions, runFullRefresh, t, upstreamConfig.upstream_account_id],
  );

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
        resetAccountDraft();
      } catch (error) {
        showError(error);
      } finally {
        setDeletingAccountId(0);
      }
    },
    [loadOptions, resetAccountDraft, t, upstreamConfig.upstream_account_id],
  );

  const loadDetailPage = useCallback(
    async (page = detailPage, pageSize = detailPageSize) => {
      if (!reportMatchesCurrentFilters || validationErrors.length > 0) {
        setDetailRows([]);
        setDetailTotal(0);
        return;
      }
      setDetailLoading(true);
      try {
        const res = await API.post('/api/profit_board/details', {
          ...queryPayload,
          include_details: true,
          detail_limit: DETAIL_LIMIT,
          page,
          page_size: pageSize,
          view_batch_id: viewBatchId,
          detail_filter: detailFilter || {},
        });
        if (!res.data.success) return showError(res.data.message);
        setDetailRows(res.data.data?.rows || []);
        setDetailTotal(res.data.data?.total || 0);
      } catch (error) {
        showError(error);
      } finally {
        setDetailLoading(false);
      }
    },
    [
      detailFilter,
      detailPage,
      detailPageSize,
      queryPayload,
      reportMatchesCurrentFilters,
      validationErrors.length,
      viewBatchId,
    ],
  );

  useEffect(() => {
    if (!reportMatchesCurrentFilters) {
      setDetailRows([]);
      setDetailTotal(0);
      return;
    }
    loadDetailPage(detailPage, detailPageSize);
  }, [
    detailFilter,
    detailPage,
    detailPageSize,
    loadDetailPage,
    reportMatchesCurrentFilters,
    viewBatchId,
  ]);

  const addOrUpdateBatch = useCallback(() => {
    const nextBatch = {
      id: editingBatchId || normalizeBatchForState({}, batches.length).id,
      name:
        draft.name?.trim() ||
        `组合 ${batches.length + (editingBatchId ? 0 : 1)}`,
      scope_type: draft.scope_type,
      channel_ids:
        draft.scope_type === 'channel' ? draft.channel_ids || [] : [],
      tags: draft.scope_type === 'tag' ? draft.tags || [] : [],
    };
    const selectedCount =
      nextBatch.scope_type === 'channel'
        ? nextBatch.channel_ids.length
        : nextBatch.tags.length;
    if (!selectedCount) return showError(t('请先选择渠道或标签'));
    setBatches((prev) =>
      editingBatchId
        ? prev.map((item) => (item.id === editingBatchId ? nextBatch : item))
        : [...prev, nextBatch],
    );
    setDraft(createDefaultDraft());
    setEditingBatchId('');
    showSuccess(editingBatchId ? t('组合已更新') : t('组合已添加'));
  }, [batches.length, draft, editingBatchId, t]);

  const editBatch = useCallback((batch) => {
    setEditingBatchId(batch.id);
    setDraft({
      id: batch.id,
      name: batch.name,
      scope_type: batch.scope_type,
      channel_ids: batch.channel_ids || [],
      tags: batch.tags || [],
    });
  }, []);
  const resetDraft = useCallback(() => {
    setDraft(createDefaultDraft());
    setEditingBatchId('');
  }, []);
  const removeBatch = useCallback(
    (batchId) => {
      setBatches((prev) => prev.filter((item) => item.id !== batchId));
      setComboConfigs((prev) =>
        prev.filter((item) => item.combo_id !== batchId),
      );
      if (editingBatchId === batchId) resetDraft();
    },
    [editingBatchId, resetDraft],
  );
  const updateComboConfig = useCallback(
    (comboId, updater) =>
      setComboConfigs((prev) =>
        prev.map((item) =>
          item.combo_id === comboId
            ? {
                ...item,
                ...(typeof updater === 'function' ? updater(item) : updater),
              }
            : item,
        ),
      ),
    [],
  );
  const addComboRule = useCallback(
    (comboId, field, initialRule = {}) =>
      updateComboConfig(comboId, (current) => ({
        [field]: [
          ...(current[field] || []),
          createDefaultPricingRule(initialRule),
        ],
      })),
    [updateComboConfig],
  );
  const updateComboRule = useCallback(
    (comboId, field, index, patch) =>
      updateComboConfig(comboId, (current) => ({
        [field]: (current[field] || []).map((item, itemIndex) =>
          itemIndex === index ? { ...item, ...patch } : item,
        ),
      })),
    [updateComboConfig],
  );
  const removeComboRule = useCallback(
    (comboId, field, index) =>
      updateComboConfig(comboId, (current) => ({
        [field]: (current[field] || []).filter(
          (_, itemIndex) => itemIndex !== index,
        ),
      })),
    [updateComboConfig],
  );

  const resolveSharedSitePreview = useCallback(
    (modelName) => {
      const model = localModelMap.get(modelName);
      if (!model) return null;
      if (
        siteConfig.group &&
        (model.enable_groups || []).length > 0 &&
        !(model.enable_groups || []).includes(siteConfig.group)
      )
        return null;
      if (model.quota_type === 1)
        return {
          input_price: clampNumber(model.model_price),
          output_price: 0,
          cache_read_price: 0,
          cache_creation_price: 0,
        };
      const factor = siteConfig.use_recharge_price
        ? clampNumber(model.model_price || 1)
        : 1;
      const baseInput = clampNumber(model.model_ratio) * 2 * factor;
      return {
        input_price: baseInput,
        output_price:
          clampNumber(model.model_ratio) *
          clampNumber(model.completion_ratio) *
          2 *
          factor,
        cache_read_price: model.supports_cache_read
          ? baseInput * clampNumber(model.cache_ratio)
          : 0,
        cache_creation_price: model.supports_cache_creation
          ? baseInput * clampNumber(model.cache_creation_ratio)
          : 0,
      };
    },
    [localModelMap, siteConfig.group, siteConfig.use_recharge_price],
  );
  const batchSummaryOptions = useMemo(
    () => [
      { label: t('全部组合'), value: 'all' },
      ...[
        ...(overviewReport?.batch_summaries || []),
        ...(report?.batch_summaries || []),
      ].map((item) => ({ label: item.batch_name, value: item.batch_id })),
    ],
    [overviewReport?.batch_summaries, report?.batch_summaries, t],
  );
  const businessMetrics = useMemo(
    () => [
      { key: 'configured_site_revenue_usd', label: t('本站配置收入') },
      { key: 'upstream_cost_usd', label: t('上游费用') },
      { key: 'configured_profit_usd', label: t('配置利润') },
    ],
    [t],
  );
  const metricLabel = useMemo(
    () =>
      t(
        metricOptions.find((item) => item.value === metricKey)?.label ||
          metricOptions[0].label,
      ),
    [metricKey, t],
  );
  const chartSubtitle =
    analysisMode === 'business_compare'
      ? t('本站配置收入 / 上游费用 / 配置利润')
      : metricLabel;
  const trendRows = useMemo(
    () =>
      !report
        ? []
        : analysisMode === 'business_compare'
          ? combineTimeseriesMetrics(
              report.timeseries || [],
              viewBatchId,
              businessMetrics,
            )
          : (viewBatchId === 'all'
              ? report.timeseries || []
              : (report.timeseries || []).filter(
                  (item) => item.batch_id === viewBatchId,
                )
            ).map((item) => ({
              bucket: item.bucket,
              value: Number(item[metricKey] || 0),
              batch_id: item.batch_id,
            })),
    [analysisMode, businessMetrics, metricKey, report, viewBatchId],
  );
  const channelRows = useMemo(() => {
    if (!report) return [];
    if (
      analysisMode === 'single_metric' &&
      metricKey === 'remote_observed_cost_usd'
    )
      return [];
    return analysisMode === 'business_compare'
      ? combineBreakdownMetrics(
          report.channel_breakdown || [],
          viewBatchId,
          businessMetrics,
        )
      : aggregateBreakdownRows(
          report.channel_breakdown || [],
          viewBatchId,
          metricKey,
        );
  }, [analysisMode, businessMetrics, metricKey, report, viewBatchId]);
  const modelRows = useMemo(() => {
    if (!report) return [];
    if (
      analysisMode === 'single_metric' &&
      metricKey === 'remote_observed_cost_usd'
    )
      return [];
    return analysisMode === 'business_compare'
      ? combineBreakdownMetrics(
          report.model_breakdown || [],
          viewBatchId,
          businessMetrics,
        )
      : aggregateBreakdownRows(
          report.model_breakdown || [],
          viewBatchId,
          metricKey,
        );
  }, [analysisMode, businessMetrics, metricKey, report, viewBatchId]);
  const trendSpec = useMemo(
    () => createTrendSpec(trendRows, chartSubtitle, statusState?.status, t),
    [chartSubtitle, statusState?.status, t, trendRows],
  );
  const channelSpec = useMemo(
    () =>
      createBarSpec(
        t('渠道分布'),
        channelRows,
        chartSubtitle,
        statusState?.status,
        t,
      ),
    [channelRows, chartSubtitle, statusState?.status, t],
  );
  const modelSpec = useMemo(
    () =>
      createBarSpec(
        t('模型分布'),
        modelRows,
        chartSubtitle,
        statusState?.status,
        t,
      ),
    [chartSubtitle, modelRows, statusState?.status, t],
  );
  const handleChartClick = useCallback(
    (type) => (event) => {
      const label = event?.datum?.label || event?.datum?.bucket;
      if (!label) return;
      setDetailPage(1);
      setDetailFilter({
        type,
        value: label,
        batchId: event?.datum?.batch_id || null,
      });
    },
    [],
  );
  const chartContent = useMemo(
    () => ({
      trend: trendRows.length ? (
        <VChart
          key={`trend-${actualTheme}-${analysisMode}-${viewBatchId}-${metricKey}`}
          spec={trendSpec}
          onClick={handleChartClick('trend')}
        />
      ) : (
        <Empty description={t('当前没有趋势数据')} />
      ),
      channel: channelRows.length ? (
        <VChart
          key={`channel-${actualTheme}-${analysisMode}-${viewBatchId}-${metricKey}`}
          spec={channelSpec}
          onClick={handleChartClick('channel')}
        />
      ) : (
        <Empty description={t('当前没有渠道数据')} />
      ),
      model: modelRows.length ? (
        <VChart
          key={`model-${actualTheme}-${analysisMode}-${viewBatchId}-${metricKey}`}
          spec={modelSpec}
          onClick={handleChartClick('model')}
        />
      ) : (
        <Empty description={t('当前没有模型数据')} />
      ),
    }),
    [
      actualTheme,
      analysisMode,
      channelRows.length,
      channelSpec,
      handleChartClick,
      metricKey,
      modelRows.length,
      modelSpec,
      t,
      trendRows.length,
      trendSpec,
      viewBatchId,
    ],
  );

  const detailColumns = useMemo(
    () => [
      {
        title: t('时间'),
        dataIndex: 'created_at',
        render: (value) => timestamp2string(value),
        width: 160,
      },
      { title: t('组合'), dataIndex: 'batch_name', width: 120 },
      {
        title: t('渠道'),
        dataIndex: 'channel_name',
        render: (value, row) => value || `#${row.channel_id}`,
        width: 140,
      },
      { title: t('模型'), dataIndex: 'model_name', width: 160 },
      {
        title: t('本站配置收入'),
        dataIndex: 'configured_site_revenue_usd',
        render: (value) => (
          <span className='font-medium text-emerald-600 dark:text-emerald-400'>
            {formatMoney(value, statusState?.status)}
          </span>
        ),
        width: 130,
      },
      {
        title: t('配置利润'),
        dataIndex: 'configured_profit_usd',
        render: (value, row) =>
          row.upstream_cost_known && row.site_pricing_known ? (
            <span className='font-medium text-sky-600 dark:text-sky-400'>
              {formatMoney(value, statusState?.status)}
            </span>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
        width: 110,
      },
      {
        title: t('上游费用'),
        dataIndex: 'upstream_cost_usd',
        render: (value, row) =>
          row.upstream_cost_known ? (
            <span className='font-medium text-amber-600 dark:text-amber-400'>
              {formatMoney(value, statusState?.status)}
            </span>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
        width: 110,
      },
      {
        title: t('本站实际收入'),
        dataIndex: 'actual_site_revenue_usd',
        render: (value) => (
          <span className='font-medium'>
            {formatMoney(value, statusState?.status)}
          </span>
        ),
        width: 130,
      },
      {
        title: t('实际利润'),
        dataIndex: 'actual_profit_usd',
        render: (value, row) =>
          row.upstream_cost_known ? (
            <span className='font-medium text-violet-600 dark:text-violet-400'>
              {formatMoney(value, statusState?.status)}
            </span>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
        width: 110,
      },
      {
        title: t('配置与实际差值'),
        dataIndex: 'configured_actual_delta_usd',
        render: (value) => (
          <span className='font-medium'>
            {formatMoney(value, statusState?.status)}
          </span>
        ),
        width: 140,
      },
      {
        title: t('本站配置来源'),
        dataIndex: 'site_pricing_source',
        render: (value, row) => (
          <Tag color={row.site_pricing_known ? 'blue' : 'grey'} size='small'>
            {sitePricingSourceLabelMap[value] || value || t('未知')}
          </Tag>
        ),
        width: 130,
      },
    ],
    [statusState?.status, t],
  );

  const summaryMetricHelp = useMemo(
    () => ({
      request_count: t('当前口径内命中的消费日志数量。'),
      remote_observed_cost_usd: t(
        '来自远端 new-api 实例的钱包已用额度 + 订阅已用额度增量，金额按本站额度口径换算。',
      ),
    }),
    [t],
  );
  const cumulativeSummaryCards = useMemo(
    () =>
      !overviewReport?.summary
        ? []
        : [
            {
              key: 'configured_site_revenue_usd',
              title: t('本站配置收入'),
              value: formatMoney(
                overviewReport.summary.configured_site_revenue_usd,
                statusState?.status,
              ),
              icon: <CircleDollarSign size={18} className='text-emerald-600 dark:text-emerald-400' />,
            },
            {
              key: 'upstream_cost_usd',
              title: t('上游费用'),
              value: formatMoney(
                overviewReport.summary.upstream_cost_usd,
                statusState?.status,
              ),
              icon: <BadgeDollarSign size={18} className='text-amber-600 dark:text-amber-400' />,
            },
            {
              key: 'remote_observed_cost_usd',
              title: t('远端观测消耗'),
              value: formatMoney(
                overviewReport.summary.remote_observed_cost_usd,
                statusState?.status,
              ),
              icon: <BadgeDollarSign size={18} className='text-rose-600 dark:text-rose-400' />,
            },
            {
              key: 'configured_profit_usd',
              title: t('配置利润'),
              value: formatMoney(
                overviewReport.summary.configured_profit_usd,
                statusState?.status,
              ),
              icon: <BarChart3 size={18} className='text-sky-600 dark:text-sky-400' />,
            },
            {
              key: 'actual_profit_usd',
              title: t('实际利润'),
              value: formatMoney(
                overviewReport.summary.actual_profit_usd,
                statusState?.status,
              ),
              icon: <BarChart3 size={18} className='text-violet-600 dark:text-violet-400' />,
            },
          ],
    [overviewReport?.summary, statusState?.status, t],
  );
  const diagnosticSummaryCards = useMemo(
    () =>
      !overviewReport?.summary
        ? []
        : [
            {
              key: 'request_count',
              title: t('请求数'),
              value: overviewReport.summary.request_count,
            },
            {
              key: 'configured_profit_coverage_rate',
              title: t('配置利润覆盖率'),
              value: formatRatio(
                overviewReport.summary.configured_profit_coverage_rate,
              ),
            },
            {
              key: 'returned_cost_count',
              title: t('上游返回费用'),
              value: overviewReport.summary.returned_cost_count,
            },
            {
              key: 'manual_cost_count',
              title: t('手动上游价格'),
              value: overviewReport.summary.manual_cost_count,
            },
            {
              key: 'missing_site_pricing_count',
              title: t('缺失本站价格'),
              value: overviewReport.summary.missing_site_pricing_count,
            },
          ],
    [overviewReport?.summary, t],
  );
  const statusSummary = useMemo(() => {
    const items = [];
    if (reportMatchesCurrentFilters)
      items.push({ key: 'fresh', color: 'blue', text: t('时间分析已同步') });
    else if (report)
      items.push({
        key: 'stale',
        color: 'grey',
        text: t('筛选已变化，等待刷新'),
      });
    if (overviewReport)
      items.push({
        key: 'overview',
        color: 'green',
        text: t('累计总览已更新'),
      });
    if (activityChecking)
      items.push({ key: 'watch', color: 'cyan', text: t('低频检查中') });
    return items;
  }, [
    activityChecking,
    overviewReport,
    report,
    reportMatchesCurrentFilters,
    t,
  ]);
  const combinedWarnings = useMemo(
    () =>
      Array.from(
        new Set([
          ...(overviewReport?.warnings || []),
          ...(report?.warnings || []),
          ...validationErrors,
        ]),
      ),
    [overviewReport?.warnings, report?.warnings, validationErrors],
  );
  const warningSummary = combinedWarnings.length
    ? t('{{count}} 个需要关注的问题', { count: combinedWarnings.length })
    : t('当前没有需要处理的收益口径提示');
  const detailFilterText = useMemo(
    () =>
      !detailFilter?.value
        ? ''
        : `${{ trend: t('时间桶'), channel: t('渠道'), model: t('模型') }[detailFilter.type] || t('筛选')}：${detailFilter.value}`,
    [detailFilter, t],
  );
  const batchDigest = useCallback(
    (batch) =>
      batch.scope_type === 'channel'
        ? (batch.channel_ids || [])
            .map((id) => channelMap.get(String(id))?.name)
            .filter(Boolean)
            .slice(0, 3)
            .join('、')
        : (batch.tags || []).slice(0, 3).join('、'),
    [channelMap],
  );
  const generatedAtText = report?.meta?.generated_at
    ? timestamp2string(report.meta.generated_at)
    : t('尚未生成');
  const sharedSiteModelCount = comboConfigs.some(
    (item) => item.site_mode === 'shared_site_model',
  )
    ? siteConfig.model_names?.length || 0
    : 0;
  const sitePriceFactorNote =
    overviewReport?.meta?.site_price_factor_note ||
    report?.meta?.site_price_factor_note ||
    '';

  return (
    <Spin spinning={loading}>
      <div className='mt-[60px] space-y-4 px-2 pb-6'>
        <ProfitBoardHeader
          querying={querying}
          overviewQuerying={overviewQuerying}
          runFullRefresh={runFullRefresh}
          saving={saving}
          saveConfig={saveConfig}
          autoRefreshMode={autoRefreshMode}
          setAutoRefreshMode={setAutoRefreshMode}
          statusSummary={statusSummary}
          hasNewActivity={hasNewActivity}
          generatedAtText={generatedAtText}
          sharedSiteModelCount={sharedSiteModelCount}
          warningSummary={warningSummary}
          combinedWarnings={combinedWarnings}
          sitePriceFactorNote={sitePriceFactorNote}
          walletModeEnabled={walletModeEnabled}
          selectedAccount={selectedAccount}
          t={t}
        />
        <Tabs type='line' size='large' className='profit-board-tabs'>
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1.5'>
                <BarChart3 size={16} />
                {t('收益分析')}
              </span>
            }
            itemKey='analysis'
          >
            <div className='mt-4 space-y-4'>
              <OverviewPanel
                overviewQuerying={overviewQuerying}
                overviewReport={overviewReport}
                report={report}
                reportMatchesCurrentFilters={reportMatchesCurrentFilters}
                cumulativeSummaryCards={cumulativeSummaryCards}
                diagnosticSummaryCards={diagnosticSummaryCards}
                summaryMetricHelp={summaryMetricHelp}
                formatMoney={formatMoney}
                status={statusState?.status}
                t={t}
              />
              <TimeRangePanel
                datePresets={createPresetRanges()}
                dateRange={dateRange}
                setDateRange={setDateRange}
                currentRangeText={formatRangeLabel(dateRange)}
                currentRangeDuration={formatRangeDuration(dateRange)}
                validationErrors={validationErrors}
                statusSummary={statusSummary}
                report={report}
                t={t}
              />
              <ChartAnalysisCard
                analysisMode={analysisMode}
                setAnalysisMode={setAnalysisMode}
                metricKey={metricKey}
                setMetricKey={setMetricKey}
                metricOptions={metricOptions}
                viewBatchId={viewBatchId}
                setViewBatchId={setViewBatchId}
                batchSummaryOptions={batchSummaryOptions}
                granularity={granularity}
                setGranularity={setGranularity}
                customIntervalMinutes={customIntervalMinutes}
                setCustomIntervalMinutes={setCustomIntervalMinutes}
                detailFilter={detailFilter}
                clearDetailFilter={() => {
                  setDetailFilter(null);
                  setDetailPage(1);
                }}
                runQuery={runQuery}
                querying={querying}
                chartTab={chartTab}
                setChartTab={setChartTab}
                report={report}
                chartContent={chartContent}
                t={t}
              />
              <DetailTableCard
                detailFilterText={detailFilterText}
                detailRows={detailRows}
                detailTotal={detailTotal}
                detailPage={detailPage}
                detailPageSize={detailPageSize}
                setDetailPage={setDetailPage}
                setDetailPageSize={setDetailPageSize}
                detailColumns={detailColumns}
                detailLoading={detailLoading}
                report={report}
                isMobile={isMobile}
                formatMoney={formatMoney}
                status={statusState?.status}
                sitePricingSourceLabelMap={sitePricingSourceLabelMap}
                t={t}
              />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1.5'>
                <CircleDollarSign size={16} />
                {t('配置管理')}
              </span>
            }
            itemKey='config'
          >
            <div className='mt-4 space-y-4'>
              <ComboManagerCard
                draft={draft}
                setDraft={setDraft}
                channelOptions={channelOptions}
                options={options}
                isMobile={isMobile}
                addOrUpdateBatch={addOrUpdateBatch}
                editingBatchId={editingBatchId}
                resetDraft={resetDraft}
                batches={batches}
                batchDigest={batchDigest}
                editBatch={editBatch}
                removeBatch={removeBatch}
                batchValidationError={duplicateBatchError}
                t={t}
              />
              <PricingRulesCard
                batches={batches}
                comboConfigs={comboConfigs}
                siteConfig={siteConfig}
                setSiteConfig={setSiteConfig}
                modelNameOptions={modelNameOptions}
                options={options}
                resolveSharedSitePreview={resolveSharedSitePreview}
                upstreamConfig={upstreamConfig}
                setUpstreamConfig={setUpstreamConfig}
                isMobile={isMobile}
                createDefaultComboPricingConfig={
                  createDefaultComboPricingConfig
                }
                updateComboConfig={updateComboConfig}
                updateComboRule={updateComboRule}
                removeComboRule={removeComboRule}
                addComboRule={addComboRule}
                localModelMap={localModelMap}
                clampNumber={clampNumber}
                t={t}
              />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1.5'>
                <BadgeDollarSign size={16} />
                {t('上游账户')}
              </span>
            }
            itemKey='wallet'
          >
            <div className='mt-4 space-y-4'>
              <UpstreamWalletCard
                accounts={options.upstream_accounts || []}
                accountDraft={accountDraft}
                setAccountDraft={setAccountDraft}
                editingAccountId={editingAccountId}
                setEditingAccountId={setEditingAccountId}
                saveAccount={saveAccount}
                syncAccount={syncAccount}
                deleteAccount={deleteAccount}
                resetAccountDraft={resetAccountDraft}
                savingAccount={savingAccount}
                syncingAccountId={syncingAccountId}
                deletingAccountId={deletingAccountId}
                formatMoney={formatMoney}
                status={statusState?.status}
                t={t}
              />
            </div>
          </Tabs.TabPane>
        </Tabs>
      </div>
    </Spin>
  );
};

export default ProfitBoardPage;
