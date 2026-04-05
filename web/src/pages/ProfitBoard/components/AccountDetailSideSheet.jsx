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
import { Empty, SideSheet, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import dayjs from 'dayjs';
import {
  formatUpstreamExpiryDate,
  formatUpstreamSubscriptionRemaining,
  getAccountResourceSummaryTones,
  getUpstreamAccountSuggestedName,
  normalizeUpstreamAccountResourceDisplayMode,
} from '../utils';

const { Text, Title } = Typography;

const SummaryItem = ({ label, value, tone = 'text-semi-color-text-0' }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <div className={`mt-2 text-lg font-semibold tabular-nums ${tone}`}>
      {value}
    </div>
  </div>
);

const SubscriptionRow = ({ item, formatMoney, t, status }) => {
  const remainingValue = item.has_unlimited
    ? t('不限额')
    : formatMoney(item.remaining_quota_usd, status);
  const totalValue = item.has_unlimited
    ? t('不限额')
    : formatMoney(item.total_quota_usd, status);

  return (
    <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div>
          <Text strong>{`${t('订阅')} #${item.subscription_id}`}</Text>
          <Text type='tertiary' size='small' className='ml-2'>
            {item.plan_id ? `${t('计划')} #${item.plan_id}` : t('未绑定计划')}
          </Text>
        </div>
        <Tag color='blue' size='small'>
          {item.status || t('运行中')}
        </Tag>
      </div>
      <div className='mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
        <SummaryItem label={t('订阅剩余')} value={remainingValue} />
        <SummaryItem
          label={t('订阅已用')}
          value={formatMoney(item.used_quota_usd, status)}
        />
        <SummaryItem label={t('订阅总额')} value={totalValue} />
        <SummaryItem
          label={t('最早到期')}
          value={formatUpstreamExpiryDate(item.end_time, t)}
        />
      </div>
      <div className='mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-semi-color-text-2'>
        <span>
          {t('下次重置')}{' '}
          <span className='font-medium text-semi-color-text-0'>
            {formatUpstreamExpiryDate(item.next_reset_time, t)}
          </span>
        </span>
        <span>
          {t('开始时间')}{' '}
          <span className='font-medium text-semi-color-text-0'>
            {item.start_time > 0
              ? dayjs.unix(item.start_time).format('YYYY-MM-DD')
              : '-'}
          </span>
        </span>
      </div>
    </div>
  );
};

const AccountDetailSideSheet = ({
  visible,
  onClose,
  account,
  accountTrend,
  accountTrendLoading,
  formatMoney,
  status,
  t,
}) => {
  const subscriptions = accountTrend?.subscriptions || [];
  const summaryTones = getAccountResourceSummaryTones(account);
  const domainLabel = getUpstreamAccountSuggestedName(account?.base_url);
  const resourceDisplayMode = normalizeUpstreamAccountResourceDisplayMode(
    account?.resource_display_mode,
  );
  const showWallet =
    resourceDisplayMode === 'both' || resourceDisplayMode === 'wallet';
  const showSubscription =
    resourceDisplayMode === 'both' || resourceDisplayMode === 'subscription';

  return (
    <SideSheet
      visible={visible}
      onCancel={onClose}
      width={720}
      title={account?.name || t('账户详情')}
    >
      <Spin spinning={accountTrendLoading}>
        {account ? (
          <div className='space-y-4'>
            <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='mb-3'>
                <Title heading={6} style={{ margin: 0 }}>
                  {t('账户总览')}
                </Title>
                <Text type='tertiary' size='small' className='mt-1 block'>
                  {domainLabel || account.base_url}
                </Text>
                {domainLabel && domainLabel !== account.base_url ? (
                  <Text type='tertiary' size='small' className='mt-1 block'>
                    {account.base_url}
                  </Text>
                ) : null}
              </div>
              <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
                {showWallet ? (
                  <SummaryItem
                    label={t('钱包剩余')}
                    value={formatMoney(account.wallet_balance_usd, status)}
                    tone={summaryTones.wallet}
                  />
                ) : null}
                {showWallet ? (
                  <SummaryItem
                    label={t('累计已用')}
                    value={formatMoney(account.wallet_used_total_usd, status)}
                  />
                ) : null}
                {showSubscription ? (
                  <SummaryItem
                    label={t('订阅剩余')}
                    value={formatUpstreamSubscriptionRemaining(account, status, t)}
                    tone={summaryTones.subscription}
                  />
                ) : null}
                {showSubscription ? (
                  <SummaryItem
                    label={t('最早到期')}
                    value={formatUpstreamExpiryDate(
                      account.subscription_earliest_expire_at,
                      t,
                    )}
                  />
                ) : null}
              </div>
            </div>

            {showSubscription ? (
              <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
                <div className='mb-3'>
                  <Title heading={6} style={{ margin: 0 }}>
                    {t('订阅明细')}
                  </Title>
                  <Text type='tertiary' size='small' className='mt-1 block'>
                    {t('按最早到期时间排序')}
                  </Text>
                </div>
                {subscriptions.length > 0 ? (
                  <div className='space-y-3'>
                    {subscriptions.map((item) => (
                      <SubscriptionRow
                        key={item.subscription_id}
                        item={item}
                        formatMoney={formatMoney}
                        status={status}
                        t={t}
                      />
                    ))}
                  </div>
                ) : (
                  <Empty image={null} description={t('未获取到订阅数据')} />
                )}
              </div>
            ) : null}
          </div>
        ) : (
          <Empty image={null} description={t('请选择一个账户')} />
        )}
      </Spin>
    </SideSheet>
  );
};

export default AccountDetailSideSheet;
