import React from 'react';
import {
  Button,
  Card,
  Empty,
  InputNumber,
  Select,
  Space,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import { Filter } from 'lucide-react';

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
  t,
}) => (
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
          {detailFilter?.value ? (
            <Button type='tertiary' icon={<Filter size={14} />} onClick={clearDetailFilter}>
              {t('清除图表筛选')}
            </Button>
          ) : null}
          <Button type='tertiary' onClick={runQuery} loading={querying}>
            {t('刷新时间分析')}
          </Button>
        </Space>
      </div>
    }
  >
    <Text type='tertiary' className='mb-3 block'>
      {analysisMode === 'business_compare'
        ? t('默认直接看本站配置收入和上游费用的金额对比，更适合判断渠道是否赚钱。')
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
      {report ? chartContent[chartTab] : <Empty description={t('设置时间范围后刷新即可查看时间分析')} />}
    </div>
  </Card>
);

export default ChartAnalysisCard;
