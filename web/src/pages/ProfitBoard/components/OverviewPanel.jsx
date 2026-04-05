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
import { Card, Empty, Spin, Table, Tag, Typography } from '@douyinfe/semi-ui';

const { Text, Title } = Typography;

const MetricCard = ({ item }) => {
  const iconColorMap = {
    configured_site_revenue_usd: 'text-emerald-600 dark:text-emerald-400',
    upstream_cost_usd: 'text-amber-600 dark:text-amber-400',
    configured_profit_usd: 'text-sky-600 dark:text-sky-400',
  };
  const bgColorMap = {
    configured_site_revenue_usd: 'bg-emerald-500/10',
    upstream_cost_usd: 'bg-amber-500/10',
    configured_profit_usd: 'bg-sky-500/10',
  };
  const iconColor =
    iconColorMap[item.key] || 'text-blue-600 dark:text-blue-400';
  const bgColor = bgColorMap[item.key] || 'bg-blue-500/10';

  return (
    <Card
      key={item.key}
      bordered={false}
      bodyStyle={{ padding: 0 }}
      className='overflow-hidden rounded-xl border border-semi-color-border'
    >
      <div className='p-4'>
        <div className='flex items-start justify-between'>
          <div className='flex-1'>
            <Text type='tertiary' size='small'>
              {item.title}
            </Text>
            <div className='mt-3 flex items-baseline gap-2'>
              <Title
                heading={2}
                style={{ margin: 0, fontWeight: 600 }}
                className={iconColor}
              >
                {item.value}
              </Title>
            </div>
          </div>
          <div
            className={`flex h-12 w-12 items-center justify-center rounded-xl ${bgColor}`}
          >
            <span className={iconColor}>{item.icon}</span>
          </div>
        </div>
      </div>
    </Card>
  );
};

const OverviewPanel = ({
  overviewQuerying,
  autoRefreshing,
  queryReady,
  overviewReport,
  overviewSummaryCards,
  formatMoney,
  status,
  t,
}) => {
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
      dataIndex: 'configured_site_revenue_usd',
      key: 'configured_site_revenue_usd',
      render: (val) => (
        <span className='text-emerald-600 dark:text-emerald-400'>
          {formatMoney(val, status)}
        </span>
      ),
    },
    {
      title: t('上游费用'),
      dataIndex: 'upstream_cost_usd',
      key: 'upstream_cost_usd',
      render: (val) => (
        <span className='text-amber-600 dark:text-amber-400'>
          {formatMoney(val, status)}
        </span>
      ),
    },
    {
      title: t('配置利润'),
      dataIndex: 'configured_profit_usd',
      key: 'configured_profit_usd',
      render: (val) => (
        <span className='text-sky-600 dark:text-sky-400'>
          {formatMoney(val, status)}
        </span>
      ),
    },
  ];

  const showEmptyState = !queryReady || !overviewReport;
  const isBusy = overviewQuerying || autoRefreshing;

  return (
    <div className='space-y-4'>
      <Card
        bordered={false}
        className='rounded-xl'
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
                <MetricCard key={item.key} item={item} />
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
          className='rounded-xl'
        >
          <Table
            columns={batchColumns}
            dataSource={overviewReport.batch_summaries}
            rowKey='batch_id'
            pagination={false}
            size='small'
            scroll={{ x: 'max-content' }}
          />
        </Card>
      ) : null}
    </div>
  );
};

export default OverviewPanel;
