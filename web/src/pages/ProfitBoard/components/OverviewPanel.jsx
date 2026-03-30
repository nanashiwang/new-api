import React from 'react';
import { Card, Empty, Spin, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { Info } from 'lucide-react';
import { timestamp2string } from '../../../helpers';

const { Text, Title } = Typography;

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
  t,
}) => {
  const remoteObserverStates =
    reportMatchesCurrentFilters && report?.remote_observer_states?.length
      ? report.remote_observer_states
      : overviewReport?.remote_observer_states || [];
  const remoteObservedCostLabel =
    reportMatchesCurrentFilters && report?.remote_observer_states?.length
      ? t('当前时间范围观测消耗')
      : t('累计观测消耗');
  const remoteStatusColorMap = {
    ready: 'green',
    needs_baseline: 'orange',
    failed: 'red',
    not_configured: 'grey',
    disabled: 'grey',
  };
  const remoteStatusLabelMap = {
    ready: t('正常'),
    needs_baseline: t('等待基线'),
    failed: t('同步失败'),
    not_configured: t('未配置'),
    disabled: t('未启用'),
  };

  return (
    <div className='space-y-4'>
      <Card bordered={false} title={t('累计总览')}>
        <Spin spinning={overviewQuerying}>
          {overviewReport ? (
            <div className='space-y-4'>
              <div className='grid gap-4 md:grid-cols-2'>
                {cumulativeSummaryCards.map((item) => (
                  <Card key={item.key} bordered={false} bodyStyle={{ padding: 18 }} className='bg-semi-color-fill-0'>
                    <div className='flex items-center justify-between gap-3'>
                      <div>
                        <Tooltip content={summaryMetricHelp[item.key] || item.title}>
                          <div className='inline-flex cursor-help items-center gap-1'>
                            <Text type='tertiary'>{item.title}</Text>
                            <Info size={14} className='text-semi-color-text-2' />
                          </div>
                        </Tooltip>
                        <Title heading={3} style={{ margin: '8px 0 0' }}>
                          {item.value}
                        </Title>
                      </div>
                      <div className='flex h-10 w-10 items-center justify-center rounded-full bg-semi-color-bg-2'>
                        {item.icon}
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
              <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-5'>
                {diagnosticSummaryCards.map((item) => (
                  <div key={item.key} className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'>
                    <Tooltip content={summaryMetricHelp[item.key] || item.title}>
                      <div className='inline-flex cursor-help items-center gap-1'>
                        <Text type='tertiary'>{item.title}</Text>
                        <Info size={13} className='text-semi-color-text-2' />
                      </div>
                    </Tooltip>
                    <div className='mt-2 text-lg font-semibold'>{item.value}</div>
                  </div>
                ))}
              </div>
              <div className='grid gap-3 md:grid-cols-2'>
                <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                  <Text type='tertiary'>{t('累计统计范围')}</Text>
                  <div className='mt-1 font-medium'>{t('已添加组合的全部历史消费日志')}</div>
                </div>
                <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
                  <Text type='tertiary'>{t('最近一条命中日志')}</Text>
                  <div className='mt-1 font-medium'>
                    {overviewReport?.meta?.latest_log_created_at
                      ? timestamp2string(overviewReport.meta.latest_log_created_at)
                      : '-'}
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <Empty description={t('添加组合后可手动刷新累计总览')} />
          )}
        </Spin>
      </Card>

      {overviewReport?.batch_summaries?.length > 0 ? (
        <Card bordered={false} title={t('组合累计收益')}>
          <div className='grid gap-3 md:grid-cols-2'>
            {overviewReport.batch_summaries.map((item) => (
              <div key={item.batch_id} className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'>
                <div className='flex items-center justify-between gap-2'>
                  <Text strong>{item.batch_name}</Text>
                  <Tag color='blue'>
                    {item.request_count} {t('次请求')}
                  </Tag>
                </div>
                <div className='mt-2 grid grid-cols-2 gap-2 text-sm'>
                  <Text>{t('本站配置收入')}：{formatMoney(item.configured_site_revenue_usd, status)}</Text>
                  <Text>{t('上游费用')}：{formatMoney(item.upstream_cost_usd, status)}</Text>
                  <Text>{t('远端观测消耗')}：{formatMoney(item.remote_observed_cost_usd, status)}</Text>
                  <Text>{t('配置利润')}：{formatMoney(item.configured_profit_usd, status)}</Text>
                  <Text>{t('实际利润')}：{formatMoney(item.actual_profit_usd, status)}</Text>
                </div>
              </div>
            ))}
          </div>
        </Card>
      ) : null}

      {remoteObserverStates.length > 0 ? (
        <Card bordered={false} title={t('远端额度观测状态')}>
          <div className='grid gap-3 md:grid-cols-2'>
            {remoteObserverStates.map((item) => (
              <div key={item.batch_id} className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'>
                <div className='flex flex-wrap items-center justify-between gap-2'>
                  <Text strong>{item.batch_name}</Text>
                  <div className='flex flex-wrap gap-2'>
                    <Tag color={remoteStatusColorMap[item.status] || 'grey'}>
                      {remoteStatusLabelMap[item.status] || item.status}
                    </Tag>
                    {item.quota_per_unit_mismatch ? (
                      <Tag color='orange'>{t('远端额度倍率不同')}</Tag>
                    ) : null}
                  </div>
                </div>
                <div className='mt-3 grid gap-2 text-sm'>
                  <Text>{remoteObservedCostLabel}：{formatMoney(item.observed_cost_usd, status)}</Text>
                  <Text>{t('钱包已用')}：{formatMoney(item.wallet_used_quota_usd, status)}</Text>
                  <Text>{t('钱包余额')}：{formatMoney((item.wallet_quota_usd || 0) - (item.wallet_used_quota_usd || 0), status)}</Text>
                  <Text>{t('订阅已用')}：{formatMoney(item.subscription_used_quota_usd, status)}</Text>
                  <Text>{t('订阅总额')}：{item.subscription_total_quota_usd > 0 ? formatMoney(item.subscription_total_quota_usd, status) : t('不限额或未知')}</Text>
                  <Text>{t('最近同步')}：{item.last_synced_at ? timestamp2string(item.last_synced_at) : '-'}</Text>
                  <Text>{t('最近成功')}：{item.last_success_at ? timestamp2string(item.last_success_at) : '-'}</Text>
                </div>
                {item.error_message ? (
                  <div className='mt-3 rounded-lg bg-semi-color-warning-light-default px-3 py-2 text-sm text-semi-color-warning'>
                    {item.error_message}
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        </Card>
      ) : null}
    </div>
  );
};

export default OverviewPanel;
