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
import { Empty, Modal, Tabs } from '@douyinfe/semi-ui';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import { BadgeDollarSign, BarChart3, CircleDollarSign } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { CHART_CONFIG } from '../../constants/dashboard.constants';
import { StatusContext } from '../../context/Status';
import { showError, showSuccess, timestamp2string } from '../../helpers';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import ChartAnalysisCard from './components/ChartAnalysisCard';
import ComboManagerCard from './components/ComboManagerCard';
import OverviewPanel from './components/OverviewPanel';
import PricingConfigModal from './components/PricingConfigModal';
import ProfitBoardHeader from './components/ProfitBoardHeader';
import ResponsiveVChart from './components/ResponsiveVChart';
import UpstreamWalletCard from './components/UpstreamWalletCard';
import { useProfitBoardBatches } from './hooks/useProfitBoardBatches';
import { useProfitBoardConfig } from './hooks/useProfitBoardConfig';
import { useProfitBoardPersist } from './hooks/useProfitBoardPersist';
import { useProfitBoardQuery } from './hooks/useProfitBoardQuery';
import { useUpstreamAccounts } from './hooks/useUpstreamAccounts';
import {
  aggregateBreakdownRows,
  aggregateChannelRowsByTag,
  clampNumber,
  combineBreakdownMetrics,
  combineChannelMetricsByTag,
  combineTimeseriesMetrics,
  createBarSpec,
  createBatchId,
  createBatchCreatedAt,
  createDefaultComboPricingConfig,
  createDefaultPricingRule,
  createMetricOptions,
  createPresetRanges,
  createSuggestedComboName,
  createTrendSpec,
  formatMoney,
  getUpstreamCostSourceLabel,
  isLikelyAutoComboName,
  mergeComboDraftWithTemplate,
  pickDominantComboModes,
  pickRecommendedUpstreamAccountId,
} from './utils';

initVChartSemiTheme({
  isWatchingThemeSwitch: true,
});

const buildBatchOverlapError = (batches, channelMap, tagChannelMap) => {
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
        return `${channelName} 同时出现在组合"${owner}"和"${batch.name}"中，请拆开后再统计`;
      }
      ownerMap.set(channelId, batch.name);
    }
  }
  return '';
};

const cloneComboDraft = (batch, comboConfig) => ({
  id: batch.id,
  name: batch.name || '',
  scope_type: batch.scope_type || 'channel',
  channel_ids: [...(batch.channel_ids || [])],
  tags: [...(batch.tags || [])],
  created_at: Number(batch.created_at || createBatchCreatedAt()),
  combo_id: comboConfig.combo_id,
  site_mode: comboConfig.site_mode,
  upstream_mode: comboConfig.upstream_mode,
  cost_source: comboConfig.cost_source || 'manual_only',
  upstream_account_id: Number(comboConfig.upstream_account_id || 0),
  shared_site: { ...(comboConfig.shared_site || {}) },
  site_rules: (comboConfig.site_rules || []).map((rule) =>
    createDefaultPricingRule(rule),
  ),
  upstream_rules: (comboConfig.upstream_rules || []).map((rule) =>
    createDefaultPricingRule(rule),
  ),
  site_fixed_total_amount: clampNumber(comboConfig.site_fixed_total_amount),
  upstream_fixed_total_amount: clampNumber(
    comboConfig.upstream_fixed_total_amount,
  ),
  remote_observer: { ...(comboConfig.remote_observer || {}) },
});

const getSiteSummaryText = (comboConfig, t) => {
  if (comboConfig.site_mode === 'log_quota') return t('智能（按日志额度）');
  if (comboConfig.site_mode !== 'shared_site_model') return t('手动定价');
  const modelCount = comboConfig.shared_site?.model_names?.length || 0;
  if (modelCount === 0) return t('本站模型价格');
  return t('本站模型价格 · {{count}} 个模型', { count: modelCount });
};

const getUpstreamSummaryText = (comboConfig, options, t) => {
  if (comboConfig.upstream_mode !== 'wallet_observer') {
    return getUpstreamCostSourceLabel(
      comboConfig.cost_source || 'manual_only',
      t,
    );
  }
  const account = (options?.upstream_accounts || []).find(
    (item) => item.id === Number(comboConfig.upstream_account_id || 0),
  );
  return account
    ? t('按钱包余额变化 · {{name}}', { name: account.name })
    : t('按钱包余额变化');
};

const ProfitBoardPage = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const isMobile = useIsMobile();

  const rechargePriceFactor = useMemo(() => {
    const price = statusState?.status?.price ?? 1;
    const usdRate = statusState?.status?.usd_exchange_rate;
    if (!usdRate || usdRate <= 0) return 1;
    return price / usdRate;
  }, [statusState?.status?.price, statusState?.status?.usd_exchange_rate]);

  const { restoredState, cachedBundle, persistState, persistReportCache } =
    useProfitBoardPersist();

  const batchesHook = useProfitBoardBatches({ restoredState });
  const [comboConfigs, setComboConfigs] = useState(
    restoredState.comboConfigs || [],
  );
  const [editorDraft, setEditorDraft] = useState(null);
  const [editorNameAuto, setEditorNameAuto] = useState(false);
  const [editingBatchId, setEditingBatchId] = useState('');
  const [editorVisible, setEditorVisible] = useState(false);
  const [editorValidationError, setEditorValidationError] = useState('');
  const [hasUnsavedConfigChanges, setHasUnsavedConfigChanges] = useState(
    !!restoredState.hasUnsavedConfigChanges,
  );
  const [activeTab, setActiveTab] = useState('wallet');
  const [builderOptionsReady, setBuilderOptionsReady] = useState(false);
  const [accountsReady, setAccountsReady] = useState(false);
  const [configReady, setConfigReady] = useState(false);
  const serverRestoredRef = useRef(false);
  const pendingAutoSaveRef = useRef(false);

  const configHook = useProfitBoardConfig({
    batchPayload: batchesHook.batchPayload,
    comboConfigs,
    setComboConfigs,
    restoredState,
    rechargePriceFactor,
  });

  const { batches, setBatches, batchPayload, upsertBatch, removeBatch } = batchesHook;

  const {
    builderLoading,
    accountsLoading,
    saving,
    options,
    siteConfig,
    upstreamConfig,
    setUpstreamConfig,
    channelOptions,
    channelMap,
    channelModelMap,
    tagChannelMap,
    localModelMap,
    modelNameOptions,
    configPayload,
    loadBuilderOptions,
    loadUpstreamAccounts,
    loadCurrentConfig,
    applyLoadedConfig,
    saveConfig,
    resolveSharedSitePreview,
    getModelsByChannelIds,
    getModelsByTags,
  } = configHook;

  const tagOptions = useMemo(
    () =>
      (options.tags || []).map((item) => ({
        label: item,
        value: item,
      })),
    [options.tags],
  );

  const availableAccountIds = useMemo(
    () =>
      new Set(
        (options.upstream_accounts || [])
          .filter((item) => item.enabled !== false)
          .map((item) => Number(item.id)),
      ),
    [options.upstream_accounts],
  );

  const availableAccounts = useMemo(
    () =>
      (options.upstream_accounts || []).filter(
        (item) => item.enabled !== false,
      ),
    [options.upstream_accounts],
  );

  const resolveComboConfig = useCallback(
    (batchId) =>
      comboConfigs.find((item) => item.combo_id === batchId) ||
      createDefaultComboPricingConfig(
        batchId,
        siteConfig,
        siteConfig,
        upstreamConfig,
      ),
    [comboConfigs, siteConfig, upstreamConfig],
  );

  useEffect(() => {
    setComboConfigs((prev) =>
      batches.map((batch) => {
        const existing = (prev || []).find(
          (item) => item.combo_id === batch.id,
        );
        const fallback = createDefaultComboPricingConfig(
          batch.id,
          siteConfig,
          siteConfig,
          upstreamConfig,
        );
        return {
          ...fallback,
          ...(existing || {}),
          combo_id: batch.id,
          shared_site: {
            ...fallback.shared_site,
            ...(existing?.shared_site || {}),
          },
          remote_observer: {
            ...fallback.remote_observer,
            ...(existing?.remote_observer || {}),
          },
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

  const duplicateBatchError = useMemo(
    () => buildBatchOverlapError(batches, channelMap, tagChannelMap),
    [batches, channelMap, tagChannelMap],
  );

  const validationErrors = useMemo(() => {
    const errors = [];
    if (configReady && !batches.length) errors.push(t('请至少添加一个组合'));
    if (duplicateBatchError) errors.push(duplicateBatchError);
    if (
      comboConfigs.some(
        (item) =>
          item.site_mode === 'shared_site_model' &&
          !(item.shared_site?.model_names || []).length,
      )
    ) {
      errors.push(t('启用了本站模型价格的组合必须至少选择一个模型'));
    }
    if (
      comboConfigs.some((item) => {
        if (item.upstream_mode !== 'wallet_observer') return false;
        const accountId = Number(item.upstream_account_id || 0);
        return accountId <= 0 || !availableAccountIds.has(accountId);
      })
    ) {
      errors.push(t('钱包扣减模式必须绑定一个上游账户'));
    }
    return Array.from(new Set(errors));
  }, [
    availableAccountIds,
    batches.length,
    comboConfigs,
    configReady,
    duplicateBatchError,
    t,
  ]);

  const queryReady = accountsReady && configReady && batchPayload.length > 0;

  const queryHook = useProfitBoardQuery({
    restoredState,
    cachedBundle,
    configPayload,
    batchPayload,
    validationErrors,
    persistReportCache,
    queryReady,
  });

  const {
    querying,
    overviewQuerying,
    dateRange,
    setDateRange,
    granularity,
    setGranularity,
    customIntervalMinutes,
    setCustomIntervalMinutes,
    chartTab,
    setChartTab,
    channelGroupMode,
    setChannelGroupMode,
    metricKey,
    setMetricKey,
    analysisMode,
    setAnalysisMode,
    viewBatchId,
    setViewBatchId,
    overviewReport,
    report,
    reportMatchesCurrentFilters,
    autoRefreshMode,
    setAutoRefreshMode,
    hasNewActivity,
    activityChecking,
    autoRefreshing,
    lastQueryKey,
    runFullRefresh,
  } = queryHook;

  const accountsHook = useUpstreamAccounts({
    options,
    loadUpstreamAccounts,
    comboConfigs,
    setComboConfigs,
    upstreamConfig,
    setUpstreamConfig,
    setHasUnsavedConfigChanges,
    runFullRefresh,
  });

  const buildDraftValidationError = useCallback(
    (draft) => {
      if (!draft) return '';
      const selectedCount =
        draft.scope_type === 'channel'
          ? (draft.channel_ids || []).length
          : (draft.tags || []).length;
      if (!selectedCount) return t('请先选择渠道或标签');
      if (
        draft.site_mode === 'shared_site_model' &&
        !(draft.shared_site?.model_names || []).length
      ) {
        return t('启用了本站模型价格的组合必须至少选择一个模型');
      }
      if (draft.upstream_mode === 'wallet_observer') {
        const accountId = Number(draft.upstream_account_id || 0);
        if (accountId <= 0 || !availableAccountIds.has(accountId)) {
          return t('钱包扣减模式必须绑定一个上游账户');
        }
      }
      const nextBatch = {
        id: draft.id,
        name: draft.name?.trim() || t('未命名组合'),
        scope_type: draft.scope_type,
        channel_ids:
          draft.scope_type === 'channel' ? draft.channel_ids || [] : [],
        tags: draft.scope_type === 'tag' ? draft.tags || [] : [],
      };
      return buildBatchOverlapError(
        [...batches.filter((item) => item.id !== draft.id), nextBatch],
        channelMap,
        tagChannelMap,
      );
    },
    [availableAccountIds, batches, channelMap, t, tagChannelMap],
  );

  useEffect(() => {
    if (!editorDraft) {
      setEditorValidationError('');
      return;
    }
    setEditorValidationError(buildDraftValidationError(editorDraft));
  }, [buildDraftValidationError, editorDraft]);

  useEffect(() => {
    let cancelled = false;
    loadUpstreamAccounts()
      .then(() => {
        if (!cancelled) setAccountsReady(true);
      })
      .catch((error) => {
        if (cancelled) return;
        setAccountsReady(false);
        showError(error);
      });
    if (!serverRestoredRef.current) {
      serverRestoredRef.current = true;
      (async () => {
        try {
          const serverConfig = await loadCurrentConfig();
          if (cancelled) return;
          if (serverConfig) {
            const serverBatches = (serverConfig.batches || []).map((batch) => ({
              id: batch.id || '',
              name: batch.name || '',
              scope_type: batch.scope_type || 'channel',
              channel_ids: (batch.channel_ids || []).map(Number).filter(Boolean),
              tags: batch.tags || [],
              created_at: Number(batch.created_at || 0),
            }));
            if (serverBatches.length > 0) {
              setBatches(serverBatches);
            }
            applyLoadedConfig(serverConfig);
            setHasUnsavedConfigChanges(false);
          }
        } catch (error) {
          if (!cancelled) showError(error);
        } finally {
          if (!cancelled) setConfigReady(true);
        }
      })();
    }
    return () => {
      cancelled = true;
    };
  }, [applyLoadedConfig, loadCurrentConfig, loadUpstreamAccounts, setBatches]);

  useEffect(() => {
    if (builderOptionsReady) return;
    if (activeTab !== 'config' && !editorVisible && channelGroupMode !== 'tag') {
      return;
    }
    let cancelled = false;
    loadBuilderOptions()
      .then(() => {
        if (!cancelled) setBuilderOptionsReady(true);
      })
      .catch((error) => {
        if (cancelled) return;
        showError(error);
      });
    return () => {
      cancelled = true;
    };
  }, [
    activeTab,
    builderOptionsReady,
    channelGroupMode,
    editorVisible,
    loadBuilderOptions,
  ]);

  useEffect(() => {
    if (viewBatchId === 'all') return;
    if (batches.some((batch) => batch.id === viewBatchId)) return;
    setViewBatchId('all');
  }, [batches, setViewBatchId, viewBatchId]);

  useEffect(() => {
    persistState({
      batches,
      dateRange,
      granularity,
      customIntervalMinutes,
      chartTab,
      channelGroupMode,
      metricKey,
      analysisMode,
      viewBatchId,
      comboConfigs,
      upstreamConfig,
      siteConfig,
      lastQueryKey,
      autoRefreshMode,
      hasUnsavedConfigChanges,
    });
  }, [
    analysisMode,
    autoRefreshMode,
    batches,
    chartTab,
    channelGroupMode,
    comboConfigs,
    customIntervalMinutes,
    dateRange,
    granularity,
    hasUnsavedConfigChanges,
    metricKey,
    persistState,
    lastQueryKey,
    siteConfig,
    upstreamConfig,
    viewBatchId,
  ]);

  useEffect(() => {
    if (!pendingAutoSaveRef.current || !configReady) return;
    pendingAutoSaveRef.current = false;
    (async () => {
      const saved = await saveConfig([]);
      if (saved) setHasUnsavedConfigChanges(false);
    })();
  }, [configPayload, configReady, saveConfig]);

  const metricOpts = useMemo(() => createMetricOptions(t), [t]);
  const batchSummaryOptions = useMemo(
    () => [
      { label: t('全部组合'), value: 'all' },
      ...batches.map((batch) => ({
        label: batch.name || t('未命名组合'),
        value: batch.id,
      })),
    ],
    [batches, t],
  );
  const businessMetrics = useMemo(
    () => [
      { key: 'configured_site_revenue_usd', label: t('本站配置收入') },
      { key: 'upstream_cost_usd', label: t('上游费用') },
      { key: 'configured_profit_usd', label: t('利润') },
    ],
    [t],
  );
  const metricLabel = useMemo(
    () =>
      metricOpts.find((item) => item.value === metricKey)?.label ||
      metricOpts[0].label,
    [metricKey, metricOpts],
  );
  const chartSubtitle =
    analysisMode === 'business_compare'
      ? t('本站配置收入 / 上游费用 / 利润')
      : metricLabel;

  const trendRows = useMemo(() => {
    if (!report) return [];
    if (analysisMode === 'business_compare') {
      return combineTimeseriesMetrics(
        report.timeseries || [],
        viewBatchId,
        businessMetrics,
      );
    }
    const filtered =
      viewBatchId === 'all'
        ? report.timeseries || []
        : (report.timeseries || []).filter(
            (row) => row.batch_id === viewBatchId,
          );
    return filtered.map((row) => ({
      bucket: row.bucket,
      value: Number(row[metricKey] || 0),
      batch_id: row.batch_id,
    }));
  }, [analysisMode, businessMetrics, metricKey, report, viewBatchId]);

  const channelTagMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((channel) => {
      map.set(String(channel.id), channel.tag || t('未设置标签'));
    });
    return map;
  }, [options.channels, t]);

  const channelRows = useMemo(() => {
    if (
      !report ||
      (analysisMode === 'single_metric' &&
        metricKey === 'remote_observed_cost_usd')
    ) {
      return [];
    }
    if (analysisMode === 'business_compare') {
      return channelGroupMode === 'tag'
        ? combineChannelMetricsByTag(
            report.channel_breakdown || [],
            viewBatchId,
            businessMetrics,
            channelTagMap,
            t('未设置标签'),
          )
        : combineBreakdownMetrics(
            report.channel_breakdown || [],
            viewBatchId,
            businessMetrics,
          );
    }
    return channelGroupMode === 'tag'
      ? aggregateChannelRowsByTag(
          report.channel_breakdown || [],
          viewBatchId,
          metricKey,
          channelTagMap,
          t('未设置标签'),
        )
      : aggregateBreakdownRows(
          report.channel_breakdown || [],
          viewBatchId,
          metricKey,
        );
  }, [
    analysisMode,
    businessMetrics,
    channelGroupMode,
    channelTagMap,
    metricKey,
    report,
    t,
    viewBatchId,
  ]);

  const modelRows = useMemo(() => {
    if (
      !report ||
      (analysisMode === 'single_metric' &&
        metricKey === 'remote_observed_cost_usd')
    ) {
      return [];
    }
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

  const tagAggregationMeta = useMemo(() => {
    if (chartTab !== 'channel' || channelGroupMode !== 'tag') {
      return { emptyReason: '', bucketCount: 0 };
    }
    if (!channelRows.length) {
      const visibleRows =
        viewBatchId === 'all'
          ? report?.channel_breakdown || []
          : (report?.channel_breakdown || []).filter(
              (row) => row.batch_id === viewBatchId,
            );
      return {
        emptyReason: visibleRows.length
          ? t('当前渠道没有可用标签，无法按标签聚合')
          : '',
        bucketCount: 0,
      };
    }
    return {
      emptyReason:
        channelRows.length <= 1 ? t('当前数据按标签聚合后只有一个分组') : '',
      bucketCount: channelRows.length,
    };
  }, [
    channelGroupMode,
    channelRows,
    chartTab,
    report?.channel_breakdown,
    t,
    viewBatchId,
  ]);

  const trendSpec = useMemo(
    () => createTrendSpec(trendRows, chartSubtitle, statusState?.status, t),
    [chartSubtitle, statusState?.status, t, trendRows],
  );
  const channelSpec = useMemo(
    () =>
      createBarSpec(
        channelGroupMode === 'tag' ? t('标签分布') : t('渠道分布'),
        channelRows,
        chartSubtitle,
        statusState?.status,
        t,
      ),
    [channelGroupMode, channelRows, chartSubtitle, statusState?.status, t],
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

  const chartHeight = isMobile ? 340 : 480;
  const renderChart = useCallback(
    (chartKey, spec) => (
      <ResponsiveVChart
        key={chartKey}
        chartKey={chartKey}
        spec={{ ...spec, height: chartHeight }}
        option={CHART_CONFIG}
        minHeight={chartHeight}
      />
    ),
    [chartHeight],
  );

  const chartContent = useMemo(
    () => ({
      trend: trendRows.length ? (
        renderChart(
          `trend-${analysisMode}-${viewBatchId}-${metricKey}-${granularity}-${customIntervalMinutes}-${lastQueryKey}-${trendRows.length}`,
          trendSpec,
        )
      ) : (
        <Empty description={t('当前没有趋势数据')} />
      ),
      channel: channelRows.length ? (
        renderChart(
          `channel-${analysisMode}-${viewBatchId}-${metricKey}-${channelGroupMode}-${granularity}-${customIntervalMinutes}-${lastQueryKey}-${channelRows.length}`,
          channelSpec,
        )
      ) : (
        <Empty
          description={tagAggregationMeta.emptyReason || t('当前没有渠道数据')}
        />
      ),
      model: modelRows.length ? (
        renderChart(
          `model-${analysisMode}-${viewBatchId}-${metricKey}-${granularity}-${customIntervalMinutes}-${lastQueryKey}-${modelRows.length}`,
          modelSpec,
        )
      ) : (
        <Empty description={t('当前没有模型数据')} />
      ),
    }),
    [
      analysisMode,
      channelGroupMode,
      channelRows.length,
      channelSpec,
      customIntervalMinutes,
      granularity,
      lastQueryKey,
      modelRows.length,
      metricKey,
      modelSpec,
      renderChart,
      tagAggregationMeta.emptyReason,
      t,
      trendRows.length,
      trendSpec,
      viewBatchId,
    ],
  );

  const overviewSummaryCards = useMemo(
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
              icon: (
                <CircleDollarSign
                  size={18}
                  className='text-emerald-600 dark:text-emerald-400'
                />
              ),
              requestCount: overviewReport.summary.request_count,
            },
            {
              key: 'upstream_cost_usd',
              title: t('上游费用'),
              value: formatMoney(
                overviewReport.summary.upstream_cost_usd,
                statusState?.status,
              ),
              icon: (
                <BadgeDollarSign
                  size={18}
                  className='text-amber-600 dark:text-amber-400'
                />
              ),
            },
            {
              key: 'configured_profit_usd',
              title: t('利润'),
              value: formatMoney(
                overviewReport.summary.configured_profit_usd,
                statusState?.status,
              ),
              icon: (
                <BarChart3
                  size={18}
                  className='text-sky-600 dark:text-sky-400'
                />
              ),
            },
          ],
    [overviewReport?.summary, statusState?.status, t],
  );

  const statusSummary = useMemo(() => {
    const items = [];
    if (autoRefreshing) {
      items.push({
        key: 'refreshing',
        color: 'cyan',
        text: t('收益分析刷新中'),
      });
    }
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
    autoRefreshing,
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

  const batchDigest = useCallback(
    (batch) => {
      const labels =
        batch.scope_type === 'channel'
          ? (batch.channel_ids || [])
              .map((id) => channelMap.get(String(id))?.name)
              .filter(Boolean)
          : batch.tags || [];
      const total = labels.length;
      if (!total) {
        return batch.scope_type === 'channel'
          ? t('未选择渠道')
          : t('未选择标签');
      }
      const preview = labels.slice(0, 3).join('、');
      return total > 3 ? `${preview}，共 ${total} 项` : preview;
    },
    [channelMap, t],
  );

  const getEditorSuggestedName = useCallback(
    (draft) => {
      const fallbackName = editingBatchId
        ? draft?.name?.trim() || t('未命名组合')
        : `组合 ${batches.length + 1}`;
      return createSuggestedComboName(draft, channelMap, t, fallbackName);
    },
    [batches.length, channelMap, editingBatchId, t],
  );

  const setEditorDraftSmart = useCallback(
    (updater) => {
      setEditorDraft((prev) => {
        const nextValue =
          typeof updater === 'function' ? updater(prev) : updater;
        if (!nextValue) return nextValue;

        const nextDraft = { ...nextValue };
        if (nextDraft.upstream_mode === 'wallet_observer') {
          const accountId = Number(nextDraft.upstream_account_id || 0);
          if (
            (!accountId || !availableAccountIds.has(accountId)) &&
            availableAccounts.length === 1
          ) {
            nextDraft.upstream_account_id = Number(availableAccounts[0].id);
          }
        } else {
          nextDraft.upstream_account_id = 0;
        }

        if (editorNameAuto) {
          nextDraft.name = getEditorSuggestedName(nextDraft);
        }

        return nextDraft;
      });
    },
    [
      availableAccountIds,
      availableAccounts,
      editorNameAuto,
      getEditorSuggestedName,
    ],
  );

  const handleEditorNameChange = useCallback((value) => {
    setEditorNameAuto(false);
    setEditorDraft((prev) => (prev ? { ...prev, name: value } : prev));
  }, []);

  const handleRegenerateEditorName = useCallback(() => {
    setEditorNameAuto(true);
    setEditorDraft((prev) =>
      prev ? { ...prev, name: getEditorSuggestedName(prev) } : prev,
    );
  }, [getEditorSuggestedName]);

  const dominantComboModes = useMemo(
    () =>
      pickDominantComboModes(
        comboConfigs,
        siteConfig?.pricing_mode === 'site_model'
          ? 'shared_site_model'
          : 'manual',
        upstreamConfig?.upstream_mode || 'manual_rules',
      ),
    [comboConfigs, siteConfig?.pricing_mode, upstreamConfig?.upstream_mode],
  );

  const recommendedAccountId = useMemo(
    () =>
      pickRecommendedUpstreamAccountId(
        comboConfigs,
        availableAccountIds,
        editingBatchId,
      ),
    [availableAccountIds, comboConfigs, editingBatchId],
  );

  const recommendedAccount = useMemo(
    () =>
      availableAccounts.find(
        (item) => Number(item.id) === Number(recommendedAccountId || 0),
      ) || null,
    [availableAccounts, recommendedAccountId],
  );

  const copyTemplateOptions = useMemo(
    () =>
      batches
        .filter((batch) => batch.id !== editingBatchId)
        .map((batch) => ({
          label: batch.name || t('未命名组合'),
          value: batch.id,
        })),
    [batches, editingBatchId, t],
  );

  const generatedAtText =
    reportMatchesCurrentFilters && report?.meta?.generated_at
      ? timestamp2string(report.meta.generated_at)
      : t('尚未生成');
  const trendBucketCount = useMemo(
    () => new Set((trendRows || []).map((row) => row.bucket)).size,
    [trendRows],
  );
  const sitePriceFactorNote =
    overviewReport?.meta?.site_price_factor_note ||
    report?.meta?.site_price_factor_note ||
    '';

  const closeEditor = useCallback(() => {
    setEditorVisible(false);
    setEditingBatchId('');
    setEditorDraft(null);
    setEditorNameAuto(false);
    setEditorValidationError('');
  }, []);

  const openCreateBatchModal = useCallback(async () => {
    try {
      if (!builderOptionsReady) {
        await loadBuilderOptions();
        setBuilderOptionsReady(true);
      }
      const batchId = createBatchId();
      const defaultBatch = {
        id: batchId,
        name: `组合 ${batches.length + 1}`,
        scope_type: 'channel',
        channel_ids: [],
        tags: [],
        created_at: createBatchCreatedAt(),
      };
      const defaultComboConfig = createDefaultComboPricingConfig(
        batchId,
        siteConfig,
        siteConfig,
        upstreamConfig,
      );
      const nextDraft = cloneComboDraft(defaultBatch, defaultComboConfig);
      setEditingBatchId('');
      setEditorNameAuto(true);
      setEditorDraft({
        ...nextDraft,
        name: createSuggestedComboName(
          nextDraft,
          channelMap,
          t,
          `组合 ${batches.length + 1}`,
        ),
      });
      setEditorVisible(true);
    } catch (error) {
      showError(error);
    }
  }, [
    batches.length,
    builderOptionsReady,
    channelMap,
    loadBuilderOptions,
    siteConfig,
    t,
    upstreamConfig,
  ]);

  const openEditBatchModal = useCallback(
    async (batch) => {
      try {
        if (!builderOptionsReady) {
          await loadBuilderOptions();
          setBuilderOptionsReady(true);
        }
        const nextDraft = cloneComboDraft(batch, resolveComboConfig(batch.id));
        const suggestedName = createSuggestedComboName(
          nextDraft,
          channelMap,
          t,
          batch.name || t('未命名组合'),
        );
        setEditingBatchId(batch.id);
        setEditorNameAuto(isLikelyAutoComboName(batch.name, suggestedName));
        setEditorDraft(nextDraft);
        setEditorVisible(true);
      } catch (error) {
        showError(error);
      }
    },
    [builderOptionsReady, channelMap, loadBuilderOptions, resolveComboConfig, t],
  );

  const handleApplyRecommendedModes = useCallback(() => {
    setEditorDraftSmart((prev) =>
      prev
        ? {
            ...prev,
            site_mode: dominantComboModes.site_mode,
            upstream_mode: dominantComboModes.upstream_mode,
          }
        : prev,
    );
  }, [
    dominantComboModes.site_mode,
    dominantComboModes.upstream_mode,
    setEditorDraftSmart,
  ]);

  const handleApplyTemplate = useCallback(
    (templateBatchId) => {
      const templateConfig = comboConfigs.find(
        (item) => item.combo_id === templateBatchId,
      );
      if (!templateConfig) return;
      setEditorDraftSmart((prev) =>
        mergeComboDraftWithTemplate(prev, templateConfig),
      );
    },
    [comboConfigs, setEditorDraftSmart],
  );

  const handleApplyRecommendedAccount = useCallback(() => {
    if (!recommendedAccountId) return;
    setEditorDraftSmart((prev) =>
      prev
        ? {
            ...prev,
            upstream_mode: 'wallet_observer',
            upstream_account_id: Number(recommendedAccountId),
          }
        : prev,
    );
  }, [recommendedAccountId, setEditorDraftSmart]);

  const handleSaveEditor = useCallback(async () => {
    const error = buildDraftValidationError(editorDraft);
    if (error) {
      setEditorValidationError(error);
      showError(error);
      return;
    }
    const nextBatch = {
      id: editorDraft.id,
      name:
        editorDraft.name?.trim() ||
        `组合 ${batches.length + (editingBatchId ? 0 : 1)}`,
      scope_type: editorDraft.scope_type,
      channel_ids:
        editorDraft.scope_type === 'channel'
          ? editorDraft.channel_ids || []
          : [],
      tags: editorDraft.scope_type === 'tag' ? editorDraft.tags || [] : [],
      created_at: Number(editorDraft.created_at || createBatchCreatedAt()),
    };
    const nextComboConfig = {
      combo_id: nextBatch.id,
      site_mode: editorDraft.site_mode,
      upstream_mode: editorDraft.upstream_mode,
      cost_source: editorDraft.cost_source || 'manual_only',
      upstream_account_id: Number(editorDraft.upstream_account_id || 0),
      shared_site: { ...(editorDraft.shared_site || {}) },
      site_rules: (editorDraft.site_rules || []).map((rule) =>
        createDefaultPricingRule(rule),
      ),
      upstream_rules: (editorDraft.upstream_rules || []).map((rule) =>
        createDefaultPricingRule(rule),
      ),
      site_fixed_total_amount: clampNumber(editorDraft.site_fixed_total_amount),
      upstream_fixed_total_amount: clampNumber(
        editorDraft.upstream_fixed_total_amount,
      ),
      remote_observer: { ...(editorDraft.remote_observer || {}) },
    };

    upsertBatch(nextBatch);
    setComboConfigs((prev) => {
      const exists = prev.some((item) => item.combo_id === nextBatch.id);
      if (!exists) return [...prev, nextComboConfig];
      return prev.map((item) =>
        item.combo_id === nextBatch.id ? nextComboConfig : item,
      );
    });
    setHasUnsavedConfigChanges(true);
    const walletAccountId =
      nextComboConfig.upstream_mode === 'wallet_observer'
        ? Number(nextComboConfig.upstream_account_id || 0)
        : 0;
    showSuccess(
      walletAccountId > 0
        ? editingBatchId
          ? '组合已更新，正在同步上游账户'
          : '组合已添加，正在同步上游账户'
        : editingBatchId
          ? '组合已更新'
          : '组合已添加',
    );
    closeEditor();
    pendingAutoSaveRef.current = true;

    if (walletAccountId > 0) {
      await accountsHook.syncAccount(walletAccountId, {
        forceRefresh: true,
        suppressReadyToast: true,
        suppressNeedsBaselineToast: true,
      });
    }
  }, [
    accountsHook.syncAccount,
    batches.length,
    buildDraftValidationError,
    closeEditor,
    editorDraft,
    editingBatchId,
    upsertBatch,
  ]);

  const handleRemoveBatch = useCallback(
    (batch) => {
      Modal.confirm({
        title: t('确认删除'),
        content: t(
          '删除后将同时移除组合”{{name}}”及其定价配置，并自动同步到服务器。',
          { name: batch.name },
        ),
        okText: t('确认删除'),
        cancelText: t('取消'),
        okButtonProps: {
          type: 'danger',
        },
        onOk: () => {
          removeBatch(batch.id);
          setComboConfigs((prev) =>
            prev.filter((item) => item.combo_id !== batch.id),
          );
          setHasUnsavedConfigChanges(true);
          pendingAutoSaveRef.current = true;
          if (editingBatchId === batch.id) {
            closeEditor();
          }
          showSuccess(t('组合已删除'));
        },
      });
    },
    [closeEditor, editingBatchId, removeBatch, t],
  );

  const handleSaveConfig = useCallback(async () => {
    const saved = await saveConfig(validationErrors);
    if (saved) {
      setHasUnsavedConfigChanges(false);
    }
  }, [saveConfig, validationErrors]);

  const editorSmartSuggestions = useMemo(() => {
    if (!editorDraft) return null;

    const suggestedName = getEditorSuggestedName(editorDraft);
    const currentAccountId = Number(editorDraft.upstream_account_id || 0);
    const recommendedModeLabel = `${dominantComboModes.site_mode === 'shared_site_model' ? t('本站模型价格') : t('手动定价')} / ${dominantComboModes.upstream_mode === 'wallet_observer' ? t('钱包余额变化') : t('模型单价')}`;

    return {
      suggestedName,
      canApplySuggestedName:
        !!suggestedName &&
        (!editorNameAuto || (editorDraft.name?.trim() || '') !== suggestedName),
      copyTemplateOptions,
      shouldRecommendModes:
        editorDraft.site_mode !== dominantComboModes.site_mode ||
        editorDraft.upstream_mode !== dominantComboModes.upstream_mode,
      recommendedModeLabel,
      recommendedAccountName: recommendedAccount?.name || '',
      shouldRecommendAccount:
        !!recommendedAccount &&
        (editorDraft.upstream_mode !== 'wallet_observer' ||
          currentAccountId !== Number(recommendedAccount.id)),
    };
  }, [
    copyTemplateOptions,
    dominantComboModes.site_mode,
    dominantComboModes.upstream_mode,
    editorDraft,
    editorNameAuto,
    getEditorSuggestedName,
    recommendedAccount,
    t,
  ]);

  const overviewPanelProps = {
    overviewQuerying,
    autoRefreshing,
    queryReady,
    overviewReport,
    overviewSummaryCards,
    formatMoney,
    status: statusState?.status,
    t,
  };

  const chartAnalysisProps = {
    analysisMode,
    setAnalysisMode,
    metricKey,
    setMetricKey,
    metricOptions: metricOpts,
    viewBatchId,
    setViewBatchId,
    batchSummaryOptions,
    granularity,
    setGranularity,
    customIntervalMinutes,
    setCustomIntervalMinutes,
    datePresets: createPresetRanges(t),
    dateRange,
    setDateRange,
    runQuery: () => runFullRefresh(),
    querying: querying || autoRefreshing,
    chartTab,
    setChartTab,
    channelGroupMode,
    setChannelGroupMode,
    report,
    reportMatchesCurrentFilters,
    queryReady,
    chartContent,
    trendBucketCount,
    tagAggregationHint: tagAggregationMeta.emptyReason,
    validationErrors,
    t,
  };

  const comboManagerProps = {
    batches,
    batchDigest,
    resolveComboConfig,
    getSiteSummary: (comboConfig) => getSiteSummaryText(comboConfig, t),
    getUpstreamSummary: (comboConfig) =>
      getUpstreamSummaryText(comboConfig, options, t),
    batchValidationError: duplicateBatchError,
    isMobile,
    onCreateBatch: openCreateBatchModal,
    onEditBatch: openEditBatchModal,
    onRemoveBatch: handleRemoveBatch,
    t,
  };

  const walletCardProps = {
    accounts: accountsHook.accounts,
    accountDraft: accountsHook.accountDraft,
    updateAccountDraftField: accountsHook.updateAccountDraftField,
    normalizeAccountDraftBaseUrl: accountsHook.normalizeAccountDraftBaseUrl,
    touchAccountDraftField: accountsHook.touchAccountDraftField,
    accountDraftErrors: accountsHook.accountDraftErrors,
    accountDraftCanSave: accountsHook.accountDraftCanSave,
    accountDraftValidation: accountsHook.accountDraftValidation,
    editingAccountId: accountsHook.editingAccountId,
    editingAccount: accountsHook.editingAccount,
    accountTrend: accountsHook.accountTrend,
    accountTrendLoading: accountsHook.accountTrendLoading,
    saveAccount: accountsHook.saveAccount,
    syncAccount: accountsHook.syncAccount,
    syncAllAccounts: accountsHook.syncAllAccounts,
    deleteAccount: accountsHook.deleteAccount,
    savingAccount: accountsHook.savingAccount,
    syncingAccountId: accountsHook.syncingAccountId,
    syncingAllAccounts: accountsHook.syncingAllAccounts,
    deletingAccountId: accountsHook.deletingAccountId,
    sideSheetVisible: accountsHook.sideSheetVisible,
    detailSideSheetVisible: accountsHook.detailSideSheetVisible,
    openCreateSideSheet: accountsHook.openCreateSideSheet,
    openEditSideSheet: accountsHook.openEditSideSheet,
    closeSideSheet: accountsHook.closeSideSheet,
    openDetailSideSheet: accountsHook.openDetailSideSheet,
    closeDetailSideSheet: accountsHook.closeDetailSideSheet,
    formatMoney,
    status: statusState?.status,
    t,
  };

  const pricingModalProps = {
    visible: editorVisible,
    isEditing: !!editingBatchId,
    comboConfig: editorDraft,
    setComboConfig: setEditorDraftSmart,
    onNameChange: handleEditorNameChange,
    onRegenerateName: handleRegenerateEditorName,
    onApplyRecommendedModes: handleApplyRecommendedModes,
    onApplyTemplate: handleApplyTemplate,
    onApplyRecommendedAccount: handleApplyRecommendedAccount,
    smartSuggestions: editorSmartSuggestions,
    channelOptions,
    tagOptions,
    modelNameOptions,
    options,
    resolveSharedSitePreview,
    getModelsByChannelIds,
    getModelsByTags,
    isMobile,
    clampNumber,
    localModelMap,
    validationError: editorValidationError,
    onOk: handleSaveEditor,
    onCancel: closeEditor,
    t,
  };

  return (
    <>
      <div className='mt-[60px] space-y-3 px-2 pb-6'>
        <ProfitBoardHeader
          querying={querying}
          overviewQuerying={overviewQuerying}
          runFullRefresh={runFullRefresh}
          saving={saving}
          saveConfig={handleSaveConfig}
          autoRefreshMode={autoRefreshMode}
          setAutoRefreshMode={setAutoRefreshMode}
          statusSummary={statusSummary}
          hasNewActivity={hasNewActivity}
          generatedAtText={generatedAtText}
          combinedWarnings={combinedWarnings}
          sitePriceFactorNote={sitePriceFactorNote}
          hasUnsavedConfigChanges={hasUnsavedConfigChanges}
          configReady={configReady}
          t={t}
        />
        <Tabs
          type='line'
          size='large'
          className='profit-board-tabs'
          activeKey={activeTab}
          onChange={setActiveTab}
        >
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1.5'>
                <BadgeDollarSign size={16} />
                {t('上游账户')}
              </span>
            }
            itemKey='wallet'
          >
            <div className='mt-3 space-y-3'>
              {accountsLoading && !options.upstream_accounts?.length ? (
                <Empty description={t('上游账户加载中')} />
              ) : (
                <UpstreamWalletCard {...walletCardProps} />
              )}
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1.5'>
                <BarChart3 size={16} />
                {t('收益分析')}
              </span>
            }
            itemKey='analysis'
          >
            <div className='mt-3 space-y-3'>
              <OverviewPanel {...overviewPanelProps} />
              <ChartAnalysisCard {...chartAnalysisProps} />
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
            <div className='mt-3 space-y-3'>
              {builderLoading && !builderOptionsReady ? (
                <Empty description={t('配置选项加载中')} />
              ) : (
                <ComboManagerCard {...comboManagerProps} />
              )}
            </div>
          </Tabs.TabPane>
        </Tabs>
      </div>

      <PricingConfigModal {...pricingModalProps} />
    </>
  );
};

export default ProfitBoardPage;
