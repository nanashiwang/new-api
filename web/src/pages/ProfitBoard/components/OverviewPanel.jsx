import React from 'react';
import { Card, Empty, Spin, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { Info } from 'lucide-react';
import { timestamp2string } from '../../../helpers';

const { Text, Title } = Typography;

const OverviewPanel = ({
  overviewQuerying,
  overviewReport,
  cumulativeSummaryCards,
  diagnosticSummaryCards,
  summaryMetricHelp,
  formatMoney,
  status,
  t,
}) => (
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
                <Text>{t('配置利润')}：{formatMoney(item.configured_profit_usd, status)}</Text>
                <Text>{t('实际利润')}：{formatMoney(item.actual_profit_usd, status)}</Text>
              </div>
            </div>
          ))}
        </div>
      </Card>
    ) : null}
  </div>
);

export default OverviewPanel;
