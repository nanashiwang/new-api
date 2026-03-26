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
import {
  ArrowDownToLine,
  BadgeDollarSign,
  BarChart3,
  CircleDollarSign,
  Database,
  Filter,
  RefreshCw,
  Save,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../context/Status';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const STORAGE_KEY = 'profit-board:state';
const REPORT_CACHE_KEY = 'profit-board:report';
const { Text, Title } = Typography;

const buildSelectionSignature = (scopeType, values) => {
  const list = [...(values || [])]
    .map((item) => item?.toString().trim())
    .filter(Boolean)
    .sort();
  return `${scopeType}:${list.join('|')}`;
};

const createDefaultUpstreamConfig = () => ({
  cost_source: 'returned_cost_first',
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

const createDefaultState = () => {
  const end = new Date();
  const start = dayjs(end).subtract(7, 'day').toDate();
  return {
    scopeType: 'channel',
    selectedChannels: [],
    selectedTags: [],
    dateRange: [start, end],
    granularity: 'day',
    chartTab: 'trend',
    metricKey: 'configured_profit_usd',
    detailFilter: null,
    upstreamConfig: createDefaultUpstreamConfig(),
    siteConfig: createDefaultSiteConfig(),
  };
};

const normalizeRestoredState = (state) => {
  const next = { ...createDefaultState(), ...(state || {}) };
  const [start, end] = next.dateRange || [];
  next.dateRange = [
    start ? new Date(start) : createDefaultState().dateRange[0],
    end ? new Date(end) : createDefaultState().dateRange[1],
  ];
  next.selectedChannels = (next.selectedChannels || []).map((item) =>
    item.toString(),
  );
  next.selectedTags = next.selectedTags || [];
  next.upstreamConfig = {
    ...createDefaultUpstreamConfig(),
    ...(next.upstreamConfig || {}),
  };
  next.siteConfig = {
    ...createDefaultSiteConfig(),
    ...(next.siteConfig || {}),
  };
  return next;
};

const safeParse = (raw, fallback) => {
  try {
    return raw ? JSON.parse(raw) : fallback;
  } catch (error) {
    return fallback;
  }
};

const clampNumber = (value) => {
  const next = Number(value || 0);
  if (!Number.isFinite(next) || next < 0) {
    return 0;
  }
  return next;
};

const metricOptions = [
  { value: 'configured_profit_usd', label: '配置利润' },
  { value: 'actual_site_revenue_usd', label: '本站实际收入' },
  { value: 'configured_site_revenue_usd', label: '本站配置收入' },
  { value: 'upstream_cost_usd', label: '上游费用' },
];

const getDisplayCurrency = (status) => {
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

const createTrendSpec = (report, metricKey, metricLabel, status, t) => ({
  type: 'line',
  data: [
    {
      id: 'trend',
      values: (report?.timeseries || []).map((item) => ({
        bucket: item.bucket,
        value: item[metricKey] || 0,
      })),
    },
  ],
  xField: 'bucket',
  yField: 'value',
  point: { visible: true, style: { size: 6, fill: '#0f766e' } },
  line: { style: { curveType: 'monotone', lineWidth: 3, stroke: '#0f766e' } },
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
        {
          key: metricLabel,
          value: (datum) => formatMoney(datum.value, status),
        },
      ],
    },
  },
});

const createBreakdownSpec = (title, rows, metricKey, metricLabel, status) => ({
  type: 'bar',
  data: [
    {
      id: 'breakdown',
      values: (rows || []).slice(0, 12).map((item) => ({
        label: item.label,
        value: item[metricKey] || 0,
      })),
    },
  ],
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
  bar: { style: { fill: '#0f172a' } },
  title: { visible: true, text: title, subtext: metricLabel },
  tooltip: {
    mark: {
      content: [
        { key: 'label', value: (datum) => datum.label },
        {
          key: metricLabel,
          value: (datum) => formatMoney(datum.value, status),
        },
      ],
    },
  },
});

const buildExcelHTML = (report, status, t) => {
  const rows = report?.detail_rows || [];
  const summary = report?.summary || {};
  const summaryItems = [
    [t('请求数'), summary.request_count || 0],
    [t('本站实际收入'), formatMoney(summary.actual_site_revenue_usd, status)],
    [
      t('本站配置收入'),
      formatMoney(summary.configured_site_revenue_usd, status),
    ],
    [t('上游费用'), formatMoney(summary.upstream_cost_usd, status)],
    [t('配置利润'), formatMoney(summary.configured_profit_usd, status)],
  ];

  const escape = (value) =>
    String(value ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;');

  return `
    <html>
      <head>
        <meta charset="utf-8" />
      </head>
      <body>
        <table border="1">
          <tr><th colspan="2">${escape(t('收益看板总览'))}</th></tr>
          ${summaryItems
            .map(
              ([label, value]) =>
                `<tr><td>${escape(label)}</td><td>${escape(value)}</td></tr>`,
            )
            .join('')}
        </table>
        <br />
        <table border="1">
          <tr>
            <th>${escape(t('时间'))}</th>
            <th>${escape(t('渠道'))}</th>
            <th>${escape(t('模型'))}</th>
            <th>${escape(t('本站实际收入'))}</th>
            <th>${escape(t('本站配置收入'))}</th>
            <th>${escape(t('上游费用'))}</th>
            <th>${escape(t('配置利润'))}</th>
          </tr>
          ${rows
            .map(
              (row) => `
              <tr>
                <td>${escape(timestamp2string(row.created_at))}</td>
                <td>${escape(row.channel_name || row.channel_id)}</td>
                <td>${escape(row.model_name)}</td>
                <td>${escape(formatMoney(row.actual_site_revenue_usd, status))}</td>
                <td>${escape(formatMoney(row.configured_site_revenue_usd, status))}</td>
                <td>${escape(formatMoney(row.upstream_cost_usd, status))}</td>
                <td>${escape(formatMoney(row.configured_profit_usd, status))}</td>
              </tr>`,
            )
            .join('')}
        </table>
      </body>
    </html>
  `;
};

const ProfitBoardPage = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [statusState] = useContext(StatusContext);
  const initialState = useMemo(
    () =>
      normalizeRestoredState(safeParse(localStorage.getItem(STORAGE_KEY), {})),
    [],
  );

  const [options, setOptions] = useState({
    channels: [],
    tags: [],
    groups: [],
    local_models: [],
  });
  const [scopeType, setScopeType] = useState(initialState.scopeType);
  const [selectedChannels, setSelectedChannels] = useState(
    initialState.selectedChannels || [],
  );
  const [selectedTags, setSelectedTags] = useState(
    initialState.selectedTags || [],
  );
  const [dateRange, setDateRange] = useState(
    initialState.dateRange || createDefaultState().dateRange,
  );
  const [granularity, setGranularity] = useState(
    initialState.granularity || 'day',
  );
  const [chartTab, setChartTab] = useState(initialState.chartTab || 'trend');
  const [metricKey, setMetricKey] = useState(
    initialState.metricKey || 'configured_profit_usd',
  );
  const [detailFilter, setDetailFilter] = useState(
    initialState.detailFilter || null,
  );
  const [upstreamConfig, setUpstreamConfig] = useState(
    initialState.upstreamConfig || createDefaultUpstreamConfig(),
  );
  const [siteConfig, setSiteConfig] = useState(
    initialState.siteConfig || createDefaultSiteConfig(),
  );
  const [loading, setLoading] = useState(true);
  const [querying, setQuerying] = useState(false);
  const [saving, setSaving] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [report, setReport] = useState(() =>
    safeParse(localStorage.getItem(REPORT_CACHE_KEY), null),
  );

  const selectionValues =
    scopeType === 'channel' ? selectedChannels : selectedTags;
  const selectionSignature = useMemo(
    () => buildSelectionSignature(scopeType, selectionValues),
    [scopeType, selectionValues],
  );

  const metricLabel = useMemo(
    () =>
      metricOptions.find((item) => item.value === metricKey)?.label ||
      t('配置利润'),
    [metricKey, t],
  );

  const validationErrors = useMemo(() => {
    const errors = [];
    if (scopeType === 'channel' && selectedChannels.length === 0) {
      errors.push(t('请至少选择一个渠道'));
    }
    if (scopeType === 'tag' && selectedTags.length === 0) {
      errors.push(t('请至少选择一个标签'));
    }
    if (!Array.isArray(dateRange) || !dateRange[0] || !dateRange[1]) {
      errors.push(t('请选择完整的时间范围'));
    }
    if (
      siteConfig.pricing_mode === 'site_model' &&
      (siteConfig.model_names || []).length === 0
    ) {
      errors.push(t('读取本站模型价格时，至少选择一个模型'));
    }
    return errors;
  }, [dateRange, scopeType, selectedChannels, selectedTags, siteConfig, t]);

  const persistState = useCallback(
    (next) => {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          scopeType,
          selectedChannels,
          selectedTags,
          dateRange,
          granularity,
          chartTab,
          metricKey,
          detailFilter,
          upstreamConfig,
          siteConfig,
          ...next,
        }),
      );
    },
    [
      chartTab,
      dateRange,
      detailFilter,
      granularity,
      metricKey,
      scopeType,
      selectedChannels,
      selectedTags,
      siteConfig,
      upstreamConfig,
    ],
  );

  const loadOptions = useCallback(async () => {
    const res = await API.get('/api/profit_board/options');
    if (!res.data.success) {
      throw new Error(res.data.message);
    }
    setOptions(res.data.data);
  }, []);

  const loadConfig = useCallback(async () => {
    if (!selectionSignature || !selectionValues.length) return;
    const params =
      scopeType === 'channel'
        ? { scope_type: scopeType, channel_ids: selectedChannels.join(',') }
        : { scope_type: scopeType, tags: selectedTags.join(',') };
    const res = await API.get('/api/profit_board/config', { params });
    if (!res.data.success) {
      throw new Error(res.data.message);
    }
    const config = res.data.data?.config;
    if (config) {
      setUpstreamConfig({
        ...createDefaultUpstreamConfig(),
        ...config.upstream,
      });
      setSiteConfig({ ...createDefaultSiteConfig(), ...config.site });
    }
  }, [
    scopeType,
    selectedChannels,
    selectedTags,
    selectionSignature,
    selectionValues.length,
  ]);

  const buildQueryPayload = useCallback(
    () => ({
      selection: {
        scope_type: scopeType,
        channel_ids: selectedChannels.map((item) => Number(item)),
        tags: selectedTags,
      },
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
      start_timestamp: Math.floor(new Date(dateRange[0]).getTime() / 1000),
      end_timestamp: Math.floor(new Date(dateRange[1]).getTime() / 1000),
      granularity: granularity,
    }),
    [
      dateRange,
      granularity,
      scopeType,
      selectedChannels,
      selectedTags,
      siteConfig,
      upstreamConfig,
    ],
  );

  const runQuery = useCallback(async () => {
    if (validationErrors.length > 0) {
      showError(validationErrors[0]);
      return;
    }
    setQuerying(true);
    try {
      const payload = buildQueryPayload();
      const res = await API.post('/api/profit_board/query', payload);
      if (!res.data.success) {
        throw new Error(res.data.message);
      }
      setReport(res.data.data);
      localStorage.setItem(REPORT_CACHE_KEY, JSON.stringify(res.data.data));
      showSuccess(t('收益看板已更新'));
    } catch (error) {
      showError(error);
    } finally {
      setQuerying(false);
    }
  }, [buildQueryPayload, t, validationErrors]);

  const saveConfig = useCallback(async () => {
    if (validationErrors.length > 0) {
      showError(validationErrors[0]);
      return;
    }
    setSaving(true);
    try {
      const res = await API.put('/api/profit_board/config', {
        selection: {
          scope_type: scopeType,
          channel_ids: selectedChannels.map((item) => Number(item)),
          tags: selectedTags,
        },
        upstream: buildQueryPayload().upstream,
        site: buildQueryPayload().site,
      });
      if (!res.data.success) {
        throw new Error(res.data.message);
      }
      showSuccess(t('配置已保存'));
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  }, [
    buildQueryPayload,
    scopeType,
    selectedChannels,
    selectedTags,
    t,
    validationErrors,
  ]);

  const exportCSV = useCallback(async () => {
    if (!report) return;
    setExporting(true);
    try {
      const res = await API.post(
        '/api/profit_board/export/csv',
        buildQueryPayload(),
        {
          responseType: 'blob',
        },
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
  }, [buildQueryPayload, report]);

  const exportExcel = useCallback(() => {
    if (!report) return;
    const html = buildExcelHTML(report, statusState?.status, t);
    downloadBlob(
      new Blob([html], { type: 'application/vnd.ms-excel;charset=utf-8' }),
      `profit-board-${dayjs().format('YYYYMMDD-HHmmss')}.xls`,
    );
  }, [report, statusState?.status, t]);

  useEffect(() => {
    const bootstrap = async () => {
      setLoading(true);
      try {
        await loadOptions();
        if (selectionValues.length > 0) {
          await loadConfig();
        }
      } catch (error) {
        showError(error);
      } finally {
        setLoading(false);
      }
    };
    bootstrap();
  }, [loadConfig, loadOptions, selectionValues.length]);

  useEffect(() => {
    persistState();
  }, [persistState]);

  useEffect(() => {
    if (selectionValues.length > 0) {
      loadConfig().catch(showError);
    }
  }, [loadConfig, selectionSignature, selectionValues.length]);

  const trendSpec = useMemo(
    () =>
      createTrendSpec(report, metricKey, metricLabel, statusState?.status, t),
    [metricKey, metricLabel, report, statusState?.status, t],
  );
  const channelSpec = useMemo(
    () =>
      createBreakdownSpec(
        t('渠道对比'),
        report?.channel_breakdown,
        metricKey,
        metricLabel,
        statusState?.status,
      ),
    [metricKey, metricLabel, report?.channel_breakdown, statusState?.status, t],
  );
  const modelSpec = useMemo(
    () =>
      createBreakdownSpec(
        t('模型对比'),
        report?.model_breakdown,
        metricKey,
        metricLabel,
        statusState?.status,
      ),
    [metricKey, metricLabel, report?.model_breakdown, statusState?.status, t],
  );

  const filteredDetailRows = useMemo(() => {
    const rows = report?.detail_rows || [];
    if (!detailFilter?.type || !detailFilter?.value) {
      return rows;
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
          dayjs
            .unix(row.created_at)
            .format(
              granularity === 'hour' ? 'YYYY-MM-DD HH:00' : 'YYYY-MM-DD',
            ) === detailFilter.value,
      );
    }
    return rows;
  }, [detailFilter, granularity, report?.detail_rows]);

  const handleChartClick = useCallback(
    (type) => (event) => {
      const label = event?.datum?.label || event?.datum?.bucket;
      if (!label) return;
      setDetailFilter({ type, value: label });
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
        title: t('上游费用'),
        dataIndex: 'upstream_cost_usd',
        render: (value, row) =>
          row.upstream_cost_known ? (
            <Space>
              <span>{formatMoney(value, statusState?.status)}</span>
              <Tag color='white'>{row.upstream_cost_source || t('未知')}</Tag>
            </Space>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('配置利润'),
        dataIndex: 'configured_profit_usd',
        render: (value, row) => (
          <Text
            style={value >= 0 ? { color: '#0f766e' } : { color: '#dc2626' }}
          >
            {row.upstream_cost_known
              ? formatMoney(value, statusState?.status)
              : '-'}
          </Text>
        ),
      },
    ],
    [statusState?.status, t],
  );

  const summaryCards = [
    {
      title: t('请求数'),
      value: report?.summary?.request_count || 0,
      icon: Database,
      raw: true,
    },
    {
      title: t('本站实际收入'),
      value: report?.summary?.actual_site_revenue_usd || 0,
      icon: CircleDollarSign,
    },
    {
      title: t('本站配置收入'),
      value: report?.summary?.configured_site_revenue_usd || 0,
      icon: BadgeDollarSign,
    },
    {
      title: t('上游费用'),
      value: report?.summary?.upstream_cost_usd || 0,
      icon: ArrowDownToLine,
    },
    {
      title: t('配置利润'),
      value: report?.summary?.configured_profit_usd || 0,
      icon: BarChart3,
      emphasize: true,
    },
  ];

  if (loading) {
    return (
      <div className='mt-[60px] px-2'>
        <div className='min-h-[60vh] flex items-center justify-center'>
          <Spin size='large' />
        </div>
      </div>
    );
  }

  const selectionOptions =
    scopeType === 'channel'
      ? options.channels.map((item) => ({
          label: `${item.name} (#${item.id})${item.tag ? ` / ${item.tag}` : ''}`,
          value: item.id.toString(),
        }))
      : options.tags.map((item) => ({ label: item, value: item }));

  const chartContent = {
    trend: <VChart spec={trendSpec} onClick={handleChartClick('trend')} />,
    channel: (
      <VChart spec={channelSpec} onClick={handleChartClick('channel')} />
    ),
    model: <VChart spec={modelSpec} onClick={handleChartClick('model')} />,
  };

  return (
    <div className='mt-[60px] px-2 pb-8'>
      <div className='rounded-[28px] border border-[rgba(15,23,42,0.08)] bg-[linear-gradient(135deg,rgba(15,23,42,0.96),rgba(15,118,110,0.92),rgba(217,119,6,0.88))] px-5 py-5 text-white shadow-[0_24px_60px_rgba(15,23,42,0.22)]'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between'>
          <div>
            <Title heading={3} style={{ color: 'white', marginBottom: 6 }}>
              {t('收益看板')}
            </Title>
            <Text style={{ color: 'rgba(255,255,255,0.82)' }}>
              {t(
                '按渠道或标签聚合渠道读取调用日志，用真实日志消费对比上游返回费用或你配置的上下游价格。',
              )}
            </Text>
          </div>
          <Space wrap>
            <Button
              theme='solid'
              type='secondary'
              icon={<RefreshCw size={16} />}
              onClick={runQuery}
              loading={querying}
            >
              {t('刷新数据')}
            </Button>
            <Button
              theme='solid'
              type='tertiary'
              icon={<Save size={16} />}
              onClick={saveConfig}
              loading={saving}
            >
              {t('保存配置')}
            </Button>
          </Space>
        </div>
      </div>

      <div className='mt-4 grid gap-4 xl:grid-cols-[1.05fr_1fr_1fr]'>
        <Card
          title={t('查询范围')}
          bordered={false}
          bodyStyle={{ paddingTop: 12 }}
        >
          <div className='space-y-4'>
            <Radio.Group
              type='button'
              value={scopeType}
              onChange={(event) => setScopeType(event.target.value)}
            >
              <Radio value='channel'>{t('渠道')}</Radio>
              <Radio value='tag'>{t('标签聚合渠道')}</Radio>
            </Radio.Group>
            <Select
              multiple
              filter
              maxTagCount={isMobile ? 2 : 4}
              optionList={selectionOptions}
              value={scopeType === 'channel' ? selectedChannels : selectedTags}
              onChange={(value) => {
                if (scopeType === 'channel') {
                  setSelectedChannels(value || []);
                } else {
                  setSelectedTags(value || []);
                }
              }}
              placeholder={
                scopeType === 'channel'
                  ? t('选择一个或多个渠道')
                  : t('选择一个或多个标签')
              }
              style={{ width: '100%' }}
            />
            <DatePicker
              type='dateTimeRange'
              value={dateRange}
              onChange={(value) => setDateRange(value)}
              style={{ width: '100%' }}
            />
            <div className='flex items-center justify-between gap-3'>
              <Select
                value={granularity}
                onChange={setGranularity}
                optionList={[
                  { label: t('按小时'), value: 'hour' },
                  { label: t('按天'), value: 'day' },
                ]}
                style={{ width: 140 }}
              />
              <Tag color='light-blue'>{selectionSignature}</Tag>
            </div>
            {validationErrors.length > 0 && (
              <Banner
                type='danger'
                description={validationErrors[0]}
                closeIcon={null}
              />
            )}
          </div>
        </Card>
        <Card
          title={t('上游价格配置')}
          bordered={false}
          bodyStyle={{ paddingTop: 12 }}
        >
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
          <Text type='tertiary' size='small' className='mt-3 block'>
            {t('如果上游没有返回真实费用，会按这里的手动价格回退。')}
          </Text>
        </Card>
        <Card
          title={t('本站价格配置')}
          bordered={false}
          bodyStyle={{ paddingTop: 12 }}
        >
          <div className='space-y-3'>
            <Select
              value={siteConfig.pricing_mode}
              onChange={(value) =>
                setSiteConfig((prev) => ({ ...prev, pricing_mode: value }))
              }
              optionList={[
                { label: t('手动价格'), value: 'manual' },
                { label: t('读取本站模型价格'), value: 'site_model' },
              ]}
            />
            {siteConfig.pricing_mode === 'site_model' && (
              <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
                <Select
                  multiple
                  filter
                  optionList={options.local_models.map((item) => ({
                    label: item.model_name,
                    value: item.model_name,
                  }))}
                  value={siteConfig.model_names || []}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({
                      ...prev,
                      model_names: value || [],
                    }))
                  }
                  placeholder={t('选择一个或多个本站模型')}
                />
                <Select
                  value={siteConfig.group}
                  onChange={(value) =>
                    setSiteConfig((prev) => ({ ...prev, group: value }))
                  }
                  optionList={[
                    { label: t('自动取最低分组倍率'), value: '' },
                  ].concat(
                    (options.groups || []).map((item) => ({
                      label: item,
                      value: item,
                    })),
                  )}
                />
                <div className='md:col-span-2 flex items-center justify-between rounded-2xl border border-[rgba(15,23,42,0.08)] px-4 py-3'>
                  <div>
                    <Text strong>{t('按充值价格读取')}</Text>
                    <Text type='tertiary' size='small' className='block'>
                      {t(
                        '读取模型广场展示价格对应的充值换算值，而不是原始美元价。',
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
              </div>
            )}
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
            <Text type='tertiary' size='small'>
              {t('读取本站模型价格时，未命中模型会回退到手动价格。')}
            </Text>
          </div>
        </Card>
      </div>

      <div className='mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-5'>
        {summaryCards.map((item) => {
          const Icon = item.icon;
          return (
            <Card
              key={item.title}
              bordered={false}
              bodyStyle={{ padding: 18 }}
              className={
                item.emphasize ? 'ring-1 ring-[rgba(15,118,110,0.16)]' : ''
              }
            >
              <div className='flex items-start justify-between gap-3'>
                <div>
                  <Text type='tertiary'>{item.title}</Text>
                  <Title
                    heading={isMobile ? 5 : 4}
                    style={{ margin: '8px 0 0' }}
                  >
                    {item.raw
                      ? item.value
                      : formatMoney(item.value, statusState?.status)}
                  </Title>
                </div>
                <div className='rounded-2xl bg-[rgba(15,118,110,0.08)] p-3 text-[#0f766e]'>
                  <Icon size={18} />
                </div>
              </div>
            </Card>
          );
        })}
      </div>

      {report?.warnings?.length > 0 && (
        <div className='mt-4 space-y-2'>
          {report.warnings.map((item) => (
            <Banner
              key={item}
              type='warning'
              description={item}
              closeIcon={null}
            />
          ))}
        </div>
      )}

      <Card
        className='mt-4'
        bordered={false}
        title={
          <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
            <Space wrap>
              <BarChart3 size={16} />
              <span>{t('收益图表')}</span>
            </Space>
            <Space wrap>
              <Select
                value={metricKey}
                onChange={setMetricKey}
                optionList={metricOptions.map((item) => ({
                  ...item,
                  label: t(item.label),
                }))}
                style={{ width: 180 }}
              />
              <Button
                type='tertiary'
                icon={<Filter size={16} />}
                onClick={() => setDetailFilter(null)}
              >
                {t('清除下钻')}
              </Button>
              <Button
                type='tertiary'
                icon={<ArrowDownToLine size={16} />}
                onClick={exportCSV}
                loading={exporting}
              >
                CSV
              </Button>
              <Button type='tertiary' onClick={exportExcel}>
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
            <Empty description={t('先选择条件并刷新数据')} />
          )}
        </div>
      </Card>

      <Card
        className='mt-4'
        bordered={false}
        title={
          <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
            <span>{t('对账明细')}</span>
            <Space wrap>
              {detailFilter?.value && (
                <Tag color='light-blue'>{detailFilter.value}</Tag>
              )}
              <Text type='tertiary'>
                {t('已展示 {{count}} 条', { count: filteredDetailRows.length })}
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
  );
};

export default ProfitBoardPage;
