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
import React from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  InputNumber,
  Radio,
  Select,
  Space,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { RefreshCw, RotateCcw } from 'lucide-react';

const { Text } = Typography;

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
  channelGroupMode,
  setChannelGroupMode,
  report,
  reportMatchesCurrentFilters,
  queryReady,
  chartContent,
  trendBucketCount,
  tagAggregationHint,
  t,
}) => {
  const clearAllFilters = () => {
    setAnalysisMode('business_compare');
    setMetricKey('configured_profit_usd');
    setViewBatchId('all');
    setGranularity('day');
    setChannelGroupMode('channel');
    if (detailFilter?.value) clearDetailFilter();
  };

  return (
    <Card
      bordered={false}
      className='rounded-xl'
      bodyStyle={{ paddingTop: 12 }}
      title={
        <Text strong className='text-base'>
          {t('图表分析')}
        </Text>
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
      {/* 筛选项 - 紧凑的行内布局 */}
      <div className='flex flex-wrap items-end gap-3'>
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

      <Tabs
        activeKey={chartTab}
        onChange={setChartTab}
        type='line'
        className='mt-4'
      >
        <Tabs.TabPane tab={t('趋势')} itemKey='trend' />
        <Tabs.TabPane tab={t('渠道')} itemKey='channel' />
        <Tabs.TabPane tab={t('模型')} itemKey='model' />
      </Tabs>

      {chartTab === 'channel' ? (
        <>
          <div className='mb-3 flex items-center justify-end'>
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

      <div className='min-h-[300px]'>
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
          <Empty description={t('添加组合并完成加载后会自动生成图表')} />
        )}
      </div>
    </Card>
  );
};

export default ChartAnalysisCard;
