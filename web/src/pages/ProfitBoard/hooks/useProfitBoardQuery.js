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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { API, showError } from '../../../helpers';
import { buildQueryKey } from '../utils';

const SECTION_TIMESERIES = 'timeseries';
const SECTION_CHANNEL_BREAKDOWN = 'channel_breakdown';
const SECTION_MODEL_BREAKDOWN = 'model_breakdown';
const SECTION_WARNING_ITEMS = 'warning_items';
const DEFAULT_REPORT_SECTIONS = [
  SECTION_TIMESERIES,
  SECTION_CHANNEL_BREAKDOWN,
  SECTION_MODEL_BREAKDOWN,
  SECTION_WARNING_ITEMS,
];

const getSectionsForChartTab = (chartTab) => {
  switch (chartTab) {
    case 'channel':
      return [SECTION_CHANNEL_BREAKDOWN, SECTION_WARNING_ITEMS];
    case 'model':
      return [SECTION_MODEL_BREAKDOWN, SECTION_WARNING_ITEMS];
    case 'trend':
    default:
      return [SECTION_TIMESERIES, SECTION_WARNING_ITEMS];
  }
};

const normalizeLoadedSections = (report) => {
  if (!report) return [];
  const loaded = report?.meta?.loaded_sections;
  return Array.isArray(loaded) && loaded.length ? loaded : DEFAULT_REPORT_SECTIONS;
};

const mergeUniqueSections = (...sectionLists) =>
  Array.from(
    new Set(
      sectionLists.flatMap((items) => (Array.isArray(items) ? items : [])),
    ),
  );

const mergeReportSections = (prevReport, nextReport) => {
  if (!prevReport) return nextReport;
  const loadedSections = mergeUniqueSections(
    normalizeLoadedSections(prevReport),
    normalizeLoadedSections(nextReport),
  );
  return {
    ...prevReport,
    ...nextReport,
    meta: {
      ...(prevReport.meta || {}),
      ...(nextReport.meta || {}),
      loaded_sections: loadedSections,
    },
    timeseries:
      nextReport?.timeseries?.length || loadedSections.includes(SECTION_TIMESERIES)
        ? nextReport?.timeseries || prevReport?.timeseries || []
        : prevReport?.timeseries || [],
    channel_breakdown:
      nextReport?.channel_breakdown?.length ||
      loadedSections.includes(SECTION_CHANNEL_BREAKDOWN)
        ? nextReport?.channel_breakdown || prevReport?.channel_breakdown || []
        : prevReport?.channel_breakdown || [],
    model_breakdown:
      nextReport?.model_breakdown?.length ||
      loadedSections.includes(SECTION_MODEL_BREAKDOWN)
        ? nextReport?.model_breakdown || prevReport?.model_breakdown || []
        : prevReport?.model_breakdown || [],
    warning_items:
      nextReport?.warning_items?.length ||
      loadedSections.includes(SECTION_WARNING_ITEMS)
        ? nextReport?.warning_items || prevReport?.warning_items || []
        : prevReport?.warning_items || [],
    warnings:
      nextReport?.warnings?.length || loadedSections.includes(SECTION_WARNING_ITEMS)
        ? nextReport?.warnings || prevReport?.warnings || []
        : prevReport?.warnings || [],
  };
};

export const useProfitBoardQuery = ({
  restoredState,
  cachedBundle,
  configPayload,
  batchPayload,
  validationErrors,
  persistReportCache,
  queryReady,
  activeTab,
}) => {
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
  const [channelGroupMode, setChannelGroupMode] = useState(
    restoredState.channelGroupMode || 'channel',
  );
  const [metricKey, setMetricKey] = useState(
    restoredState.metricKey || 'configured_profit_cny',
  );
  const [analysisMode, setAnalysisMode] = useState(
    restoredState.analysisMode || 'business_compare',
  );
  const [viewBatchId, setViewBatchId] = useState(
    restoredState.viewBatchId || 'all',
  );
  const [overviewReport, setOverviewReport] = useState(null);
  const [report, setReport] = useState(cachedBundle?.report || null);
  const [lastQueryKey, setLastQueryKey] = useState(
    cachedBundle?.queryKey || restoredState.lastQueryKey || '',
  );
  const [lastOverviewKey, setLastOverviewKey] = useState('');
  const [autoRefreshMode, setAutoRefreshMode] = useState(
    restoredState.autoRefreshMode || false,
  );
  const [hasNewActivity, setHasNewActivity] = useState(false);
  const [activityChecking, setActivityChecking] = useState(false);
  const [autoRefreshing, setAutoRefreshing] = useState(false);
  const lastActivityWatermarkRef = useRef(
    cachedBundle?.activityWatermark || '',
  );
  const activeQueryKeyRef = useRef('');
  const activeOverviewKeyRef = useRef('');
  const autoRefreshTimerRef = useRef(null);
  const scheduledAutoQueryKeyRef = useRef('');
  const scheduledAutoOverviewKeyRef = useRef('');
  const overviewRequestIdRef = useRef(0);
  const queryRequestIdRef = useRef(0);

  const currentQueryKey = useMemo(
    () =>
      buildQueryKey({
        batches: batchPayload,
        shared_site: configPayload.shared_site,
        combo_configs: configPayload.combo_configs,
        excluded_user_ids: configPayload.excluded_user_ids,
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

  const overviewConfigKey = useMemo(
    () =>
      buildQueryKey({
        batches: batchPayload,
        shared_site: configPayload.shared_site,
        combo_configs: configPayload.combo_configs,
        excluded_user_ids: configPayload.excluded_user_ids,
        upstream: configPayload.upstream,
        site: configPayload.site,
      }),
    [batchPayload, configPayload],
  );

  useEffect(() => {
    activeQueryKeyRef.current = currentQueryKey;
  }, [currentQueryKey]);

  useEffect(() => {
    activeOverviewKeyRef.current = overviewConfigKey;
  }, [overviewConfigKey]);

  useEffect(
    () => () => {
      if (autoRefreshTimerRef.current) {
        window.clearTimeout(autoRefreshTimerRef.current);
      }
    },
    [],
  );

  const reportMatchesCurrentFilters =
    !!report && lastQueryKey === currentQueryKey;
  const overviewMatchesCurrentConfig =
    !!overviewReport && lastOverviewKey === overviewConfigKey;
  const loadedReportSections = useMemo(
    () => normalizeLoadedSections(report),
    [report],
  );
  const activeChartSection = useMemo(() => {
    switch (chartTab) {
      case 'channel':
        return SECTION_CHANNEL_BREAKDOWN;
      case 'model':
        return SECTION_MODEL_BREAKDOWN;
      case 'trend':
      default:
        return SECTION_TIMESERIES;
    }
  }, [chartTab]);
  const activeChartSectionLoaded = loadedReportSections.includes(
    activeChartSection,
  );

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

  const runOverviewQuery = useCallback(async (options = {}) => {
    const { expectedOverviewKey = overviewConfigKey } = options;
    if (!queryReady || validationErrors.length > 0) return false;
    const requestId = ++overviewRequestIdRef.current;
    setOverviewQuerying(true);
    try {
      const res = await API.post('/api/profit_board/overview', configPayload);
      if (!res.data.success) return showError(res.data.message);
      if (
        overviewRequestIdRef.current !== requestId ||
        activeOverviewKeyRef.current !== expectedOverviewKey
      ) {
        return false;
      }
      setOverviewReport(res.data.data);
      setLastOverviewKey(expectedOverviewKey);
      return true;
    } catch (error) {
      showError(error);
      return false;
    } finally {
      if (overviewRequestIdRef.current === requestId) {
        setOverviewQuerying(false);
      }
    }
  }, [configPayload, overviewConfigKey, queryReady, validationErrors.length]);

  const runQuery = useCallback(async (options = {}) => {
    const {
      expectedQueryKey = currentQueryKey,
      showValidationError = true,
      sections = getSectionsForChartTab(chartTab),
      merge = false,
    } = options;
    if (!queryReady || validationErrors.length > 0) {
      if (showValidationError && validationErrors.length > 0) {
        showError(validationErrors[0]);
      }
      return false;
    }
    const requestId = ++queryRequestIdRef.current;
    setQuerying(true);
    try {
      const res = await API.post('/api/profit_board/query', {
        ...queryPayload,
        sections,
      });
      if (!res.data.success) return showError(res.data.message);
      if (
        queryRequestIdRef.current !== requestId ||
        activeQueryKeyRef.current !== expectedQueryKey
      ) {
        return false;
      }
      const nextReport = merge
        ? mergeReportSections(report, res.data.data)
        : res.data.data;
      setReport(nextReport);
      setLastQueryKey(expectedQueryKey);
      setHasNewActivity(false);
      lastActivityWatermarkRef.current =
        nextReport?.meta?.activity_watermark || '';
      persistReportCache(nextReport, expectedQueryKey);
      return true;
    } catch (error) {
      showError(error);
      return false;
    } finally {
      if (queryRequestIdRef.current === requestId) {
        setQuerying(false);
      }
    }
  }, [
    currentQueryKey,
    chartTab,
    persistReportCache,
    queryPayload,
    queryReady,
    report,
    validationErrors,
  ]);

  const syncAggregate = useCallback(async () => {
    if (!queryReady || validationErrors.length > 0) return false;
    try {
      const res = await API.post('/api/profit_board/sync_aggregate', {});
      if (!res.data.success) return showError(res.data.message);
      return true;
    } catch (error) {
      showError(error);
      return false;
    }
  }, [queryReady, validationErrors.length]);

  const runFullRefresh = useCallback(
    async (options = {}) => {
      const {
        expectedQueryKey = currentQueryKey,
        expectedOverviewKey = overviewConfigKey,
        showValidationError = true,
      } = options;
      if (autoRefreshTimerRef.current) {
        window.clearTimeout(autoRefreshTimerRef.current);
        autoRefreshTimerRef.current = null;
      }
      setAutoRefreshing(false);
      if (!queryReady || validationErrors.length > 0) {
        if (showValidationError && validationErrors.length > 0) {
          showError(validationErrors[0]);
        }
        return false;
      }
      const syncOk = await syncAggregate();
      if (!syncOk) {
        return false;
      }
      const reportPromise = runQuery({
        expectedQueryKey,
        showValidationError: false,
        sections: getSectionsForChartTab(chartTab),
      });
      const overviewPromise = runOverviewQuery({ expectedOverviewKey });
      const [reportOk, overviewOk] = await Promise.all([
        reportPromise,
        overviewPromise,
      ]);
      return !!reportOk && !!overviewOk;
    },
    [
      chartTab,
      currentQueryKey,
      overviewConfigKey,
      queryReady,
      runOverviewQuery,
      runQuery,
      syncAggregate,
      validationErrors,
    ],
  );

  const checkActivity = useCallback(async () => {
    if (
      !autoRefreshMode ||
      !autoRefreshEligible ||
      activeTab !== 'analysis' ||
      validationErrors.length > 0
    )
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
    activeTab,
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

  useEffect(() => {
    if (autoRefreshTimerRef.current) {
      window.clearTimeout(autoRefreshTimerRef.current);
      autoRefreshTimerRef.current = null;
    }

    if (!queryReady || validationErrors.length > 0) {
      scheduledAutoQueryKeyRef.current = '';
      scheduledAutoOverviewKeyRef.current = '';
      setAutoRefreshing(false);
      setOverviewReport(null);
      setLastOverviewKey('');
      setReport(null);
      setLastQueryKey('');
      setHasNewActivity(false);
      lastActivityWatermarkRef.current = '';
      return undefined;
    }

    if (activeTab !== 'analysis') {
      scheduledAutoQueryKeyRef.current = '';
      scheduledAutoOverviewKeyRef.current = '';
      setAutoRefreshing(false);
      return undefined;
    }

    const needsReport = !reportMatchesCurrentFilters;
    const needsOverview = !overviewMatchesCurrentConfig;
    if (!needsReport && !needsOverview) {
      setAutoRefreshing(false);
      return undefined;
    }

    setAutoRefreshing(true);
    scheduledAutoQueryKeyRef.current = currentQueryKey;
    scheduledAutoOverviewKeyRef.current = overviewConfigKey;
    autoRefreshTimerRef.current = window.setTimeout(async () => {
      const expectedQueryKey = scheduledAutoQueryKeyRef.current;
      const expectedOverviewKey = scheduledAutoOverviewKeyRef.current;
      if (needsReport) {
        await runQuery({
          expectedQueryKey,
          showValidationError: false,
          sections: getSectionsForChartTab(chartTab),
        });
      }
      if (needsOverview) {
        await runOverviewQuery({ expectedOverviewKey });
      }
      if (
        scheduledAutoQueryKeyRef.current === expectedQueryKey &&
        scheduledAutoOverviewKeyRef.current === expectedOverviewKey
      ) {
        setAutoRefreshing(false);
      }
    }, 400);

    return () => {
      if (autoRefreshTimerRef.current) {
        window.clearTimeout(autoRefreshTimerRef.current);
        autoRefreshTimerRef.current = null;
      }
    };
  }, [
    activeTab,
    chartTab,
    currentQueryKey,
    overviewConfigKey,
    overviewMatchesCurrentConfig,
    queryReady,
    reportMatchesCurrentFilters,
    runOverviewQuery,
    runQuery,
    validationErrors.length,
  ]);

  useEffect(() => {
    if (
      activeTab !== 'analysis' ||
      !queryReady ||
      validationErrors.length > 0 ||
      !reportMatchesCurrentFilters ||
      !report ||
      activeChartSectionLoaded
    ) {
      return;
    }
    runQuery({
      expectedQueryKey: currentQueryKey,
      showValidationError: false,
      sections: getSectionsForChartTab(chartTab),
      merge: true,
    });
  }, [
    activeTab,
    activeChartSectionLoaded,
    chartTab,
    currentQueryKey,
    queryReady,
    report,
    reportMatchesCurrentFilters,
    runQuery,
    validationErrors.length,
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
    lastQueryKey,
    lastOverviewKey,
    currentQueryKey,
    overviewConfigKey,
    reportMatchesCurrentFilters,
    loadedReportSections,
    activeChartSectionLoaded,
    autoRefreshMode,
    setAutoRefreshMode,
    hasNewActivity,
    activityChecking,
    autoRefreshing,
    runOverviewQuery,
    runQuery,
    runFullRefresh,
    queryPayload,
  };
};
