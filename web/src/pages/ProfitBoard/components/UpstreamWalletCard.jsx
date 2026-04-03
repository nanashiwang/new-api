import React, { useMemo } from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  InputNumber,
  Progress,
  Space,
  Switch,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
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

const { Text, Title } = Typography;

const statusColorMap = {
  ready: 'green',
  needs_baseline: 'orange',
  failed: 'red',
  not_configured: 'grey',
  disabled: 'grey',
};

const statusLabelMap = (t) => ({
  ready: t('正常'),
  needs_baseline: t('等待首次同步'),
  failed: t('同步失败'),
  not_configured: t('未配置'),
  disabled: t('未启用'),
});

const balanceLevel = (balance, total) => {
  if (total <= 0) return 'unknown';
  const ratio = balance / total;
  if (ratio <= 0.15) return 'critical';
  if (ratio <= 0.3) return 'low';
  return 'healthy';
};

const balanceLevelColor = (level) => ({
  critical: 'red',
  low: 'orange',
  healthy: 'green',
  unknown: 'grey',
})[level] || 'grey';

const balanceLevelBorder = (level) => ({
  critical: 'border-red-500/40',
  low: 'border-orange-500/30',
  healthy: 'border-semi-color-border',
  unknown: 'border-semi-color-border',
})[level] || 'border-semi-color-border';

const balanceLevelBg = (level) => ({
  critical: 'bg-red-500/5',
  low: 'bg-orange-500/5',
  healthy: '',
  unknown: '',
})[level] || '';

const usageColor = (percent) => {
  if (percent >= 85) return 'var(--semi-color-danger)';
  if (percent >= 60) return 'var(--semi-color-warning)';
  return 'var(--semi-color-success)';
};

const UpstreamWalletCard = ({
  accounts,
  accountDraft,
  setAccountDraft,
  editingAccountId,
  setEditingAccountId,
  saveAccount,
  syncAccount,
  deleteAccount,
  resetAccountDraft,
  savingAccount,
  syncingAccountId,
  deletingAccountId,
  formatMoney,
  status,
  t,
}) => {
  const labels = statusLabelMap(t);

  const enabledAccounts = useMemo(
    () => accounts.filter((a) => a.enabled !== false),
    [accounts],
  );

  const walletSummary = useMemo(() => {
    let totalQuota = 0;
    let totalUsed = 0;
    enabledAccounts.forEach((a) => {
      totalQuota += a.wallet_quota_usd || 0;
      totalUsed += a.wallet_used_quota_usd || 0;
    });
    return {
      totalQuota,
      totalUsed,
      totalBalance: totalQuota - totalUsed,
      count: enabledAccounts.length,
    };
  }, [enabledAccounts]);

  const selectedAccount =
    accounts.find((item) => item.id === editingAccountId) || null;

  return (
    <div className='space-y-3'>
      {/* 钱包总览仪表盘 */}
      {enabledAccounts.length > 0 && (
        <Card
          bordered={false}
          bodyStyle={{ padding: '16px' }}
          title={
            <div className='flex items-center gap-2'>
              <Wallet size={16} />
              <span>{t('钱包总览')}</span>
              <Tag color='blue' size='small'>
                {enabledAccounts.length} {t('个账户')}
              </Tag>
            </div>
          }
        >
          {/* 汇总数字 */}
          <div className='mb-4 grid gap-3 sm:grid-cols-3'>
            <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
              <Text type='tertiary' size='small'>
                {t('总余额')}
              </Text>
              <div
                className={`mt-1 text-xl font-bold ${
                  balanceLevel(walletSummary.totalBalance, walletSummary.totalQuota) === 'critical'
                    ? 'text-red-500'
                    : balanceLevel(walletSummary.totalBalance, walletSummary.totalQuota) === 'low'
                      ? 'text-orange-500'
                      : 'text-emerald-600 dark:text-emerald-400'
                }`}
              >
                {formatMoney(walletSummary.totalBalance, status)}
              </div>
            </div>
            <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
              <Text type='tertiary' size='small'>
                {t('总额度')}
              </Text>
              <div className='mt-1 text-xl font-bold'>
                {formatMoney(walletSummary.totalQuota, status)}
              </div>
            </div>
            <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
              <Text type='tertiary' size='small'>
                {t('总已用')}
              </Text>
              <div className='mt-1 text-xl font-bold text-amber-600 dark:text-amber-400'>
                {formatMoney(walletSummary.totalUsed, status)}
              </div>
            </div>
          </div>

          {/* 各账户余额一览 */}
          <div className='space-y-1.5'>
            {enabledAccounts.map((account) => {
              const walletTotal = account.wallet_quota_usd || 0;
              const walletUsed = account.wallet_used_quota_usd || 0;
              const balance = walletTotal - walletUsed;
              const percent =
                walletTotal > 0
                  ? Math.min(Math.round((walletUsed / walletTotal) * 100), 100)
                  : 0;
              const level = balanceLevel(balance, walletTotal);

              return (
                <div
                  key={account.id}
                  className={`flex items-center gap-3 rounded-lg border px-3 py-2 ${balanceLevelBorder(level)} ${balanceLevelBg(level)}`}
                >
                  <div className='min-w-0 flex-1'>
                    <div className='flex items-center gap-2'>
                      <Text strong size='small' className='truncate'>
                        {account.name}
                      </Text>
                      {level === 'critical' && (
                        <Tooltip content={t('余额不足，请及时充值')}>
                          <AlertCircle
                            size={14}
                            className='shrink-0 text-red-500'
                          />
                        </Tooltip>
                      )}
                      {level === 'low' && (
                        <Tooltip content={t('余额偏低，建议充值')}>
                          <AlertCircle
                            size={14}
                            className='shrink-0 text-orange-500'
                          />
                        </Tooltip>
                      )}
                    </div>
                  </div>
                  <div className='w-28 shrink-0'>
                    <Progress
                      percent={percent}
                      stroke={usageColor(percent)}
                      showInfo
                      size='small'
                      format={() => `${percent}%`}
                    />
                  </div>
                  <div className='w-24 shrink-0 text-right'>
                    <Text
                      strong
                      className={
                        level === 'critical'
                          ? 'text-red-500'
                          : level === 'low'
                            ? 'text-orange-500'
                            : 'text-emerald-600 dark:text-emerald-400'
                      }
                    >
                      {formatMoney(balance, status)}
                    </Text>
                  </div>
                  <Tag
                    color={statusColorMap[account.status] || 'grey'}
                    size='small'
                  >
                    {labels[account.status] || account.status || t('未同步')}
                  </Tag>
                </div>
              );
            })}
          </div>
        </Card>
      )}

      {/* 账户管理 */}
      <Card
        bordered={false}
        title={
          <div className='flex items-center justify-between gap-3'>
            <div className='flex items-center gap-2'>
              <Wallet size={16} />
              <span>{t('上游账户管理')}</span>
            </div>
            <Button
              theme='solid'
              type='primary'
              icon={<Plus size={14} />}
              size='small'
              onClick={resetAccountDraft}
            >
              {t('新建账户')}
            </Button>
          </div>
        }
      >
        <div className='grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]'>
          {/* 左侧账户列表 */}
          <div className='space-y-2'>
            {accounts.length > 0 ? (
              accounts.map((item) => {
                const walletTotal = item.wallet_quota_usd || 0;
                const walletUsed = item.wallet_used_quota_usd || 0;
                const balance = walletTotal - walletUsed;
                const percent =
                  walletTotal > 0
                    ? Math.min(
                        Math.round((walletUsed / walletTotal) * 100),
                        100,
                      )
                    : 0;
                const level = balanceLevel(balance, walletTotal);
                const isSelected = editingAccountId === item.id;

                return (
                  <button
                    key={item.id}
                    type='button'
                    onClick={() => {
                      setEditingAccountId(item.id);
                      setAccountDraft({
                        id: item.id,
                        name: item.name || '',
                        remark: item.remark || '',
                        account_type: item.account_type || 'newapi',
                        base_url: item.base_url || '',
                        user_id: item.user_id || 0,
                        access_token: '',
                        access_token_masked: item.access_token_masked || '',
                        enabled: item.enabled !== false,
                      });
                    }}
                    className={`w-full rounded-lg border p-3 text-left transition ${
                      isSelected
                        ? 'border-semi-color-primary bg-semi-color-primary-light-default'
                        : `${balanceLevelBorder(level)} bg-semi-color-fill-0 hover:border-semi-color-primary-hover`
                    }`}
                  >
                    <div className='mb-2 flex items-start justify-between gap-2'>
                      <div className='min-w-0'>
                        <div className='truncate text-sm font-semibold'>
                          {item.name}
                        </div>
                        <div className='truncate text-xs text-semi-color-text-2'>
                          {item.base_url}
                        </div>
                      </div>
                      <Tag
                        color={statusColorMap[item.status] || 'grey'}
                        size='small'
                      >
                        {labels[item.status] || item.status || t('未同步')}
                      </Tag>
                    </div>
                    <div className='mb-1.5 flex items-center justify-between text-xs'>
                      <span className='text-semi-color-text-2'>
                        {t('余额')}
                      </span>
                      <span
                        className={`text-sm font-bold ${
                          level === 'critical'
                            ? 'text-red-500'
                            : level === 'low'
                              ? 'text-orange-500'
                              : 'text-emerald-600 dark:text-emerald-400'
                        }`}
                      >
                        {formatMoney(balance, status)}
                      </span>
                    </div>
                    <div className='flex items-center justify-between text-xs text-semi-color-text-2'>
                      <span>{t('本期消耗')}</span>
                      <span className='font-semibold text-semi-color-text-0'>
                        {formatMoney(item.observed_cost_usd, status)}
                      </span>
                    </div>
                    {walletTotal > 0 && (
                      <div className='mt-2'>
                        <Progress
                          percent={percent}
                          stroke={usageColor(percent)}
                          showInfo
                          size='small'
                          format={() => `${percent}%`}
                        />
                      </div>
                    )}
                  </button>
                );
              })
            ) : (
              <Empty
                image={null}
                description={t('还没有上游账户，先新建一个 new-api 账户')}
              />
            )}
          </div>

          {/* 右侧编辑区 */}
          <div className='space-y-3'>
            {/* 编辑表单 */}
            <div className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 p-4'>
              <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                <Text strong>
                  {accountDraft.id ? t('编辑上游账户') : t('新建上游账户')}
                </Text>
                <Space>
                  {accountDraft.id ? (
                    <Button
                      type='danger'
                      theme='light'
                      icon={<Trash2 size={14} />}
                      size='small'
                      loading={deletingAccountId === accountDraft.id}
                      onClick={() => deleteAccount(accountDraft.id)}
                    >
                      {t('删除')}
                    </Button>
                  ) : null}
                  {accountDraft.id ? (
                    <Button
                      theme='light'
                      type='warning'
                      icon={<RefreshCw size={14} />}
                      size='small'
                      loading={syncingAccountId === accountDraft.id}
                      onClick={() => syncAccount(accountDraft.id)}
                    >
                      {t('同步钱包')}
                    </Button>
                  ) : null}
                  <Button
                    theme='solid'
                    type='primary'
                    icon={<Save size={14} />}
                    size='small'
                    loading={savingAccount}
                    onClick={saveAccount}
                  >
                    {accountDraft.id ? t('保存账户') : t('创建账户')}
                  </Button>
                </Space>
              </div>

              <div className='space-y-3'>
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
                      placeholder={t('例如：newapi 上海主账户')}
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
                </div>
                <div className='grid gap-3 lg:grid-cols-2'>
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
                      placeholder={t('远端用户 ID')}
                      style={{ width: '100%' }}
                    />
                  </div>
                </div>
                <div className='grid gap-3 lg:grid-cols-2'>
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
                        {t('当前密钥：')} {accountDraft.access_token_masked}
                      </Text>
                    ) : null}
                  </div>
                  <div>
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
                      placeholder={t('备注，例如：主站、备用、包月账户')}
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* 选中账户的钱包详情 */}
            {selectedAccount ? (
              <div className='rounded-lg border border-semi-color-border bg-semi-color-bg-1 p-4'>
                <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                  <div>
                    <Title heading={6} style={{ margin: 0 }}>
                      {selectedAccount.name}
                    </Title>
                    <Text type='tertiary' size='small'>
                      {selectedAccount.remark || selectedAccount.base_url}
                    </Text>
                  </div>
                  <Space>
                    <Tag
                      color={statusColorMap[selectedAccount.status] || 'grey'}
                      size='small'
                    >
                      {labels[selectedAccount.status] ||
                        selectedAccount.status ||
                        t('未同步')}
                    </Tag>
                    {selectedAccount.quota_per_unit_mismatch ? (
                      <Tag color='orange' size='small'>
                        {t('额度倍率不同')}
                      </Tag>
                    ) : null}
                  </Space>
                </div>

                <div className='grid gap-2 sm:grid-cols-3'>
                  <div className='rounded-md bg-semi-color-fill-0 px-3 py-2'>
                    <Text type='tertiary' size='small'>
                      {t('钱包总额')}
                    </Text>
                    <div className='mt-1 text-lg font-bold'>
                      {formatMoney(selectedAccount.wallet_quota_usd, status)}
                    </div>
                  </div>
                  <div className='rounded-md bg-semi-color-fill-0 px-3 py-2'>
                    <Text type='tertiary' size='small'>
                      {t('累计已用')}
                    </Text>
                    <div className='mt-1 text-lg font-bold text-amber-600 dark:text-amber-400'>
                      {formatMoney(
                        selectedAccount.wallet_used_quota_usd,
                        status,
                      )}
                    </div>
                  </div>
                  <div className='rounded-md bg-semi-color-fill-0 px-3 py-2'>
                    <Text type='tertiary' size='small'>
                      {t('本期消耗')}
                    </Text>
                    <div className='mt-1 text-lg font-bold text-rose-600 dark:text-rose-400'>
                      {formatMoney(selectedAccount.observed_cost_usd, status)}
                    </div>
                  </div>
                </div>

                {(selectedAccount.subscription_total_quota_usd > 0 ||
                  selectedAccount.subscription_used_quota_usd > 0) && (
                  <div className='mt-3 rounded-lg border border-semi-color-border bg-semi-color-fill-0 p-3'>
                    <div className='mb-1.5 text-sm font-semibold'>
                      {t('订阅额度')}
                    </div>
                    <div className='grid gap-2 sm:grid-cols-2'>
                      <div>
                        <Text type='tertiary' size='small'>
                          {t('总额')}
                        </Text>
                        <div className='mt-0.5 font-semibold'>
                          {selectedAccount.subscription_total_quota_usd > 0
                            ? formatMoney(
                                selectedAccount.subscription_total_quota_usd,
                                status,
                              )
                            : t('不限额或未知')}
                        </div>
                      </div>
                      <div>
                        <Text type='tertiary' size='small'>
                          {t('已用')}
                        </Text>
                        <div className='mt-0.5 font-semibold'>
                          {formatMoney(
                            selectedAccount.subscription_used_quota_usd,
                            status,
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                <div className='mt-3 grid gap-2 sm:grid-cols-2'>
                  <div className='rounded-md bg-semi-color-fill-0 px-3 py-2'>
                    <Text type='tertiary' size='small'>
                      {t('最近同步')}
                    </Text>
                    <div className='mt-0.5 text-sm font-medium'>
                      {selectedAccount.last_synced_at
                        ? timestamp2string(selectedAccount.last_synced_at)
                        : '-'}
                    </div>
                  </div>
                  <div className='rounded-md bg-semi-color-fill-0 px-3 py-2'>
                    <Text type='tertiary' size='small'>
                      {t('最近成功')}
                    </Text>
                    <div className='mt-0.5 text-sm font-medium'>
                      {selectedAccount.last_success_at
                        ? timestamp2string(selectedAccount.last_success_at)
                        : '-'}
                    </div>
                  </div>
                </div>

                {selectedAccount.error_message ? (
                  <div className='mt-3 flex items-start gap-2 rounded-md border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm'>
                    <AlertCircle
                      size={14}
                      className='mt-0.5 shrink-0 text-red-500'
                    />
                    <span>{selectedAccount.error_message}</span>
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
        </div>
      </Card>
    </div>
  );
};

export default UpstreamWalletCard;
