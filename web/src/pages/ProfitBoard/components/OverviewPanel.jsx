import React from 'react';
import { Card, Empty, Spin, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { Info, TrendingUp, TrendingDown } from 'lucide-react';
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
  const iconColor = iconColorMap[item.key] || 'text-blue-600 dark:text-blue-400';
  const bgColor = bgColorMap[item.key] || 'bg-blue-500/10';

  return (
    <Card
      key={item.key}
      bordered={false}
      bodyStyle={{ padding: 0 }}
      className='overflow-hidden rounded-xl border border-semi-color-border'
    >
      <div className='p-5'>
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

const BatchSummaryCard = ({ item, formatMoney, status, t }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4 transition-all hover:border-semi-color-primary-hover'>
    <div className='flex items-center justify-between gap-2'>
      <Text strong className='text-base'>
        {item.batch_name}
      </Text>
      <Tag color='blue' size='small'>
        {item.request_count} {t('次请求')}
      </Tag>
    </div>
    <div className='mt-3 grid grid-cols-2 gap-x-4 gap-y-2'>
      <div className='flex items-center justify-between'>
        <Text type='tertiary' size='small'>
          {t('本站配置收入')}
        </Text>
        <Text strong className='text-emerald-600 dark:text-emerald-400'>
          {formatMoney(item.configured_site_revenue_usd, status)}
        </Text>
      </div>
      <div className='flex items-center justify-between'>
        <Text type='tertiary' size='small'>
          {t('上游费用')}
        </Text>
        <Text strong className='text-amber-600 dark:text-amber-400'>
          {formatMoney(item.upstream_cost_usd, status)}
        </Text>
      </div>
      <div className='flex items-center justify-between'>
        <Text type='tertiary' size='small'>
          {t('远端观测消耗')}
        </Text>
        <Text strong className='text-rose-600 dark:text-rose-400'>
          {formatMoney(item.remote_observed_cost_usd, status)}
        </Text>
      </div>
      <div className='flex items-center justify-between'>
        <Text type='tertiary' size='small'>
          {t('配置利润')}
        </Text>
        <Text strong className='text-sky-600 dark:text-sky-400'>
          {formatMoney(item.configured_profit_usd, status)}
        </Text>
      </div>
      <div className='col-span-2 flex items-center justify-between border-t border-semi-color-border pt-2 mt-1'>
        <Text type='tertiary' size='small'>
          {t('实际利润')}
        </Text>
        <Text strong className='text-violet-600 dark:text-violet-400 text-base'>
          {formatMoney(item.actual_profit_usd, status)}
        </Text>
      </div>
    </div>
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
  remoteObserverStates,
  t,
}) => {
  return (
    <div className='space-y-4'>
      <Card
        bordered={false}
        title={
          <Text strong className='text-base'>
            {t('累计总览')}
          </Text>
        }
        className='rounded-xl'
      >
        <Spin spinning={overviewQuerying}>
          {overviewReport ? (
            <div className='space-y-5'>
              <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5'>
                {cumulativeSummaryCards.map((item) => (
                  <MetricCard
                    key={item.key}
                    item={item}
                    summaryMetricHelp={summaryMetricHelp}
                    status={status}
                  />
                ))}
              </div>
              <div className='border-t border-semi-color-border pt-4'>
                <Text type='tertiary' size='small' className='mb-3 block'>
                  {t('诊断指标')}
                </Text>
                <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5'>
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
          ) : (
            <Empty description={t('添加组合后可手动刷新累计总览')} />
          )}
        </Spin>
      </Card>

      {overviewReport?.batch_summaries?.length > 0 ? (
        <Card
          bordered={false}
          title={
            <Text strong className='text-base'>
              {t('组合累计收益')}
            </Text>
          }
          className='rounded-xl'
        >
          <div className='grid gap-4 sm:grid-cols-2'>
            {overviewReport.batch_summaries.map((item) => (
              <BatchSummaryCard
                key={item.batch_id}
                item={item}
                formatMoney={formatMoney}
                status={status}
                t={t}
              />
            ))}
          </div>
        </Card>
      ) : null}
    </div>
  );
};

export default OverviewPanel;
