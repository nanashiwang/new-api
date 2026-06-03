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

import React, { Suspense, lazy, useMemo } from 'react';
import { Card, Empty, Spin, Table, Tag } from '@douyinfe/semi-ui';
import { TrendingUp, WalletCards } from 'lucide-react';
import { renderNumber, renderQuota } from '../../helpers/dashboardFormat';

const VChart = lazy(() =>
  import('@visactor/react-vchart').then((module) => ({
    default: module.VChart,
  })),
);

const formatPercent = (value) => `${(Number(value || 0) * 100).toFixed(1)}%`;

const formatSnapshotTime = (timestamp) => {
  if (!timestamp) return '-';
  const date = new Date(Number(timestamp) * 1000);
  if (Number.isNaN(date.getTime())) return '-';
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(
    2,
    '0',
  )}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const buildMetricCards = (report, t) => {
  const latest = report?.latest || {};
  const deltaQuota = Number(report?.delta_quota || 0);
  return [
    {
      label: t('当前总余额'),
      value: renderQuota(latest.total_quota || 0, 2),
      color: 'blue',
    },
    {
      label: t('较上次统计'),
      value: renderQuota(deltaQuota, 2),
      color: deltaQuota >= 0 ? 'green' : 'red',
    },
    {
      label: t('有余额用户'),
      value: renderNumber(latest.positive_user_count || 0),
      color: 'cyan',
    },
    {
      label: t('Top10 占比'),
      value: formatPercent(latest.top10_share || 0),
      color: 'orange',
    },
    {
      label: t('负余额用户'),
      value: renderNumber(latest.negative_user_count || 0),
      color: latest.negative_user_count > 0 ? 'red' : 'grey',
    },
  ];
};

const UserBalanceTrendPanel = ({ report, loading, CARD_PROPS, CHART_CONFIG, t }) => {
  const snapshots = report?.snapshots || [];
  const metricCards = useMemo(() => buildMetricCards(report, t), [report, t]);
  const trendSpec = useMemo(
    () => ({
      type: 'area',
      data: [
        {
          id: 'userBalanceTrend',
          values: snapshots.map((item) => ({
            date: item.snapshot_date,
            quota: item.total_quota,
            positiveUsers: item.positive_user_count,
          })),
        },
      ],
      xField: 'date',
      yField: 'quota',
      title: {
        visible: true,
        text: t('全站用户余额走势'),
        subtext: `${t('每日 04:00 自动统计')} · ${t('最近统计')}：${formatSnapshotTime(
          report?.latest?.snapshot_at,
        )}`,
      },
      line: { style: { stroke: '#1664FF', lineWidth: 2 } },
      area: {
        style: {
          fill: {
            gradient: 'linear',
            x0: 0,
            y0: 0,
            x1: 0,
            y1: 1,
            stops: [
              { offset: 0, color: 'rgba(22, 100, 255, 0.28)' },
              { offset: 1, color: 'rgba(22, 100, 255, 0.02)' },
            ],
          },
        },
      },
      point: { visible: true, style: { size: 4 } },
      axes: [
        { orient: 'bottom', type: 'band' },
        {
          orient: 'left',
          type: 'linear',
          label: { formatMethod: (value) => renderQuota(value, 0) },
        },
      ],
      tooltip: {
        mark: {
          content: [
            { key: t('统计日期'), value: (datum) => datum.date },
            {
              key: t('总余额'),
              value: (datum) => renderQuota(datum.quota || 0, 2),
            },
            {
              key: t('有余额用户'),
              value: (datum) => renderNumber(datum.positiveUsers || 0),
            },
          ],
        },
      },
    }),
    [report?.latest?.snapshot_at, snapshots, t],
  );

  const topUserColumns = [
    {
      title: t('用户'),
      dataIndex: 'username',
      render: (text, record) => record.display_name || text || '-',
    },
    {
      title: t('余额'),
      dataIndex: 'quota',
      align: 'right',
      render: (value) => renderQuota(value || 0, 2),
    },
  ];

  return (
    <Card
      {...CARD_PROPS}
      className='!rounded-2xl mb-4'
      title={
        <div className='flex items-center gap-2'>
          <WalletCards size={16} />
          {t('用户余额走势')}
        </div>
      }
    >
      <Spin spinning={loading}>
        {snapshots.length > 0 ? (
          <div className='space-y-4'>
            <div className='grid grid-cols-2 md:grid-cols-5 gap-3'>
              {metricCards.map((item) => (
                <div key={item.label} className='rounded-xl bg-gray-50 p-3'>
                  <div className='mb-2 flex items-center justify-between gap-2'>
                    <span className='text-xs text-gray-500'>{item.label}</span>
                    <Tag color={item.color} shape='circle' size='small'>
                      <TrendingUp size={12} />
                    </Tag>
                  </div>
                  <div className='text-lg font-semibold text-gray-900'>
                    {item.value}
                  </div>
                </div>
              ))}
            </div>

            <div className='grid grid-cols-1 xl:grid-cols-3 gap-4'>
              <div className='h-80 xl:col-span-2 rounded-xl border border-gray-100 p-2'>
                <Suspense
                  fallback={
                    <div className='h-full flex items-center justify-center text-gray-400 text-sm'>
                      {t('加载中...')}
                    </div>
                  }
                >
                  <VChart spec={trendSpec} option={CHART_CONFIG} />
                </Suspense>
              </div>
              <div className='rounded-xl border border-gray-100 p-3'>
                <div className='mb-3 text-sm font-semibold text-gray-700'>
                  {t('高余额用户 Top10')}
                </div>
                <Table
                  columns={topUserColumns}
                  dataSource={report?.top_users || []}
                  rowKey='id'
                  pagination={false}
                  size='small'
                />
                {(report?.negative_users || []).length > 0 ? (
                  <div className='mt-3 rounded-lg bg-red-50 p-2 text-xs text-red-600'>
                    {t('发现负余额用户')}：
                    {(report?.negative_users || [])
                      .slice(0, 3)
                      .map((item) => item.display_name || item.username)
                      .join('、')}
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        ) : (
          <div className='py-10'>
            <Empty
              title={t('暂无余额快照')}
              description={t('系统会在每天 04:00 自动统计用户余额')}
            />
          </div>
        )}
      </Spin>
    </Card>
  );
};

export default UserBalanceTrendPanel;
