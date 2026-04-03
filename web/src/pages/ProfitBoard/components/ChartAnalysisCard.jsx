import React, { useState, useMemo } from 'react';
import {
  Button,
  Card,
  Collapse,
  Empty,
  InputNumber,
  Select,
  Space,
  Tag,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { Filter, RefreshCw, Settings2, X } from 'lucide-react';

const { Text } = Typography;

const FilterTag = ({ label, value, onClear }) => (
  <Tag
    color='blue'
    closable
    onClose={onClear}
    className='flex items-center gap-1'
  >
    <span className='text-semi-color-text-2'>{label}:</span>
    <span>{value}</span>
  </Tag>
);

const ChartAnalysisCard = ({
  analysisMode,
  setAnalysisMode,
  metricKey,
  setMetricKey,
  metricOptions,
  viewBatchId,
  setViewBatchId,
  batchSummaryOptions,
  granularity,
  setGranularity,
  customIntervalMinutes,
  setCustomIntervalMinutes,
  detailFilter,
  clearDetailFilter,
  runQuery,
  querying,
  chartTab,
  setChartTab,
  report,
  chartContent,
  t,
}) => {
  const [filterExpanded, setFilterExpanded] = useState(false);

  const activeFilters = useMemo(() => {
    const filters = [];
    filters.push({
      key: 'analysisMode',
      label: t('分析模式'),
      value:
        analysisMode === 'business_compare' ? t('经营对比') : t('单指标分析'),
      onClear: () => setAnalysisMode('business_compare'),
    });
    if (analysisMode === 'single_metric') {
      const metric = metricOptions.find((m) => m.value === metricKey);
      filters.push({
        key: 'metricKey',
        label: t('指标'),
        value: metric ? t(metric.label) : metricKey,
        onClear: () => setMetricKey('configured_profit_usd'),
      });
    }
    const batch = batchSummaryOptions.find((b) => b.value === viewBatchId);
    if (batch && batch.value !== 'all') {
      filters.push({
        key: 'viewBatchId',
        label: t('组合'),
        value: batch.label,
        onClear: () => setViewBatchId('all'),
      });
    }
    const granularityLabels = {
      hour: t('按小时'),
      day: t('按天'),
      week: t('按周'),
      month: t('按月'),
      custom: t('自定义'),
    };
    if (granularity !== 'day') {
      filters.push({
        key: 'granularity',
        label: t('粒度'),
        value:
          granularity === 'custom'
            ? `${t('自定义')} ${customIntervalMinutes}${t('分钟')}`
            : granularityLabels[granularity],
        onClear: () => setGranularity('day'),
      });
    }
    if (detailFilter?.value) {
      const typeLabels = {
        trend: t('时间桶'),
        channel: t('渠道'),
        model: t('模型'),
      };
      filters.push({
        key: 'detailFilter',
        label: t('图表筛选'),
        value: `${typeLabels[detailFilter.type] || t('筛选')}: ${detailFilter.value}`,
        onClear: clearDetailFilter,
      });
    }
    return filters;
  }, [
    analysisMode,
    metricKey,
    viewBatchId,
    granularity,
    customIntervalMinutes,
    detailFilter,
    batchSummaryOptions,
    metricOptions,
    t,
    setAnalysisMode,
    setMetricKey,
    setViewBatchId,
    setGranularity,
    clearDetailFilter,
  ]);

  const hasActiveFilters = activeFilters.length > 0;

  const clearAllFilters = () => {
    setAnalysisMode('business_compare');
    setMetricKey('configured_profit_usd');
    setViewBatchId('all');
    setGranularity('day');
    if (detailFilter?.value) clearDetailFilter();
  };

  return (
    <Card
      bordered={false}
      className='rounded-xl'
      bodyStyle={{ paddingTop: 12 }}
      title={
        <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <div className='flex items-center gap-2'>
            <Filter size={16} />
            <span className='font-medium'>{t('图表分析')}</span>
          </div>
          <Space wrap>
            <Button
              type='tertiary'
              icon={<Settings2 size={14} />}
              onClick={() => setFilterExpanded(!filterExpanded)}
            >
              {filterExpanded ? t('收起筛选') : t('展开筛选')}
            </Button>
            <Button
              type='primary'
              theme='solid'
              icon={<RefreshCw size={14} />}
              loading={querying}
              onClick={runQuery}
            >
              {t('刷新')}
            </Button>
          </Space>
        </div>
      }
    >
      <Collapse
        activeKey={filterExpanded ? ['filters'] : []}
        onChange={() => setFilterExpanded(!filterExpanded)}
      >
        <Collapse.Panel header={t('筛选条件')} itemKey='filters'>
          <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-4 xl:grid-cols-6'>
            <div>
              <Text type='tertiary' size='small' className='mb-1.5 block'>
                {t('分析模式')}
              </Text>
              <Select
                value={analysisMode}
                onChange={setAnalysisMode}
                optionList={[
                  { label: t('经营对比'), value: 'business_compare' },
                  { label: t('单指标分析'), value: 'single_metric' },
                ]}
                style={{ width: '100%' }}
              />
            </div>
            {analysisMode === 'single_metric' && (
              <div>
                <Text type='tertiary' size='small' className='mb-1.5 block'>
                  {t('指标')}
                </Text>
                <Select
                  value={metricKey}
                  onChange={setMetricKey}
                  optionList={metricOptions.map((item) => ({
                    label: t(item.label),
                    value: item.value,
                  }))}
                  style={{ width: '100%' }}
                />
              </div>
            )}
            <div>
              <Text type='tertiary' size='small' className='mb-1.5 block'>
                {t('组合')}
              </Text>
              <Select
                value={viewBatchId}
                onChange={setViewBatchId}
                optionList={batchSummaryOptions}
                style={{ width: '100%' }}
              />
            </div>
            <div>
              <Text type='tertiary' size='small' className='mb-1.5 block'>
                {t('粒度')}
              </Text>
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
                style={{ width: '100%' }}
              />
            </div>
            {granularity === 'custom' && (
              <div>
                <Text type='tertiary' size='small' className='mb-1.5 block'>
                  {t('自定义间隔')}
                </Text>
                <InputNumber
                  min={1}
                  value={customIntervalMinutes}
                  onChange={(value) =>
                    setCustomIntervalMinutes(Math.max(Number(value || 1), 1))
                  }
                  suffix={t('分钟')}
                  style={{ width: '100%' }}
                />
              </div>
            )}
          </div>
        </Collapse.Panel>
      </Collapse>

      {hasActiveFilters && (
        <div className='mt-4 flex flex-wrap items-center gap-2'>
          <Text type='tertiary' size='small'>
            {t('当前筛选')}:
          </Text>
          {activeFilters.map((filter) => (
            <FilterTag
              key={filter.key}
              label={filter.label}
              value={filter.value}
              onClear={filter.onClear}
            />
          ))}
          {hasActiveFilters && (
            <Button size='small' type='tertiary' onClick={clearAllFilters}>
              {t('清除全部')}
            </Button>
          )}
        </div>
      )}

      <Text type='tertiary' size='small' className='mt-4 mb-3 block'>
        {analysisMode === 'business_compare'
          ? t(
              '默认直接看本站配置收入和上游费用的金额对比，更适合判断渠道是否赚钱。',
            )
          : metricKey === 'remote_observed_cost_usd'
            ? t('远端观测消耗目前只支持趋势图，渠道和模型分布暂不拆分。')
            : t('单指标分析适合单独观察某一个指标在时间、渠道或模型上的变化。')}
      </Text>

      <Tabs activeKey={chartTab} onChange={setChartTab} type='line'>
        <Tabs.TabPane tab={t('趋势')} itemKey='trend' />
        <Tabs.TabPane tab={t('渠道')} itemKey='channel' />
        <Tabs.TabPane tab={t('模型')} itemKey='model' />
      </Tabs>

      <div className='min-h-[360px]'>
        {report ? (
          chartContent[chartTab]
        ) : (
          <Empty description={t('设置时间范围后刷新即可查看时间分析')} />
        )}
      </div>
    </Card>
  );
};

export default ChartAnalysisCard;
