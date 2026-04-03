import React from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Input,
  InputNumber,
  Progress,
  Space,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
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
  needs_baseline: t('等待基线'),
  failed: t('同步失败'),
  not_configured: t('未配置'),
  disabled: t('未启用'),
});

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
  const selectedAccount =
    accounts.find((item) => item.id === editingAccountId) || accounts[0] || null;

  return (
    <Card
      bordered={false}
      title={
        <div className='flex items-center justify-between gap-3'>
          <div className='flex items-center gap-2'>
            <Wallet size={18} />
            <span>{t('上游账户与钱包')}</span>
          </div>
          <Button
            theme='solid'
            type='primary'
            icon={<Plus size={14} />}
            onClick={resetAccountDraft}
          >
            {t('新建账户')}
          </Button>
        </div>
      }
    >
      <div className='grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)]'>
        <div className='space-y-3'>
          {accounts.length > 0 ? (
            accounts.map((item) => {
              const walletTotal = item.wallet_quota_usd || 0;
              const walletUsed = item.wallet_used_quota_usd || 0;
              const percent =
                walletTotal > 0
                  ? Math.min(Math.round((walletUsed / walletTotal) * 100), 100)
                  : 0;
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
                  className={`w-full rounded-2xl border p-4 text-left transition ${
                    editingAccountId === item.id
                      ? 'border-[#0f766e] bg-[#ecfeff]'
                      : 'border-semi-color-border bg-semi-color-fill-0 hover:border-[#5eead4]'
                  }`}
                >
                  <div className='mb-3 flex items-start justify-between gap-3'>
                    <div className='min-w-0'>
                      <div className='truncate text-sm font-semibold'>
                        {item.name}
                      </div>
                      <div className='truncate text-xs text-semi-color-text-2'>
                        {item.base_url}
                      </div>
                    </div>
                    <Tag color={statusColorMap[item.status] || 'grey'}>
                      {labels[item.status] || item.status || t('未同步')}
                    </Tag>
                  </div>
                  <div className='grid gap-2 text-xs text-semi-color-text-2'>
                    <div className='flex items-center justify-between gap-2'>
                      <span>{t('钱包余额')}</span>
                      <span className='font-semibold text-semi-color-text-0'>
                        {formatMoney(walletTotal - walletUsed, status)}
                      </span>
                    </div>
                    <div className='flex items-center justify-between gap-2'>
                      <span>{t('观测扣减')}</span>
                      <span className='font-semibold text-semi-color-text-0'>
                        {formatMoney(item.observed_cost_usd, status)}
                      </span>
                    </div>
                  </div>
                  {walletTotal > 0 && (
                    <div className='mt-3'>
                      <Progress
                        percent={percent}
                        stroke={usageColor(percent)}
                        showInfo={false}
                        size='small'
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

        <div className='space-y-4'>
          <div className='rounded-[28px] border border-semi-color-border bg-[linear-gradient(135deg,#ecfeff_0%,#f8fafc_100%)] p-5'>
            <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
              <div>
                <Text strong>
                  {accountDraft.id ? t('编辑上游账户') : t('新建上游账户')}
                </Text>
                <Text type='tertiary' className='mt-1 block'>
                  {t('统一维护 URL、用户 ID 和密钥，收益看板里直接选择，不再每个组合重复填写。')}
                </Text>
              </div>
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
                {accountDraft.id ? (
                  <Button
                    theme='light'
                    type='warning'
                    icon={<RefreshCw size={14} />}
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
                  loading={savingAccount}
                  onClick={saveAccount}
                >
                  {accountDraft.id ? t('保存账户') : t('创建账户')}
                </Button>
              </Space>
            </div>

            <div className='grid gap-3 lg:grid-cols-2'>
              <Input
                value={accountDraft.name}
                onChange={(value) =>
                  setAccountDraft((prev) => ({ ...prev, name: value }))
                }
                placeholder={t('例如：newapi 上海主账户')}
                prefix={<Pencil size={14} />}
              />
              <div className='flex items-center justify-between rounded-2xl border border-semi-color-border bg-white px-4 py-3'>
                <div>
                  <Text strong>{t('启用账户')}</Text>
                  <Text type='tertiary' className='mt-1 block'>
                    {t('禁用后仍保留账户资料，但不会参与选择和同步。')}
                  </Text>
                </div>
                <Switch
                  checked={accountDraft.enabled !== false}
                  onChange={(checked) =>
                    setAccountDraft((prev) => ({ ...prev, enabled: checked }))
                  }
                />
              </div>
              <Input
                value={accountDraft.base_url}
                onChange={(value) =>
                  setAccountDraft((prev) => ({ ...prev, base_url: value }))
                }
                placeholder='https://your-new-api.example.com'
              />
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
              <Input
                value={accountDraft.access_token}
                onChange={(value) =>
                  setAccountDraft((prev) => ({ ...prev, access_token: value }))
                }
                mode='password'
                prefix={<KeyRound size={14} />}
                placeholder={
                  accountDraft.access_token_masked
                    ? t('留空则保留当前密钥')
                    : t('输入上游 access token')
                }
              />
              <Input
                value={accountDraft.remark}
                onChange={(value) =>
                  setAccountDraft((prev) => ({ ...prev, remark: value }))
                }
                placeholder={t('备注，例如：主站、备用、包月账户')}
              />
            </div>

            {accountDraft.access_token_masked ? (
              <Text type='tertiary' className='mt-3 block'>
                {t('当前密钥：')} {accountDraft.access_token_masked}
              </Text>
            ) : null}
          </div>

          {selectedAccount ? (
            <div className='rounded-[28px] border border-semi-color-border bg-white p-5'>
              <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
                <div>
                  <Title heading={5} style={{ margin: 0 }}>
                    {selectedAccount.name}
                  </Title>
                  <Text type='tertiary'>
                    {selectedAccount.remark || selectedAccount.base_url}
                  </Text>
                </div>
                <Space wrap>
                  <Tag color={statusColorMap[selectedAccount.status] || 'grey'}>
                    {labels[selectedAccount.status] ||
                      selectedAccount.status ||
                      t('未同步')}
                  </Tag>
                  {selectedAccount.quota_per_unit_mismatch ? (
                    <Tag color='orange'>{t('额度倍率不同')}</Tag>
                  ) : null}
                </Space>
              </div>

              <div className='grid gap-3 md:grid-cols-3'>
                <div className='rounded-2xl bg-[#f8fafc] px-4 py-3'>
                  <Text type='tertiary' size='small'>
                    {t('钱包总额')}
                  </Text>
                  <Title heading={4} style={{ margin: '6px 0 0' }}>
                    {formatMoney(selectedAccount.wallet_quota_usd, status)}
                  </Title>
                </div>
                <div className='rounded-2xl bg-[#f8fafc] px-4 py-3'>
                  <Text type='tertiary' size='small'>
                    {t('累计已用')}
                  </Text>
                  <Title heading={4} style={{ margin: '6px 0 0' }}>
                    {formatMoney(selectedAccount.wallet_used_quota_usd, status)}
                  </Title>
                </div>
                <div className='rounded-2xl bg-[#f8fafc] px-4 py-3'>
                  <Text type='tertiary' size='small'>
                    {t('本期观测扣减')}
                  </Text>
                  <Title heading={4} style={{ margin: '6px 0 0' }}>
                    {formatMoney(selectedAccount.observed_cost_usd, status)}
                  </Title>
                </div>
              </div>

              {(selectedAccount.subscription_total_quota_usd > 0 ||
                selectedAccount.subscription_used_quota_usd > 0) && (
                <div className='mt-4 rounded-2xl border border-semi-color-border bg-[#f8fafc] p-4'>
                  <div className='mb-2 text-sm font-semibold'>
                    {t('订阅额度')}
                  </div>
                  <div className='grid gap-3 md:grid-cols-2'>
                    <div>
                      <Text type='tertiary' size='small'>
                        {t('总额')}
                      </Text>
                      <div className='mt-1 font-semibold'>
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
                      <div className='mt-1 font-semibold'>
                        {formatMoney(
                          selectedAccount.subscription_used_quota_usd,
                          status,
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              )}

              <div className='mt-4 grid gap-3 md:grid-cols-2'>
                <div>
                  <Text type='tertiary'>{t('最近同步')}</Text>
                  <div className='mt-1'>
                    {selectedAccount.last_synced_at
                      ? timestamp2string(selectedAccount.last_synced_at)
                      : '-'}
                  </div>
                </div>
                <div>
                  <Text type='tertiary'>{t('最近成功')}</Text>
                  <div className='mt-1'>
                    {selectedAccount.last_success_at
                      ? timestamp2string(selectedAccount.last_success_at)
                      : '-'}
                  </div>
                </div>
              </div>

              {selectedAccount.error_message ? (
                <Banner
                  className='mt-4'
                  type='warning'
                  closeIcon={null}
                  description={selectedAccount.error_message}
                />
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </Card>
  );
};

export default UpstreamWalletCard;
