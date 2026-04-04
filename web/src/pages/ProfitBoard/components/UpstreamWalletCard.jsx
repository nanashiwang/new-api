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
import {
  Button,
  Card,
  Empty,
  Space,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  AlertCircle,
  Pencil,
  Plus,
  RefreshCw,
  Wallet,
} from 'lucide-react';
import { timestamp2string } from '../../../helpers';
import { getBalanceHealthLevel, getWalletStatusMeta } from '../utils';
import AccountEditSideSheet from './AccountEditSideSheet';

const { Text, Title } = Typography;

const InfoMetric = ({ label, value, emphasis }) => (
  <div className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-3 py-2.5'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <div className={`mt-1 text-sm font-semibold ${emphasis || ''}`}>
      {value}
    </div>
  </div>
);

const UpstreamWalletCard = ({
  accounts,
  accountDraft,
  setAccountDraft,
  editingAccountId,
  setEditingAccountId,
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
}) => {
  const enabledAccounts = useMemo(
    () => accounts.filter((item) => item.enabled !== false),
    [accounts],
  );
  const selectedAccount =
    accounts.find((item) => item.id === editingAccountId) || null;

  const summary = useMemo(() => {
    let totalBalance = 0;
    let totalUsed = 0;
    let latestSyncedAt = 0;
    let criticalCount = 0;
    let warningCount = 0;
    enabledAccounts.forEach((item) => {
      totalBalance += Number(item.wallet_balance_usd || 0);
      totalUsed += Number(item.wallet_used_total_usd || 0);
      latestSyncedAt = Math.max(
        latestSyncedAt,
        Number(item.last_synced_at || 0),
      );
      const health = getBalanceHealthLevel(item, t);
      if (health.level === 'critical') criticalCount++;
      else if (health.level === 'warning') warningCount++;
    });
    return {
      totalBalance,
      totalUsed,
      latestSyncedAt,
      count: enabledAccounts.length,
      criticalCount,
      warningCount,
    };
  }, [enabledAccounts, t]);

  return (
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
          <Tag color='blue' size='small'>
            {summary.count} {t('个账户')}
          </Tag>
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
        <div className='space-y-4'>
          {/* 汇总指标 */}
          <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
            <InfoMetric
              label={t('当前余额')}
              value={formatMoney(summary.totalBalance, status)}
              emphasis='text-emerald-600 dark:text-emerald-400'
            />
            <InfoMetric
              label={t('历史累计已用')}
              value={formatMoney(summary.totalUsed, status)}
              emphasis='text-amber-600 dark:text-amber-400'
            />
            <InfoMetric
              label={t('最近同步')}
              value={
                summary.latestSyncedAt
                  ? timestamp2string(summary.latestSyncedAt)
                  : '-'
              }
            />
            {/* 余额健康汇总 */}
            <div
              className={`rounded-lg border px-3 py-2.5 ${
                summary.criticalCount > 0
                  ? 'border-red-500/30 bg-red-500/5'
                  : summary.warningCount > 0
                    ? 'border-amber-500/30 bg-amber-500/5'
                    : 'border-emerald-500/30 bg-emerald-500/5'
              }`}
            >
              <Text type='tertiary' size='small'>
                {t('余额状态')}
              </Text>
              <div className='mt-1 text-sm font-semibold'>
                {summary.criticalCount > 0 ? (
                  <span className='text-red-600 dark:text-red-400'>
                    {t('{{count}} 个偏低', { count: summary.criticalCount })}
                  </span>
                ) : summary.warningCount > 0 ? (
                  <span className='text-amber-600 dark:text-amber-400'>
                    {t('{{count}} 个需注意', { count: summary.warningCount })}
                  </span>
                ) : (
                  <span className='text-emerald-600 dark:text-emerald-400'>
                    {t('全部正常')}
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* 账户列表 + 详情 */}
          <div className='grid gap-4 xl:grid-cols-[340px_minmax(0,1fr)]'>
            <div className='space-y-2'>
              {accounts.map((item) => {
                const statusMeta = getWalletStatusMeta(item.status, t);
                const health = getBalanceHealthLevel(item, t);
                const isSelected = editingAccountId === item.id;
                return (
                  <button
                    key={item.id}
                    type='button'
                    onClick={() => setEditingAccountId(item.id)}
                    className={`w-full rounded-xl border p-3 text-left transition ${
                      isSelected
                        ? 'border-semi-color-primary bg-semi-color-primary-light-default'
                        : 'border-semi-color-border bg-semi-color-bg-1 hover:border-semi-color-primary-hover'
                    }`}
                  >
                    <div className='flex items-start justify-between gap-2'>
                      <div className='min-w-0'>
                        <div className='flex items-center gap-2'>
                          <Text strong className='truncate'>
                            {item.name}
                          </Text>
                          {/* 余额健康色点 */}
                          <Tooltip content={health.label}>
                            <div
                              className={`h-2.5 w-2.5 rounded-full ${
                                health.level === 'critical'
                                  ? 'bg-red-500'
                                  : health.level === 'warning'
                                    ? 'bg-amber-500'
                                    : 'bg-emerald-500'
                              }`}
                            />
                          </Tooltip>
                        </div>
                        <Tooltip content={item.base_url || '-'}>
                          <div className='mt-1 truncate text-xs text-semi-color-text-2'>
                            {item.base_url || '-'}
                          </div>
                        </Tooltip>
                      </div>
                      <div
                        className='flex items-center gap-2'
                        onClick={(event) => event.stopPropagation()}
                      >
                        <Tag color={statusMeta.color} size='small'>
                          {statusMeta.label}
                        </Tag>
                        <Button
                          type='tertiary'
                          icon={<RefreshCw size={14} />}
                          loading={syncingAccountId === item.id}
                          onClick={() => syncAccount(item.id)}
                          size='small'
                        />
                      </div>
                    </div>

                    <div className='mt-2 flex items-baseline gap-2'>
                      <span className={`text-sm font-semibold ${health.textColor}`}>
                        {formatMoney(item.wallet_balance_usd, status)}
                      </span>
                      <Text type='tertiary' size='small'>
                        {health.label}
                      </Text>
                    </div>
                  </button>
                );
              })}
            </div>

            <div className='space-y-3'>
              {selectedAccount ? (
                <>
                  <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
                    <div className='flex flex-wrap items-start justify-between gap-3'>
                      <div className='min-w-0'>
                        <Title heading={6} style={{ margin: 0 }}>
                          {selectedAccount.name}
                        </Title>
                        <Text type='tertiary' size='small'>
                          {selectedAccount.remark || selectedAccount.base_url}
                        </Text>
                      </div>
                      <Space wrap>
                        <Tag
                          color={
                            getWalletStatusMeta(selectedAccount.status, t).color
                          }
                          size='small'
                        >
                          {getWalletStatusMeta(selectedAccount.status, t).label}
                        </Tag>
                        <Button
                          type='tertiary'
                          icon={<RefreshCw size={14} />}
                          loading={syncingAccountId === selectedAccount.id}
                          onClick={() => syncAccount(selectedAccount.id)}
                          size='small'
                        >
                          {t('同步')}
                        </Button>
                        <Button
                          type='tertiary'
                          icon={<Pencil size={14} />}
                          onClick={() => openEditSideSheet(selectedAccount.id)}
                          size='small'
                        >
                          {t('编辑')}
                        </Button>
                      </Space>
                    </div>

                    {/* 关键指标：余额 + 累计已用 */}
                    <div className='mt-4 grid gap-3 sm:grid-cols-2'>
                      {(() => {
                        const health = getBalanceHealthLevel(selectedAccount, t);
                        return (
                          <div
                            className={`rounded-lg border px-3 py-2.5 ${
                              health.level === 'critical'
                                ? 'border-red-500/30 bg-red-500/5'
                                : health.level === 'warning'
                                  ? 'border-amber-500/30 bg-amber-500/5'
                                  : 'border-emerald-500/30 bg-emerald-500/5'
                            }`}
                          >
                            <Text type='tertiary' size='small'>
                              {t('当前余额')}
                            </Text>
                            <div
                              className={`mt-1 text-lg font-bold ${health.textColor}`}
                            >
                              {formatMoney(
                                selectedAccount.wallet_balance_usd,
                                status,
                              )}
                            </div>
                            <Text
                              size='small'
                              className={`mt-0.5 block ${health.textColor}`}
                            >
                              {health.label}
                            </Text>
                          </div>
                        );
                      })()}
                      <InfoMetric
                        label={t('历史累计已用')}
                        value={formatMoney(
                          selectedAccount.wallet_used_total_usd,
                          status,
                        )}
                        emphasis='text-amber-600 dark:text-amber-400'
                      />
                    </div>

                    <div className='mt-3 grid gap-3 sm:grid-cols-2'>
                      <InfoMetric
                        label={t('最近同步')}
                        value={
                          selectedAccount.last_synced_at
                            ? timestamp2string(selectedAccount.last_synced_at)
                            : '-'
                        }
                      />
                      <InfoMetric
                        label={t('最近成功')}
                        value={
                          selectedAccount.last_success_at
                            ? timestamp2string(selectedAccount.last_success_at)
                            : '-'
                        }
                      />
                    </div>

                    {(selectedAccount.subscription_total_quota_usd > 0 ||
                      selectedAccount.subscription_used_quota_usd > 0 ||
                      selectedAccount.low_balance_threshold_usd > 0) && (
                      <div className='mt-3 grid gap-3 sm:grid-cols-3'>
                        <InfoMetric
                          label={t('订阅总额')}
                          value={
                            selectedAccount.subscription_total_quota_usd > 0
                              ? formatMoney(
                                  selectedAccount.subscription_total_quota_usd,
                                  status,
                                )
                              : t('不限额或未知')
                          }
                        />
                        <InfoMetric
                          label={t('订阅已用')}
                          value={formatMoney(
                            selectedAccount.subscription_used_quota_usd,
                            status,
                          )}
                        />
                        <InfoMetric
                          label={t('低余额提醒线')}
                          value={
                            selectedAccount.low_balance_threshold_usd > 0
                              ? formatMoney(
                                  selectedAccount.low_balance_threshold_usd,
                                  status,
                                )
                              : t('未设置')
                          }
                        />
                      </div>
                    )}

                    {selectedAccount.status === 'needs_baseline' ? (
                      <div className='mt-3 rounded-lg border border-blue-500/20 bg-blue-500/5 px-3 py-2 text-sm text-semi-color-text-1'>
                        {t(
                          '首次同步只会拿到当前余额和历史累计已用，下一次同步后数据更完整。',
                        )}
                      </div>
                    ) : null}

                    {selectedAccount.error_message ? (
                      <div className='mt-3 flex items-start gap-2 rounded-lg border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm'>
                        <AlertCircle
                          size={14}
                          className='mt-0.5 shrink-0 text-red-500'
                        />
                        <span>{selectedAccount.error_message}</span>
                      </div>
                    ) : null}
                  </div>
                </>
              ) : (
                <div className='rounded-xl border border-dashed border-semi-color-border bg-semi-color-bg-1 p-8'>
                  <Empty image={null} description={t('选择一个账户查看详情')} />
                </div>
              )}
            </div>
          </div>
        </div>
      ) : (
        <Empty image={null} description={t('点击右上角新建账户')} />
      )}

      <AccountEditSideSheet
        visible={sideSheetVisible}
        onClose={closeSideSheet}
        accountDraft={accountDraft}
        setAccountDraft={setAccountDraft}
        saveAccount={saveAccount}
        deleteAccount={deleteAccount}
        savingAccount={savingAccount}
        deletingAccountId={deletingAccountId}
      />
    </Card>
  );
};

export default UpstreamWalletCard;
