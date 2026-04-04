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
import React, { useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Collapsible,
  DatePicker,
  Empty,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { ChevronDown, ChevronRight, Info } from 'lucide-react';
import { timestamp2string } from '../../../helpers';

const { Text, Title } = Typography;

const MetricCard = ({ item, summaryMetricHelp, status }) => {
  const iconColorMap = {
    configured_site_revenue_usd: 'text-emerald-600 dark:text-emerald-400',
    upstream_cost_usd: 'text-amber-600 dark:text-amber-400',
    remote_observed_cost_usd: 'text-rose-600 dark:text-rose-400',
    configured_profit_usd: 'text-sky-600 dark:text-sky-400',
    actual_profit_usd: 'text-violet-600 dark:text-violet-400',
  };
  const bgColorMap = {
    configured_site_revenue_usd: 'bg-emerald-500/10',
    upstream_cost_usd: 'bg-amber-500/10',
    remote_observed_cost_usd: 'bg-rose-500/10',
    configured_profit_usd: 'bg-sky-500/10',
    actual_profit_usd: 'bg-violet-500/10',
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
            <Tooltip content={summaryMetricHelp[item.key] || item.title}>
              <div className='inline-flex cursor-help items-center gap-1.5'>
                <Text type='tertiary' size='small'>
                  {item.title}
                </Text>
                <Info size={13} className='text-semi-color-text-3' />
              </div>
            </Tooltip>
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

const DiagnosticMetric = ({ item, summaryMetricHelp }) => (
  <div className='rounded-lg border border-semi-color-border bg-semi-color-bg-1 px-4 py-3'>
    <Tooltip content={summaryMetricHelp[item.key] || item.title}>
      <div className='inline-flex cursor-help items-center gap-1'>
        <Text type='tertiary' size='small'>
          {item.title}
        </Text>
        <Info size={12} className='text-semi-color-text-3' />
      </div>
    </Tooltip>
    <div className='mt-1.5 text-lg font-semibold'>{item.value}</div>
  </div>
);

const OverviewPanel = ({
  overviewQuerying,
  overviewReport,
  report,
  reportMatchesCurrentFilters,
  cumulativeSummaryCards,
  diagnosticSummaryCards,
  summaryMetricHelp,
  formatMoney,
  status,
  datePresets,
  dateRange,
  setDateRange,
  validationErrors,
  t,
}) => {
  const [moreOpen, setMoreOpen] = useState(false);
  const coreCards = cumulativeSummaryCards.slice(0, 3);
  const extraCards = cumulativeSummaryCards.slice(3);

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
      title: t('收入'),
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
    {
      title: t('实际利润'),
      dataIndex: 'actual_profit_usd',
      key: 'actual_profit_usd',
      render: (val) => (
        <span className='text-violet-600 dark:text-violet-400'>
          {formatMoney(val, status)}
        </span>
      ),
    },
  ];

  return (
    <div className='space-y-4'>
      <Card
        bordered={false}
        className='rounded-xl'
        title={
          <Text strong className='text-base'>
            {t('数据总览')}
          </Text>
        }
      >
        {/* 时间范围选择 - 内联在总览头部 */}
        <div className='mb-5 rounded-lg border border-semi-color-border bg-semi-color-fill-0 p-3'>
          <div className='mb-2'>
            <Text type='tertiary' size='small'>
              {t('时间范围')}
            </Text>
          </div>
          <div className='flex flex-col gap-3 lg:flex-row lg:items-center'>
            <div className='flex flex-wrap gap-1.5'>
              {datePresets.map((item) => (
                <Button
                  key={item.label}
                  type='tertiary'
                  size='small'
                  onClick={() => setDateRange(item.value)}
                >
                  {t(item.label)}
                </Button>
              ))}
            </div>
            <DatePicker
              type='dateTimeRange'
              value={dateRange}
              onChange={(value) => setDateRange(value)}
              style={{ minWidth: 340 }}
              className='flex-1'
            />
          </div>
          {validationErrors.length > 0 && (
            <Banner
              type='danger'
              description={validationErrors[0]}
              closeIcon={null}
              className='mt-2'
            />
          )}
        </div>

        <Spin spinning={overviewQuerying}>
          {overviewReport ? (
            <div className='space-y-4'>
              <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
                {coreCards.map((item) => (
                  <MetricCard
                    key={item.key}
                    item={item}
                    summaryMetricHelp={summaryMetricHelp}
                    status={status}
                  />
                ))}
              </div>
              {(extraCards.length > 0 || diagnosticSummaryCards.length > 0) && (
                <button
                  type='button'
                  onClick={() => setMoreOpen(!moreOpen)}
                  className='flex items-center gap-1.5 text-sm text-semi-color-primary hover:opacity-80'
                >
                  {moreOpen ? (
                    <ChevronDown size={14} />
                  ) : (
                    <ChevronRight size={14} />
                  )}
                  {moreOpen ? t('收起') : t('查看更多')}
                </button>
              )}
              <Collapsible collapseHeight={0} isOpen={moreOpen} keepDOM>
                <div className='space-y-4 pt-1'>
                  {extraCards.length > 0 && (
                    <div className='grid gap-4 sm:grid-cols-2'>
                      {extraCards.map((item) => (
                        <MetricCard
                          key={item.key}
                          item={item}
                          summaryMetricHelp={summaryMetricHelp}
                          status={status}
                        />
                      ))}
                    </div>
                  )}
                  <div className='border-t border-semi-color-border pt-4'>
                    <Text type='tertiary' size='small' className='mb-3 block'>
                      {t('辅助指标')}
                    </Text>
                    <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'>
                      {diagnosticSummaryCards.map((item) => (
                        <DiagnosticMetric
                          key={item.key}
                          item={item}
                          summaryMetricHelp={summaryMetricHelp}
                        />
                      ))}
                    </div>
                  </div>
                  <div className='rounded-lg border border-semi-color-border bg-semi-color-bg-1 px-4 py-3'>
                    <Text type='tertiary' size='small'>
                      {t('最近一条命中日志')}
                    </Text>
                    <div className='mt-1 font-medium'>
                      {overviewReport?.meta?.latest_log_created_at
                        ? timestamp2string(
                            overviewReport.meta.latest_log_created_at,
                          )
                        : '-'}
                    </div>
                  </div>
                </div>
              </Collapsible>
            </div>
          ) : (
            <Empty description={t('添加组合后可手动刷新数据')} />
          )}
        </Spin>
      </Card>

      {overviewReport?.batch_summaries?.length > 0 ? (
        <Card
          bordered={false}
          title={
            <Text strong className='text-base'>
              {t('各组合收益')}
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
