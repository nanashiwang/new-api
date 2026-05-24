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

import React, { Suspense, lazy } from 'react';
import { Card, Tabs, TabPane, Table } from '@douyinfe/semi-ui';
import { PieChart } from 'lucide-react';

const VChart = lazy(() =>
  import('@visactor/react-vchart').then((module) => ({
    default: module.VChart,
  })),
);

const ChartFallback = ({ t }) => (
  <div className='h-full flex items-center justify-center text-gray-400 text-sm'>
    {t('加载中...')}
  </div>
);

const ChartsPanel = ({
  activeChartTab,
  setActiveChartTab,
  spec_line,
  spec_model_line,
  spec_pie,
  spec_rank_bar,
  spec_user_rank,
  spec_user_trend,
  perfMetricsSummary,
  perfMetricsLoading,
  isAdminUser,
  CARD_PROPS,
  CHART_CONFIG,
  FLEX_CENTER_GAP2,
  hasApiInfoPanel,
  t,
}) => {
  const perfMetricColumns = [
    {
      title: t('模型'),
      dataIndex: 'model_name',
      render: (text, record) => (
        <span className='font-medium'>
          {record.rank}. {text}
        </span>
      ),
    },
    {
      title: t('请求数'),
      dataIndex: 'request_count',
      render: (value) => Number(value || 0).toLocaleString(),
    },
    {
      title: t('平均延迟'),
      dataIndex: 'avg_latency_ms',
      render: (value) => `${Number(value || 0).toLocaleString()} ms`,
    },
    {
      title: 'TTFT',
      dataIndex: 'avg_ttft_ms',
      render: (value) => `${Number(value || 0).toLocaleString()} ms`,
    },
    {
      title: t('成功率'),
      dataIndex: 'success_rate',
      render: (value) => `${Number(value || 0).toFixed(2)}%`,
    },
    {
      title: t('平均TPS'),
      dataIndex: 'avg_tps',
      render: (value) => Number(value || 0).toFixed(2),
    },
  ];
  const perfMetricData = (perfMetricsSummary || [])
    .slice(0, 10)
    .map((item, index) => ({
      ...item,
      rank: index + 1,
    }));

  return (
    <Card
      {...CARD_PROPS}
      className={`!rounded-2xl ${hasApiInfoPanel ? 'lg:col-span-3' : ''}`}
      title={
        <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between w-full gap-3'>
          <div className={FLEX_CENTER_GAP2}>
            <PieChart size={16} />
            {t('模型数据分析')}
          </div>
          <Tabs
            type='slash'
            activeKey={activeChartTab}
            onChange={setActiveChartTab}
          >
            <TabPane tab={<span>{t('消耗分布')}</span>} itemKey='1' />
            <TabPane tab={<span>{t('调用趋势')}</span>} itemKey='2' />
            <TabPane tab={<span>{t('调用次数分布')}</span>} itemKey='3' />
            <TabPane tab={<span>{t('调用次数排行')}</span>} itemKey='4' />
            {isAdminUser && (
              <TabPane tab={<span>{t('用户消耗排行')}</span>} itemKey='5' />
            )}
            {isAdminUser && (
              <TabPane tab={<span>{t('用户消耗趋势')}</span>} itemKey='6' />
            )}
            <TabPane tab={<span>{t('性能表现')}</span>} itemKey='7' />
          </Tabs>
        </div>
      }
      bodyStyle={{ padding: 0 }}
    >
      <div className='h-96 p-2'>
        <Suspense fallback={<ChartFallback t={t} />}>
          {activeChartTab === '1' && (
            <VChart spec={spec_line} option={CHART_CONFIG} />
          )}
          {activeChartTab === '2' && (
            <VChart spec={spec_model_line} option={CHART_CONFIG} />
          )}
          {activeChartTab === '3' && (
            <VChart spec={spec_pie} option={CHART_CONFIG} />
          )}
          {activeChartTab === '4' && (
            <VChart spec={spec_rank_bar} option={CHART_CONFIG} />
          )}
          {activeChartTab === '5' && isAdminUser && (
            <VChart spec={spec_user_rank} option={CHART_CONFIG} />
          )}
          {activeChartTab === '6' && isAdminUser && (
            <VChart spec={spec_user_trend} option={CHART_CONFIG} />
          )}
        </Suspense>
        {activeChartTab === '7' && (
          <div className='h-full overflow-auto p-2'>
            <Table
              columns={perfMetricColumns}
              dataSource={perfMetricData}
              pagination={false}
              loading={perfMetricsLoading}
              rowKey='model_name'
              size='small'
            />
          </div>
        )}
      </div>
    </Card>
  );
};

export default ChartsPanel;
