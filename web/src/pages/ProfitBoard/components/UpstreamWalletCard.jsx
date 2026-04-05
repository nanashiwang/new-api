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
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  AlertCircle,
  Pencil,
  Plus,
  RefreshCw,
  TrendingUp,
  Wallet,
} from 'lucide-react';
import { CHART_CONFIG } from '../../../constants/dashboard.constants';
import { timestamp2string } from '../../../helpers';
import {
  createAccountUsageTrendSpec,
  getAccountBalanceVisualMeta,
  getWalletStatusMeta,
} from '../utils';
import AccountEditSideSheet from './AccountEditSideSheet';
import ResponsiveVChart from './ResponsiveVChart';

const { Text, Title } = Typography;

/* ── 指标行 ─────────────────────────────────────────── */

const MetricRow = ({ label, value, truncate }) => (
  <div className='flex items-baseline justify-between gap-3'>
    <span className='shrink-0 text-semi-color-text-2'>{label}</span>
    <span
      className={`font-medium text-semi-color-text-0 text-right${truncate ? ' truncate' : ''}`}
    >
      {value}
    </span>
  </div>
);

/* ── 账户列表行 ─────────────────────────────────────── */

const AccountListRow = ({
  item,
  isSelected,
  balanceMeta,
  balanceValue,
  syncingAccountId,
  syncAccount,
  onSelect,
}) => (
  <div
    role='button'
    tabIndex={0}
    onClick={() => onSelect(item.id)}
    onKeyDown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') onSelect(item.id);
    }}
    className={`flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 transition-colors ${
      isSelected
        ? 'bg-semi-color-primary-light-default ring-1 ring-semi-color-primary'
        : 'hover:bg-semi-color-fill-0'
    }`}
  >
    {/* 状态圆点 */}
    <span
      className={`h-2.5 w-2.5 shrink-0 rounded-full ${balanceMeta.accentColor.replace('border-l-', 'bg-')}`}
    />

    {/* 名称 */}
    <span className='min-w-0 flex-1 truncate text-sm font-medium text-semi-color-text-0'>
      {item.name}
    </span>

    {/* 余额 */}
    <span
      className={`shrink-0 text-sm font-semibold tabular-nums ${balanceMeta.amountTone}`}
    >
      {balanceValue}
    </span>

    {/* 健康徽章 */}
    <span
      className={`hidden shrink-0 rounded-full px-2 py-0.5 text-xs font-medium sm:inline-flex ${balanceMeta.badgeTone}`}
    >
      {balanceMeta.label}
    </span>

    {/* 同步 */}
    <Button
      type='tertiary'
      icon={<RefreshCw size={14} />}
      loading={syncingAccountId === item.id}
      onClick={(e) => {
        e.stopPropagation();
        syncAccount(item.id);
      }}
      size='small'
      className='shrink-0'
    />
  </div>
);

/* ── 选中账户详情面板 ───────────────────────────────── */

const AccountDetailSection = ({
  account,
  balanceMeta,
  statusMeta,
  balanceValue,
  syncTime,
  trendSpec,
  accountTrendLoading,
  hasTrendRows,
  syncingAccountId,
  syncAccount,
  openEditSideSheet,
  formatMoney,
  status,
  isMobile,
  t,
}) => (
  <div
    className={`mt-3 rounded-xl border border-semi-color-border bg-semi-color-bg-2 p-4 ${
      isMobile ? '' : 'grid grid-cols-5 gap-6'
    }`}
  >
    {/* 左侧：指标 */}
    <div className={isMobile ? '' : 'col-span-2 space-y-4'}>
      {/* 状态 + 健康 */}
      <div className='flex items-center gap-2'>
        <Tag color={statusMeta.color} size='small'>
          {statusMeta.label}
        </Tag>
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium ${balanceMeta.badgeTone}`}
        >
          {balanceMeta.label}
        </span>
      </div>

      {/* 余额大字 */}
      <div>
        <div className='flex items-baseline gap-2'>
          <span
            className={`text-3xl font-bold tabular-nums ${balanceMeta.amountTone}`}
          >
            {balanceValue}
          </span>
        </div>
        {balanceMeta.helper && (
          <Text type='tertiary' size='small' className='mt-1 block'>
            {balanceMeta.helper}
          </Text>
        )}
      </div>

      {/* 描述列表 */}
      <div className='space-y-2 text-sm'>
        <MetricRow
          label={t('累计消耗')}
          value={formatMoney(account.wallet_used_total_usd, status)}
        />
        <MetricRow label={t('最后同步')} value={syncTime} />
        {account.base_url && (
          <MetricRow
            label={t('来源')}
            value={account.base_url.replace(/^https?:\/\//, '')}
            truncate
          />
        )}
        {(account.subscription_total_quota_usd > 0 ||
          account.subscription_used_quota_usd > 0) && (
          <>
            <MetricRow
              label={t('订阅总额')}
              value={
                account.subscription_total_quota_usd > 0
                  ? formatMoney(account.subscription_total_quota_usd, status)
                  : t('不限额')
              }
            />
            <MetricRow
              label={t('订阅已用')}
              value={formatMoney(account.subscription_used_quota_usd, status)}
            />
          </>
        )}
      </div>

      {/* 提示 */}
      {account.status === 'needs_baseline' && (
        <div className='rounded-lg bg-blue-500/5 px-3 py-1.5 text-xs text-semi-color-text-1'>
          {t('首次同步后，下次开始统计近 7 天已用')}
        </div>
      )}
      {account.error_message && (
        <div className='flex items-start gap-1.5 rounded-lg bg-red-500/5 px-3 py-1.5 text-xs text-red-600 dark:text-red-300'>
          <AlertCircle size={12} className='mt-0.5 shrink-0 text-red-500' />
          <span>{account.error_message}</span>
        </div>
      )}

      {/* 操作 */}
      <div className='flex gap-2 pt-1'>
        <Button
          type='tertiary'
          icon={<Pencil size={14} />}
          onClick={() => openEditSideSheet(account.id)}
          size='small'
        >
          {t('编辑')}
        </Button>
        <Button
          type='tertiary'
          icon={<RefreshCw size={14} />}
          loading={syncingAccountId === account.id}
          onClick={() => syncAccount(account.id)}
          size='small'
        >
          {t('同步')}
        </Button>
      </div>
    </div>

    {/* 右侧：趋势图 */}
    <div className={isMobile ? 'mt-4' : 'col-span-3'}>
      <div className='mb-2 flex items-center gap-2'>
        <TrendingUp size={14} className='text-semi-color-text-2' />
        <span className='text-sm font-medium text-semi-color-text-1'>
          {t('近 7 天已用趋势')}
        </span>
      </div>
      <Spin spinning={accountTrendLoading}>
        {hasTrendRows && trendSpec ? (
          <ResponsiveVChart
            chartKey={`account-trend-${account.id}`}
            spec={trendSpec}
            option={CHART_CONFIG}
            minHeight={240}
          />
        ) : (
          <div className='flex h-[240px] items-center justify-center'>
            <Empty
              image={null}
              description={
                accountTrendLoading
                  ? t('加载中…')
                  : account.status === 'needs_baseline'
                    ? t('首次同步后才会有趋势数据')
                    : t('暂无趋势数据')
              }
            />
          </div>
        )}
      </Spin>
    </div>
  </div>
);

/* ── 主组件 ─────────────────────────────────────────── */

const UpstreamWalletCard = ({
  accounts,
  editingAccountId,
  setEditingAccountId,
  editingAccount,
  accountTrend,
  accountTrendLoading,
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
  sideSheetVisible,
  openEditSideSheet,
  openCreateSideSheet,
  closeSideSheet,
  formatMoney,
  status,
  isMobile,
  t,
}) => {
  // 趋势数据：兼容数组或 { rows } 两种结构
  const trendRows = useMemo(() => {
    if (!accountTrend) return [];
    return Array.isArray(accountTrend)
      ? accountTrend
      : accountTrend?.rows || [];
  }, [accountTrend]);

  const trendSpec = useMemo(() => {
    if (!trendRows.length) return null;
    return createAccountUsageTrendSpec(trendRows, status, t);
  }, [trendRows, status, t]);

  // 选中账户的显示元数据
  const selectedMeta = useMemo(() => {
    if (!editingAccount) return null;
    return {
      statusMeta: getWalletStatusMeta(editingAccount.status, t),
      balanceMeta: getAccountBalanceVisualMeta(editingAccount, status, t),
      balanceValue:
        editingAccount.status === 'failed' && !editingAccount.last_success_at
          ? '--'
          : formatMoney(editingAccount.wallet_balance_usd, status),
      syncTime: editingAccount.last_synced_at
        ? timestamp2string(editingAccount.last_synced_at)
        : '-',
    };
  }, [editingAccount, formatMoney, status, t]);

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
        <>
          {/* 列表区 */}
          <div className='space-y-1'>
            {accounts.map((item) => {
              const balanceMeta = getAccountBalanceVisualMeta(
                item,
                status,
                t,
              );
              const balanceValue =
                item.status === 'failed' && !item.last_success_at
                  ? '--'
                  : formatMoney(item.wallet_balance_usd, status);
              return (
                <AccountListRow
                  key={item.id}
                  item={item}
                  isSelected={editingAccountId === item.id}
                  balanceMeta={balanceMeta}
                  balanceValue={balanceValue}
                  syncingAccountId={syncingAccountId}
                  syncAccount={syncAccount}
                  onSelect={setEditingAccountId}
                />
              );
            })}
          </div>

          {/* 详情区 */}
          {editingAccount && selectedMeta && (
            <AccountDetailSection
              account={editingAccount}
              balanceMeta={selectedMeta.balanceMeta}
              statusMeta={selectedMeta.statusMeta}
              balanceValue={selectedMeta.balanceValue}
              syncTime={selectedMeta.syncTime}
              trendSpec={trendSpec}
              accountTrendLoading={accountTrendLoading}
              hasTrendRows={trendRows.length > 0}
              syncingAccountId={syncingAccountId}
              syncAccount={syncAccount}
              openEditSideSheet={openEditSideSheet}
              formatMoney={formatMoney}
              status={status}
              isMobile={isMobile}
              t={t}
            />
          )}
        </>
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
};

export default UpstreamWalletCard;
