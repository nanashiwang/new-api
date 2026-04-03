import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../../helpers';
import { DETAIL_LIMIT, buildQueryKey } from '../utils';

export const useProfitBoardQuery = ({
  restoredState,
  cachedBundle,
  configPayload,
  batchPayload,
  validationErrors,
  persistReportCache,
}) => {
  const { t } = useTranslation();
  const [querying, setQuerying] = useState(false);
  const [overviewQuerying, setOverviewQuerying] = useState(false);
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
  const [overviewReport, setOverviewReport] = useState(null);
  const [report, setReport] = useState(cachedBundle?.report || null);
  const [lastQueryKey, setLastQueryKey] = useState(
    cachedBundle?.queryKey || restoredState.lastQueryKey || '',
  );
  const [autoRefreshMode, setAutoRefreshMode] = useState(
    restoredState.autoRefreshMode || false,
  );
  const [hasNewActivity, setHasNewActivity] = useState(false);
  const [activityChecking, setActivityChecking] = useState(false);
  const [detailRows, setDetailRows] = useState([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailPage, setDetailPage] = useState(restoredState.detailPage || 1);
  const [detailPageSize, setDetailPageSize] = useState(
    restoredState.detailPageSize || 12,
  );
  const [detailTotal, setDetailTotal] = useState(0);
  const lastActivityWatermarkRef = useRef(
    cachedBundle?.activityWatermark || '',
  );

  const currentQueryKey = useMemo(
    () =>
      buildQueryKey({
        batches: batchPayload,
        shared_site: configPayload.shared_site,
        combo_configs: configPayload.combo_configs,
        upstream: configPayload.upstream,
        site: configPayload.site,
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
    [batchPayload, configPayload, customIntervalMinutes, dateRange, granularity],
  );

  const reportMatchesCurrentFilters =
    !!report && lastQueryKey === currentQueryKey;

  const autoRefreshEligible = useMemo(
    () =>
      !!dateRange?.[1] &&
      Math.abs(Date.now() - new Date(dateRange[1]).getTime()) <=
        15 * 60 * 1000,
    [dateRange],
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
      persistReportCache(nextReport, currentQueryKey);
    } catch (error) {
      showError(error);
    } finally {
      setQuerying(false);
    }
  }, [currentQueryKey, persistReportCache, queryPayload, validationErrors]);

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
    } catch {
      // silently ignore
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

  return {
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
    lastQueryKey,
    currentQueryKey,
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
    runOverviewQuery,
    runQuery,
    runFullRefresh,
    loadDetailPage,
    queryPayload,
  };
};
