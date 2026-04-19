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
import { Empty, Skeleton, Tabs } from '@douyinfe/semi-ui';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import { BadgeDollarSign, BarChart3, CircleDollarSign } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { CHART_CONFIG } from '../../constants/dashboard.constants';
import { StatusContext } from '../../context/Status';
import { showError, timestamp2string } from '../../helpers';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import ChartAnalysisCard from './components/ChartAnalysisCard';
import ComboManagerCard from './components/ComboManagerCard';
import ExcludedAdminUsersCard from './components/ExcludedAdminUsersCard';
import OverviewPanel from './components/OverviewPanel';
import PricingConfigModal from './components/PricingConfigModal';
import ProfitBoardHeader from './components/ProfitBoardHeader';
import ResponsiveVChart from './components/ResponsiveVChart';
import UpstreamWalletCard from './components/UpstreamWalletCard';
import { useComboEditor } from './hooks/useComboEditor';
import { useProfitBoardBatches } from './hooks/useProfitBoardBatches';
import { useProfitBoardConfig } from './hooks/useProfitBoardConfig';
import { useProfitBoardPersist } from './hooks/useProfitBoardPersist';
import { useProfitBoardQuery } from './hooks/useProfitBoardQuery';
import { useUpstreamAccounts } from './hooks/useUpstreamAccounts';
import {
  aggregateBreakdownRows,
  aggregateChannelRowsByTag,
  buildBatchOverlapError,
  clampNumber,
  combineBreakdownMetrics,
  combineChannelMetricsByTag,
  combineTimeseriesMetrics,
  createBarSpec,
  createDefaultComboPricingConfig,
  createDefaultPricingRule,
  createMetricOptions,
  createPresetRanges,
  createTrendSpec,
  formatBoardExchangeRate,
  formatBoardMetricPair,
  formatMoney,
  getUpstreamCostSourceLabel,
  normalizeBatchForState,
} from './utils';

initVChartSemiTheme({
  isWatchingThemeSwitch: true,
});

const getSiteSummaryText = (comboConfig, t) => {
  const rateText = formatBoardExchangeRate(comboConfig.site_exchange_rate);
  if (comboConfig.site_mode === 'log_quota')
    return `${t('智能（按日志额度）')} · ${rateText}`;
  if (comboConfig.site_mode !== 'shared_site_model')
    return `${t('手动定价')} · ${rateText}`;
  const modelCount = comboConfig.shared_site?.model_names?.length || 0;
  if (modelCount === 0) return `${t('本站模型价格')} · ${rateText}`;
  return `${t('本站模型价格 · {{count}} 个模型', { count: modelCount })} · ${rateText}`;
};

const getUpstreamSummaryText = (comboConfig, options, t) => {
  const rateText = formatBoardExchangeRate(comboConfig.upstream_exchange_rate);
  if (comboConfig.upstream_mode !== 'wallet_observer') {
    return `${getUpstreamCostSourceLabel('manual_only', t)} · ${rateText}`;
  }
  const account = (options?.upstream_accounts || []).find(
    (item) => item.id === Number(comboConfig.upstream_account_id || 0),
  );
  return account
    ? `${t('按钱包余额变化 · {{name}}', { name: account.name })} · ${rateText}`
    : `${t('按钱包余额变化')} · ${rateText}`;
};

const normalizeServerBatchState = (batch, index, fallbackCreatedAt = 0) => {
  const normalized = normalizeBatchForState(
    {
      id: batch?.id || '',
      name: batch?.name || '',
      scope_type: batch?.scope_type || 'channel',
      channel_ids: (batch?.channel_ids || []).map(Number).filter(Boolean),
      tags: batch?.tags || [],
      created_at: Number(batch?.created_at || fallbackCreatedAt || 0),
    },
    index,
  );
  return {
    ...normalized,
    channel_ids: (normalized.channel_ids || []).map(Number).filter(Boolean),
  };
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
  const [hasUnsavedConfigChanges, setHasUnsavedConfigChanges] = useState(
    !!restoredState.hasUnsavedConfigChanges,
  );
  const [activeTab, setActiveTab] = useState('wallet');
  const [builderOptionsReady, setBuilderOptionsReady] = useState(false);
  const [configReady, setConfigReady] = useState(false);
  const serverRestoredRef = useRef(false);
  const pendingAutoSaveRef = useRef(false);

  const configHook = useProfitBoardConfig({
    batchPayload: batchesHook.batchPayload,
    comboConfigs,
    setComboConfigs,
    restoredState,
    rechargePriceFactor,
    usdExchangeRate: statusState?.status?.usd_exchange_rate || 0,
  });

  const { batches, setBatches, batchPayload, upsertBatch, removeBatch } =
    batchesHook;

  const {
    builderLoading,
    accountsLoading,
    saving,
    options,
    siteConfig,
    excludedUserIDs,
    setExcludedUserIDs,
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
    subscriptionPlans,
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

  const duplicateBatchError = useMemo(
    () => buildBatchOverlapError(batches, channelMap, tagChannelMap),
    [batches, channelMap, tagChannelMap],
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
          cost_source: 'manual_only',
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

  const validationErrors = useMemo(() => {
    const errors = [];
    if (configReady && !batches.length) errors.push(t('请至少添加一个组合'));
    if (duplicateBatchError) errors.push(duplicateBatchError);
    if (
      comboConfigs.some(
        (item) =>
          (item.site_mode === 'shared_site_model' ||
            item.site_mode === 'log_quota') &&
          !(item.shared_site?.model_names || []).length,
      )
    ) {
      errors.push(t('启用了本站模型价格或智能模式的组合必须至少选择一个模型'));
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

  const queryReady = configReady && batchPayload.length > 0;

  const queryHook = useProfitBoardQuery({
    restoredState,
    cachedBundle,
    configPayload,
    batchPayload,
    validationErrors,
    persistReportCache,
    queryReady,
    activeTab,
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
    activeChartSectionLoaded,
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

  const onConfigChanged = useCallback(() => {
    setHasUnsavedConfigChanges(true);
    pendingAutoSaveRef.current = true;
  }, []);

  const editorHook = useComboEditor({
    batches,
    upsertBatch,
    removeBatch,
    comboConfigs,
    setComboConfigs,
    availableAccountIds,
    availableAccounts,
    channelMap,
    tagChannelMap,
    siteConfig,
    upstreamConfig,
    builderOptionsReady,
    setBuilderOptionsReady,
    loadBuilderOptions,
    resolveComboConfig,
    onConfigChanged,
    syncAccount: accountsHook.syncAccount,
    t,
  });

  // --- Initialization effects ---

  useEffect(() => {
    let cancelled = false;
    loadUpstreamAccounts()
      .catch((error) => {
        if (cancelled) return;
        showError(error);
      });
    if (!serverRestoredRef.current) {
      serverRestoredRef.current = true;
      (async () => {
        try {
          const serverConfig = await loadCurrentConfig();
          if (cancelled) return;
          if (serverConfig) {
            const restoredBatchCreatedAtMap = new Map(
              (restoredState.batches || []).map((batch) => [
                batch.id,
                Number(batch.created_at || 0),
              ]),
            );
            let migratedLegacyBatchCreatedAt = false;
            const serverBatches = (serverConfig.batches || []).map(
              (batch, index) => {
                const rawCreatedAt = Number(batch?.created_at || 0);
                const restoredCreatedAt = Number(
                  restoredBatchCreatedAtMap.get(batch?.id || '') || 0,
                );
                if (rawCreatedAt <= 0) {
                  migratedLegacyBatchCreatedAt = true;
                }
                return normalizeServerBatchState(
                  batch,
                  index,
                  rawCreatedAt > 0 ? rawCreatedAt : restoredCreatedAt,
                );
              },
            );
            if (serverBatches.length > 0) {
              setBatches(serverBatches);
            }
            applyLoadedConfig(serverConfig);
            setHasUnsavedConfigChanges(migratedLegacyBatchCreatedAt);
            if (migratedLegacyBatchCreatedAt) {
              pendingAutoSaveRef.current = true;
            }
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
  }, [
    applyLoadedConfig,
    loadCurrentConfig,
    loadUpstreamAccounts,
    restoredState.batches,
    setBatches,
  ]);

  useEffect(() => {
    if (builderOptionsReady) return;
    if (
      activeTab !== 'config' &&
      !editorHook.editorVisible &&
      channelGroupMode !== 'tag'
    ) {
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
    editorHook.editorVisible,
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
      excludedUserIDs,
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
    excludedUserIDs,
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

  // --- Derived data for charts ---

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
      { key: 'configured_site_revenue_cny', label: t('本站配置收入') },
      { key: 'upstream_cost_cny', label: t('上游费用') },
      { key: 'configured_profit_cny', label: t('利润') },
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
    const bucketMap = new Map();
    filtered.forEach((row) => {
      const bucket = row.bucket;
      const current = bucketMap.get(bucket) || {
        bucket,
        value: 0,
        batch_id: row.batch_id,
        __batchCount: 0,
      };
      current.value += Number(row[metricKey] || 0);
      current.__batchCount += 1;
      if (current.batch_id && current.batch_id !== row.batch_id) {
        current.batch_id = null;
      }
      bucketMap.set(bucket, current);
    });
    return Array.from(bucketMap.values())
      .sort((a, b) => String(a.bucket).localeCompare(String(b.bucket)))
      .map(({ __batchCount, ...rest }) => rest);
  }, [analysisMode, businessMetrics, metricKey, report, viewBatchId]);

  const channelTagMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((channel) => {
      map.set(String(channel.id), channel.tag || t('未设置标签'));
    });
    return map;
  }, [options.channels, t]);

  const channelRows = useMemo(() => {
    if (!report) {
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
    if (!report) {
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
    () => createTrendSpec(trendRows, chartSubtitle, t),
    [chartSubtitle, t, trendRows],
  );
  const channelSpec = useMemo(
    () =>
      createBarSpec(
        channelGroupMode === 'tag' ? t('标签分布') : t('渠道分布'),
        channelRows,
        chartSubtitle,
        t,
      ),
    [channelGroupMode, channelRows, chartSubtitle, t],
  );
  const modelSpec = useMemo(
    () =>
      createBarSpec(
        t('模型分布'),
        modelRows,
        chartSubtitle,
        t,
      ),
    [chartSubtitle, modelRows, t],
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
      trend: !activeChartSectionLoaded && chartTab === 'trend' ? (
        <Empty description={t('当前收益分析正在加载')} />
      ) : trendRows.length ? (
        renderChart(
          `trend-${analysisMode}-${viewBatchId}-${metricKey}-${granularity}-${customIntervalMinutes}-${lastQueryKey}-${trendRows.length}`,
          trendSpec,
        )
      ) : (
        <Empty description={t('当前没有趋势数据')} />
      ),
      channel: !activeChartSectionLoaded && chartTab === 'channel' ? (
        <Empty description={t('当前收益分析正在加载')} />
      ) : channelRows.length ? (
        renderChart(
          `channel-${analysisMode}-${viewBatchId}-${metricKey}-${channelGroupMode}-${granularity}-${customIntervalMinutes}-${lastQueryKey}-${channelRows.length}`,
          channelSpec,
        )
      ) : (
        <Empty
          description={tagAggregationMeta.emptyReason || t('当前没有渠道数据')}
        />
      ),
      model: !activeChartSectionLoaded && chartTab === 'model' ? (
        <Empty description={t('当前收益分析正在加载')} />
      ) : modelRows.length ? (
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
      activeChartSectionLoaded,
      chartTab,
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

  // --- Summary data ---

  const overviewSummaryCards = useMemo(
    () =>
      !overviewReport?.summary
        ? []
        : [
            {
              key: 'configured_site_revenue_cny',
              title: t('本站配置收入'),
              ...formatBoardMetricPair(
                overviewReport.summary.configured_site_revenue_cny,
                overviewReport.summary.configured_site_revenue_usd,
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
              key: 'upstream_cost_cny',
              title: t('上游费用'),
              ...formatBoardMetricPair(
                overviewReport.summary.upstream_cost_cny,
                overviewReport.summary.upstream_cost_usd,
              ),
              icon: (
                <BadgeDollarSign
                  size={18}
                  className='text-amber-600 dark:text-amber-400'
                />
              ),
            },
            {
              key: 'configured_profit_cny',
              title: t('利润'),
              ...formatBoardMetricPair(
                overviewReport.summary.configured_profit_cny,
                overviewReport.summary.configured_profit_usd,
              ),
              icon: (
                <BarChart3
                  size={18}
                  className='text-sky-600 dark:text-sky-400'
                />
              ),
            },
          ],
    [overviewReport?.summary, t],
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

  const sitePriceFactorNote =
    overviewReport?.meta?.site_price_factor_note ||
    report?.meta?.site_price_factor_note ||
    '';

  const combinedMessages = useMemo(() => {
    const messages = [];
    const seenKeys = new Set();
    const structuredWarningTexts = new Set();

    const pushMessage = (message) => {
      if (!message?.text) return;
      const key = message.key || `${message.type}:${message.code || message.text}`;
      if (seenKeys.has(key)) return;
      seenKeys.add(key);
      messages.push(message);
    };

    const pushWarningItems = (items = []) => {
      items.forEach((item, index) => {
        if (!item?.message) return;
        structuredWarningTexts.add(item.message);
        pushMessage({
          key: `warning-item:${item.code || item.message}:${index}`,
          type: 'warning',
          code: item.code || item.message,
          text: item.message,
          totalCount: Number(item.total_count || 0),
          details: Array.isArray(item.details) ? item.details : [],
        });
      });
    };

    pushWarningItems(overviewReport?.warning_items || []);
    pushWarningItems(report?.warning_items || []);

    [...(overviewReport?.warnings || []), ...(report?.warnings || [])].forEach(
      (warningText) => {
        if (!warningText || structuredWarningTexts.has(warningText)) return;
        pushMessage({
          key: `warning-text:${warningText}`,
          type: 'warning',
          text: warningText,
        });
      },
    );

    validationErrors.forEach((warningText) => {
      pushMessage({
        key: `validation:${warningText}`,
        type: 'warning',
        text: warningText,
      });
    });

    if (sitePriceFactorNote) {
      pushMessage({
        key: 'info:site-price-factor-note',
        type: 'info',
        text: sitePriceFactorNote,
      });
    }

    return messages;
  }, [
    overviewReport?.warning_items,
    report?.warning_items,
    overviewReport?.warnings,
    report?.warnings,
    validationErrors,
    sitePriceFactorNote,
  ]);

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

  const generatedAtText =
    reportMatchesCurrentFilters && report?.meta?.generated_at
      ? timestamp2string(report.meta.generated_at)
      : t('尚未生成');
  const trendBucketCount = useMemo(
    () => new Set((trendRows || []).map((row) => row.bucket)).size,
    [trendRows],
  );

  const batchMetrics = useMemo(() => {
    const summaries = overviewReport?.batch_summaries;
    if (!summaries?.length) return null;
    const map = {};
    summaries.forEach((s) => {
      map[s.batch_id] = {
        revenue: formatBoardMetricPair(
          s.configured_site_revenue_cny,
          s.configured_site_revenue_usd,
        ),
        cost: formatBoardMetricPair(s.upstream_cost_cny, s.upstream_cost_usd),
        profit: formatBoardMetricPair(
          s.configured_profit_cny,
          s.configured_profit_usd,
        ),
      };
    });
    return map;
  }, [overviewReport?.batch_summaries]);

  const handleSaveConfig = useCallback(async () => {
    const saved = await saveConfig(validationErrors);
    if (saved) {
      setHasUnsavedConfigChanges(false);
    }
  }, [saveConfig, validationErrors]);

  const handleExcludedUserIDsChange = useCallback((nextIDs) => {
    setExcludedUserIDs(nextIDs);
    setHasUnsavedConfigChanges(true);
    pendingAutoSaveRef.current = true;
  }, [setExcludedUserIDs]);

  const handleMoveBatch = useCallback(
    (index, direction) => {
      const target = index + direction;
      if (target < 0 || target >= batches.length) return;
      const next = [...batches];
      [next[index], next[target]] = [next[target], next[index]];
      setBatches(next);
      onConfigChanged();
    },
    [batches, onConfigChanged, setBatches],
  );

  // --- Render ---

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
          combinedMessages={combinedMessages}
          hasUnsavedConfigChanges={hasUnsavedConfigChanges}
          configReady={configReady}
          t={t}
        />
        <Tabs
          type='line'
          size='large'
          keepDOM
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
                <div className='space-y-3'>
                  <Skeleton.Title style={{ width: 200 }} />
                  <div className='grid gap-4 md:grid-cols-2'>
                    <Skeleton.Paragraph rows={4} />
                    <Skeleton.Paragraph rows={4} />
                  </div>
                </div>
              ) : (
                <UpstreamWalletCard
                  accounts={accountsHook.accounts}
                  accountDraft={accountsHook.accountDraft}
                  updateAccountDraftField={accountsHook.updateAccountDraftField}
                  normalizeAccountDraftBaseUrl={
                    accountsHook.normalizeAccountDraftBaseUrl
                  }
                  touchAccountDraftField={accountsHook.touchAccountDraftField}
                  accountDraftErrors={accountsHook.accountDraftErrors}
                  accountDraftCanSave={accountsHook.accountDraftCanSave}
                  accountDraftValidation={accountsHook.accountDraftValidation}
                  editingAccountId={accountsHook.editingAccountId}
                  editingAccount={accountsHook.editingAccount}
                  accountTrend={accountsHook.accountTrend}
                  accountTrendLoading={accountsHook.accountTrendLoading}
                  saveAccount={accountsHook.saveAccount}
                  syncAccount={accountsHook.syncAccount}
                  syncAllAccounts={accountsHook.syncAllAccounts}
                  deleteAccount={accountsHook.deleteAccount}
                  savingAccount={accountsHook.savingAccount}
                  syncingAccountId={accountsHook.syncingAccountId}
                  syncingAllAccounts={accountsHook.syncingAllAccounts}
                  deletingAccountId={accountsHook.deletingAccountId}
                  sideSheetVisible={accountsHook.sideSheetVisible}
                  detailSideSheetVisible={accountsHook.detailSideSheetVisible}
                  openCreateSideSheet={accountsHook.openCreateSideSheet}
                  openEditSideSheet={accountsHook.openEditSideSheet}
                  closeSideSheet={accountsHook.closeSideSheet}
                  openDetailSideSheet={accountsHook.openDetailSideSheet}
                  closeDetailSideSheet={accountsHook.closeDetailSideSheet}
                  formatMoney={formatMoney}
                  status={statusState?.status}
                  t={t}
                />
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
              <OverviewPanel
                overviewQuerying={overviewQuerying}
                autoRefreshing={autoRefreshing}
                queryReady={queryReady}
                overviewReport={overviewReport}
                overviewSummaryCards={overviewSummaryCards}
                formatMoney={formatMoney}
                status={statusState?.status}
                timeseries={report?.timeseries}
                onMetricClick={(metricKey) => {
                  setAnalysisMode('single_metric');
                  setMetricKey(metricKey);
                }}
                t={t}
              />
              <ChartAnalysisCard
                analysisMode={analysisMode}
                setAnalysisMode={setAnalysisMode}
                metricKey={metricKey}
                setMetricKey={setMetricKey}
                metricOptions={metricOpts}
                viewBatchId={viewBatchId}
                setViewBatchId={setViewBatchId}
                batchSummaryOptions={batchSummaryOptions}
                granularity={granularity}
                setGranularity={setGranularity}
                customIntervalMinutes={customIntervalMinutes}
                setCustomIntervalMinutes={setCustomIntervalMinutes}
                datePresets={createPresetRanges(t)}
                dateRange={dateRange}
                setDateRange={setDateRange}
                runQuery={() => runFullRefresh()}
                querying={querying || autoRefreshing}
                chartTab={chartTab}
                setChartTab={setChartTab}
                channelGroupMode={channelGroupMode}
                setChannelGroupMode={setChannelGroupMode}
                report={report}
                reportMatchesCurrentFilters={reportMatchesCurrentFilters}
                queryReady={queryReady}
                chartContent={chartContent}
                trendBucketCount={trendBucketCount}
                tagAggregationHint={tagAggregationMeta.emptyReason}
                validationErrors={validationErrors}
                onNavigateToConfig={() => setActiveTab('config')}
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
            <div className='mt-3 space-y-3'>
              {builderLoading && !builderOptionsReady ? (
                <div className='space-y-3'>
                  <Skeleton.Title style={{ width: 200 }} />
                  <Skeleton.Paragraph rows={3} />
                </div>
              ) : (
                <>
                  <ExcludedAdminUsersCard
                    adminUsers={options.admin_users || []}
                    excludedUserIDs={excludedUserIDs}
                    onChange={handleExcludedUserIDsChange}
                    t={t}
                  />
                  <ComboManagerCard
                    batches={batches}
                    batchDigest={batchDigest}
                    resolveComboConfig={resolveComboConfig}
                    getSiteSummary={(comboConfig) =>
                      getSiteSummaryText(comboConfig, t)
                    }
                    getUpstreamSummary={(comboConfig) =>
                      getUpstreamSummaryText(comboConfig, options, t)
                    }
                    batchValidationError={duplicateBatchError}
                    batchMetrics={batchMetrics}
                    isMobile={isMobile}
                    onCreateBatch={editorHook.openCreateBatchModal}
                    onEditBatch={editorHook.openEditBatchModal}
                    onRemoveBatch={editorHook.handleRemoveBatch}
                    onMoveBatch={handleMoveBatch}
                    t={t}
                  />
                </>
              )}
            </div>
          </Tabs.TabPane>
        </Tabs>
      </div>

      <PricingConfigModal
        visible={editorHook.editorVisible}
        isEditing={!!editorHook.editingBatchId}
        comboConfig={editorHook.editorDraft}
        setComboConfig={editorHook.setEditorDraftSmart}
        onNameChange={editorHook.handleEditorNameChange}
        onRegenerateName={editorHook.handleRegenerateEditorName}
        onApplyRecommendedModes={editorHook.handleApplyRecommendedModes}
        onApplyTemplate={editorHook.handleApplyTemplate}
        onApplyRecommendedAccount={editorHook.handleApplyRecommendedAccount}
        smartSuggestions={editorHook.editorSmartSuggestions}
        channelOptions={channelOptions}
        tagOptions={tagOptions}
        modelNameOptions={modelNameOptions}
        options={options}
        resolveSharedSitePreview={resolveSharedSitePreview}
        getModelsByChannelIds={getModelsByChannelIds}
        getModelsByTags={getModelsByTags}
        isMobile={isMobile}
        clampNumber={clampNumber}
        localModelMap={localModelMap}
        subscriptionPlans={subscriptionPlans}
        usdExchangeRate={statusState?.status?.usd_exchange_rate || 0}
        validationError={editorHook.editorValidationError}
        onOk={editorHook.handleSaveEditor}
        onCancel={editorHook.closeEditor}
        t={t}
      />
    </>
  );
};

export default ProfitBoardPage;
