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
import { Button, Card, Empty, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { AlertCircle, Pencil, Plus, RefreshCw, Wallet } from 'lucide-react';
import { timestamp2string } from '../../../helpers';
import { getAccountBalanceVisualMeta, getWalletStatusMeta } from '../utils';
import AccountEditSideSheet from './AccountEditSideSheet';

const { Text, Title } = Typography;

const InfoMetric = ({ label, value }) => (
  <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-3'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <div className='mt-1 text-sm font-semibold tabular-nums'>{value}</div>
  </div>
);

const UpstreamWalletCard = ({
  accounts,
  accountDraft,
  updateAccountDraftField,
  normalizeAccountDraftBaseUrl,
  touchAccountDraftField,
  accountDraftErrors,
  accountDraftCanSave,
  accountDraftValidation,
  saveAccount,
  syncAccount,
  syncAllAccounts,
  deleteAccount,
  savingAccount,
  syncingAccountId,
  syncingAllAccounts,
  deletingAccountId,
  formatMoney,
  status,
  openEditSideSheet,
  openCreateSideSheet,
  sideSheetVisible,
  closeSideSheet,
  t,
}) => (
  <Card
    bordered={false}
    className='rounded-xl'
    title={
      <div className='flex items-center gap-2'>
        <Wallet size={16} />
        <span>{t('上游账户')}</span>
      </div>
    }
    headerExtraContent={
      <Space wrap>
        <Button
          type='tertiary'
          icon={<RefreshCw size={14} />}
          loading={syncingAllAccounts}
          onClick={syncAllAccounts}
          size='small'
        >
          {t('全部同步')}
        </Button>
        <Button
          theme='solid'
          type='primary'
          icon={<Plus size={14} />}
          onClick={openCreateSideSheet}
          size='small'
        >
          {t('新建')}
        </Button>
      </Space>
    }
  >
    {accounts.length > 0 ? (
      <div className='grid gap-4 md:grid-cols-2 2xl:grid-cols-3'>
        {accounts.map((item) => {
          const statusMeta = getWalletStatusMeta(item.status, t);
          const balanceMeta = getAccountBalanceVisualMeta(item, status, t);
          const balanceTitle =
            item.status === 'failed' && item.last_success_at
              ? t('最近有效余额')
              : t('当前余额');
          const balanceValue =
            item.status === 'failed' && !item.last_success_at
              ? '--'
              : formatMoney(item.wallet_balance_usd, status);
          const subtitle = item.remark || item.base_url || '-';
          const syncTime = item.last_synced_at
            ? timestamp2string(item.last_synced_at)
            : '-';

          return (
            <div
              key={item.id}
              className='rounded-3xl border border-semi-color-border bg-semi-color-bg-1 p-5 shadow-sm transition-colors'
            >
              <div className='flex items-start justify-between gap-4'>
                <div className='min-w-0 flex-1 space-y-2'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Title heading={5} style={{ margin: 0 }}>
                      {item.name}
                    </Title>
                    <Tag color={statusMeta.color} size='small'>
                      {statusMeta.label}
                    </Tag>
                  </div>
                  <Text
                    type='tertiary'
                    size='small'
                    className='block break-all leading-5'
                  >
                    {subtitle}
                  </Text>
                </div>

                <Space>
                  <Button
                    type='tertiary'
                    icon={<RefreshCw size={14} />}
                    loading={syncingAccountId === item.id}
                    onClick={() => syncAccount(item.id)}
                    size='small'
                  />
                  <Button
                    type='tertiary'
                    icon={<Pencil size={14} />}
                    onClick={() => openEditSideSheet(item.id)}
                    size='small'
                  />
                </Space>
              </div>

              <div
                className={`mt-5 rounded-3xl border px-4 py-4 ${balanceMeta.panelTone}`}
              >
                <div className='flex items-start justify-between gap-3'>
                  <div className='min-w-0 flex-1'>
                    <span
                      className={`inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-xs font-medium ${balanceMeta.eyebrowTone}`}
                    >
                      <span
                        className={`h-2 w-2 rounded-full ${balanceMeta.dotTone}`}
                      />
                      {balanceTitle}
                    </span>
                    <div
                      className={`mt-3 text-4xl font-semibold tracking-tight tabular-nums ${balanceMeta.amountTone}`}
                    >
                      {balanceValue}
                    </div>
                    <div className='mt-3 flex flex-wrap gap-2'>
                      <span
                        className={`rounded-full border px-2.5 py-1 text-xs font-medium ${balanceMeta.badgeTone}`}
                      >
                        {balanceMeta.label}
                      </span>
                      <span
                        className={`rounded-full border px-2.5 py-1 text-xs font-medium ${balanceMeta.rangeTone}`}
                      >
                        {balanceMeta.rangeLabel}
                      </span>
                    </div>
                  </div>
                </div>

                <Text
                  size='small'
                  className={`mt-4 block leading-5 ${balanceMeta.helperTone}`}
                >
                  {balanceMeta.helper}
                </Text>
              </div>

              <div className='mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3'>
                <InfoMetric
                  label={t('累计消耗')}
                  value={formatMoney(item.wallet_used_total_usd, status)}
                />
                <InfoMetric label={t('最近同步')} value={syncTime} />
                <InfoMetric
                  label={t('同步来源')}
                  value={
                    item.base_url
                      ? item.base_url.replace(/^https?:\/\//, '')
                      : '-'
                  }
                />
              </div>

              {(item.subscription_total_quota_usd > 0 ||
                item.subscription_used_quota_usd > 0) && (
                <div className='mt-3 grid gap-3 sm:grid-cols-2'>
                  <InfoMetric
                    label={t('订阅总额')}
                    value={
                      item.subscription_total_quota_usd > 0
                        ? formatMoney(item.subscription_total_quota_usd, status)
                        : t('不限额或未知')
                    }
                  />
                  <InfoMetric
                    label={t('订阅已用')}
                    value={formatMoney(
                      item.subscription_used_quota_usd,
                      status,
                    )}
                  />
                </div>
              )}

              {item.status === 'needs_baseline' ? (
                <div className='mt-3 rounded-xl border border-blue-500/20 bg-blue-500/5 px-3 py-2 text-sm text-semi-color-text-1'>
                  {t('首次同步后，下次开始统计近 7 天已用')}
                </div>
              ) : null}

              {item.error_message ? (
                <div className='mt-3 flex items-start gap-2 rounded-2xl border border-red-500/20 bg-red-500/5 px-3 py-2.5 text-sm text-red-600 dark:text-red-300'>
                  <AlertCircle
                    size={14}
                    className='mt-0.5 shrink-0 text-red-500'
                  />
                  <span>{item.error_message}</span>
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    ) : (
      <Empty image={null} description={t('点击右上角新建账户')} />
    )}

    <AccountEditSideSheet
      visible={sideSheetVisible}
      onClose={closeSideSheet}
      accountDraft={accountDraft}
      updateAccountDraftField={updateAccountDraftField}
      normalizeAccountDraftBaseUrl={normalizeAccountDraftBaseUrl}
      touchAccountDraftField={touchAccountDraftField}
      accountDraftErrors={accountDraftErrors}
      accountDraftCanSave={accountDraftCanSave}
      accountDraftValidation={accountDraftValidation}
      saveAccount={saveAccount}
      deleteAccount={deleteAccount}
      savingAccount={savingAccount}
      deletingAccountId={deletingAccountId}
      t={t}
    />
  </Card>
);

export default UpstreamWalletCard;
