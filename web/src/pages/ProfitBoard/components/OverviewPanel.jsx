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
import { Avatar, Card, Empty, Spin, Table, Tag, Typography } from '@douyinfe/semi-ui';

const { Text, Title } = Typography;

const cardThemeMap = {
  configured_site_revenue_usd: {
    bg: 'bg-emerald-50 dark:bg-emerald-950/30',
    text: 'text-emerald-600 dark:text-emerald-400',
    avatarColor: 'green',
  },
  upstream_cost_usd: {
    bg: 'bg-amber-50 dark:bg-amber-950/30',
    text: 'text-amber-600 dark:text-amber-400',
    avatarColor: 'amber',
  },
  configured_profit_usd: {
    bg: 'bg-sky-50 dark:bg-sky-950/30',
    text: 'text-sky-600 dark:text-sky-400',
    avatarColor: 'blue',
  },
};

const MetricCard = ({ item, t }) => {
  const theme = cardThemeMap[item.key] || {
    bg: 'bg-blue-50 dark:bg-blue-950/30',
    text: 'text-blue-600 dark:text-blue-400',
    avatarColor: 'blue',
  };

  return (
    <div className={`rounded-2xl p-5 ${theme.bg}`}>
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
              {item.value}
            </Title>
          </div>
        </div>
        <Avatar size='small' color={theme.avatarColor} className='shrink-0'>
          {item.icon}
        </Avatar>
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
      title: t('利润'),
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
                <MetricCard key={item.key} item={item} t={t} />
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
