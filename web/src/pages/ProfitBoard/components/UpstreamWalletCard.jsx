import React, { useMemo } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  InputNumber,
  Space,
  Switch,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import {
  AlertCircle,
  KeyRound,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Trash2,
  Wallet,
} from 'lucide-react';
import { timestamp2string } from '../../../helpers';
import { createAccountUsageTrendSpec, getWalletStatusMeta } from '../utils';

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
  accountTrend,
  accountTrendLoading,
  accountDraft,
  setAccountDraft,
  editingAccountId,
  setEditingAccountId,
  saveAccount,
  syncAccount,
  syncAllAccounts,
  deleteAccount,
  resetAccountDraft,
  savingAccount,
  syncingAccountId,
  syncingAllAccounts,
  deletingAccountId,
  formatMoney,
  status,
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
    let totalPeriod = 0;
    let latestSyncedAt = 0;
    enabledAccounts.forEach((item) => {
      totalBalance += Number(item.wallet_balance_usd || 0);
      totalUsed += Number(item.wallet_used_total_usd || 0);
      totalPeriod += Number(item.period_used_usd || 0);
      latestSyncedAt = Math.max(
        latestSyncedAt,
        Number(item.last_synced_at || 0),
      );
    });
    return {
      totalBalance,
      totalUsed,
      totalPeriod,
      latestSyncedAt,
      count: enabledAccounts.length,
    };
  }, [enabledAccounts]);

  const trendSpec = useMemo(() => {
    const rows = accountTrend?.points || [];
    if (!rows.length) return null;
    return createAccountUsageTrendSpec(rows, status, t);
  }, [accountTrend?.points, status, t]);

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
          >
            {t('全部同步')}
          </Button>
          <Button
            theme='solid'
            type='primary'
            icon={<Plus size={14} />}
            onClick={resetAccountDraft}
          >
            {t('新建账户')}
          </Button>
        </Space>
      }
    >
      {accounts.length > 0 ? (
        <div className='space-y-4'>
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
              label={t('近 7 天已用')}
              value={formatMoney(summary.totalPeriod, status)}
              emphasis='text-rose-600 dark:text-rose-400'
            />
            <InfoMetric
              label={t('最近同步')}
              value={
                summary.latestSyncedAt
                  ? timestamp2string(summary.latestSyncedAt)
                  : '-'
              }
            />
          </div>

          <div className='grid gap-4 xl:grid-cols-[340px_minmax(0,1fr)]'>
            <div className='space-y-3'>
              {accounts.map((item) => {
                const statusMeta = getWalletStatusMeta(item.status, t);
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
                          {item.low_balance_alert ? (
                            <Tooltip content={t('当前余额已经低于提醒线')}>
                              <AlertCircle size={14} className='text-red-500' />
                            </Tooltip>
                          ) : null}
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
                        />
                      </div>
                    </div>

                    <div className='mt-3 grid gap-2 sm:grid-cols-2'>
                      <InfoMetric
                        label={t('当前余额')}
                        value={formatMoney(item.wallet_balance_usd, status)}
                        emphasis='text-emerald-600 dark:text-emerald-400'
                      />
                      <InfoMetric
                        label={t('近 7 天已用')}
                        value={formatMoney(item.period_used_usd, status)}
                        emphasis='text-rose-600 dark:text-rose-400'
                      />
                      <InfoMetric
                        label={t('历史累计已用')}
                        value={formatMoney(item.wallet_used_total_usd, status)}
                        emphasis='text-amber-600 dark:text-amber-400'
                      />
                      <InfoMetric
                        label={t('上次同步')}
                        value={
                          item.last_synced_at
                            ? timestamp2string(item.last_synced_at)
                            : '-'
                        }
                      />
                    </div>

                    {item.low_balance_threshold_usd > 0 ? (
                      <div className='mt-2 text-xs text-semi-color-text-2'>
                        {t('提醒线')}:{' '}
                        {formatMoney(item.low_balance_threshold_usd, status)}
                      </div>
                    ) : null}
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
                        {selectedAccount.low_balance_alert ? (
                          <Tag color='red' size='small'>
                            {t('余额偏低')}
                          </Tag>
                        ) : null}
                        {selectedAccount.quota_per_unit_mismatch ? (
                          <Tag color='orange' size='small'>
                            {t('额度倍率不同')}
                          </Tag>
                        ) : null}
                        <Button
                          type='tertiary'
                          icon={<RefreshCw size={14} />}
                          loading={syncingAccountId === selectedAccount.id}
                          onClick={() => syncAccount(selectedAccount.id)}
                        >
                          {t('刷新')}
                        </Button>
                      </Space>
                    </div>

                    <div className='mt-4 grid gap-3 sm:grid-cols-3'>
                      <InfoMetric
                        label={t('当前余额')}
                        value={formatMoney(
                          selectedAccount.wallet_balance_usd,
                          status,
                        )}
                        emphasis='text-emerald-600 dark:text-emerald-400'
                      />
                      <InfoMetric
                        label={t('历史累计已用')}
                        value={formatMoney(
                          selectedAccount.wallet_used_total_usd,
                          status,
                        )}
                        emphasis='text-amber-600 dark:text-amber-400'
                      />
                      <InfoMetric
                        label={t('近 7 天已用')}
                        value={formatMoney(
                          selectedAccount.period_used_usd,
                          status,
                        )}
                        emphasis='text-rose-600 dark:text-rose-400'
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
                          '首次同步只会拿到当前余额和历史累计已用，下一次同步后才会开始统计近 7 天已用。',
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

                  <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
                    <div className='mb-3 flex items-center justify-between gap-2'>
                      <Text strong>{t('近 7 天已用趋势')}</Text>
                      <Text type='tertiary' size='small'>
                        {t('按同步快照增量统计')}
                      </Text>
                    </div>
                    {trendSpec ? (
                      <VChart
                        forceInit
                        spec={trendSpec}
                        style={{ width: '100%', height: 260 }}
                      />
                    ) : accountTrendLoading ? (
                      <div className='py-12 text-center text-sm text-semi-color-text-2'>
                        {t('正在加载趋势')}
                      </div>
                    ) : (
                      <Empty
                        image={null}
                        description={t('至少同步两次后，才会出现已用趋势')}
                      />
                    )}
                  </div>
                </>
              ) : (
                <div className='rounded-xl border border-dashed border-semi-color-border bg-semi-color-bg-1 p-8'>
                  <Empty image={null} description={t('选择一个账户查看详情')} />
                </div>
              )}

              <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
                <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                  <Text strong>
                    {accountDraft.id ? t('编辑账户') : t('新建账户')}
                  </Text>
                  <Space wrap>
                    {accountDraft.id ? (
                      <Button
                        type='danger'
                        theme='light'
                        icon={<Trash2 size={14} />}
                        loading={deletingAccountId === accountDraft.id}
                        onClick={() => deleteAccount(accountDraft.id)}
                      >
                        {t('删除')}
                      </Button>
                    ) : null}
                    <Button
                      theme='solid'
                      type='primary'
                      icon={<Save size={14} />}
                      loading={savingAccount}
                      onClick={saveAccount}
                    >
                      {accountDraft.id ? t('保存账户') : t('创建账户')}
                    </Button>
                  </Space>
                </div>

                <div className='grid gap-3 lg:grid-cols-2'>
                  <div>
                    <Text type='tertiary' size='small' className='mb-1 block'>
                      {t('名称')}
                    </Text>
                    <Input
                      value={accountDraft.name}
                      onChange={(value) =>
                        setAccountDraft((prev) => ({ ...prev, name: value }))
                      }
                      placeholder={t('例如：上海主账户')}
                      prefix={<Pencil size={14} />}
                    />
                  </div>
                  <div className='flex items-end'>
                    <div className='flex w-full items-center justify-between rounded-lg border border-semi-color-border bg-semi-color-bg-1 px-3 py-2'>
                      <Text strong>{t('启用账户')}</Text>
                      <Switch
                        checked={accountDraft.enabled !== false}
                        onChange={(checked) =>
                          setAccountDraft((prev) => ({
                            ...prev,
                            enabled: checked,
                          }))
                        }
                      />
                    </div>
                  </div>

                  <div>
                    <Text type='tertiary' size='small' className='mb-1 block'>
                      URL
                    </Text>
                    <Input
                      value={accountDraft.base_url}
                      onChange={(value) =>
                        setAccountDraft((prev) => ({
                          ...prev,
                          base_url: value,
                        }))
                      }
                      placeholder='https://your-new-api.example.com'
                    />
                  </div>
                  <div>
                    <Text type='tertiary' size='small' className='mb-1 block'>
                      {t('用户 ID')}
                    </Text>
                    <InputNumber
                      min={0}
                      value={accountDraft.user_id || 0}
                      onChange={(value) =>
                        setAccountDraft((prev) => ({
                          ...prev,
                          user_id: Number(value || 0),
                        }))
                      }
                      style={{ width: '100%' }}
                    />
                  </div>

                  <div>
                    <Text type='tertiary' size='small' className='mb-1 block'>
                      {t('密钥')}
                    </Text>
                    <Input
                      value={accountDraft.access_token}
                      onChange={(value) =>
                        setAccountDraft((prev) => ({
                          ...prev,
                          access_token: value,
                        }))
                      }
                      mode='password'
                      prefix={<KeyRound size={14} />}
                      placeholder={
                        accountDraft.access_token_masked
                          ? t('留空则保留当前密钥')
                          : t('输入上游 access token')
                      }
                    />
                    {accountDraft.access_token_masked ? (
                      <Text type='tertiary' size='small' className='mt-1 block'>
                        {t('当前密钥')}: {accountDraft.access_token_masked}
                      </Text>
                    ) : null}
                  </div>
                  <div>
                    <Text type='tertiary' size='small' className='mb-1 block'>
                      {t('低余额提醒线')}
                    </Text>
                    <InputNumber
                      min={0}
                      value={accountDraft.low_balance_threshold_usd || 0}
                      onChange={(value) =>
                        setAccountDraft((prev) => ({
                          ...prev,
                          low_balance_threshold_usd: Number(value || 0),
                        }))
                      }
                      placeholder={t('不填则不提醒')}
                      style={{ width: '100%' }}
                    />
                  </div>

                  <div className='lg:col-span-2'>
                    <Text type='tertiary' size='small' className='mb-1 block'>
                      {t('备注')}
                    </Text>
                    <Input
                      value={accountDraft.remark}
                      onChange={(value) =>
                        setAccountDraft((prev) => ({
                          ...prev,
                          remark: value,
                        }))
                      }
                      placeholder={t('例如：主站、备用、包月账户')}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <Empty image={null} description={t('点击右上角新建账户')} />
      )}
    </Card>
  );
};

export default UpstreamWalletCard;
