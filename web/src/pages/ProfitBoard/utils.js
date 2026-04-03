import dayjs from 'dayjs';

export const STORAGE_KEY = 'profit-board:state';
export const REPORT_CACHE_KEY = 'profit-board:report';
export const DETAIL_LIMIT = 600;

export const metricOptions = [
  { value: 'configured_profit_usd', label: '配置利润' },
  { value: 'actual_profit_usd', label: '实际利润' },
  { value: 'actual_site_revenue_usd', label: '本站实际收入' },
  { value: 'configured_site_revenue_usd', label: '本站配置收入' },
  { value: 'upstream_cost_usd', label: '上游费用' },
  { value: 'remote_observed_cost_usd', label: '远端观测消耗' },
];

export const sitePricingSourceLabelMap = {
  manual: '手动价格',
  manual_rule: '手动价格',
  manual_default: '手动默认规则',
  manual_fallback: '手动价格回退',
  site_model_standard: '读取本站模型原价',
  site_model_recharge: '读取本站模型充值价',
  site_model_missing: '未命中本站模型',
};

export const createBatchId = () =>
  `batch-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

export const clampNumber = (value) => {
  const next = Number(value || 0);
  if (!Number.isFinite(next) || next < 0) return 0;
  return next;
};

export const createDefaultUpstreamConfig = () => ({
  cost_source: 'manual_only',
  upstream_mode: 'manual_rules',
  upstream_account_id: 0,
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  fixed_amount: 0,
  fixed_total_amount: 0,
});

export const createDefaultSiteConfig = () => ({
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

export const createDefaultPricingRule = (overrides = {}) => ({
  model_name: '',
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  is_default: false,
  is_custom: false,
  ...overrides,
});

export const createDefaultRemoteObserverConfig = () => ({
  enabled: false,
  base_url: '',
  user_id: 0,
  access_token: '',
  access_token_masked: '',
});

export const createDefaultComboPricingConfig = (
  comboId,
  legacySite,
  legacyUpstream,
) => ({
  combo_id: comboId,
  site_mode:
    legacySite?.pricing_mode === 'site_model' ? 'shared_site_model' : 'manual',
  site_rules: [
    createDefaultPricingRule({
      is_default: true,
      input_price: clampNumber(legacySite?.input_price),
      output_price: clampNumber(legacySite?.output_price),
      cache_read_price: clampNumber(legacySite?.cache_read_price),
      cache_creation_price: clampNumber(legacySite?.cache_creation_price),
    }),
  ],
  upstream_rules: [
    createDefaultPricingRule({
      is_default: true,
      input_price: clampNumber(legacyUpstream?.input_price),
      output_price: clampNumber(legacyUpstream?.output_price),
      cache_read_price: clampNumber(legacyUpstream?.cache_read_price),
      cache_creation_price: clampNumber(legacyUpstream?.cache_creation_price),
    }),
  ],
  site_fixed_total_amount: clampNumber(legacySite?.fixed_total_amount),
  upstream_fixed_total_amount: clampNumber(legacyUpstream?.fixed_total_amount),
  remote_observer: createDefaultRemoteObserverConfig(),
});

export const createDefaultDraft = () => ({
  id: '',
  name: '',
  scope_type: 'channel',
  channel_ids: [],
  tags: [],
});

export const createDefaultUpstreamAccountDraft = () => ({
  id: 0,
  name: '',
  remark: '',
  account_type: 'newapi',
  base_url: '',
  user_id: 0,
  access_token: '',
  access_token_masked: '',
  enabled: true,
});

export const createDefaultState = () => {
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
    comboConfigs: [],
    upstreamConfig: createDefaultUpstreamConfig(),
    siteConfig: createDefaultSiteConfig(),
    lastQueryKey: '',
    detailPage: 1,
    detailPageSize: 12,
    autoRefreshMode: false,
  };
};

export const safeParse = (raw, fallback) => {
  try {
    return raw ? JSON.parse(raw) : fallback;
  } catch (error) {
    return fallback;
  }
};

export const normalizeBatchForState = (batch, index) => ({
  id: batch?.id || createBatchId(),
  name: batch?.name || `组合 ${index + 1}`,
  scope_type: batch?.scope_type || 'channel',
  channel_ids: (batch?.channel_ids || []).map((item) => item.toString()),
  tags: batch?.tags || [],
});

export const normalizeRestoredState = (state) => {
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
          name: '组合 1',
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
    upstream_mode:
      next.upstreamConfig?.upstream_mode ||
      (next.upstreamConfig?.cost_source &&
      next.upstreamConfig.cost_source !== 'manual_only'
        ? 'wallet_observer'
        : 'manual_rules'),
    fixed_amount: 0,
  };
  if (next.upstreamConfig.upstream_mode !== 'wallet_observer') {
    next.upstreamConfig.upstream_account_id = 0;
    next.upstreamConfig.cost_source = 'manual_only';
  }
  next.siteConfig = {
    ...createDefaultSiteConfig(),
    ...(next.siteConfig || {}),
    model_names: next.siteConfig?.model_names || [],
    fixed_amount: 0,
  };
  next.comboConfigs = (next.comboConfigs || []).map((item) => ({
    ...createDefaultComboPricingConfig(item?.combo_id || ''),
    ...item,
    site_rules: (item?.site_rules || []).map((rule) =>
      createDefaultPricingRule(rule),
    ),
    upstream_rules: (item?.upstream_rules || []).map((rule) =>
      createDefaultPricingRule(rule),
    ),
    remote_observer: {
      ...createDefaultRemoteObserverConfig(),
      ...(item?.remote_observer || {}),
    },
  }));
  next.viewBatchId = next.viewBatchId || 'all';
  next.lastQueryKey = next.lastQueryKey || '';
  next.analysisMode = next.analysisMode || 'business_compare';
  next.detailPage = Math.max(Number(next.detailPage || 1), 1);
  next.detailPageSize = Math.min(
    Math.max(Number(next.detailPageSize || defaults.detailPageSize), 1),
    100,
  );
  next.autoRefreshMode = !!next.autoRefreshMode;
  return next;
};

export const getDisplayCurrency = (status) => {
  const displayType = status?.quota_display_type || 'USD';
  if (displayType === 'CNY') {
    return { symbol: '¥', rate: status?.usd_exchange_rate || 1 };
  }
  if (displayType === 'CUSTOM') {
    return {
      symbol: status?.custom_currency_symbol || '¤',
      rate: status?.custom_currency_exchange_rate || 1,
    };
  }
  return { symbol: '$', rate: 1 };
};

export const formatMoney = (value, status, digits = 3) => {
  const amount = Number(value || 0);
  const { symbol, rate } = getDisplayCurrency(status);
  return `${symbol}${(amount * rate).toFixed(digits)}`;
};

export const createPresetRanges = () => {
  const now = dayjs();
  return [
    {
      label: '今天',
      value: [now.startOf('day').toDate(), now.endOf('day').toDate()],
    },
    {
      label: '最近 24 小时',
      value: [now.subtract(24, 'hour').toDate(), now.toDate()],
    },
    {
      label: '近 7 天',
      value: [now.subtract(7, 'day').toDate(), now.toDate()],
    },
    {
      label: '近 30 天',
      value: [now.subtract(30, 'day').toDate(), now.toDate()],
    },
    {
      label: '本月',
      value: [now.startOf('month').toDate(), now.endOf('month').toDate()],
    },
    {
      label: '上月',
      value: [
        now.subtract(1, 'month').startOf('month').toDate(),
        now.subtract(1, 'month').endOf('month').toDate(),
      ],
    },
  ];
};

export const formatRangeLabel = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return '-';
  return `${dayjs(range[0]).format('YYYY-MM-DD HH:mm')} ~ ${dayjs(range[1]).format('YYYY-MM-DD HH:mm')}`;
};

const hoursToFixed = (minutes) => (minutes % 60 === 0 ? 0 : 1);

export const formatRangeDuration = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return '-';
  const minutes = dayjs(range[1]).diff(dayjs(range[0]), 'minute');
  if (minutes < 60) return `${minutes} 分钟`;
  const hours = (minutes / 60).toFixed(hoursToFixed(minutes));
  if (minutes < 1440) return `${hours} 小时`;
  return `${(minutes / 1440).toFixed(1)} 天`;
};

export const buildQueryKey = (payload) => JSON.stringify(payload);

export const normalizeCachedReportBundle = (raw) => {
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

export const formatRatio = (value) =>
  `${(Number(value || 0) * 100).toFixed(1)}%`;

export const formatBucketLabel = (
  timestamp,
  granularity,
  customIntervalMinutes,
) => {
  const current = dayjs.unix(timestamp);
  if (granularity === 'hour') return current.format('YYYY-MM-DD HH:00');
  if (granularity === 'week')
    return current.startOf('week').add(1, 'day').format('GGGG-[W]WW');
  if (granularity === 'month')
    return current.startOf('month').format('YYYY-MM');
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

export const buildPreviousPeriodRange = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return [];
  const start = dayjs(range[0]);
  const end = dayjs(range[1]);
  const duration = Math.max(end.diff(start, 'second'), 0);
  const previousEnd = start.subtract(1, 'second');
  const previousStart = previousEnd.subtract(duration, 'second');
  return [previousStart.toDate(), previousEnd.toDate()];
};

export const aggregateBreakdownRows = (rows, viewBatchId, metricKey) => {
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

export const combineTimeseriesMetrics = (rows, viewBatchId, metrics) => {
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

export const combineBreakdownMetrics = (rows, viewBatchId, metrics) => {
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

export const createTrendSpec = (rows, metricLabel, status, t) => {
  const isSparse = rows.length <= 3;
  const hasSeries = rows.some((item) => item.series);
  return {
    type: 'line',
    background: 'transparent',
    data: [{ id: 'trend', values: rows }],
    xField: 'bucket',
    yField: 'value',
    seriesField: hasSeries ? 'series' : undefined,
    legends: { visible: hasSeries },
    point: {
      visible: true,
      style: { size: isSparse ? 10 : 5 },
    },
    line: {
      style: { curveType: 'monotone', lineWidth: isSparse ? 3 : 2 },
    },
    label: isSparse
      ? {
          visible: true,
          position: 'top',
          formatter: (datum) => formatMoney(datum.value, status),
          style: { fontSize: 12 },
        }
      : { visible: false },
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
          ...(hasSeries
            ? [{ key: t('组合'), value: (datum) => datum.series }]
            : []),
          {
            key: metricLabel,
            value: (datum) => formatMoney(datum.value, status),
          },
        ],
      },
    },
  };
};

export const createBarSpec = (title, rows, metricLabel, status, t) => ({
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
