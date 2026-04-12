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
import React, { useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Collapsible,
  DatePicker,
  Empty,
  InputNumber,
  Radio,
  Select,
  Space,
  Tag,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { BarChart3, ChevronDown, ChevronUp, RefreshCw, RotateCcw, Settings } from 'lucide-react';

const { Text } = Typography;

const DEFAULTS = {
  analysisMode: 'business_compare',
  metricKey: 'configured_profit_usd',
  viewBatchId: 'all',
  granularity: 'day',
  channelGroupMode: 'channel',
};

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
  datePresets,
  dateRange,
  setDateRange,
  runQuery,
  querying,
  chartTab,
  setChartTab,
  channelGroupMode,
  setChannelGroupMode,
  report,
  reportMatchesCurrentFilters,
  queryReady,
  chartContent,
  trendBucketCount,
  tagAggregationHint,
  validationErrors,
  onNavigateToConfig,
  t,
}) => {
  const [filtersExpanded, setFiltersExpanded] = useState(false);

  const hasNonDefaultFilters =
    analysisMode !== DEFAULTS.analysisMode ||
    viewBatchId !== DEFAULTS.viewBatchId ||
    granularity !== DEFAULTS.granularity ||
    channelGroupMode !== DEFAULTS.channelGroupMode;

  const clearAllFilters = () => {
    setAnalysisMode(DEFAULTS.analysisMode);
    setMetricKey(DEFAULTS.metricKey);
    setViewBatchId(DEFAULTS.viewBatchId);
    setGranularity(DEFAULTS.granularity);
    setChannelGroupMode(DEFAULTS.channelGroupMode);
  };

  const datePickerPresets = useMemo(
    () =>
      datePresets.map((item) => ({
        text: t(item.label),
        start: item.value[0],
        end: item.value[1],
      })),
    [datePresets, t],
  );

  return (
    <Card
      bordered={false}
      className='!rounded-2xl'
      bodyStyle={{ paddingTop: 0 }}
      title={
        <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between w-full gap-3'>
          <div className='flex items-center gap-2'>
            <BarChart3 size={16} />
            <Text strong className='text-base'>
              {t('时间范围分析')}
            </Text>
          </div>
          <Tabs
            type='slash'
            activeKey={chartTab}
            onChange={setChartTab}
          >
            <Tabs.TabPane tab={t('趋势')} itemKey='trend' />
            <Tabs.TabPane tab={t('渠道')} itemKey='channel' />
            <Tabs.TabPane tab={t('模型')} itemKey='model' />
          </Tabs>
        </div>
      }
      headerExtraContent={
        <Space>
          <Button
            type='tertiary'
            size='small'
            icon={<RotateCcw size={13} />}
            onClick={clearAllFilters}
          >
            {t('重置')}
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
      }
    >
      <div className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 p-3'>
        {/* 第一行: 日期范围 + 高级筛选切换 */}
        <div className='flex items-center gap-2'>
          <div className='flex-1'>
            <DatePicker
              type='dateTimeRange'
              value={dateRange}
              onChange={(value) => setDateRange(value)}
              presets={datePickerPresets}
              presetPosition='left'
              style={{ width: '100%' }}
            />
          </div>
          <Button
            size='small'
            theme='borderless'
            type={hasNonDefaultFilters ? 'primary' : 'tertiary'}
            icon={filtersExpanded ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
            iconPosition='right'
            onClick={() => setFiltersExpanded((v) => !v)}
          >
            {t('筛选')}
            {hasNonDefaultFilters && !filtersExpanded ? (
              <Tag color='blue' size='small' className='ml-1'>{t('已调整')}</Tag>
            ) : null}
          </Button>
        </div>

        {validationErrors.length > 0 ? (
          <Banner
            type='danger'
            description={validationErrors[0]}
            closeIcon={null}
            className='mt-3'
          />
        ) : null}

        {/* 高级筛选区 */}
        <Collapsible isOpen={filtersExpanded}>
          <div className='mt-3 flex flex-wrap items-end gap-3 border-t border-semi-color-border pt-3'>
            <div className='min-w-[140px]'>
              <Text type='tertiary' size='small' className='mb-1 block'>
                {t('分析模式')}
              </Text>
              <Select
                value={analysisMode}
                onChange={setAnalysisMode}
                optionList={[
                  { label: t('经营对比'), value: 'business_compare' },
                  { label: t('单指标分析'), value: 'single_metric' },
                ]}
                size='small'
                style={{ width: 140 }}
              />
            </div>
            {analysisMode === 'single_metric' && (
              <div className='min-w-[140px]'>
                <Text type='tertiary' size='small' className='mb-1 block'>
                  {t('指标')}
                </Text>
                <Select
                  value={metricKey}
                  onChange={setMetricKey}
                  optionList={metricOptions.map((item) => ({
                    label: t(item.label),
                    value: item.value,
                  }))}
                  size='small'
                  style={{ width: 140 }}
                />
              </div>
            )}
            <div className='min-w-[130px]'>
              <Text type='tertiary' size='small' className='mb-1 block'>
                {t('组合')}
              </Text>
              <Select
                value={viewBatchId}
                onChange={setViewBatchId}
                optionList={batchSummaryOptions}
                size='small'
                style={{ width: 130 }}
              />
            </div>
            <div className='min-w-[120px]'>
              <Text type='tertiary' size='small' className='mb-1 block'>
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
                  { label: t('自定义'), value: 'custom' },
                ]}
                size='small'
                style={{ width: 120 }}
              />
            </div>
            {granularity === 'custom' && (
              <div className='min-w-[120px]'>
                <Text type='tertiary' size='small' className='mb-1 block'>
                  {t('间隔')}
                </Text>
                <InputNumber
                  min={1}
                  value={customIntervalMinutes}
                  onChange={(value) =>
                    setCustomIntervalMinutes(Math.max(Number(value || 1), 1))
                  }
                  suffix={t('分钟')}
                  size='small'
                  style={{ width: 120 }}
                />
              </div>
            )}
          </div>
        </Collapsible>
      </div>

      {chartTab === 'channel' ? (
        <>
          <div className='mt-4 mb-3 flex items-center justify-end'>
            <Radio.Group
              type='button'
              value={channelGroupMode}
              onChange={(event) => setChannelGroupMode(event.target.value)}
              size='small'
            >
              <Radio value='channel'>{t('单渠道')}</Radio>
              <Radio value='tag'>{t('标签聚合')}</Radio>
            </Radio.Group>
          </div>
          {channelGroupMode === 'tag' && tagAggregationHint ? (
            <Banner
              type='info'
              description={tagAggregationHint}
              closeIcon={null}
              className='mb-3'
            />
          ) : null}
        </>
      ) : null}

      <div className='mt-4 min-h-[420px]'>
        {queryReady ? (
          <>
            {!reportMatchesCurrentFilters && querying ? (
              <Banner
                type='info'
                description={t('筛选条件已变化，正在刷新收益分析')}
                closeIcon={null}
                className='mb-3'
              />
            ) : null}
            {chartTab === 'trend' &&
              trendBucketCount > 0 &&
              trendBucketCount < 4 &&
              granularity !== 'month' && (
                <Banner
                  type='info'
                  description={t('数据点较少，可尝试扩大时间范围')}
                  closeIcon={null}
                  className='mb-3'
                />
              )}
            {report && reportMatchesCurrentFilters ? (
              chartContent[chartTab]
            ) : (
              <Empty description={t('当前收益分析正在加载')} />
            )}
          </>
        ) : (
          <Empty description={t('添加组合并完成加载后会自动生成图表')}>
            {onNavigateToConfig ? (
              <Button
                type='primary'
                theme='light'
                icon={<Settings size={14} />}
                onClick={onNavigateToConfig}
                className='mt-2'
              >
                {t('前往配置管理')}
              </Button>
            ) : null}
          </Empty>
        )}
      </div>
    </Card>
  );
};

export default ChartAnalysisCard;
