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

import React, { useCallback, useContext, useEffect, useMemo } from 'react';
import { Empty, Spin, Tabs, Tag, Typography } from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import { BadgeDollarSign, BarChart3, CircleDollarSign } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { showError, timestamp2string } from '../../helpers';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import ChartAnalysisCard from './components/ChartAnalysisCard';
import ComboManagerCard from './components/ComboManagerCard';
import DetailTableCard from './components/DetailTableCard';
import OverviewPanel from './components/OverviewPanel';
import PricingRulesCard from './components/PricingRulesCard';
import ProfitBoardHeader from './components/ProfitBoardHeader';
import UpstreamWalletCard from './components/UpstreamWalletCard';
import TimeRangePanel from './components/TimeRangePanel';
import { useProfitBoardBatches } from './hooks/useProfitBoardBatches';
import { useProfitBoardConfig } from './hooks/useProfitBoardConfig';
import { useProfitBoardPersist } from './hooks/useProfitBoardPersist';
import { useProfitBoardQuery } from './hooks/useProfitBoardQuery';
import { useUpstreamAccounts } from './hooks/useUpstreamAccounts';
import {
  aggregateBreakdownRows,
  clampNumber,
  combineBreakdownMetrics,
  combineTimeseriesMetrics,
  createBarSpec,
  createDefaultComboPricingConfig,
  createMetricOptions,
  createPresetRanges,
  createSitePricingSourceLabelMap,
  createTrendSpec,
  formatMoney,
  formatRatio,
} from './utils';

const { Text } = Typography;
initVChartSemiTheme();

const ProfitBoardPage = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const isMobile = useIsMobile();

  const { restoredState, cachedBundle, persistState, persistReportCache } =
    useProfitBoardPersist();

  const batchesHook = useProfitBoardBatches({
    restoredState,
    channelMap: null,
    tagChannelMap: null,
    siteConfig: restoredState.siteConfig || {},
    upstreamConfig: restoredState.upstreamConfig || {},
  });

  const configHook = useProfitBoardConfig({
    batchPayload: batchesHook.batchPayload,
    comboConfigs: batchesHook.comboConfigs,
    setComboConfigs: batchesHook.setComboConfigs,
    restoredState,
  });

  const {
    batches,
    draft,
    setDraft,
    editingBatchId,
    comboConfigs,
    batchPayload,
    addOrUpdateBatch,
    editBatch,
    resetDraft,
    removeBatch,
    updateComboConfig,
    addComboRule,
    updateComboRule,
    removeComboRule,
    duplicateBatchError,
  } = batchesHook;

  const {
    loading,
    setLoading,
    saving,
    options,
    siteConfig,
    setSiteConfig,
    upstreamConfig,
    setUpstreamConfig,
    channelOptions,
    channelMap,
    localModelMap,
    modelNameOptions,
    configPayload,
    configLookupKey,
    walletModeEnabled,
    selectedAccount,
    loadOptions,
    loadConfig,
    saveConfig,
    resolveSharedSitePreview,
  } = configHook;

  const validationErrors = useMemo(() => {
    const errors = [];
    if (!batches.length) errors.push(t('请至少添加一个组合'));
    if (duplicateBatchError) errors.push(duplicateBatchError);
    if (
      comboConfigs.some(
        (c) =>
          c.site_mode === 'shared_site_model' &&
          !(c.shared_site?.model_names || []).length,
      )
    )
      errors.push(t('启用了本站模型价格的组合必须至少选择一个模型'));
    if (
      upstreamConfig.upstream_mode === 'wallet_observer' &&
      !Number(upstreamConfig.upstream_account_id || 0)
    )
      errors.push(t('钱包扣减模式必须绑定一个上游账户'));
    return errors;
  }, [batches.length, comboConfigs, duplicateBatchError, upstreamConfig, t]);

  const queryHook = useProfitBoardQuery({
    restoredState,
    cachedBundle,
    configPayload,
    batchPayload,
    validationErrors,
    persistReportCache,
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
    metricKey,
    setMetricKey,
    analysisMode,
    setAnalysisMode,
    viewBatchId,
    setViewBatchId,
    detailFilter,
    setDetailFilter,
    overviewReport,
    report,
    reportMatchesCurrentFilters,
    autoRefreshMode,
    setAutoRefreshMode,
    hasNewActivity,
    activityChecking,
    detailRows,
    detailLoading,
    detailPage,
    setDetailPage,
    detailPageSize,
    setDetailPageSize,
    detailTotal,
    runFullRefresh,
    runQuery,
  } = queryHook;

  const accountsHook = useUpstreamAccounts({
    options,
    loadOptions,
    upstreamConfig,
    setUpstreamConfig,
    runFullRefresh,
  });

  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        await loadOptions();
      } catch (e) {
        showError(e);
      } finally {
        setLoading(false);
      }
    })();
  }, [loadOptions, setLoading]);

  useEffect(() => {
    if (batchPayload.length) loadConfig().catch(showError);
  }, [batchPayload.length, configLookupKey, loadConfig]);

  useEffect(() => {
    persistState({
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
      lastQueryKey: queryHook.lastQueryKey,
      detailPage,
      detailPageSize,
      autoRefreshMode,
    });
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
    metricKey,
    persistState,
    queryHook.lastQueryKey,
    siteConfig,
    upstreamConfig,
    viewBatchId,
  ]);

  const metricOpts = useMemo(() => createMetricOptions(t), [t]);
  const sitePricingLabels = useMemo(
    () => createSitePricingSourceLabelMap(t),
    [t],
  );
  const batchSummaryOptions = useMemo(
    () => [
      { label: t('全部组合'), value: 'all' },
      ...[
        ...(overviewReport?.batch_summaries || []),
        ...(report?.batch_summaries || []),
      ].map((s) => ({ label: s.batch_name, value: s.batch_id })),
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
      metricOpts.find((m) => m.value === metricKey)?.label ||
      metricOpts[0].label,
    [metricKey, metricOpts],
  );
  const chartSubtitle =
    analysisMode === 'business_compare'
      ? t('本站配置收入 / 上游费用 / 配置利润')
      : metricLabel;

  const trendRows = useMemo(() => {
    if (!report) return [];
    if (analysisMode === 'business_compare')
      return combineTimeseriesMetrics(
        report.timeseries || [],
        viewBatchId,
        businessMetrics,
      );
    const filtered =
      viewBatchId === 'all'
        ? report.timeseries || []
        : (report.timeseries || []).filter((r) => r.batch_id === viewBatchId);
    return filtered.map((r) => ({
      bucket: r.bucket,
      value: Number(r[metricKey] || 0),
      batch_id: r.batch_id,
    }));
  }, [analysisMode, businessMetrics, metricKey, report, viewBatchId]);

  const channelRows = useMemo(() => {
    if (
      !report ||
      (analysisMode === 'single_metric' &&
        metricKey === 'remote_observed_cost_usd')
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
    if (
      !report ||
      (analysisMode === 'single_metric' &&
        metricKey === 'remote_observed_cost_usd')
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
    [setDetailFilter, setDetailPage],
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

  const summaryMetricHelp = useMemo(
    () => ({
      request_count: t('当前口径内命中的消费日志数量。'),
      remote_observed_cost_usd: t(
        '来自上游账户钱包和订阅已用额度的增量，按本站额度口径换算。',
      ),
    }),
    [t],
  );
  const cumulativeSummaryCards = useMemo(
    () =>
      !overviewReport?.summary
        ? []
        : [
            { key: 'configured_site_revenue_usd', title: t('本站配置收入'), value: formatMoney(overviewReport.summary.configured_site_revenue_usd, statusState?.status), icon: <CircleDollarSign size={18} className='text-emerald-600 dark:text-emerald-400' /> },
            { key: 'upstream_cost_usd', title: t('上游费用'), value: formatMoney(overviewReport.summary.upstream_cost_usd, statusState?.status), icon: <BadgeDollarSign size={18} className='text-amber-600 dark:text-amber-400' /> },
            { key: 'configured_profit_usd', title: t('配置利润'), value: formatMoney(overviewReport.summary.configured_profit_usd, statusState?.status), icon: <BarChart3 size={18} className='text-sky-600 dark:text-sky-400' /> },
            { key: 'remote_observed_cost_usd', title: t('上游实际消耗'), value: formatMoney(overviewReport.summary.remote_observed_cost_usd, statusState?.status), icon: <BadgeDollarSign size={18} className='text-rose-600 dark:text-rose-400' /> },
            { key: 'actual_profit_usd', title: t('实际利润'), value: formatMoney(overviewReport.summary.actual_profit_usd, statusState?.status), icon: <BarChart3 size={18} className='text-violet-600 dark:text-violet-400' /> },
          ],
    [overviewReport?.summary, statusState?.status, t],
  );
  const diagnosticSummaryCards = useMemo(
    () =>
      !overviewReport?.summary
        ? []
        : [
            { key: 'request_count', title: t('请求数'), value: overviewReport.summary.request_count },
            { key: 'configured_profit_coverage_rate', title: t('配置利润覆盖率'), value: formatRatio(overviewReport.summary.configured_profit_coverage_rate) },
            { key: 'returned_cost_count', title: t('上游返回费用'), value: overviewReport.summary.returned_cost_count },
            { key: 'manual_cost_count', title: t('手动上游价格'), value: overviewReport.summary.manual_cost_count },
            { key: 'missing_site_pricing_count', title: t('缺失本站价格'), value: overviewReport.summary.missing_site_pricing_count },
          ],
    [overviewReport?.summary, t],
  );

  const statusSummary = useMemo(() => {
    const items = [];
    if (reportMatchesCurrentFilters) items.push({ key: 'fresh', color: 'blue', text: t('时间分析已同步') });
    else if (report) items.push({ key: 'stale', color: 'grey', text: t('筛选已变化，等待刷新') });
    if (overviewReport) items.push({ key: 'overview', color: 'green', text: t('累计总览已更新') });
    if (activityChecking) items.push({ key: 'watch', color: 'cyan', text: t('低频检查中') });
    return items;
  }, [activityChecking, overviewReport, report, reportMatchesCurrentFilters, t]);

  const combinedWarnings = useMemo(
    () => Array.from(new Set([...(overviewReport?.warnings || []), ...(report?.warnings || []), ...validationErrors])),
    [overviewReport?.warnings, report?.warnings, validationErrors],
  );

  const detailFilterText = useMemo(() => {
    if (!detailFilter?.value) return '';
    const typeLabels = { trend: t('时间桶'), channel: t('渠道'), model: t('模型') };
    return `${typeLabels[detailFilter.type] || t('筛选')}：${detailFilter.value}`;
  }, [detailFilter, t]);

  const batchDigest = useCallback(
    (batch) =>
      batch.scope_type === 'channel'
        ? (batch.channel_ids || []).map((id) => channelMap.get(String(id))?.name).filter(Boolean).slice(0, 3).join('、')
        : (batch.tags || []).slice(0, 3).join('、'),
    [channelMap],
  );

  const generatedAtText = report?.meta?.generated_at ? timestamp2string(report.meta.generated_at) : t('尚未生成');
  const trendBucketCount = useMemo(() => new Set((trendRows || []).map((r) => r.bucket)).size, [trendRows]);
  const sitePriceFactorNote = overviewReport?.meta?.site_price_factor_note || report?.meta?.site_price_factor_note || '';

  const detailColumns = useMemo(() => [
    { title: t('时间'), dataIndex: 'created_at', render: (v) => timestamp2string(v), width: 160 },
    { title: t('组合'), dataIndex: 'batch_name', width: 120 },
    { title: t('渠道'), dataIndex: 'channel_name', render: (v, r) => v || `#${r.channel_id}`, width: 140 },
    { title: t('模型'), dataIndex: 'model_name', width: 160 },
    { title: t('本站配置收入'), dataIndex: 'configured_site_revenue_usd', render: (v) => <span className='font-medium text-emerald-600 dark:text-emerald-400'>{formatMoney(v, statusState?.status)}</span>, width: 130 },
    { title: t('配置利润'), dataIndex: 'configured_profit_usd', render: (v, r) => r.upstream_cost_known && r.site_pricing_known ? <span className='font-medium text-sky-600 dark:text-sky-400'>{formatMoney(v, statusState?.status)}</span> : <Text type='tertiary'>-</Text>, width: 110 },
    { title: t('上游费用'), dataIndex: 'upstream_cost_usd', render: (v, r) => r.upstream_cost_known ? <span className='font-medium text-amber-600 dark:text-amber-400'>{formatMoney(v, statusState?.status)}</span> : <Text type='tertiary'>-</Text>, width: 110 },
    { title: t('本站实际收入'), dataIndex: 'actual_site_revenue_usd', render: (v) => <span className='font-medium'>{formatMoney(v, statusState?.status)}</span>, width: 130 },
    { title: t('实际利润'), dataIndex: 'actual_profit_usd', render: (v, r) => r.upstream_cost_known ? <span className='font-medium text-violet-600 dark:text-violet-400'>{formatMoney(v, statusState?.status)}</span> : <Text type='tertiary'>-</Text>, width: 110 },
    { title: t('配置与实际差值'), dataIndex: 'configured_actual_delta_usd', render: (v) => <span className='font-medium'>{formatMoney(v, statusState?.status)}</span>, width: 140 },
    { title: t('本站配置来源'), dataIndex: 'site_pricing_source', render: (v, r) => <Tag color={r.site_pricing_known ? 'blue' : 'grey'} size='small'>{sitePricingLabels[v] || v || t('未知')}</Tag>, width: 130 },
  ], [sitePricingLabels, statusState?.status, t]);

  return (
    <Spin spinning={loading}>
      <div className='mt-[60px] space-y-3 px-2 pb-6'>
        <ProfitBoardHeader
          querying={querying} overviewQuerying={overviewQuerying} runFullRefresh={runFullRefresh}
          saving={saving} saveConfig={() => saveConfig(validationErrors)}
          autoRefreshMode={autoRefreshMode} setAutoRefreshMode={setAutoRefreshMode}
          statusSummary={statusSummary} hasNewActivity={hasNewActivity}
          generatedAtText={generatedAtText} combinedWarnings={combinedWarnings}
          sitePriceFactorNote={sitePriceFactorNote} walletModeEnabled={walletModeEnabled}
          selectedAccount={selectedAccount} t={t}
        />
        <Tabs type='line' size='large' className='profit-board-tabs'>
          <Tabs.TabPane tab={<span className='flex items-center gap-1.5'><BarChart3 size={16} />{t('收益分析')}</span>} itemKey='analysis'>
            <div className='mt-3 space-y-3'>
              <OverviewPanel overviewQuerying={overviewQuerying} overviewReport={overviewReport} report={report} reportMatchesCurrentFilters={reportMatchesCurrentFilters} cumulativeSummaryCards={cumulativeSummaryCards} diagnosticSummaryCards={diagnosticSummaryCards} summaryMetricHelp={summaryMetricHelp} formatMoney={formatMoney} status={statusState?.status} t={t} />
              <TimeRangePanel datePresets={createPresetRanges(t)} dateRange={dateRange} setDateRange={setDateRange} validationErrors={validationErrors} t={t} />
              <ChartAnalysisCard analysisMode={analysisMode} setAnalysisMode={setAnalysisMode} metricKey={metricKey} setMetricKey={setMetricKey} metricOptions={metricOpts} viewBatchId={viewBatchId} setViewBatchId={setViewBatchId} batchSummaryOptions={batchSummaryOptions} granularity={granularity} setGranularity={setGranularity} customIntervalMinutes={customIntervalMinutes} setCustomIntervalMinutes={setCustomIntervalMinutes} detailFilter={detailFilter} clearDetailFilter={() => { setDetailFilter(null); setDetailPage(1); }} runQuery={runQuery} querying={querying} chartTab={chartTab} setChartTab={setChartTab} report={report} chartContent={chartContent} trendRowCount={trendRows.length} trendBucketCount={trendBucketCount} t={t} />
              <DetailTableCard detailFilterText={detailFilterText} detailRows={detailRows} detailTotal={detailTotal} detailPage={detailPage} detailPageSize={detailPageSize} setDetailPage={setDetailPage} setDetailPageSize={setDetailPageSize} detailColumns={detailColumns} detailLoading={detailLoading} report={report} isMobile={isMobile} formatMoney={formatMoney} status={statusState?.status} sitePricingSourceLabelMap={sitePricingLabels} t={t} />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane tab={<span className='flex items-center gap-1.5'><CircleDollarSign size={16} />{t('配置管理')}</span>} itemKey='config'>
            <div className='mt-3 space-y-3'>
              <ComboManagerCard draft={draft} setDraft={setDraft} channelOptions={channelOptions} options={options} isMobile={isMobile} addOrUpdateBatch={addOrUpdateBatch} editingBatchId={editingBatchId} resetDraft={resetDraft} batches={batches} batchDigest={batchDigest} editBatch={editBatch} removeBatch={removeBatch} batchValidationError={duplicateBatchError} t={t} />
              <PricingRulesCard batches={batches} comboConfigs={comboConfigs} siteConfig={siteConfig} setSiteConfig={setSiteConfig} modelNameOptions={modelNameOptions} options={options} resolveSharedSitePreview={resolveSharedSitePreview} upstreamConfig={upstreamConfig} setUpstreamConfig={setUpstreamConfig} isMobile={isMobile} createDefaultComboPricingConfig={createDefaultComboPricingConfig} updateComboConfig={updateComboConfig} updateComboRule={updateComboRule} removeComboRule={removeComboRule} addComboRule={addComboRule} localModelMap={localModelMap} clampNumber={clampNumber} t={t} />
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane tab={<span className='flex items-center gap-1.5'><BadgeDollarSign size={16} />{t('上游账户')}</span>} itemKey='wallet'>
            <div className='mt-3 space-y-3'>
              <UpstreamWalletCard accounts={accountsHook.accounts} accountTrend={accountsHook.accountTrend} accountTrendLoading={accountsHook.accountTrendLoading} accountDraft={accountsHook.accountDraft} setAccountDraft={accountsHook.setAccountDraft} editingAccountId={accountsHook.editingAccountId} setEditingAccountId={accountsHook.setEditingAccountId} saveAccount={accountsHook.saveAccount} syncAccount={accountsHook.syncAccount} syncAllAccounts={accountsHook.syncAllAccounts} deleteAccount={accountsHook.deleteAccount} resetAccountDraft={accountsHook.resetAccountDraft} savingAccount={accountsHook.savingAccount} syncingAccountId={accountsHook.syncingAccountId} syncingAllAccounts={accountsHook.syncingAllAccounts} deletingAccountId={accountsHook.deletingAccountId} sideSheetVisible={accountsHook.sideSheetVisible} openCreateSideSheet={accountsHook.openCreateSideSheet} openEditSideSheet={accountsHook.openEditSideSheet} closeSideSheet={accountsHook.closeSideSheet} formatMoney={formatMoney} status={statusState?.status} t={t} />
            </div>
          </Tabs.TabPane>
        </Tabs>
      </div>
    </Spin>
  );
};

export default ProfitBoardPage;
