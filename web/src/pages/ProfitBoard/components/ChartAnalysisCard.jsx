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
  Select,
  Space,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { Filter, RefreshCw, RotateCcw } from 'lucide-react';

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
  report,
  chartContent,
  trendRowCount,
  trendBucketCount,
  t,
}) => {
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
        <div className='flex items-end'>
          <Button
            type='tertiary'
            size='small'
            icon={<RotateCcw size={13} />}
            onClick={clearAllFilters}
          >
            {t('重置筛选')}
          </Button>
        </div>
      </div>

      <Tabs activeKey={chartTab} onChange={setChartTab} type='line' className='mt-4'>
        <Tabs.TabPane tab={t('趋势')} itemKey='trend' />
        <Tabs.TabPane tab={t('渠道')} itemKey='channel' />
        <Tabs.TabPane tab={t('模型')} itemKey='model' />
      </Tabs>

      <div className='min-h-[300px]'>
        {report ? (
          <>
            {chartTab === 'trend' &&
              trendBucketCount > 0 &&
              trendBucketCount < 4 && (
                <Banner
                  type='info'
                  description={t(
                    '当前趋势图只有 {{count}} 个时间点。即使请求很多，也会先汇总进这些时间桶；想看更明显的趋势，请扩大时间范围或调整粒度。',
                    { count: trendBucketCount },
                  )}
                  closeIcon={null}
                  className='mb-3'
                />
              )}
            {chartContent[chartTab]}
          </>
        ) : (
          <Empty description={t('设置时间范围后刷新即可查看时间分析')} />
        )}
      </div>
    </Card>
  );
};

export default ChartAnalysisCard;
