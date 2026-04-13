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
import React, { useMemo } from 'react';
import { Avatar, Card, Empty, Spin, Table, Tag, Typography } from '@douyinfe/semi-ui';

const { Text, Title } = Typography;

const MetricValuePair = ({ primary, secondary, className = '' }) => (
  <div className={className}>
    <div>{primary}</div>
    <Text type='tertiary' size='small'>
      {secondary}
    </Text>
  </div>
);

const Sparkline = ({ data, color = '#6366f1', width = 80, height = 24 }) => {
  if (!data || data.length < 2) return null;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const points = data.map(
    (v, i) =>
      `${(i / (data.length - 1)) * width},${height - ((v - min) / range) * (height - 2) - 1}`,
  );
  return (
    <svg width={width} height={height} className='shrink-0 opacity-60'>
      <polyline
        fill='none'
        stroke={color}
        strokeWidth='1.5'
        strokeLinecap='round'
        strokeLinejoin='round'
        points={points.join(' ')}
      />
    </svg>
  );
};

const sparklineColorMap = {
  configured_site_revenue_cny: '#059669',
  upstream_cost_cny: '#d97706',
  configured_profit_cny: '#0284c7',
};

const cardThemeMap = {
  configured_site_revenue_cny: {
    bg: 'bg-emerald-50 dark:bg-emerald-950/30',
    text: 'text-emerald-600 dark:text-emerald-400',
    avatarColor: 'green',
  },
  upstream_cost_cny: {
    bg: 'bg-amber-50 dark:bg-amber-950/30',
    text: 'text-amber-600 dark:text-amber-400',
    avatarColor: 'amber',
  },
  configured_profit_cny: {
    bg: 'bg-sky-50 dark:bg-sky-950/30',
    text: 'text-sky-600 dark:text-sky-400',
    avatarColor: 'blue',
  },
};

const MetricCard = ({ item, onClick, sparklineData, t }) => {
  const theme = cardThemeMap[item.key] || {
    bg: 'bg-blue-50 dark:bg-blue-950/30',
    text: 'text-blue-600 dark:text-blue-400',
    avatarColor: 'blue',
  };

  return (
    <div
      className={`rounded-2xl p-5 ${theme.bg} ${onClick ? 'cursor-pointer transition-shadow hover:shadow-md' : ''}`}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      <div className='flex items-start justify-between'>
        <div className='flex-1'>
          <Text type='tertiary' size='small'>
            {item.title}
          </Text>
          <div className='mt-2'>
            <Title
              heading={1}
              style={{ margin: 0, fontWeight: 700 }}
              className={theme.text}
            >
              {item.primary}
            </Title>
            {item.secondary ? (
              <Text type='tertiary' size='small'>
                {item.secondary}
              </Text>
            ) : null}
          </div>
        </div>
        <div className='flex flex-col items-end gap-2'>
          <Avatar size='small' color={theme.avatarColor} className='shrink-0'>
            {item.icon}
          </Avatar>
          <Sparkline
            data={sparklineData}
            color={sparklineColorMap[item.key] || '#6366f1'}
          />
        </div>
      </div>
      {item.requestCount != null && (
        <div className='mt-3 flex items-center gap-1.5'>
          <Text type='tertiary' size='small'>
            {t('累计请求')}
          </Text>
          <Tag color='blue' size='small'>
            {Number(item.requestCount).toLocaleString()}
          </Tag>
        </div>
      )}
    </div>
  );
};

const OverviewPanel = ({
  overviewQuerying,
  autoRefreshing,
  queryReady,
  overviewReport,
  overviewSummaryCards,
  timeseries,
  onMetricClick,
  t,
}) => {
  // 从 timeseries 提取每个指标的 sparkline 数据（聚合全部 batch）
  const sparklineMap = useMemo(() => {
    if (!timeseries?.length) return {};
    const bucketMap = new Map();
    timeseries.forEach((row) => {
      const existing = bucketMap.get(row.bucket);
      if (!existing) {
        bucketMap.set(row.bucket, { ...row });
      } else {
        existing.configured_site_revenue_cny =
          (existing.configured_site_revenue_cny || 0) + (row.configured_site_revenue_cny || 0);
        existing.upstream_cost_cny =
          (existing.upstream_cost_cny || 0) + (row.upstream_cost_cny || 0);
        existing.configured_profit_cny =
          (existing.configured_profit_cny || 0) + (row.configured_profit_cny || 0);
      }
    });
    const sorted = Array.from(bucketMap.values()).sort((a, b) =>
      a.bucket < b.bucket ? -1 : 1,
    );
    return {
      configured_site_revenue_cny: sorted.map((r) => r.configured_site_revenue_cny || 0),
      upstream_cost_cny: sorted.map((r) => r.upstream_cost_cny || 0),
      configured_profit_cny: sorted.map((r) => r.configured_profit_cny || 0),
    };
  }, [timeseries]);
  const batchColumns = [
    {
      title: t('组合名称'),
      dataIndex: 'batch_name',
      key: 'batch_name',
    },
    {
      title: t('请求数'),
      dataIndex: 'request_count',
      key: 'request_count',
      render: (val) => (
        <Tag color='blue' size='small'>
          {val}
        </Tag>
      ),
    },
    {
      title: t('本站配置收入'),
      dataIndex: 'configured_site_revenue_cny',
      key: 'configured_site_revenue_cny',
      render: (val, record) => (
        <MetricValuePair
          primary={`¥${Number(val || 0).toFixed(3)}`}
          secondary={`$${Number(record?.configured_site_revenue_usd || 0).toFixed(3)}`}
          className='text-emerald-600 dark:text-emerald-400'
        />
      ),
    },
    {
      title: t('上游费用'),
      dataIndex: 'upstream_cost_cny',
      key: 'upstream_cost_cny',
      render: (val, record) => (
        <MetricValuePair
          primary={`¥${Number(val || 0).toFixed(3)}`}
          secondary={`$${Number(record?.upstream_cost_usd || 0).toFixed(3)}`}
          className='text-amber-600 dark:text-amber-400'
        />
      ),
    },
    {
      title: t('利润'),
      dataIndex: 'configured_profit_cny',
      key: 'configured_profit_cny',
      render: (val, record) => (
        <MetricValuePair
          primary={`¥${Number(val || 0).toFixed(3)}`}
          secondary={`$${Number(record?.configured_profit_usd || 0).toFixed(3)}`}
          className='text-sky-600 dark:text-sky-400'
        />
      ),
    },
  ];

  const showEmptyState = !queryReady || !overviewReport;
  const isBusy = overviewQuerying || autoRefreshing;

  return (
    <div className='space-y-4'>
      <Card
        bordered={false}
        className='!rounded-2xl'
        title={
          <div className='flex flex-col gap-1 py-1'>
            <Text strong className='text-base'>
              {t('累计总览')}
            </Text>
            <Text type='tertiary' size='small'>
              {t('按当前组合配置累计统计，不受下方分析时间范围影响')}
            </Text>
          </div>
        }
      >
        <Spin spinning={isBusy}>
          {overviewReport ? (
            <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
              {overviewSummaryCards.map((item) => (
                <MetricCard
                  key={item.key}
                  item={item}
                  onClick={onMetricClick ? () => onMetricClick(item.key) : undefined}
                  sparklineData={sparklineMap[item.key]}
                  t={t}
                />
              ))}
            </div>
          ) : (
            <Empty
              description={
                !queryReady
                  ? t('添加组合并完成加载后会自动生成累计总览')
                  : t('当前还没有累计总览数据')
              }
            />
          )}
        </Spin>
      </Card>

      {!showEmptyState && overviewReport?.batch_summaries?.length > 0 ? (
        <Card
          bordered={false}
          title={
            <Text strong className='text-base'>
              {t('累计组合对比')}
            </Text>
          }
          className='!rounded-2xl'
        >
          <Table
            columns={batchColumns}
            dataSource={overviewReport.batch_summaries}
            rowKey='batch_id'
            pagination={
              overviewReport.batch_summaries.length > 10
                ? { pageSize: 10, showSizeChanger: false }
                : false
            }
            size='small'
            scroll={{ x: 'max-content' }}
          />
        </Card>
      ) : null}
    </div>
  );
};

export default OverviewPanel;
