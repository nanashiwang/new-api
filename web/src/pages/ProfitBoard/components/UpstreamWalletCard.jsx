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

/* ── 单张账户卡片 ───────────────────────────────────── */

const AccountCard = ({
  item,
  balanceMeta,
  statusMeta,
  balanceValue,
  syncTime,
  syncingAccountId,
  syncAccount,
  openEditSideSheet,
  formatMoney,
  status,
  t,
}) => (
  <div
    className={`rounded-xl border border-l-4 bg-semi-color-bg-1 ${balanceMeta.accentColor} transition-colors`}
  >
    {/* 头部：名称 + 操作 */}
    <div className='flex items-center justify-between gap-2 px-4 pt-3 pb-2'>
      <div className='flex min-w-0 flex-1 items-center gap-2'>
        <Title heading={6} ellipsis style={{ margin: 0, maxWidth: '60%' }}>
          {item.name}
        </Title>
        <Tag color={statusMeta.color} size='small'>
          {statusMeta.label}
        </Tag>
      </div>
      <Space spacing={4}>
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

    {/* 余额（核心信息，大字 + 颜色） */}
    <div className='px-4 pb-3'>
      <div className='flex items-baseline gap-2'>
        <span
          className={`text-2xl font-bold tabular-nums ${balanceMeta.amountTone}`}
        >
          {balanceValue}
        </span>
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium ${balanceMeta.badgeTone}`}
        >
          {balanceMeta.label}
        </span>
      </div>

      {/* 次要指标：累计消耗 + 同步时间 */}
      <div className='mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-semi-color-text-2'>
        <span>
          {t('累计消耗')}{' '}
          <span className='font-medium text-semi-color-text-0'>
            {formatMoney(item.wallet_used_total_usd, status)}
          </span>
        </span>
        <span>
          {t('同步')}{' '}
          <span className='font-medium text-semi-color-text-0'>
            {syncTime}
          </span>
        </span>
      </div>
    </div>

    {/* 错误信息（仅异常时显示） */}
    {item.error_message && (
      <div className='mx-4 mb-3 flex items-start gap-1.5 rounded-lg bg-red-500/5 px-3 py-1.5 text-xs text-red-600 dark:text-red-300'>
        <AlertCircle size={12} className='mt-0.5 shrink-0 text-red-500' />
        <span>{item.error_message}</span>
      </div>
    )}
  </div>
);

/* ── 主组件 ─────────────────────────────────────────── */

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
      <div className='grid gap-3 md:grid-cols-2 2xl:grid-cols-3'>
        {accounts.map((item) => {
          const statusMeta = getWalletStatusMeta(item.status, t);
          const balanceMeta = getAccountBalanceVisualMeta(item, status, t);
          const balanceValue =
            item.status === 'failed' && !item.last_success_at
              ? '--'
              : formatMoney(item.wallet_balance_usd, status);
          const syncTime = item.last_synced_at
            ? timestamp2string(item.last_synced_at)
            : '-';

          return (
            <AccountCard
              key={item.id}
              item={item}
              balanceMeta={balanceMeta}
              statusMeta={statusMeta}
              balanceValue={balanceValue}
              syncTime={syncTime}
              syncingAccountId={syncingAccountId}
              syncAccount={syncAccount}
              openEditSideSheet={openEditSideSheet}
              formatMoney={formatMoney}
              status={status}
              t={t}
            />
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
