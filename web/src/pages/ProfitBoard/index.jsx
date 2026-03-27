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
  Typography,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import {
  ArrowDownToLine,
  BadgeDollarSign,
  BarChart3,
  CircleDollarSign,
  Database,
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

const upstreamCostSourceLabelMap = {
  returned_cost: '上游返回费用',
  manual: '手动价格回退',
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
});
const createDefaultSiteConfig = () => ({
  pricing_mode: 'manual',
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  fixed_amount: 0,
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
    chartTab: 'trend',
    metricKey: 'configured_profit_usd',
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
  next.draft = normalizeBatchForState(next.draft || {}, 0);
  next.editingBatchId = next.editingBatchId || '';
  next.upstreamConfig = {
    ...createDefaultUpstreamConfig(),
    ...(next.upstreamConfig || {}),
  };
  next.siteConfig = {
    ...createDefaultSiteConfig(),
    ...(next.siteConfig || {}),
    model_names: next.siteConfig?.model_names || [],
  };
  next.viewBatchId = next.viewBatchId || 'all';
  next.lastQueryKey = next.lastQueryKey || '';
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

const downloadBlob = (blob, filename) => {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  window.URL.revokeObjectURL(url);
};
const buildQueryKey = (payload) => JSON.stringify(payload);

const formatRatio = (value) => `${(Number(value || 0) * 100).toFixed(1)}%`;

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
  const cachedReport = useMemo(
    () => safeParse(localStorage.getItem(REPORT_CACHE_KEY), null),
    [],
  );
  const restoredState = useMemo(
    () =>
      normalizeRestoredState(safeParse(localStorage.getItem(STORAGE_KEY), {})),
    [],
  );

  const [loading, setLoading] = useState(false);
  const [querying, setQuerying] = useState(false);
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
  const [chartTab, setChartTab] = useState(restoredState.chartTab || 'trend');
  const [metricKey, setMetricKey] = useState(
    restoredState.metricKey || 'configured_profit_usd',
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
  const [report, setReport] = useState(cachedReport);
  const [lastQueryKey, setLastQueryKey] = useState(
    restoredState.lastQueryKey || '',
  );
  const [reportLoadedFromCache, setReportLoadedFromCache] = useState(
    !!cachedReport,
  );

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

  const validationErrors = useMemo(() => {
    const errors = [];
    if ((batches || []).length === 0) errors.push(t('请至少添加一个批次'));
    if (!Array.isArray(dateRange) || !dateRange[0] || !dateRange[1])
      errors.push(t('请选择完整的时间范围'));
    if (
      siteConfig.pricing_mode === 'site_model' &&
      (siteConfig.model_names || []).length === 0
    )
      errors.push(t('读取本站模型价格时，至少选择一个模型'));
    if (duplicateBatchError) errors.push(duplicateBatchError);
    return errors;
  }, [batches, dateRange, duplicateBatchError, siteConfig, t]);

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
          chartTab,
          metricKey,
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
      dateRange,
      detailFilter,
      draft,
      editingBatchId,
      granularity,
      metricKey,
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
      },
      site: {
        ...siteConfig,
        input_price: clampNumber(siteConfig.input_price),
        output_price: clampNumber(siteConfig.output_price),
        cache_read_price: clampNumber(siteConfig.cache_read_price),
        cache_creation_price: clampNumber(siteConfig.cache_creation_price),
        fixed_amount: clampNumber(siteConfig.fixed_amount),
      },
      start_timestamp: dayjs(dateRange?.[0]).unix(),
      end_timestamp: dayjs(dateRange?.[1]).unix(),
      granularity,
      detail_limit: DETAIL_LIMIT,
    }),
    [batchPayload, dateRange, granularity, siteConfig, upstreamConfig],
  );
  const currentQueryKey = useMemo(
    () => buildQueryKey(buildQueryPayload()),
    [buildQueryPayload],
  );
  const reportIsFresh =
    !!report &&
    !reportLoadedFromCache &&
    currentQueryKey === lastQueryKey;
  const hasCachedReport = !!report && reportLoadedFromCache;
  const reportIsStale =
    !!report && !!lastQueryKey && currentQueryKey !== lastQueryKey;

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
    });
    setSiteConfig({
      ...createDefaultSiteConfig(),
      ...(config.site || {}),
      model_names: config.site?.model_names || [],
    });
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
      (report?.batch_summaries || []).map((item) => item.batch_id),
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
            dayjs
              .unix(row.created_at)
              .format(
                granularity === 'hour'
                  ? 'YYYY-MM-DD HH:00'
                  : granularity === 'week'
                    ? 'GGGG-[W]WW'
                    : 'YYYY-MM-DD',
              ) === prev.value
          );
        }
        return false;
      });
      return matched ? prev : null;
    });
  }, [granularity, report, viewBatchId]);

  const batchSummaryOptions = useMemo(
    () => [
      { label: t('全部批次'), value: 'all' },
      ...(report?.batch_summaries || []).map((item) => ({
        label: item.batch_name,
        value: item.batch_id,
      })),
    ],
    [report?.batch_summaries, t],
  );

  const addOrUpdateBatch = useCallback(() => {
    const name =
      draft.name?.trim() || `批次 ${batches.length + (editingBatchId ? 0 : 1)}`;
    if (
      draft.scope_type === 'channel' &&
      (draft.channel_ids || []).length === 0
    )
      return showError(t('请至少选择一个渠道'));
    if (draft.scope_type === 'tag' && (draft.tags || []).length === 0)
      return showError(t('请至少选择一个标签'));
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

  const runQuery = useCallback(
    async (silent = false) => {
      if (validationErrors.length > 0) return showError(validationErrors[0]);
      setQuerying(true);
      try {
        const payload = buildQueryPayload();
        const queryKey = buildQueryKey(payload);
        const res = await API.post('/api/profit_board/query', payload);
        if (!res.data.success) throw new Error(res.data.message);
        setReport(res.data.data);
        setLastQueryKey(queryKey);
        setReportLoadedFromCache(false);
        localStorage.setItem(REPORT_CACHE_KEY, JSON.stringify(res.data.data));
        if (!silent) {
          showSuccess(t('收益看板已更新'));
        }
      } catch (error) {
        showError(error);
      } finally {
        setQuerying(false);
      }
    },
    [buildQueryPayload, t, validationErrors],
  );

  useEffect(() => {
    if (!report || validationErrors.length > 0) return;
    runQuery(true);
  }, [granularity]);

  const saveConfig = useCallback(async () => {
    if (validationErrors.length > 0) return showError(validationErrors[0]);
    setSaving(true);
    try {
      const payload = buildQueryPayload();
      const res = await API.put('/api/profit_board/config', {
        batches: batchPayload,
        upstream: payload.upstream,
        site: payload.site,
      });
      if (!res.data.success) throw new Error(res.data.message);
      showSuccess(t('配置已保存'));
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  }, [batchPayload, buildQueryPayload, t, validationErrors]);

  const exportCSV = useCallback(async () => {
    if (!reportIsFresh) return showError(t('当前结果已过期，请重新刷新数据'));
    setExporting(true);
    try {
      const res = await API.post(
        '/api/profit_board/export/csv',
        buildQueryPayload(),
        { responseType: 'blob' },
      );
      const disposition = res.headers?.['content-disposition'] || '';
      const matched = disposition.match(/filename="(.+)"/);
      downloadBlob(
        new Blob([res.data], { type: 'text/csv;charset=utf-8' }),
        matched?.[1] || 'profit-board.csv',
      );
    } catch (error) {
      showError(error);
    } finally {
      setExporting(false);
    }
  }, [buildQueryPayload, reportIsFresh, t]);

  const exportExcel = useCallback(async () => {
    if (!reportIsFresh) return showError(t('当前结果已过期，请重新刷新数据'));
    setExporting(true);
    try {
      const res = await API.post(
        '/api/profit_board/export/excel',
        buildQueryPayload(),
        { responseType: 'blob' },
      );
      const disposition = res.headers?.['content-disposition'] || '';
      const matched = disposition.match(/filename="(.+)"/);
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
  }, [buildQueryPayload, reportIsFresh, t]);

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
  const channelChartRows = useMemo(
    () =>
      aggregateBreakdownRows(
        report?.channel_breakdown || [],
        viewBatchId,
        metricKey,
      ),
    [metricKey, report?.channel_breakdown, viewBatchId],
  );
  const modelChartRows = useMemo(
    () =>
      aggregateBreakdownRows(
        report?.model_breakdown || [],
        viewBatchId,
        metricKey,
      ),
    [metricKey, report?.model_breakdown, viewBatchId],
  );
  const filteredDetailRows = useMemo(() => {
    let rows = report?.detail_rows || [];
    if (viewBatchId !== 'all')
      rows = rows.filter((row) => row.batch_id === viewBatchId);
    if (!detailFilter?.type || !detailFilter?.value) return rows;
    if (detailFilter.batchId)
      rows = rows.filter((row) => row.batch_id === detailFilter.batchId);
    if (detailFilter.type === 'channel')
      return rows.filter((row) => row.channel_name === detailFilter.value);
    if (detailFilter.type === 'model')
      return rows.filter((row) => row.model_name === detailFilter.value);
    if (detailFilter.type === 'trend') {
      return rows.filter(
        (row) =>
          dayjs
            .unix(row.created_at)
            .format(
              granularity === 'hour'
                ? 'YYYY-MM-DD HH:00'
                : granularity === 'week'
                  ? 'GGGG-[W]WW'
                  : 'YYYY-MM-DD',
            ) === detailFilter.value,
      );
    }
    return rows;
  }, [detailFilter, granularity, report?.detail_rows, viewBatchId]);

  const detailFilterText = useMemo(() => {
    if (!detailFilter?.value) return '';
    const sourceMap = {
      trend: t('图表时间桶'),
      channel: t('图表渠道'),
      model: t('图表模型'),
    };
    return `${sourceMap[detailFilter.type] || t('图表筛选')}：${detailFilter.value}`;
  }, [detailFilter, t]);

  const trendSpec = useMemo(
    () => createTrendSpec(trendRows, metricLabel, statusState?.status, t),
    [metricLabel, statusState?.status, t, trendRows],
  );
  const channelSpec = useMemo(
    () =>
      createBarSpec(
        t('渠道对比'),
        channelChartRows,
        metricLabel,
        statusState?.status,
        t,
      ),
    [channelChartRows, metricLabel, statusState?.status, t],
  );
  const modelSpec = useMemo(
    () =>
      createBarSpec(
        t('模型对比'),
        modelChartRows,
        metricLabel,
        statusState?.status,
        t,
      ),
    [metricLabel, modelChartRows, statusState?.status, t],
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
        title: t('上游费用来源'),
        dataIndex: 'upstream_cost_source',
        render: (value, row) =>
          row.upstream_cost_known ? (
            <Tag color='teal'>
              {upstreamCostSourceLabelMap[value] || value || t('未知')}
            </Tag>
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

  const summaryCards = useMemo(
    () => [
      {
        key: 'request_count',
        title: t('请求数'),
        value: report?.summary?.request_count || 0,
        icon: <Database size={16} />,
      },
      {
        key: 'actual_site_revenue_usd',
        title: t('本站实际收入'),
        value: formatMoney(
          report?.summary?.actual_site_revenue_usd,
          statusState?.status,
        ),
        icon: <CircleDollarSign size={16} />,
      },
      {
        key: 'configured_site_revenue_usd',
        title: t('本站配置收入'),
        value: formatMoney(
          report?.summary?.configured_site_revenue_usd,
          statusState?.status,
        ),
        icon: <BadgeDollarSign size={16} />,
      },
      {
        key: 'upstream_cost_usd',
        title: t('上游费用'),
        value: formatMoney(
          report?.summary?.upstream_cost_usd,
          statusState?.status,
        ),
        icon: <BarChart3 size={16} />,
      },
      {
        key: 'configured_profit_usd',
        title: t('配置利润'),
        value: formatMoney(
          report?.summary?.configured_profit_usd,
          statusState?.status,
        ),
        icon: <BadgeDollarSign size={16} />,
      },
      {
        key: 'actual_profit_usd',
        title: t('实际利润'),
        value: formatMoney(
          report?.summary?.actual_profit_usd,
          statusState?.status,
        ),
        icon: <CircleDollarSign size={16} />,
      },
      {
        key: 'configured_profit_coverage_rate',
        title: t('费用覆盖率'),
        value: formatRatio(report?.summary?.configured_profit_coverage_rate),
        icon: <Database size={16} />,
      },
      {
        key: 'missing_upstream_cost_count',
        title: t('缺失上游费用'),
        value: report?.summary?.missing_upstream_cost_count || 0,
        icon: <Database size={16} />,
      },
      {
        key: 'site_model_match_count',
        title: t('命中本站模型价格'),
        value: report?.summary?.site_model_match_count || 0,
        icon: <Database size={16} />,
      },
      {
        key: 'missing_site_pricing_count',
        title: t('缺失本站价格'),
        value: report?.summary?.missing_site_pricing_count || 0,
        icon: <Database size={16} />,
      },
      {
        key: 'returned_cost_count',
        title: t('上游返回费用条数'),
        value: report?.summary?.returned_cost_count || 0,
        icon: <Database size={16} />,
      },
      {
        key: 'manual_cost_count',
        title: t('手动回退条数'),
        value: report?.summary?.manual_cost_count || 0,
        icon: <Database size={16} />,
      },
    ],
    [report?.summary, statusState?.status, t],
  );

  const chartContent = {
    trend: trendRows.length ? (
      <VChart
        key={`trend-${actualTheme}-${viewBatchId}-${metricKey}`}
        spec={trendSpec}
        onClick={handleChartClick('trend')}
      />
    ) : (
      <Empty description={t('当前没有趋势数据')} />
    ),
    channel: channelChartRows.length ? (
      <VChart
        key={`channel-${actualTheme}-${viewBatchId}-${metricKey}`}
        spec={channelSpec}
        onClick={handleChartClick('channel')}
      />
    ) : (
      <Empty description={t('当前没有渠道数据')} />
    ),
    model: modelChartRows.length ? (
      <VChart
        key={`model-${actualTheme}-${viewBatchId}-${metricKey}`}
        spec={modelSpec}
        onClick={handleChartClick('model')}
      />
    ) : (
      <Empty description={t('当前没有模型数据')} />
    ),
  };

  return (
    <Spin spinning={loading}>
      <div className='mt-[60px] space-y-4 px-2'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <div>
            <Title heading={4} style={{ marginBottom: 4 }}>
              {t('收益看板')}
            </Title>
            <Paragraph type='tertiary' style={{ margin: 0 }}>
              {t(
                '先添加批次，再按时间范围查询日志。收益看板按日志实时重算，不读取渠道已用余额。',
              )}
            </Paragraph>
          </div>
          <Space wrap>
            <Button
              theme='solid'
              type='primary'
              icon={<RefreshCw size={16} />}
              loading={querying}
              onClick={runQuery}
            >
              {t('刷新数据')}
            </Button>
            <Button
              theme='solid'
              type='tertiary'
              icon={<Save size={16} />}
              loading={saving}
              onClick={saveConfig}
            >
              {t('保存配置')}
            </Button>
          </Space>
        </div>

        {hasCachedReport ? (
          <Banner
            type='warning'
            description={t('当前显示的是缓存结果，请重新刷新数据')}
            closeIcon={null}
          />
        ) : null}
        {reportIsStale ? (
          <Banner
            type='danger'
            description={t('当前结果已过期，请重新刷新数据')}
            closeIcon={null}
          />
        ) : null}
        {report?.warnings?.length ? (
          <div className='space-y-2'>
            {report.warnings.map((warning) => (
              <Banner
                key={warning}
                type='warning'
                description={warning}
                closeIcon={null}
              />
            ))}
          </div>
        ) : null}
        {siteConfig.use_recharge_price &&
        report?.meta?.site_price_factor_note ? (
          <Banner
            type='info'
            icon={<Info size={16} />}
            description={report.meta.site_price_factor_note}
            closeIcon={null}
          />
        ) : null}

        {report ? (
          <>
            <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-4'>
              {summaryCards.map((item) => (
                <Card key={item.key} bordered={false} bodyStyle={{ padding: 18 }}>
                  <div className='flex items-center justify-between gap-3'>
                    <div>
                      <Text type='tertiary'>{item.title}</Text>
                      <Title heading={3} style={{ margin: '8px 0 0' }}>
                        {item.value}
                      </Title>
                    </div>
                    <div className='flex h-10 w-10 items-center justify-center rounded-full bg-semi-color-fill-0'>
                      {item.icon}
                    </div>
                  </div>
                </Card>
              ))}
            </div>

            {report.batch_summaries?.length > 1 ? (
              <Card bordered={false} title={t('批次总览')}>
                <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
                  {report.batch_summaries.map((item) => (
                    <div
                      key={item.batch_id}
                      className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'
                    >
                      <div className='flex items-center justify-between gap-2'>
                        <Text strong>{item.batch_name}</Text>
                        <Tag color='blue'>
                          {item.request_count} {t('次请求')}
                        </Tag>
                      </div>
                      <div className='mt-2 grid grid-cols-2 gap-2 text-sm'>
                        <Text>
                          {t('配置利润')}：
                          {formatMoney(
                            item.configured_profit_usd,
                            statusState?.status,
                          )}
                        </Text>
                        <Text>
                          {t('实际利润')}：
                          {formatMoney(item.actual_profit_usd, statusState?.status)}
                        </Text>
                        <Text>
                          {t('本站配置收入')}：
                          {formatMoney(
                            item.configured_site_revenue_usd,
                            statusState?.status,
                          )}
                        </Text>
                        <Text>
                          {t('上游费用')}：
                          {formatMoney(item.upstream_cost_usd, statusState?.status)}
                        </Text>
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            ) : null}
          </>
        ) : null}

        <div className='grid gap-4 xl:grid-cols-[1.15fr_1fr_1fr]'>
          <Card bordered={false} title={t('查询范围与批次')}>
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
                  onChange={(value) =>
                    setDraft((prev) => ({ ...prev, name: value }))
                  }
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
                    : (options.tags || []).map((item) => ({
                        label: item,
                        value: item,
                      }))
                }
                value={
                  draft.scope_type === 'channel'
                    ? draft.channel_ids || []
                    : draft.tags || []
                }
                onChange={(value) =>
                  draft.scope_type === 'channel'
                    ? setDraft((prev) => ({
                        ...prev,
                        channel_ids: value || [],
                      }))
                    : setDraft((prev) => ({ ...prev, tags: value || [] }))
                }
                placeholder={
                  draft.scope_type === 'channel'
                    ? t('选择一个或多个渠道')
                    : t('选择一个或多个标签')
                }
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
                  <Text strong>{t('已添加批次')}</Text>
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
                            <Tag
                              color={
                                batch.scope_type === 'channel' ? 'blue' : 'cyan'
                              }
                            >
                              {batch.scope_type === 'channel'
                                ? t('渠道')
                                : t('标签聚合渠道')}
                            </Tag>
                          </Space>
                          <Text type='tertiary' className='block mt-1'>
                            {batchDigest(batch)}
                          </Text>
                        </div>
                        <Space>
                          <Button
                            icon={<Pencil size={14} />}
                            size='small'
                            type='tertiary'
                            onClick={() => editBatch(batch)}
                          >
                            {t('编辑')}
                          </Button>
                          <Button
                            icon={<Trash2 size={14} />}
                            size='small'
                            type='danger'
                            onClick={() => removeBatch(batch.id)}
                          >
                            {t('删除')}
                          </Button>
                        </Space>
                      </div>
                    ))}
                  </div>
                ) : (
                  <Empty
                    image={null}
                    description={t('还没有批次，先添加一批渠道或标签')}
                  />
                )}
              </div>
              <DatePicker
                type='dateTimeRange'
                value={dateRange}
                onChange={(value) => setDateRange(value)}
                style={{ width: '100%' }}
              />
              {validationErrors.length > 0 ? (
                <Banner
                  type='danger'
                  description={validationErrors[0]}
                  closeIcon={null}
                />
              ) : null}
            </div>
          </Card>
          <Card bordered={false} title={t('上游价格配置')}>
            <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
              <Select
                value={upstreamConfig.cost_source}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({ ...prev, cost_source: value }))
                }
                optionList={[
                  {
                    label: t('优先读上游返回费用'),
                    value: 'returned_cost_first',
                  },
                  { label: t('只读上游返回费用'), value: 'returned_cost_only' },
                  { label: t('只按手动价格计算'), value: 'manual_only' },
                ]}
              />
              <InputNumber
                min={0}
                value={upstreamConfig.fixed_amount}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    fixed_amount: clampNumber(value),
                  }))
                }
                suffix={t('固定 / 次')}
              />
              <InputNumber
                min={0}
                value={upstreamConfig.input_price}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    input_price: clampNumber(value),
                  }))
                }
                suffix='USD / 1M 输入'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.output_price}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    output_price: clampNumber(value),
                  }))
                }
                suffix='USD / 1M 输出'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.cache_read_price}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    cache_read_price: clampNumber(value),
                  }))
                }
                suffix='USD / 1M 缓存读'
              />
              <InputNumber
                min={0}
                value={upstreamConfig.cache_creation_price}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    cache_creation_price: clampNumber(value),
                  }))
                }
                suffix='USD / 1M 缓存写'
              />
            </div>
            <Text type='tertiary' className='block mt-3'>
              {t(
                '优先读取日志里的上游真实费用；如果上游没返回，再按这里的手动价格回退。',
              )}
            </Text>
          </Card>

          <Card bordered={false} title={t('本站价格配置')}>
            <div className='space-y-3'>
              <Select
                value={siteConfig.pricing_mode}
                onChange={(value) =>
                  setSiteConfig((prev) => ({ ...prev, pricing_mode: value }))
                }
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
                      onChange={(value) =>
                        setSiteConfig((prev) => ({
                          ...prev,
                          model_names: value || [],
                        }))
                      }
                      optionList={(options.local_models || []).map((item) => ({
                        label: item.model_name,
                        value: item.model_name,
                      }))}
                      placeholder={t('选择一个或多个本站模型')}
                    />
                    <Select
                      value={siteConfig.group}
                      onChange={(value) =>
                        setSiteConfig((prev) => ({ ...prev, group: value }))
                      }
                      optionList={[
                        { label: t('自动取最低分组倍率'), value: '' },
                        ...(options.groups || []).map((item) => ({
                          label: item,
                          value: item,
                        })),
                      ]}
                    />
                  </div>
                  <div className='flex items-center justify-between rounded-lg bg-semi-color-fill-0 px-3 py-2'>
                    <div>
                      <Text strong>{t('按充值价格读取')}</Text>
                      <Text type='tertiary' className='block'>
                        {t(
                          '开启后按充值倍率重算；如果倍率正好是 1，就会和原价一样。',
                        )}
                      </Text>
                    </div>
                    <Switch
                      checked={siteConfig.use_recharge_price}
                      onChange={(checked) =>
                        setSiteConfig((prev) => ({
                          ...prev,
                          use_recharge_price: checked,
                        }))
                      }
                    />
                  </div>
                </>
              ) : null}
              <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
                <InputNumber
                  min={0}
                  value={siteConfig.fixed_amount}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({
                      ...prev,
                      fixed_amount: clampNumber(value),
                    }))
                  }
                  suffix={t('固定 / 次')}
                />
                <InputNumber
                  min={0}
                  value={siteConfig.input_price}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({
                      ...prev,
                      input_price: clampNumber(value),
                    }))
                  }
                  suffix='USD / 1M 输入'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.output_price}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({
                      ...prev,
                      output_price: clampNumber(value),
                    }))
                  }
                  suffix='USD / 1M 输出'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.cache_read_price}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({
                      ...prev,
                      cache_read_price: clampNumber(value),
                    }))
                  }
                  suffix='USD / 1M 缓存读'
                />
                <InputNumber
                  min={0}
                  value={siteConfig.cache_creation_price}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({
                      ...prev,
                      cache_creation_price: clampNumber(value),
                    }))
                  }
                  suffix='USD / 1M 缓存写'
                />
              </div>
              <Text type='tertiary' className='block'>
                {t(
                  '本站实际收入 = 当时真实扣费；本站配置收入 = 按你当前收益看板配置重新模拟出来的收入。',
                )}
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
                  value={metricKey}
                  onChange={setMetricKey}
                  optionList={metricOptions.map((item) => ({
                    label: t(item.label),
                    value: item.value,
                  }))}
                  style={{ width: 170 }}
                />
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
                  ]}
                  style={{ width: 120 }}
                />
                <Button
                  type='tertiary'
                  icon={<Filter size={14} />}
                  disabled={!detailFilter?.value}
                  onClick={() => setDetailFilter(null)}
                >
                  {t('清除图表筛选')}
                </Button>
                <Button
                  type='tertiary'
                  icon={<ArrowDownToLine size={14} />}
                  onClick={exportCSV}
                  loading={exporting}
                  disabled={!reportIsFresh}
                >
                  CSV
                </Button>
                <Button
                  type='tertiary'
                  onClick={exportExcel}
                  loading={exporting}
                  disabled={!reportIsFresh}
                >
                  Excel
                </Button>
              </Space>
            </div>
          }
        >
          <Tabs activeKey={chartTab} onChange={setChartTab} type='line'>
            <Tabs.TabPane tab={t('趋势')} itemKey='trend' />
            <Tabs.TabPane tab={t('渠道')} itemKey='channel' />
            <Tabs.TabPane tab={t('模型')} itemKey='model' />
          </Tabs>
          <div className='min-h-[360px]'>
            {report ? (
              chartContent[chartTab]
            ) : (
              <Empty description={t('先添加批次并刷新数据')} />
            )}
          </div>
        </Card>

        <Card
          bordered={false}
          title={
            <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
              <span>{t('对账明细')}</span>
              <Space wrap>
                {detailFilterText ? (
                  <Tag color='light-blue'>{detailFilterText}</Tag>
                ) : null}
                <Text type='tertiary'>
                  {t('已展示 {{count}} 条', {
                    count: filteredDetailRows.length,
                  })}
                  {report?.detail_truncated ? ` · ${t('结果已截断')}` : ''}
                </Text>
              </Space>
            </div>
          }
        >
          <Table
            columns={detailColumns}
            dataSource={filteredDetailRows}
            rowKey='id'
            pagination={{ pageSize: isMobile ? 8 : 12 }}
            empty={t('暂无明细')}
          />
        </Card>
      </div>
    </Spin>
  );
};

export default ProfitBoardPage;
