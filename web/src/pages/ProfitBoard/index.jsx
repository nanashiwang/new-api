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
import dayjs from 'dayjs';
import {
  Banner,
  Button,
  Card,
  DatePicker,
  Empty,
  Input,
  InputNumber,
  Radio,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import {
  ArrowDownToLine,
  BadgeDollarSign,
  BarChart3,
  CircleDollarSign,
  Filter,
  Info,
  Layers3,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Trash2,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import {
  API,
  copy,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const STORAGE_KEY = 'profit-board:state';
const REPORT_CACHE_KEY = 'profit-board:report';
const DETAIL_LIMIT = 600;
const { Text, Title, Paragraph } = Typography;

const metricOptions = [
  { value: 'configured_profit_usd', label: '配置利润' },
  { value: 'actual_profit_usd', label: '实际利润' },
  { value: 'actual_site_revenue_usd', label: '本站实际收入' },
  { value: 'configured_site_revenue_usd', label: '本站配置收入' },
  { value: 'upstream_cost_usd', label: '上游费用' },
];

const sitePricingSourceLabelMap = {
  manual: '手动价格',
  manual_fallback: '手动价格回退',
  site_model_standard: '读取本站模型原价',
  site_model_recharge: '读取本站模型充值价',
  site_model_missing: '未命中本站模型',
};

const createBatchId = () =>
  `batch-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
const createDefaultUpstreamConfig = () => ({
  cost_source: 'manual_only',
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  fixed_amount: 0,
  fixed_total_amount: 0,
});
const createDefaultSiteConfig = () => ({
  pricing_mode: 'manual',
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  fixed_amount: 0,
  fixed_total_amount: 0,
  model_names: [],
  group: '',
  use_recharge_price: false,
});
const createDefaultDraft = () => ({
  id: '',
  name: '',
  scope_type: 'channel',
  channel_ids: [],
  tags: [],
});

const createDefaultState = () => {
  const end = new Date();
  const start = dayjs(end).subtract(7, 'day').toDate();
  return {
    batches: [],
    draft: createDefaultDraft(),
    editingBatchId: '',
    dateRange: [start, end],
    granularity: 'day',
    customIntervalMinutes: 15,
    chartTab: 'trend',
    compareMode: 'none',
    comparePeriod: 'previous',
    compareDateRange: [],
    metricKey: 'configured_profit_usd',
    analysisMode: 'business_compare',
    viewBatchId: 'all',
    detailFilter: null,
    upstreamConfig: createDefaultUpstreamConfig(),
    siteConfig: createDefaultSiteConfig(),
    lastQueryKey: '',
  };
};

const clampNumber = (value) => {
  const next = Number(value || 0);
  if (!Number.isFinite(next) || next < 0) return 0;
  return next;
};

const safeParse = (raw, fallback) => {
  try {
    return raw ? JSON.parse(raw) : fallback;
  } catch (error) {
    return fallback;
  }
};

const normalizeBatchForState = (batch, index) => ({
  id: batch?.id || createBatchId(),
  name: batch?.name || `批次 ${index + 1}`,
  scope_type: batch?.scope_type || 'channel',
  channel_ids: (batch?.channel_ids || []).map((item) => item.toString()),
  tags: batch?.tags || [],
});

const normalizeRestoredState = (state) => {
  const defaults = createDefaultState();
  const next = { ...defaults, ...(state || {}) };
  const legacyHasSelection =
    !next.batches?.length &&
    ((next.scopeType === 'channel' &&
      (next.selectedChannels || []).length > 0) ||
      (next.scopeType === 'tag' && (next.selectedTags || []).length > 0));

  if (legacyHasSelection) {
    next.batches = [
      normalizeBatchForState(
        {
          id: createBatchId(),
          name: '批次 1',
          scope_type: next.scopeType || 'channel',
          channel_ids: next.selectedChannels || [],
          tags: next.selectedTags || [],
        },
        0,
      ),
    ];
  } else {
    next.batches = (next.batches || []).map((item, index) =>
      normalizeBatchForState(item, index),
    );
  }

  const [start, end] = next.dateRange || [];
  next.dateRange = [
    start ? new Date(start) : defaults.dateRange[0],
    end ? new Date(end) : defaults.dateRange[1],
  ];
  const [compareStart, compareEnd] = next.compareDateRange || [];
  next.compareDateRange = [
    compareStart ? new Date(compareStart) : null,
    compareEnd ? new Date(compareEnd) : null,
  ].filter(Boolean);
  next.draft = normalizeBatchForState(next.draft || {}, 0);
  next.editingBatchId = next.editingBatchId || '';
  next.customIntervalMinutes = Math.max(
    Number(next.customIntervalMinutes || defaults.customIntervalMinutes),
    1,
  );
  next.compareMode = next.compareMode || 'none';
  next.comparePeriod = next.comparePeriod || 'previous';
  next.upstreamConfig = {
    ...createDefaultUpstreamConfig(),
    ...(next.upstreamConfig || {}),
    cost_source: 'manual_only',
    fixed_amount: 0,
  };
  next.siteConfig = {
    ...createDefaultSiteConfig(),
    ...(next.siteConfig || {}),
    model_names: next.siteConfig?.model_names || [],
    fixed_amount: 0,
  };
  next.viewBatchId = next.viewBatchId || 'all';
  next.lastQueryKey = next.lastQueryKey || '';
  next.analysisMode = next.analysisMode || 'business_compare';
  return next;
};

const getDisplayCurrency = (status) => {
  const displayType = status?.quota_display_type || 'USD';
  if (displayType === 'CNY')
    return { symbol: '¥', rate: status?.usd_exchange_rate || 1 };
  if (displayType === 'CUSTOM') {
    return {
      symbol: status?.custom_currency_symbol || '¤',
      rate: status?.custom_currency_exchange_rate || 1,
    };
  }
  return { symbol: '$', rate: 1 };
};

const formatMoney = (value, status, digits = 3) => {
  const amount = Number(value || 0);
  const { symbol, rate } = getDisplayCurrency(status);
  return `${symbol}${(amount * rate).toFixed(digits)}`;
};

const createPresetRanges = () => {
  const now = dayjs();
  return [
    { label: '今天', value: [now.startOf('day').toDate(), now.endOf('day').toDate()] },
    { label: '最近 24 小时', value: [now.subtract(24, 'hour').toDate(), now.toDate()] },
    { label: '近 7 天', value: [now.subtract(7, 'day').toDate(), now.toDate()] },
    { label: '近 30 天', value: [now.subtract(30, 'day').toDate(), now.toDate()] },
    { label: '本月', value: [now.startOf('month').toDate(), now.endOf('month').toDate()] },
    { label: '上月', value: [now.subtract(1, 'month').startOf('month').toDate(), now.subtract(1, 'month').endOf('month').toDate()] },
  ];
};

const formatRangeLabel = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return '-';
  return `${dayjs(range[0]).format('YYYY-MM-DD HH:mm')} ~ ${dayjs(range[1]).format('YYYY-MM-DD HH:mm')}`;
};

const formatRangeDuration = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return '-';
  const minutes = dayjs(range[1]).diff(dayjs(range[0]), 'minute');
  if (minutes < 60) return `${minutes} 分钟`;
  const hours = (minutes / 60).toFixed(hoursToFixed(minutes));
  if (minutes < 1440) return `${hours} 小时`;
  return `${(minutes / 1440).toFixed(1)} 天`;
};

const hoursToFixed = (minutes) => (minutes % 60 === 0 ? 0 : 1);

const downloadBlob = (blob, filename) => {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  window.URL.revokeObjectURL(url);
};
const buildQueryKey = (payload) => JSON.stringify(payload);

const normalizeCachedReportBundle = (raw) => {
  if (!raw) return null;
  if (raw.report) {
    return {
      report: raw.report,
      queryKey: raw.queryKey || '',
      activityWatermark:
        raw.activityWatermark || raw.report?.meta?.activity_watermark || '',
      generatedAt: raw.generatedAt || raw.report?.meta?.generated_at || 0,
    };
  }
  return {
    report: raw,
    queryKey: '',
    activityWatermark: raw?.meta?.activity_watermark || '',
    generatedAt: raw?.meta?.generated_at || 0,
  };
};

const formatRatio = (value) => `${(Number(value || 0) * 100).toFixed(1)}%`;

const formatBucketLabel = (timestamp, granularity, customIntervalMinutes) => {
  const current = dayjs.unix(timestamp);
  if (granularity === 'hour') return current.format('YYYY-MM-DD HH:00');
  if (granularity === 'week') return current.startOf('week').add(1, 'day').format('GGGG-[W]WW');
  if (granularity === 'month') return current.startOf('month').format('YYYY-MM');
  if (granularity === 'custom') {
    const interval = Math.max(Number(customIntervalMinutes || 1), 1);
    const totalMinutes = current.hour() * 60 + current.minute();
    const alignedMinutes = Math.floor(totalMinutes / interval) * interval;
    return current
      .startOf('day')
      .add(alignedMinutes, 'minute')
      .format('YYYY-MM-DD HH:mm');
  }
  return current.format('YYYY-MM-DD');
};

const buildPreviousPeriodRange = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return [];
  const start = dayjs(range[0]);
  const end = dayjs(range[1]);
  const duration = Math.max(end.diff(start, 'second'), 0);
  const previousEnd = start.subtract(1, 'second');
  const previousStart = previousEnd.subtract(duration, 'second');
  return [previousStart.toDate(), previousEnd.toDate()];
};

const aggregateBreakdownRows = (rows, viewBatchId, metricKey) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();
  filtered.forEach((item) => {
    const key = item.label || item.key;
    const current = grouped.get(key) || { label: key, value: 0 };
    current.value += Number(item[metricKey] || 0);
    grouped.set(key, current);
  });
  return Array.from(grouped.values())
    .sort((a, b) => b.value - a.value)
    .slice(0, 12);
};

const combineTimeseriesMetrics = (rows, viewBatchId, metrics) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();
  filtered.forEach((item) => {
    metrics.forEach((metric) => {
      const key = `${item.bucket}::${metric.key}`;
      const current = grouped.get(key) || {
        bucket: item.bucket,
        value: 0,
        series: metric.label,
      };
      current.value += Number(item[metric.key] || 0);
      grouped.set(key, current);
    });
  });
  return Array.from(grouped.values()).sort((a, b) => {
    if (a.bucket === b.bucket) return a.series.localeCompare(b.series);
    return a.bucket.localeCompare(b.bucket);
  });
};

const combineBreakdownMetrics = (rows, viewBatchId, metrics) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();
  filtered.forEach((item) => {
    const label = item.label || item.key;
    metrics.forEach((metric) => {
      const key = `${label}::${metric.key}`;
      const current = grouped.get(key) || {
        label,
        value: 0,
        series: metric.label,
      };
      current.value += Number(item[metric.key] || 0);
      grouped.set(key, current);
    });
  });
  return Array.from(grouped.values())
    .sort((a, b) => {
      if (a.label === b.label) return a.series.localeCompare(b.series);
      return b.value - a.value;
    })
    .slice(0, 24);
};

const createTrendSpec = (rows, metricLabel, status, t) => ({
  type: 'line',
  background: 'transparent',
  data: [{ id: 'trend', values: rows }],
  xField: 'bucket',
  yField: 'value',
  seriesField: rows.some((item) => item.series) ? 'series' : undefined,
  legends: { visible: rows.some((item) => item.series) },
  point: { visible: true, style: { size: 5 } },
  line: { style: { curveType: 'monotone', lineWidth: 2 } },
  axes: [
    {
      orient: 'bottom',
      type: 'band',
      label: { visible: true, style: { angle: -18 } },
    },
    { orient: 'left', nice: true },
  ],
  title: { visible: true, text: t('收益趋势'), subtext: metricLabel },
  tooltip: {
    mark: {
      content: [
        { key: t('时间桶'), value: (datum) => datum.bucket },
        ...(rows.some((item) => item.series)
          ? [{ key: t('批次'), value: (datum) => datum.series }]
          : []),
        {
          key: metricLabel,
          value: (datum) => formatMoney(datum.value, status),
        },
      ],
    },
  },
});

const createBarSpec = (title, rows, metricLabel, status, t) => ({
  type: 'bar',
  background: 'transparent',
  data: [{ id: 'bar', values: rows }],
  xField: 'label',
  yField: 'value',
  seriesField: rows.some((item) => item.series) ? 'series' : undefined,
  legends: { visible: rows.some((item) => item.series) },
  axes: [
    {
      orient: 'bottom',
      type: 'band',
      label: { visible: true, style: { angle: -20 } },
    },
    { orient: 'left', nice: true },
  ],
  title: { visible: true, text: title, subtext: metricLabel },
  tooltip: {
    mark: {
      content: [
        { key: t('名称'), value: (datum) => datum.label },
        ...(rows.some((item) => item.series)
          ? [{ key: t('对比项'), value: (datum) => datum.series }]
          : []),
        {
          key: metricLabel,
          value: (datum) => formatMoney(datum.value, status),
        },
      ],
    },
  },
});

const ProfitBoardPage = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const isMobile = useIsMobile();
  const cachedReportBundle = useMemo(
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
  const [exporting, setExporting] = useState(false);
  const [options, setOptions] = useState({
    channels: [],
    tags: [],
    groups: [],
    local_models: [],
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
  const [compareMode, setCompareMode] = useState(
    restoredState.compareMode || 'none',
  );
  const [comparePeriod, setComparePeriod] = useState(
    restoredState.comparePeriod || 'previous',
  );
  const [compareDateRange, setCompareDateRange] = useState(
    restoredState.compareDateRange || [],
  );
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
  const [upstreamConfig, setUpstreamConfig] = useState(
    restoredState.upstreamConfig || createDefaultUpstreamConfig(),
  );
  const [siteConfig, setSiteConfig] = useState(
    restoredState.siteConfig || createDefaultSiteConfig(),
  );
  const [overviewReport, setOverviewReport] = useState(null);
  const [report, setReport] = useState(cachedReportBundle?.report || null);
  const [compareReport, setCompareReport] = useState(null);
  const [lastQueryKey, setLastQueryKey] = useState(
    restoredState.lastQueryKey || cachedReportBundle?.queryKey || '',
  );
  const [reportLoadedFromCache, setReportLoadedFromCache] = useState(
    !!cachedReportBundle?.report,
  );
  const [activityChecking, setActivityChecking] = useState(false);
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true);
  const lastActivityWatermarkRef = useRef(
    cachedReportBundle?.activityWatermark || '',
  );
  const autoLoadedBatchKeyRef = useRef('');

  useEffect(() => {
    initVChartSemiTheme({ isWatchingThemeSwitch: true });
  }, []);

  const channelOptions = useMemo(
    () =>
      (options.channels || []).map((item) => ({
        label: item.name,
        value: item.id.toString(),
      })),
    [options.channels],
  );
  const channelMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((item) =>
      map.set(item.id.toString(), item),
    );
    return map;
  }, [options.channels]);
  const tagChannelMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((item) => {
      const tag = item.tag || '';
      if (!tag) return;
      if (!map.has(tag)) map.set(tag, []);
      map.get(tag).push(item.id.toString());
    });
    return map;
  }, [options.channels]);

  const batchPayload = useMemo(
    () =>
      batches.map((batch) => ({
        id: batch.id,
        name: batch.name,
        scope_type: batch.scope_type,
        channel_ids:
          batch.scope_type === 'channel'
            ? (batch.channel_ids || []).map((item) => Number(item))
            : [],
        tags: batch.scope_type === 'tag' ? batch.tags || [] : [],
      })),
    [batches],
  );
  const configLookupKey = useMemo(
    () =>
      JSON.stringify(
        batchPayload.map((batch) => ({
          scope_type: batch.scope_type,
          channel_ids: batch.channel_ids,
          tags: batch.tags,
        })),
      ),
    [batchPayload],
  );
  const currentBatchSnapshotKey = useMemo(
    () =>
      JSON.stringify(
        batches.map((batch) => ({
          id: batch.id,
          name: batch.name,
          scope_type: batch.scope_type,
          channel_ids: batch.channel_ids || [],
          tags: batch.tags || [],
        })),
      ),
    [batches],
  );

  const metricLabel = useMemo(
    () =>
      metricOptions.find((item) => item.value === metricKey)?.label ||
      '配置利润',
    [metricKey],
  );
  const duplicateBatchError = useMemo(() => {
    const ownerMap = new Map();
    for (const batch of batches) {
      const channelIds =
        batch.scope_type === 'channel'
          ? batch.channel_ids || []
          : (batch.tags || []).flatMap((tag) => tagChannelMap.get(tag) || []);
      const uniqueIds = Array.from(new Set(channelIds));
      for (const channelId of uniqueIds) {
        const owner = ownerMap.get(channelId);
        if (owner) {
          const channelName =
            channelMap.get(channelId)?.name || `渠道 #${channelId}`;
          return `${channelName} 同时出现在批次“${owner}”和“${batch.name}”中，请拆开后再统计`;
        }
        ownerMap.set(channelId, batch.name);
      }
    }
    return '';
  }, [batches, channelMap, tagChannelMap]);

  const batchValidationError = useMemo(() => {
    if ((batches || []).length === 0) return t('请至少添加一个批次');
    if (duplicateBatchError) return duplicateBatchError;
    return '';
  }, [batches, duplicateBatchError, t]);

  const validationErrors = useMemo(() => {
    const errors = [];
    if (batchValidationError) errors.push(batchValidationError);
    if (!Array.isArray(dateRange) || !dateRange[0] || !dateRange[1])
      errors.push(t('请选择完整的时间分析时间范围'));
    if (granularity === 'custom' && Number(customIntervalMinutes || 0) <= 0) {
      errors.push(t('自定义时间粒度必须大于 0 分钟'));
    }
    if (
      siteConfig.pricing_mode === 'site_model' &&
      (siteConfig.model_names || []).length === 0
    )
      errors.push(t('读取本站模型价格时，至少选择一个模型'));
    if (
      compareMode === 'time' &&
      comparePeriod === 'custom' &&
      (!Array.isArray(compareDateRange) ||
        !compareDateRange[0] ||
        !compareDateRange[1])
    ) {
      errors.push(t('请选择完整的对比时间范围'));
    }
    return errors;
  }, [
    batches,
    compareDateRange,
    compareMode,
    comparePeriod,
    customIntervalMinutes,
    dateRange,
    granularity,
    siteConfig,
    batchValidationError,
    t,
  ]);

  const persistState = useCallback(
    (next = {}) => {
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
          compareMode,
          comparePeriod,
          compareDateRange,
          metricKey,
          analysisMode,
          viewBatchId,
          detailFilter,
          upstreamConfig,
          siteConfig,
          lastQueryKey,
          ...next,
        }),
      );
    },
    [
      batches,
      chartTab,
      compareDateRange,
      compareMode,
      comparePeriod,
      customIntervalMinutes,
      dateRange,
      detailFilter,
      draft,
      editingBatchId,
      granularity,
      metricKey,
      analysisMode,
      siteConfig,
      upstreamConfig,
      viewBatchId,
      lastQueryKey,
    ],
  );

  const resetDraft = useCallback(() => {
    setDraft(createDefaultDraft());
    setEditingBatchId('');
  }, []);

  const buildQueryPayload = useCallback(
    () => ({
      batches: batchPayload,
      upstream: {
        ...upstreamConfig,
        input_price: clampNumber(upstreamConfig.input_price),
        output_price: clampNumber(upstreamConfig.output_price),
        cache_read_price: clampNumber(upstreamConfig.cache_read_price),
        cache_creation_price: clampNumber(upstreamConfig.cache_creation_price),
        fixed_amount: clampNumber(upstreamConfig.fixed_amount),
        fixed_total_amount: clampNumber(upstreamConfig.fixed_total_amount),
      },
      site: {
        ...siteConfig,
        input_price: clampNumber(siteConfig.input_price),
        output_price: clampNumber(siteConfig.output_price),
        cache_read_price: clampNumber(siteConfig.cache_read_price),
        cache_creation_price: clampNumber(siteConfig.cache_creation_price),
        fixed_amount: clampNumber(siteConfig.fixed_amount),
        fixed_total_amount: clampNumber(siteConfig.fixed_total_amount),
      },
      start_timestamp: dayjs(dateRange?.[0]).unix(),
      end_timestamp: dayjs(dateRange?.[1]).unix(),
      granularity,
      custom_interval_minutes:
        granularity === 'custom' ? Number(customIntervalMinutes || 0) : 0,
      detail_limit: DETAIL_LIMIT,
    }),
    [
      batchPayload,
      customIntervalMinutes,
      dateRange,
      granularity,
      siteConfig,
      upstreamConfig,
    ],
  );
  const buildOverviewPayload = useCallback(() => {
    const payload = buildQueryPayload();
    return {
      batches: payload.batches,
      upstream: payload.upstream,
      site: payload.site,
    };
  }, [buildQueryPayload]);

  const buildActivityPayload = useCallback(() => {
    const payload = buildQueryPayload();
    return {
      batches: payload.batches,
      upstream: payload.upstream,
      site: payload.site,
      start_timestamp: payload.start_timestamp,
      end_timestamp: payload.end_timestamp,
      granularity: payload.granularity,
      custom_interval_minutes: payload.custom_interval_minutes,
    };
  }, [buildQueryPayload]);
  const resolvedCompareDateRange = useMemo(() => {
    if (compareMode !== 'time') return [];
    if (comparePeriod === 'custom') return compareDateRange || [];
    return buildPreviousPeriodRange(dateRange);
  }, [compareDateRange, compareMode, comparePeriod, dateRange]);
  const compareQueryPayload = useMemo(() => {
    if (
      compareMode !== 'time' ||
      !resolvedCompareDateRange?.[0] ||
      !resolvedCompareDateRange?.[1]
    ) {
      return null;
    }
    const payload = buildQueryPayload();
    return {
      ...payload,
      start_timestamp: dayjs(resolvedCompareDateRange[0]).unix(),
      end_timestamp: dayjs(resolvedCompareDateRange[1]).unix(),
    };
  }, [buildQueryPayload, compareMode, resolvedCompareDateRange]);
  const currentQueryKey = useMemo(
    () => buildQueryKey(buildQueryPayload()),
    [buildQueryPayload],
  );
  const compareQueryKey = useMemo(
    () => (compareQueryPayload ? buildQueryKey(compareQueryPayload) : ''),
    [compareQueryPayload],
  );
  const reportMatchesCurrentFilters =
    !!report && !!lastQueryKey && currentQueryKey === lastQueryKey;
  const reportHasPendingChanges =
    !!report && !!lastQueryKey && currentQueryKey !== lastQueryKey;
  const reportIsFresh =
    !!report && reportMatchesCurrentFilters && !reportLoadedFromCache;
  const autoRefreshEligible = useMemo(() => {
    const end = dateRange?.[1];
    if (!end) return false;
    return dayjs(end).isAfter(dayjs().subtract(15, 'minute'));
  }, [dateRange]);

  const loadOptions = useCallback(async () => {
    const res = await API.get('/api/profit_board/options');
    if (!res.data.success) throw new Error(res.data.message);
    setOptions(res.data.data || {});
  }, []);

  const loadConfig = useCallback(async () => {
    if (!configLookupKey || configLookupKey === '[]') return;
    const res = await API.get('/api/profit_board/config', {
      params: { batches: configLookupKey },
    });
    if (!res.data.success) throw new Error(res.data.message);
    const config = res.data.data?.config;
    if (!config) return;
    if (Array.isArray(config.batches) && config.batches.length > 0) {
      const nextBatches = config.batches.map((item, index) =>
        normalizeBatchForState(item, index),
      );
      const nextBatchSnapshotKey = JSON.stringify(
        nextBatches.map((item) => ({
          id: item.id,
          name: item.name,
          scope_type: item.scope_type,
          channel_ids: item.channel_ids || [],
          tags: item.tags || [],
        })),
      );
      if (nextBatchSnapshotKey !== currentBatchSnapshotKey) {
        setBatches(nextBatches);
      }
    }
    setUpstreamConfig({
      ...createDefaultUpstreamConfig(),
      ...(config.upstream || {}),
      cost_source: 'manual_only',
      fixed_amount: 0,
    });
    setSiteConfig({
      ...createDefaultSiteConfig(),
      ...(config.site || {}),
      model_names: config.site?.model_names || [],
      fixed_amount: 0,
    });
    autoLoadedBatchKeyRef.current = '';
  }, [configLookupKey, currentBatchSnapshotKey]);

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
    persistState();
  }, [persistState]);

  useEffect(() => {
    if (!batchPayload.length) return;
    loadConfig().catch(showError);
  }, [configLookupKey, batchPayload.length, loadConfig]);

  useEffect(() => {
    const validBatchIds = new Set(
      [
        ...(overviewReport?.batch_summaries || []),
        ...(report?.batch_summaries || []),
      ].map((item) => item.batch_id),
    );
    if (viewBatchId !== 'all' && !validBatchIds.has(viewBatchId)) {
      setViewBatchId('all');
    }
    setDetailFilter((prev) => {
      if (!prev?.value) return prev;
      const matched = (report?.detail_rows || []).some((row) => {
        if (prev.batchId && row.batch_id !== prev.batchId) return false;
        if (prev.type === 'channel') return row.channel_name === prev.value;
        if (prev.type === 'model') return row.model_name === prev.value;
        if (prev.type === 'trend') {
          return (
            formatBucketLabel(
              row.created_at,
              granularity,
              customIntervalMinutes,
            ) === prev.value
          );
        }
        return false;
      });
      return matched ? prev : null;
    });
  }, [customIntervalMinutes, granularity, overviewReport, report, viewBatchId]);

  useEffect(() => {
    if (analysisMode === 'business_compare' && compareMode === 'batch') {
      setCompareMode('none');
    }
  }, [analysisMode, compareMode]);

  const configValidationMessage = useMemo(() => {
    if (batchValidationError) return batchValidationError;
    if (
      siteConfig.pricing_mode === 'site_model' &&
      (siteConfig.model_names || []).length === 0
    ) {
      return t('读取本站模型价格时，至少选择一个模型');
    }
    return '';
  }, [batchValidationError, siteConfig, t]);

  const batchSummaryOptions = useMemo(
    () => [
      { label: t('全部批次'), value: 'all' },
      ...(
        overviewReport?.batch_summaries ||
        report?.batch_summaries ||
        []
      ).map((item) => ({
        label: item.batch_name,
        value: item.batch_id,
      })),
    ],
    [overviewReport, report, t],
  );

  const addOrUpdateBatch = useCallback(() => {
    const name =
      draft.name?.trim() || `批次 ${batches.length + (editingBatchId ? 0 : 1)}`;
    if (
      draft.scope_type === 'channel' &&
      (draft.channel_ids || []).length === 0
    ) {
      return showError(t('请至少选择一个渠道'));
    }
    if (draft.scope_type === 'tag' && (draft.tags || []).length === 0) {
      return showError(t('请至少选择一个标签'));
    }
    const nextBatch = {
      id: editingBatchId || createBatchId(),
      name,
      scope_type: draft.scope_type,
      channel_ids:
        draft.scope_type === 'channel'
          ? Array.from(new Set(draft.channel_ids || []))
          : [],
      tags:
        draft.scope_type === 'tag' ? Array.from(new Set(draft.tags || [])) : [],
    };
    if (editingBatchId) {
      setBatches((prev) =>
        prev.map((item) => (item.id === editingBatchId ? nextBatch : item)),
      );
      showSuccess(t('批次已更新'));
    } else {
      setBatches((prev) => [...prev, nextBatch]);
      showSuccess(t('批次已添加'));
    }
    resetDraft();
  }, [batches.length, draft, editingBatchId, resetDraft, t]);

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

  const removeBatch = useCallback(
    (batchId) => {
      setBatches((prev) => prev.filter((item) => item.id !== batchId));
      if (editingBatchId === batchId) resetDraft();
    },
    [editingBatchId, resetDraft],
  );

  const fetchCompareReport = useCallback(async (payload, silent = true) => {
    if (!payload) {
      setCompareReport(null);
      return null;
    }
    try {
      const res = await API.post('/api/profit_board/query', payload);
      if (!res.data.success) throw new Error(res.data.message);
      setCompareReport(res.data.data);
      return res.data.data;
    } catch (error) {
      setCompareReport(null);
      if (!silent) showError(error);
      return null;
    }
  }, []);

  const runOverviewQuery = useCallback(
    async (silent = false) => {
      if (configValidationMessage) {
        if (!silent) showError(configValidationMessage);
        return null;
      }
      setOverviewQuerying(true);
      try {
        const res = await API.post(
          '/api/profit_board/overview',
          buildOverviewPayload(),
        );
        if (!res.data.success) throw new Error(res.data.message);
        setOverviewReport(res.data.data);
        if (!silent) showSuccess(t('累计总览已更新'));
        return res.data.data;
      } catch (error) {
        showError(error);
        return null;
      } finally {
        setOverviewQuerying(false);
      }
    },
    [buildOverviewPayload, configValidationMessage, t],
  );

  const runQuery = useCallback(
    async (silent = false) => {
      if (validationErrors.length > 0) {
        if (!silent) showError(validationErrors[0]);
        return null;
      }
      setQuerying(true);
      try {
        const payload = buildQueryPayload();
        const queryKey = buildQueryKey(payload);
        const res = await API.post('/api/profit_board/query', payload);
        if (!res.data.success) throw new Error(res.data.message);
        setReport(res.data.data);
        setLastQueryKey(queryKey);
        setReportLoadedFromCache(false);
        lastActivityWatermarkRef.current =
          res.data.data?.meta?.activity_watermark || '';
        localStorage.setItem(
          REPORT_CACHE_KEY,
          JSON.stringify({
            report: res.data.data,
            queryKey,
            activityWatermark: res.data.data?.meta?.activity_watermark || '',
            generatedAt: res.data.data?.meta?.generated_at || 0,
          }),
        );
        if (compareMode === 'time' && compareQueryPayload) {
          await fetchCompareReport(compareQueryPayload);
        } else {
          setCompareReport(null);
        }
        if (!silent) showSuccess(t('时间分析已更新'));
        return res.data.data;
      } catch (error) {
        showError(error);
        return null;
      } finally {
        setQuerying(false);
      }
    },
    [
      buildQueryPayload,
      compareMode,
      compareQueryPayload,
      fetchCompareReport,
      t,
      validationErrors,
    ],
  );

  const runFullRefresh = useCallback(async () => {
    const [nextOverview, nextReport] = await Promise.all([
      runOverviewQuery(true),
      runQuery(true),
    ]);
    if (nextOverview && nextReport) {
      showSuccess(t('收益看板已更新'));
      return;
    }
    if (nextOverview && !nextReport) {
      showSuccess(t('累计总览已更新，时间分析暂未刷新'));
      if (validationErrors.length > 0) showError(validationErrors[0]);
      return;
    }
    if (configValidationMessage) {
      showError(configValidationMessage);
      return;
    }
    if (validationErrors.length > 0) {
      showError(validationErrors[0]);
    }
  }, [
    configValidationMessage,
    runOverviewQuery,
    runQuery,
    t,
    validationErrors,
  ]);

  useEffect(() => {
    if (!batchPayload.length || validationErrors.length > 0) return;
    if (autoLoadedBatchKeyRef.current === currentBatchSnapshotKey) return;
    autoLoadedBatchKeyRef.current = currentBatchSnapshotKey;
    runOverviewQuery(true);
    runQuery(true);
  }, [
    batchPayload.length,
    currentBatchSnapshotKey,
    runOverviewQuery,
    runQuery,
    validationErrors.length,
  ]);

  useEffect(() => {
    if (!report || validationErrors.length > 0) return;
    runQuery(true);
  }, [customIntervalMinutes, granularity]);

  useEffect(() => {
    if (compareMode !== 'time') {
      setCompareReport(null);
      return;
    }
    if (!reportMatchesCurrentFilters || !compareQueryPayload) return;
    fetchCompareReport(compareQueryPayload);
  }, [
    compareMode,
    comparePeriod,
    compareQueryKey,
    compareQueryPayload,
    fetchCompareReport,
    reportMatchesCurrentFilters,
  ]);

  useEffect(() => {
    if (reportHasPendingChanges) {
      setCompareReport(null);
    }
  }, [reportHasPendingChanges]);

  const checkActivity = useCallback(
    async (notifyOnError = false) => {
      if (!reportMatchesCurrentFilters) return null;
      setActivityChecking(true);
      try {
        const res = await API.post('/api/profit_board/activity', buildActivityPayload());
        if (!res.data.success) throw new Error(res.data.message);
        const activity = res.data.data;
        if (
          activity?.activity_watermark &&
          activity.activity_watermark !== lastActivityWatermarkRef.current
        ) {
          await Promise.all([runOverviewQuery(true), runQuery(true)]);
        } else if (activity?.activity_watermark) {
          lastActivityWatermarkRef.current = activity.activity_watermark;
        }
        return activity;
      } catch (error) {
        if (notifyOnError) showError(error);
        return null;
      } finally {
        setActivityChecking(false);
      }
    },
    [buildActivityPayload, reportMatchesCurrentFilters, runOverviewQuery, runQuery],
  );

  useEffect(() => {
    if (!report || !reportMatchesCurrentFilters) return;
    if (!reportLoadedFromCache) return;
    checkActivity();
  }, [checkActivity, report, reportLoadedFromCache, reportMatchesCurrentFilters]);

  useEffect(() => {
    const handleVisibility = () => {
      setAutoRefreshEnabled(document.visibilityState === 'visible');
    };
    handleVisibility();
    document.addEventListener('visibilitychange', handleVisibility);
    return () =>
      document.removeEventListener('visibilitychange', handleVisibility);
  }, []);

  useEffect(() => {
    if (
      !report ||
      !reportMatchesCurrentFilters ||
      !autoRefreshEligible ||
      !autoRefreshEnabled
    ) {
      return undefined;
    }
    const timer = window.setInterval(() => {
      checkActivity();
    }, 25000);
    return () => window.clearInterval(timer);
  }, [
    autoRefreshEligible,
    autoRefreshEnabled,
    checkActivity,
    report,
    reportMatchesCurrentFilters,
  ]);

  const saveConfig = useCallback(async () => {
    if (configValidationMessage) return showError(configValidationMessage);
    setSaving(true);
    try {
      const res = await API.put(
        '/api/profit_board/config',
        buildOverviewPayload(),
      );
      if (!res.data.success) throw new Error(res.data.message);
      showSuccess(t('长期配置已保存'));
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  }, [buildOverviewPayload, configValidationMessage, t]);

  const ensureFreshReport = useCallback(async () => {
    if (!report || !reportMatchesCurrentFilters || reportLoadedFromCache) {
      return !!(await runQuery(true));
    }
    await checkActivity(true);
    return true;
  }, [
    checkActivity,
    report,
    reportLoadedFromCache,
    reportMatchesCurrentFilters,
    runQuery,
  ]);

  const exportCSV = useCallback(async () => {
    setExporting(true);
    try {
      const ready = await ensureFreshReport();
      if (!ready) return;
      const res = await API.post(
        '/api/profit_board/export/csv',
        buildQueryPayload(),
        { responseType: 'blob' },
      );
      const disposition = res.headers?.['content-disposition'] || '';
      const matched = disposition.match(/filename=\"(.+)\"/);
      downloadBlob(
        new Blob([res.data], { type: 'text/csv;charset=utf-8' }),
        matched?.[1] || 'profit-board.csv',
      );
    } catch (error) {
      showError(error);
    } finally {
      setExporting(false);
    }
  }, [buildQueryPayload, ensureFreshReport]);

  const exportExcel = useCallback(async () => {
    setExporting(true);
    try {
      const ready = await ensureFreshReport();
      if (!ready) return;
      const res = await API.post(
        '/api/profit_board/export/excel',
        buildQueryPayload(),
        { responseType: 'blob' },
      );
      const disposition = res.headers?.['content-disposition'] || '';
      const matched = disposition.match(/filename=\"(.+)\"/);
      downloadBlob(
        new Blob([res.data], {
          type: 'application/vnd.ms-excel;charset=utf-8',
        }),
        matched?.[1] || 'profit-board.xls',
      );
    } catch (error) {
      showError(error);
    } finally {
      setExporting(false);
    }
  }, [buildQueryPayload, ensureFreshReport]);

  const businessMetrics = useMemo(
    () => [
      { key: 'configured_site_revenue_usd', label: t('本站配置收入') },
      { key: 'upstream_cost_usd', label: t('上游费用') },
    ],
    [t],
  );

  const trendRows = useMemo(
    () =>
      (report?.timeseries || [])
        .filter(
          (item) => viewBatchId === 'all' || item.batch_id === viewBatchId,
        )
        .map((item) => ({
          bucket: item.bucket,
          value: item[metricKey] || 0,
          series: item.batch_name,
          batch_id: item.batch_id,
        })),
    [metricKey, report?.timeseries, viewBatchId],
  );

  const compareTrendRows = useMemo(
    () =>
      (compareReport?.timeseries || [])
        .filter(
          (item) => viewBatchId === 'all' || item.batch_id === viewBatchId,
        )
        .map((item) => ({
          bucket: item.bucket,
          value: item[metricKey] || 0,
          series: item.batch_name,
          batch_id: item.batch_id,
        })),
    [compareReport?.timeseries, metricKey, viewBatchId],
  );

  const batchChannelChartRows = useMemo(() => {
    if (compareMode !== 'batch') return [];
    const filtered = (report?.channel_breakdown || []).filter(
      (item) => viewBatchId === 'all' || item.batch_id === viewBatchId,
    );
    return filtered
      .map((item) => ({
        label: item.label || item.key,
        value: Number(item[metricKey] || 0),
        series: item.batch_name,
        batch_id: item.batch_id,
      }))
      .sort((a, b) => b.value - a.value)
      .slice(0, 24);
  }, [compareMode, metricKey, report?.channel_breakdown, viewBatchId]);

  const channelChartRows = useMemo(
    () =>
      aggregateBreakdownRows(
        report?.channel_breakdown || [],
        viewBatchId,
        metricKey,
      ),
    [metricKey, report?.channel_breakdown, viewBatchId],
  );

  const batchModelChartRows = useMemo(() => {
    if (compareMode !== 'batch') return [];
    const filtered = (report?.model_breakdown || []).filter(
      (item) => viewBatchId === 'all' || item.batch_id === viewBatchId,
    );
    return filtered
      .map((item) => ({
        label: item.label || item.key,
        value: Number(item[metricKey] || 0),
        series: item.batch_name,
        batch_id: item.batch_id,
      }))
      .sort((a, b) => b.value - a.value)
      .slice(0, 24);
  }, [compareMode, metricKey, report?.model_breakdown, viewBatchId]);

  const modelChartRows = useMemo(
    () =>
      aggregateBreakdownRows(
        report?.model_breakdown || [],
        viewBatchId,
        metricKey,
      ),
    [metricKey, report?.model_breakdown, viewBatchId],
  );

  const timeCompareChannelRows = useMemo(() => {
    if (compareMode !== 'time' || !compareReport) return [];
    const currentRows = aggregateBreakdownRows(
      report?.channel_breakdown || [],
      viewBatchId,
      metricKey,
    ).map((item) => ({ ...item, series: t('当前周期') }));
    const previousRows = aggregateBreakdownRows(
      compareReport?.channel_breakdown || [],
      viewBatchId,
      metricKey,
    ).map((item) => ({ ...item, series: t('对比周期') }));
    return [...currentRows, ...previousRows].slice(0, 24);
  }, [compareMode, compareReport, metricKey, report?.channel_breakdown, t, viewBatchId]);

  const timeCompareModelRows = useMemo(() => {
    if (compareMode !== 'time' || !compareReport) return [];
    const currentRows = aggregateBreakdownRows(
      report?.model_breakdown || [],
      viewBatchId,
      metricKey,
    ).map((item) => ({ ...item, series: t('当前周期') }));
    const previousRows = aggregateBreakdownRows(
      compareReport?.model_breakdown || [],
      viewBatchId,
      metricKey,
    ).map((item) => ({ ...item, series: t('对比周期') }));
    return [...currentRows, ...previousRows].slice(0, 24);
  }, [compareMode, compareReport, metricKey, report?.model_breakdown, t, viewBatchId]);

  const businessTrendRows = useMemo(
    () =>
      combineTimeseriesMetrics(
        report?.timeseries || [],
        viewBatchId,
        businessMetrics,
      ),
    [businessMetrics, report?.timeseries, viewBatchId],
  );

  const compareBusinessTrendRows = useMemo(
    () =>
      combineTimeseriesMetrics(
        compareReport?.timeseries || [],
        viewBatchId,
        businessMetrics,
      ),
    [businessMetrics, compareReport?.timeseries, viewBatchId],
  );

  const businessChannelRows = useMemo(
    () =>
      combineBreakdownMetrics(
        report?.channel_breakdown || [],
        viewBatchId,
        businessMetrics,
      ),
    [businessMetrics, report?.channel_breakdown, viewBatchId],
  );

  const compareBusinessChannelRows = useMemo(
    () =>
      combineBreakdownMetrics(
        compareReport?.channel_breakdown || [],
        viewBatchId,
        businessMetrics,
      ),
    [businessMetrics, compareReport?.channel_breakdown, viewBatchId],
  );

  const businessModelRows = useMemo(
    () =>
      combineBreakdownMetrics(
        report?.model_breakdown || [],
        viewBatchId,
        businessMetrics,
      ),
    [businessMetrics, report?.model_breakdown, viewBatchId],
  );

  const compareBusinessModelRows = useMemo(
    () =>
      combineBreakdownMetrics(
        compareReport?.model_breakdown || [],
        viewBatchId,
        businessMetrics,
      ),
    [businessMetrics, compareReport?.model_breakdown, viewBatchId],
  );

  const timeCompareBusinessChannelRows = useMemo(() => {
    if (compareMode !== 'time' || !compareReport) return [];
    const currentRows = businessChannelRows.map((item) => ({
      ...item,
      series: `${t('当前周期')} · ${item.series}`,
    }));
    const previousRows = compareBusinessChannelRows.map((item) => ({
      ...item,
      series: `${t('对比周期')} · ${item.series}`,
    }));
    return [...currentRows, ...previousRows].slice(0, 32);
  }, [
    businessChannelRows,
    compareBusinessChannelRows,
    compareMode,
    compareReport,
    t,
  ]);

  const timeCompareBusinessModelRows = useMemo(() => {
    if (compareMode !== 'time' || !compareReport) return [];
    const currentRows = businessModelRows.map((item) => ({
      ...item,
      series: `${t('当前周期')} · ${item.series}`,
    }));
    const previousRows = compareBusinessModelRows.map((item) => ({
      ...item,
      series: `${t('对比周期')} · ${item.series}`,
    }));
    return [...currentRows, ...previousRows].slice(0, 32);
  }, [
    businessModelRows,
    compareBusinessModelRows,
    compareMode,
    compareReport,
    t,
  ]);

  const filteredDetailRows = useMemo(() => {
    let rows = report?.detail_rows || [];
    if (viewBatchId !== 'all') {
      rows = rows.filter((row) => row.batch_id === viewBatchId);
    }
    if (!detailFilter?.type || !detailFilter?.value) return rows;
    if (detailFilter.batchId) {
      rows = rows.filter((row) => row.batch_id === detailFilter.batchId);
    }
    if (detailFilter.type === 'channel') {
      return rows.filter((row) => row.channel_name === detailFilter.value);
    }
    if (detailFilter.type === 'model') {
      return rows.filter((row) => row.model_name === detailFilter.value);
    }
    if (detailFilter.type === 'trend') {
      return rows.filter(
        (row) =>
          formatBucketLabel(
            row.created_at,
            granularity,
            customIntervalMinutes,
          ) === detailFilter.value,
      );
    }
    return rows;
  }, [
    customIntervalMinutes,
    detailFilter,
    granularity,
    report?.detail_rows,
    viewBatchId,
  ]);

  const detailFilterText = useMemo(() => {
    if (!detailFilter?.value) return '';
    const sourceMap = {
      trend: t('图表时间桶'),
      channel: t('图表渠道'),
      model: t('图表模型'),
    };
    return `${sourceMap[detailFilter.type] || t('图表筛选')}：${detailFilter.value}`;
  }, [detailFilter, t]);

  const businessMetricSubtitle = t('本站配置收入 vs 上游费用');

  const trendSpec = useMemo(
    () => createTrendSpec(trendRows, metricLabel, statusState?.status, t),
    [metricLabel, statusState?.status, t, trendRows],
  );

  const compareTrendSpec = useMemo(
    () =>
      createTrendSpec(compareTrendRows, metricLabel, statusState?.status, t),
    [compareTrendRows, metricLabel, statusState?.status, t],
  );

  const businessTrendSpec = useMemo(
    () =>
      createTrendSpec(
        businessTrendRows,
        businessMetricSubtitle,
        statusState?.status,
        t,
      ),
    [businessMetricSubtitle, businessTrendRows, statusState?.status, t],
  );

  const compareBusinessTrendSpec = useMemo(
    () =>
      createTrendSpec(
        compareBusinessTrendRows,
        businessMetricSubtitle,
        statusState?.status,
        t,
      ),
    [
      businessMetricSubtitle,
      compareBusinessTrendRows,
      statusState?.status,
      t,
    ],
  );

  const channelSpec = useMemo(
    () =>
      createBarSpec(
        t('渠道对比'),
        compareMode === 'batch'
          ? batchChannelChartRows
          : compareMode === 'time'
            ? timeCompareChannelRows
            : channelChartRows,
        metricLabel,
        statusState?.status,
        t,
      ),
    [
      batchChannelChartRows,
      channelChartRows,
      compareMode,
      metricLabel,
      statusState?.status,
      t,
      timeCompareChannelRows,
    ],
  );

  const modelSpec = useMemo(
    () =>
      createBarSpec(
        t('模型对比'),
        compareMode === 'batch'
          ? batchModelChartRows
          : compareMode === 'time'
            ? timeCompareModelRows
            : modelChartRows,
        metricLabel,
        statusState?.status,
        t,
      ),
    [
      batchModelChartRows,
      compareMode,
      metricLabel,
      modelChartRows,
      statusState?.status,
      t,
      timeCompareModelRows,
    ],
  );

  const businessChannelSpec = useMemo(
    () =>
      createBarSpec(
        t('渠道经营对比'),
        compareMode === 'time'
          ? timeCompareBusinessChannelRows
          : businessChannelRows,
        businessMetricSubtitle,
        statusState?.status,
        t,
      ),
    [
      businessChannelRows,
      businessMetricSubtitle,
      compareMode,
      statusState?.status,
      t,
      timeCompareBusinessChannelRows,
    ],
  );

  const businessModelSpec = useMemo(
    () =>
      createBarSpec(
        t('模型经营对比'),
        compareMode === 'time'
          ? timeCompareBusinessModelRows
          : businessModelRows,
        businessMetricSubtitle,
        statusState?.status,
        t,
      ),
    [
      businessMetricSubtitle,
      businessModelRows,
      compareMode,
      statusState?.status,
      t,
      timeCompareBusinessModelRows,
    ],
  );

  const handleChartClick = useCallback(
    (type) => (event) => {
      const label = event?.datum?.label || event?.datum?.bucket;
      if (!label) return;
      setDetailFilter({
        type,
        value: label,
        batchId: event?.datum?.batch_id || null,
      });
    },
    [],
  );

  const detailColumns = useMemo(
    () => [
      {
        title: t('时间'),
        dataIndex: 'created_at',
        render: (value) => timestamp2string(value),
      },
      { title: t('批次'), dataIndex: 'batch_name' },
      {
        title: t('请求 ID'),
        dataIndex: 'request_id',
        render: (value) =>
          value ? (
            <Space wrap>
              <Text code>{value}</Text>
              <Button
                size='small'
                type='tertiary'
                onClick={async () => {
                  const ok = await copy(value);
                  if (ok) {
                    showSuccess(t('请求 ID 已复制到剪贴板'));
                    return;
                  }
                  showError(t('复制失败'));
                }}
              >
                {t('复制请求 ID')}
              </Button>
            </Space>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('渠道'),
        dataIndex: 'channel_name',
        render: (value, row) => value || `#${row.channel_id}`,
      },
      { title: t('模型'), dataIndex: 'model_name' },
      {
        title: t('本站实际收入'),
        dataIndex: 'actual_site_revenue_usd',
        render: (value) => formatMoney(value, statusState?.status),
      },
      {
        title: t('本站配置收入'),
        dataIndex: 'configured_site_revenue_usd',
        render: (value) => formatMoney(value, statusState?.status),
      },
      {
        title: t('配置与实际差值'),
        dataIndex: 'configured_actual_delta_usd',
        render: (value) => formatMoney(value, statusState?.status),
      },
      {
        title: t('本站配置来源'),
        dataIndex: 'site_pricing_source',
        render: (value, row) => (
          <Tag color={row.site_pricing_known ? 'blue' : 'grey'}>
            {sitePricingSourceLabelMap[value] || value || t('未知')}
          </Tag>
        ),
      },
      {
        title: t('上游费用'),
        dataIndex: 'upstream_cost_usd',
        render: (value, row) =>
          row.upstream_cost_known ? (
            formatMoney(value, statusState?.status)
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('配置利润'),
        dataIndex: 'configured_profit_usd',
        render: (value, row) =>
          row.upstream_cost_known && row.site_pricing_known ? (
            formatMoney(value, statusState?.status)
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('实际利润'),
        dataIndex: 'actual_profit_usd',
        render: (value, row) =>
          row.upstream_cost_known ? (
            formatMoney(value, statusState?.status)
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
    ],
    [statusState?.status, t],
  );

  const batchDigest = useCallback(
    (batch) => {
      if (batch.scope_type === 'channel') {
        const names = (batch.channel_ids || [])
          .map((id) => channelMap.get(id)?.name || `#${id}`)
          .slice(0, 3);
        const extra = Math.max(
          (batch.channel_ids || []).length - names.length,
          0,
        );
        return `${t('渠道')} · ${names.join(' / ')}${extra > 0 ? ` +${extra}` : ''}`;
      }
      const tags = (batch.tags || []).slice(0, 3);
      const extra = Math.max((batch.tags || []).length - tags.length, 0);
      return `${t('标签聚合渠道')} · ${tags.join(' / ')}${extra > 0 ? ` +${extra}` : ''}`;
    },
    [channelMap, t],
  );

  const summaryMetricHelp = useMemo(
    () => ({
      actual_site_revenue_usd: t('日志里当时真实扣费换算出的本站收入。'),
      configured_site_revenue_usd: t(
        '按你现在填写的本站价格配置重新模拟出来的收入。',
      ),
      upstream_cost_usd: t(
        '按当前手动填写的上游价格和固定总金额重算出来的上游费用。',
      ),
      configured_profit_usd: t(
        '本站配置收入减去上游费用，反映当前定价口径下的利润。',
      ),
      actual_profit_usd: t('真实收入减去上游费用，反映历史实际利润。'),
      configured_profit_coverage_rate: t(
        '有已知上游费用的请求数占总请求数的比例，越高说明利润统计越完整。',
      ),
      missing_upstream_cost_count: t('按当前口径仍无法确认上游费用的请求数。'),
      site_model_match_count: t('成功命中本站模型价格配置的请求数。'),
      missing_site_pricing_count: t('未命中本站模型价格且无法确认本站配置收入的请求数。'),
      manual_cost_count: t('按手动填写的上游价格计算出成本的请求数。'),
      request_count: t('当前统计范围内参与计算的请求总数。'),
    }),
    [t],
  );

  const cumulativeSummaryCards = useMemo(
    () => [
      {
        key: 'configured_site_revenue_usd',
        title: t('本站配置收入'),
        value: formatMoney(
          overviewReport?.summary?.configured_site_revenue_usd,
          statusState?.status,
        ),
        icon: <CircleDollarSign size={16} />,
      },
      {
        key: 'upstream_cost_usd',
        title: t('上游费用'),
        value: formatMoney(
          overviewReport?.summary?.upstream_cost_usd,
          statusState?.status,
        ),
        icon: <BarChart3 size={16} />,
      },
      {
        key: 'configured_profit_usd',
        title: t('配置利润'),
        value: formatMoney(
          overviewReport?.summary?.configured_profit_usd,
          statusState?.status,
        ),
        icon: <BadgeDollarSign size={16} />,
      },
      {
        key: 'actual_profit_usd',
        title: t('实际利润'),
        value: formatMoney(
          overviewReport?.summary?.actual_profit_usd,
          statusState?.status,
        ),
        icon: <BadgeDollarSign size={16} />,
      },
    ],
    [overviewReport?.summary, statusState?.status, t],
  );

  const diagnosticSummaryCards = useMemo(
    () => [
      {
        key: 'request_count',
        title: t('请求数'),
        value: overviewReport?.summary?.request_count || 0,
      },
      {
        key: 'configured_profit_coverage_rate',
        title: t('费用覆盖率'),
        value: formatRatio(overviewReport?.summary?.configured_profit_coverage_rate),
      },
      {
        key: 'missing_upstream_cost_count',
        title: t('缺失上游费用'),
        value: overviewReport?.summary?.missing_upstream_cost_count || 0,
      },
      {
        key: 'site_model_match_count',
        title: t('命中本站模型价格'),
        value: overviewReport?.summary?.site_model_match_count || 0,
      },
      {
        key: 'missing_site_pricing_count',
        title: t('缺失本站价格'),
        value: overviewReport?.summary?.missing_site_pricing_count || 0,
      },
    ],
    [overviewReport?.summary, t],
  );

  const statusSummary = useMemo(() => {
    if (!report) return [];
    const items = [];
    if (reportLoadedFromCache) {
      items.push({
        key: 'cache',
        color: 'blue',
        text: t('已从本地缓存恢复，正在校验是否有新账单'),
      });
    }
    if (reportHasPendingChanges) {
      items.push({
        key: 'dirty',
        color: 'orange',
        text: t('时间范围或价格口径已变更，当前图表仍是上一版结果'),
      });
    }
    if (activityChecking) {
      items.push({
        key: 'checking',
        color: 'cyan',
        text: t('正在检查新的请求与计费活动'),
      });
    }
    if (reportIsFresh && autoRefreshEligible && autoRefreshEnabled) {
      items.push({
        key: 'live',
        color: 'green',
        text: t('自动更新已开启'),
      });
    }
    if (reportIsFresh && !autoRefreshEligible) {
      items.push({
        key: 'history',
        color: 'grey',
        text: t('历史时间范围，自动更新已暂停'),
      });
    }
    return items;
  }, [
    activityChecking,
    autoRefreshEligible,
    autoRefreshEnabled,
    report,
    reportHasPendingChanges,
    reportIsFresh,
    reportLoadedFromCache,
    t,
  ]);

  const currentRangeText = useMemo(() => formatRangeLabel(dateRange), [dateRange]);
  const currentRangeDuration = useMemo(
    () => formatRangeDuration(dateRange),
    [dateRange],
  );
  const compareRangeText = useMemo(
    () => formatRangeLabel(resolvedCompareDateRange),
    [resolvedCompareDateRange],
  );

  const combinedWarnings = useMemo(
    () =>
      Array.from(
        new Set([
          ...(overviewReport?.warnings || []),
          ...(report?.warnings || []),
        ]),
      ),
    [overviewReport?.warnings, report?.warnings],
  );

  const chartContent = {
    trend:
      analysisMode === 'business_compare'
        ? compareMode === 'time'
          ? businessTrendRows.length || compareBusinessTrendRows.length
            ? (
              <div className='grid gap-4 xl:grid-cols-2'>
                <Card bordered={false} bodyStyle={{ padding: 8 }}>
                  <div className='mb-2 px-2'>
                    <Text strong>{t('当前周期')}</Text>
                  </div>
                  <VChart
                    key={`business-trend-current-${actualTheme}-${viewBatchId}`}
                    spec={businessTrendSpec}
                    onClick={handleChartClick('trend')}
                  />
                </Card>
                <Card bordered={false} bodyStyle={{ padding: 8 }}>
                  <div className='mb-2 px-2'>
                    <Text strong>{t('对比周期')}</Text>
                  </div>
                  <VChart
                    key={`business-trend-compare-${actualTheme}-${viewBatchId}-${compareQueryKey}`}
                    spec={compareBusinessTrendSpec}
                  />
                </Card>
              </div>
            )
            : <Empty description={t('当前没有经营趋势数据')} />
          : businessTrendRows.length
            ? (
              <VChart
                key={`business-trend-${actualTheme}-${viewBatchId}`}
                spec={businessTrendSpec}
                onClick={handleChartClick('trend')}
              />
            )
            : <Empty description={t('当前没有经营趋势数据')} />
        : compareMode === 'time'
          ? trendRows.length || compareTrendRows.length
            ? (
              <div className='grid gap-4 xl:grid-cols-2'>
                <Card bordered={false} bodyStyle={{ padding: 8 }}>
                  <div className='mb-2 px-2'>
                    <Text strong>{t('当前周期')}</Text>
                  </div>
                  <VChart
                    key={`trend-current-${actualTheme}-${viewBatchId}-${metricKey}`}
                    spec={trendSpec}
                    onClick={handleChartClick('trend')}
                  />
                </Card>
                <Card bordered={false} bodyStyle={{ padding: 8 }}>
                  <div className='mb-2 px-2'>
                    <Text strong>{t('对比周期')}</Text>
                  </div>
                  <VChart
                    key={`trend-compare-${actualTheme}-${viewBatchId}-${metricKey}-${compareQueryKey}`}
                    spec={compareTrendSpec}
                  />
                </Card>
              </div>
            )
            : <Empty description={t('当前没有趋势数据')} />
          : trendRows.length
            ? (
              <VChart
                key={`trend-${actualTheme}-${viewBatchId}-${metricKey}`}
                spec={trendSpec}
                onClick={handleChartClick('trend')}
              />
            )
            : <Empty description={t('当前没有趋势数据')} />,
    channel:
      analysisMode === 'business_compare'
        ? (compareMode === 'time'
            ? timeCompareBusinessChannelRows.length
            : businessChannelRows.length)
          ? (
            <VChart
              key={`business-channel-${actualTheme}-${viewBatchId}-${compareMode}-${compareQueryKey}`}
              spec={businessChannelSpec}
              onClick={handleChartClick('channel')}
            />
          )
          : <Empty description={t('当前没有渠道经营数据')} />
        : (compareMode === 'batch'
            ? batchChannelChartRows.length
            : compareMode === 'time'
              ? timeCompareChannelRows.length
              : channelChartRows.length)
          ? (
            <VChart
              key={`channel-${actualTheme}-${viewBatchId}-${metricKey}`}
              spec={channelSpec}
              onClick={handleChartClick('channel')}
            />
          )
          : <Empty description={t('当前没有渠道数据')} />,
    model:
      analysisMode === 'business_compare'
        ? (compareMode === 'time'
            ? timeCompareBusinessModelRows.length
            : businessModelRows.length)
          ? (
            <VChart
              key={`business-model-${actualTheme}-${viewBatchId}-${compareMode}-${compareQueryKey}`}
              spec={businessModelSpec}
              onClick={handleChartClick('model')}
            />
          )
          : <Empty description={t('当前没有模型经营数据')} />
        : (compareMode === 'batch'
            ? batchModelChartRows.length
            : compareMode === 'time'
              ? timeCompareModelRows.length
              : modelChartRows.length)
          ? (
            <VChart
              key={`model-${actualTheme}-${viewBatchId}-${metricKey}`}
              spec={modelSpec}
              onClick={handleChartClick('model')}
            />
          )
          : <Empty description={t('当前没有模型数据')} />,
  };

  const datePresets = createPresetRanges();

  return (
    <Spin spinning={loading}>
      <div className='mt-[60px] space-y-4 px-2 pb-6'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <div>
            <Title heading={4} style={{ marginBottom: 4 }}>
              {t('收益看板')}
            </Title>
            <Paragraph type='tertiary' style={{ margin: 0 }}>
              {t('上面看长期累计结果，下面看指定时间段的收入、成本和利润变化。')}
            </Paragraph>
          </div>
          <Space wrap>
            <Button
              theme='solid'
              type='primary'
              icon={<RefreshCw size={16} />}
              loading={querying || overviewQuerying}
              onClick={runFullRefresh}
            >
              {t('刷新收益看板')}
            </Button>
            <Button
              theme='solid'
              type='tertiary'
              icon={<Save size={16} />}
              loading={saving}
              onClick={saveConfig}
            >
              {t('保存长期配置')}
            </Button>
          </Space>
        </div>

        {combinedWarnings.length > 0 ? (
          <div className='space-y-2'>
            {combinedWarnings.map((warning) => (
              <Banner key={warning} type='warning' description={warning} closeIcon={null} />
            ))}
          </div>
        ) : null}
        {siteConfig.use_recharge_price && (overviewReport?.meta?.site_price_factor_note || report?.meta?.site_price_factor_note) ? (
          <Banner
            type='info'
            icon={<Info size={16} />}
            description={overviewReport?.meta?.site_price_factor_note || report?.meta?.site_price_factor_note}
            closeIcon={null}
          />
        ) : null}

        <div className='grid gap-4 xl:grid-cols-[0.96fr_1.04fr]'>
          <Card bordered={false} title={t('关注批次')}>
            <div className='space-y-4'>
              <div className='grid gap-3 md:grid-cols-[150px_1fr]'>
                <Radio.Group
                  type='button'
                  value={draft.scope_type}
                  onChange={(event) =>
                    setDraft((prev) => ({
                      ...prev,
                      scope_type: event.target.value,
                      channel_ids: [],
                      tags: [],
                    }))
                  }
                >
                  <Radio value='channel'>{t('渠道')}</Radio>
                  <Radio value='tag'>{t('标签聚合渠道')}</Radio>
                </Radio.Group>
                <Input
                  value={draft.name}
                  onChange={(value) => setDraft((prev) => ({ ...prev, name: value }))}
                  placeholder={t('批次名称，例如：OpenAI 第一批')}
                />
              </div>
              <Select
                multiple
                filter
                maxTagCount={isMobile ? 2 : 4}
                optionList={
                  draft.scope_type === 'channel'
                    ? channelOptions
                    : (options.tags || []).map((item) => ({ label: item, value: item }))
                }
                value={draft.scope_type === 'channel' ? draft.channel_ids || [] : draft.tags || []}
                onChange={(value) =>
                  draft.scope_type === 'channel'
                    ? setDraft((prev) => ({ ...prev, channel_ids: value || [] }))
                    : setDraft((prev) => ({ ...prev, tags: value || [] }))
                }
                placeholder={draft.scope_type === 'channel' ? t('选择一个或多个渠道') : t('选择一个或多个标签')}
                style={{ width: '100%' }}
              />
              <div className='flex flex-wrap items-center gap-2'>
                <Button icon={<Plus size={16} />} onClick={addOrUpdateBatch}>
                  {editingBatchId ? t('保存修改') : t('添加批次')}
                </Button>
                {editingBatchId ? (
                  <Button type='tertiary' onClick={resetDraft}>
                    {t('取消编辑')}
                  </Button>
                ) : null}
              </div>
              <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0/60 p-3'>
                <div className='mb-2 flex items-center gap-2'>
                  <Layers3 size={15} />
                  <Text strong>{t('当前长期关注的批次')}</Text>
                </div>
                {batches.length > 0 ? (
                  <div className='space-y-2'>
                    {batches.map((batch) => (
                      <div
                        key={batch.id}
                        className='flex flex-col gap-2 rounded-lg border border-semi-color-border bg-semi-color-bg-2 p-3 lg:flex-row lg:items-center lg:justify-between'
                      >
                        <div>
                          <Space wrap>
                            <Text strong>{batch.name}</Text>
                            <Tag color={batch.scope_type === 'channel' ? 'blue' : 'cyan'}>
                              {batch.scope_type === 'channel' ? t('渠道') : t('标签聚合渠道')}
                            </Tag>
                          </Space>
                          <Text type='tertiary' className='mt-1 block'>
                            {batchDigest(batch)}
                          </Text>
                        </div>
                        <Space>
                          <Button icon={<Pencil size={14} />} size='small' type='tertiary' onClick={() => editBatch(batch)}>
                            {t('编辑')}
                          </Button>
                          <Button icon={<Trash2 size={14} />} size='small' type='danger' onClick={() => removeBatch(batch.id)}>
                            {t('删除')}
                          </Button>
                        </Space>
                      </div>
                    ))}
                  </div>
                ) : (
                  <Empty image={null} description={t('还没有批次，先添加一批渠道或标签')} />
                )}
              </div>
              {batchValidationError ? (
                <Banner type='danger' description={batchValidationError} closeIcon={null} />
              ) : (
                <Text type='tertiary'>
                  {t('这里决定你长期盯哪些渠道；顶部累计总览会一直按这些批次统计。')}
                </Text>
              )}
            </div>
          </Card>

          <div className='space-y-4'>
            <Card bordered={false} title={t('累计总览')}>
              <Spin spinning={overviewQuerying}>
                {overviewReport ? (
                  <div className='space-y-4'>
                    <div className='grid gap-4 md:grid-cols-2'>
                      {cumulativeSummaryCards.map((item) => (
                        <Card key={item.key} bordered={false} bodyStyle={{ padding: 18 }} className='bg-semi-color-fill-0'>
                          <div className='flex items-center justify-between gap-3'>
                            <div>
                              <Tooltip content={summaryMetricHelp[item.key] || item.title}>
                                <div className='inline-flex cursor-help items-center gap-1'>
                                  <Text type='tertiary'>{item.title}</Text>
                                  <Info size={14} className='text-semi-color-text-2' />
                                </div>
                              </Tooltip>
                              <Title heading={3} style={{ margin: '8px 0 0' }}>
                                {item.value}
                              </Title>
                            </div>
                            <div className='flex h-10 w-10 items-center justify-center rounded-full bg-semi-color-bg-2'>
                              {item.icon}
                            </div>
                          </div>
                        </Card>
                      ))}
                    </div>
                    <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-5'>
                      {diagnosticSummaryCards.map((item) => (
                        <div key={item.key} className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'>
                          <Tooltip content={summaryMetricHelp[item.key] || item.title}>
                            <div className='inline-flex cursor-help items-center gap-1'>
                              <Text type='tertiary'>{item.title}</Text>
                              <Info size={13} className='text-semi-color-text-2' />
                            </div>
                          </Tooltip>
                          <div className='mt-2 text-lg font-semibold'>{item.value}</div>
                        </div>
                      ))}
                    </div>
                    <div className='grid gap-3 md:grid-cols-2'>
                      <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                        <Text type='tertiary'>{t('累计统计范围')}</Text>
                        <div className='mt-1 font-medium'>{t('已添加批次的全部历史消费日志')}</div>
                      </div>
                      <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                        <Text type='tertiary'>{t('最近一条命中日志')}</Text>
                        <div className='mt-1 font-medium'>
                          {overviewReport?.meta?.latest_log_created_at ? timestamp2string(overviewReport.meta.latest_log_created_at) : '-'}
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <Empty description={t('添加批次后会自动加载累计总览')} />
                )}
              </Spin>
            </Card>

            {overviewReport?.batch_summaries?.length > 0 ? (
              <Card bordered={false} title={t('批次累计收益')}>
                <div className='grid gap-3 md:grid-cols-2'>
                  {overviewReport.batch_summaries.map((item) => (
                    <div key={item.batch_id} className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'>
                      <div className='flex items-center justify-between gap-2'>
                        <Text strong>{item.batch_name}</Text>
                        <Tag color='blue'>{item.request_count} {t('次请求')}</Tag>
                      </div>
                      <div className='mt-2 grid grid-cols-2 gap-2 text-sm'>
                        <Text>{t('本站配置收入')}：{formatMoney(item.configured_site_revenue_usd, statusState?.status)}</Text>
                        <Text>{t('上游费用')}：{formatMoney(item.upstream_cost_usd, statusState?.status)}</Text>
                        <Text>{t('配置利润')}：{formatMoney(item.configured_profit_usd, statusState?.status)}</Text>
                        <Text>{t('实际利润')}：{formatMoney(item.actual_profit_usd, statusState?.status)}</Text>
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            ) : null}
          </div>
        </div>

        <div className='grid gap-4 xl:grid-cols-[1.15fr_0.85fr]'>
          <Card bordered={false} title={t('时间分析范围')}>
            <div className='space-y-4'>
              <div className='flex flex-wrap gap-2'>
                {datePresets.map((item) => (
                  <Button key={item.label} type='tertiary' size='small' onClick={() => setDateRange(item.value)}>
                    {t(item.label)}
                  </Button>
                ))}
              </div>
              <DatePicker
                type='dateTimeRange'
                value={dateRange}
                onChange={(value) => setDateRange(value)}
                style={{ width: '100%' }}
              />
              <div className='grid gap-3 md:grid-cols-2'>
                <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                  <Text type='tertiary'>{t('当前时间范围')}</Text>
                  <div className='mt-1 font-medium'>{currentRangeText}</div>
                </div>
                <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                  <Text type='tertiary'>{t('时长')}</Text>
                  <div className='mt-1 font-medium'>{currentRangeDuration}</div>
                </div>
              </div>
              {validationErrors.length > 0 ? (
                <Banner type='danger' description={validationErrors[0]} closeIcon={null} />
              ) : (
                <Text type='tertiary'>
                  {t('这里的时间只影响图表和对账明细，不影响上面的累计总览。')}
                </Text>
              )}
            </div>
          </Card>

          <Card bordered={false} title={t('时间分析状态')}>
            <div className='space-y-3'>
              <div className='flex flex-wrap gap-2'>
                {statusSummary.length > 0 ? statusSummary.map((item) => (
                  <Tag key={item.key} color={item.color}>{item.text}</Tag>
                )) : <Tag color='grey'>{t('等待时间分析结果')}</Tag>}
              </div>
              <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                <Text type='tertiary'>{t('时间分析上次更新时间')}</Text>
                <div className='mt-1 font-medium'>
                  {report?.meta?.generated_at ? timestamp2string(report.meta.generated_at) : '-'}
                </div>
              </div>
              {compareMode === 'time' ? (
                <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                  <Text type='tertiary'>{t('对比周期')}</Text>
                  <div className='mt-1 font-medium'>{compareRangeText}</div>
                </div>
              ) : null}
              <Text type='tertiary'>
                {t('自动更新只会在时间范围接近现在且页面可见时工作。')}
              </Text>
            </div>
          </Card>
        </div>

        <Card
          bordered={false}
          bodyStyle={{ paddingTop: 12 }}
          title={
            <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
              <Space>
                <Filter size={16} />
                <span>{t('图表分析')}</span>
              </Space>
              <Space wrap>
                <Select
                  value={analysisMode}
                  onChange={setAnalysisMode}
                  optionList={[
                    { label: t('经营对比'), value: 'business_compare' },
                    { label: t('单指标分析'), value: 'single_metric' },
                  ]}
                  style={{ width: 150 }}
                />
                {analysisMode === 'single_metric' ? (
                  <Select
                    value={metricKey}
                    onChange={setMetricKey}
                    optionList={metricOptions.map((item) => ({ label: t(item.label), value: item.value }))}
                    style={{ width: 170 }}
                  />
                ) : null}
                <Select
                  value={viewBatchId}
                  onChange={setViewBatchId}
                  optionList={batchSummaryOptions}
                  style={{ width: 180 }}
                />
                <Select
                  value={granularity}
                  onChange={setGranularity}
                  optionList={[
                    { label: t('按小时'), value: 'hour' },
                    { label: t('按天'), value: 'day' },
                    { label: t('按周'), value: 'week' },
                    { label: t('按月'), value: 'month' },
                    { label: t('自定义分钟'), value: 'custom' },
                  ]}
                  style={{ width: 120 }}
                />
                {granularity === 'custom' ? (
                  <InputNumber
                    min={1}
                    value={customIntervalMinutes}
                    onChange={(value) => setCustomIntervalMinutes(Math.max(Number(value || 1), 1))}
                    suffix={t('分钟')}
                    style={{ width: 140 }}
                  />
                ) : null}
                <Select
                  value={compareMode}
                  onChange={setCompareMode}
                  optionList={
                    analysisMode === 'business_compare'
                      ? [
                          { label: t('不对比'), value: 'none' },
                          { label: t('时间对比'), value: 'time' },
                        ]
                      : [
                          { label: t('不对比'), value: 'none' },
                          { label: t('批次对比'), value: 'batch' },
                          { label: t('时间对比'), value: 'time' },
                        ]
                  }
                  style={{ width: 120 }}
                />
                {compareMode === 'time' ? (
                  <>
                    <Select
                      value={comparePeriod}
                      onChange={setComparePeriod}
                      optionList={[
                        { label: t('上一等长周期'), value: 'previous' },
                        { label: t('自定义对比周期'), value: 'custom' },
                      ]}
                      style={{ width: 150 }}
                    />
                    {comparePeriod === 'custom' ? (
                      <DatePicker
                        type='dateTimeRange'
                        value={compareDateRange}
                        onChange={setCompareDateRange}
                        style={{ width: 280 }}
                      />
                    ) : null}
                  </>
                ) : null}
                {detailFilter?.value ? (
                  <Button type='tertiary' icon={<Filter size={14} />} onClick={() => setDetailFilter(null)}>
                    {t('清除图表筛选')}
                  </Button>
                ) : null}
                <Button type='tertiary' onClick={runQuery} loading={querying}>
                  {t('刷新时间分析')}
                </Button>
                <Button type='tertiary' icon={<ArrowDownToLine size={14} />} onClick={exportCSV} loading={exporting} disabled={!report}>
                  CSV
                </Button>
                <Button type='tertiary' onClick={exportExcel} loading={exporting} disabled={!report}>
                  Excel
                </Button>
              </Space>
            </div>
          }
        >
          <Text type='tertiary' className='mb-3 block'>
            {analysisMode === 'business_compare'
              ? t('默认直接看本站配置收入和上游费用的金额对比，更适合判断渠道是否赚钱。')
              : t('单指标分析适合单独观察某一个指标在时间、渠道或模型上的变化。')}
          </Text>
          <Tabs activeKey={chartTab} onChange={setChartTab} type='line'>
            <Tabs.TabPane tab={t('趋势')} itemKey='trend' />
            <Tabs.TabPane tab={t('渠道')} itemKey='channel' />
            <Tabs.TabPane tab={t('模型')} itemKey='model' />
          </Tabs>
          <div className='min-h-[360px]'>
            {report ? chartContent[chartTab] : <Empty description={t('设置时间范围后刷新即可查看时间分析')} />}
          </div>
        </Card>

        <div className='grid gap-4 xl:grid-cols-2'>
          <Card bordered={false} title={t('上游价格配置')}>
            <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
              <InputNumber
                min={0}
                value={upstreamConfig.input_price}
                onChange={(value) => setUpstreamConfig((prev) => ({ ...prev, input_price: clampNumber(value) }))}
                suffix='USD / 1M 输入'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.output_price}
                onChange={(value) => setUpstreamConfig((prev) => ({ ...prev, output_price: clampNumber(value) }))}
                suffix='USD / 1M 输出'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.cache_read_price}
                onChange={(value) => setUpstreamConfig((prev) => ({ ...prev, cache_read_price: clampNumber(value) }))}
                suffix='USD / 1M 缓存读'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.cache_creation_price}
                onChange={(value) => setUpstreamConfig((prev) => ({ ...prev, cache_creation_price: clampNumber(value) }))}
                suffix='USD / 1M 缓存写'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.fixed_total_amount}
                onChange={(value) => setUpstreamConfig((prev) => ({ ...prev, fixed_total_amount: clampNumber(value) }))}
                suffix='USD'
              />
            </div>
            <Text type='tertiary' className='mt-3 block'>
              {t('上游费用只按这里的手动价格计算；固定总金额只参与当前时间分析，不参与累计总览。')}
            </Text>
          </Card>

          <Card bordered={false} title={t('本站价格配置')}>
            <div className='space-y-3'>
              <Select
                value={siteConfig.pricing_mode}
                onChange={(value) => setSiteConfig((prev) => ({ ...prev, pricing_mode: value }))}
                optionList={[
                  { label: t('手动输入本站价格'), value: 'manual' },
                  { label: t('读取本站模型价格'), value: 'site_model' },
                ]}
              />
              {siteConfig.pricing_mode === 'site_model' ? (
                <>
                  <div className='grid gap-3 md:grid-cols-2'>
                    <Select
                      multiple
                      filter
                      value={siteConfig.model_names || []}
                      onChange={(value) => setSiteConfig((prev) => ({ ...prev, model_names: value || [] }))}
                      optionList={(options.local_models || []).map((item) => ({ label: item.model_name, value: item.model_name }))}
                      placeholder={t('选择一个或多个本站模型')}
                    />
                    <Select
                      value={siteConfig.group}
                      onChange={(value) => setSiteConfig((prev) => ({ ...prev, group: value }))}
                      optionList={[
                        { label: t('自动取最低分组倍率'), value: '' },
                        ...(options.groups || []).map((item) => ({ label: item, value: item })),
                      ]}
                    />
                  </div>
                  <div className='flex items-center justify-between rounded-lg bg-semi-color-fill-0 px-3 py-2'>
                    <div>
                      <Text strong>{t('按充值价格读取')}</Text>
                      <Text type='tertiary' className='block'>
                        {t('开启后按充值倍率重算；如果倍率正好是 1，就会和原价一样。')}
                      </Text>
                    </div>
                    <Switch
                      checked={siteConfig.use_recharge_price}
                      onChange={(checked) => setSiteConfig((prev) => ({ ...prev, use_recharge_price: checked }))}
                    />
                  </div>
                </>
              ) : null}
              <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
                <InputNumber
                  min={0}
                  value={siteConfig.input_price}
                  onChange={(value) => setSiteConfig((prev) => ({ ...prev, input_price: clampNumber(value) }))}
                  suffix='USD / 1M 输入'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.output_price}
                  onChange={(value) => setSiteConfig((prev) => ({ ...prev, output_price: clampNumber(value) }))}
                  suffix='USD / 1M 输出'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.cache_read_price}
                  onChange={(value) => setSiteConfig((prev) => ({ ...prev, cache_read_price: clampNumber(value) }))}
                  suffix='USD / 1M 缓存读'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.cache_creation_price}
                  onChange={(value) => setSiteConfig((prev) => ({ ...prev, cache_creation_price: clampNumber(value) }))}
                  suffix='USD / 1M 缓存写'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.fixed_total_amount}
                  onChange={(value) => setSiteConfig((prev) => ({ ...prev, fixed_total_amount: clampNumber(value) }))}
                  suffix='USD'
                />
              </div>
              <Text type='tertiary' className='block'>
                {t('本站配置收入 = 按你当前配置重算的收入；固定总金额适合买断、包月、保底等场景，只参与当前时间分析。')}
              </Text>
            </div>
          </Card>
        </div>

        <Card
          bordered={false}
          title={
            <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
              <span>{t('请求对账明细')}</span>
              <Space wrap>
                {detailFilterText ? <Tag color='light-blue'>{detailFilterText}</Tag> : null}
                <Text type='tertiary'>
                  {t('已展示 {{count}} 条', { count: filteredDetailRows.length })}
                  {report?.detail_truncated ? ` · ${t('结果已截断')}` : ''}
                </Text>
              </Space>
            </div>
          }
        >
          <Text type='tertiary' className='mb-3 block'>
            {t('固定总金额只参与当前时间范围的汇总和图表，不会摊到单条请求明细。')}
          </Text>
          <Table
            columns={detailColumns}
            dataSource={filteredDetailRows}
            rowKey='id'
            pagination={{ pageSize: isMobile ? 8 : 12 }}
            empty={t('当前时间范围暂无明细')}
          />
        </Card>
      </div>
    </Spin>
  );

};

export default ProfitBoardPage;
