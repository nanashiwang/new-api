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
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Empty,
  Input,
  Pagination,
  Progress,
  Select,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { RefreshCw, Search, WalletCards } from 'lucide-react';
import { useDebounce } from 'use-debounce';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import { timestamp2string } from '../../../helpers/date';
import {
  CRS_ACCOUNT_STALE_AFTER_SECONDS,
  buildCRSUsageWindows,
  formatCRSDailyCost,
  formatCRSRequestCount,
  formatCRSTokenCount,
  getCRSAccountHealth,
  getCRSPlatformBadgeLabel,
  getCRSUsagePercentages,
  sortCRSAccountsByAttention,
} from './crsDashboard.utils';

const { Text, Title } = Typography;

const HEALTH_META = {
  available: { color: 'green' },
  warning: { color: 'amber' },
  critical: { color: 'orange' },
  empty: { color: 'red' },
  rate_limited: { color: 'red' },
  inactive: { color: 'grey' },
  unschedulable: { color: 'orange' },
  sync_error: { color: 'red' },
  stale: { color: 'grey' },
};

const getHealthLabel = (key, t) => {
  switch (key) {
    case 'warning':
      return t('余量偏低');
    case 'critical':
      return t('余量紧张');
    case 'empty':
      return t('额度耗尽');
    case 'rate_limited':
      return t('限速中');
    case 'inactive':
      return t('未激活');
    case 'unschedulable':
      return t('不可调度');
    case 'sync_error':
      return t('同步异常');
    case 'stale':
      return t('数据过期');
    default:
      return t('可用');
  }
};

const toneForRemaining = (remaining) => {
  if (remaining === null) return { stroke: 'grey', type: 'tertiary' };
  if (remaining <= 10) return { stroke: 'red', type: 'danger' };
  if (remaining <= 30) return { stroke: 'amber', type: 'warning' };
  return { stroke: 'green', type: 'success' };
};

const formatPercent = (value) => {
  if (!Number.isFinite(value)) return '--';
  const rounded = Number(value.toFixed(1));
  return `${rounded}%`;
};

const HealthTag = ({ account, t }) => {
  const health = getCRSAccountHealth(account);
  const meta = HEALTH_META[health.key] || HEALTH_META.available;
  return (
    <Tag color={meta.color} size='small'>
      {getHealthLabel(health.key, t)}
    </Tag>
  );
};

const AccountStatus = ({ account, t, showHealth = true }) => (
  <div className='flex flex-wrap items-center gap-1'>
    {showHealth ? <HealthTag account={account} t={t} /> : null}
    <Tag color='cyan' size='small'>
      {getCRSPlatformBadgeLabel(account)}
    </Tag>
    {account?.rate_limited && account?.rate_limit_minutes_remaining > 0 ? (
      <Tag color='orange' size='small'>
        {t('{{minutes}} 分钟后恢复', {
          minutes: account.rate_limit_minutes_remaining,
        })}
      </Tag>
    ) : null}
  </div>
);

const UsageWindow = ({ window, t }) => {
  const { usedPercent, remainingPercent } = getCRSUsagePercentages(
    window?.progress,
  );
  const normalizedProgress = usedPercent;
  const remaining = remainingPercent;
  const tone = toneForRemaining(remaining);
  const unlimited =
    String(window?.remainingText || '').toLowerCase() === 'unlimited';

  if (unlimited) {
    return (
      <div className='flex items-center justify-between gap-2 text-xs'>
        <Text type='tertiary' size='small'>
          {window?.label || t('额度')}
        </Text>
        <Tag color='green' size='small'>
          {t('不限额')}
        </Tag>
      </div>
    );
  }

  return (
    <div className='flex flex-col gap-1'>
      <div className='flex items-center justify-between gap-3 text-xs'>
        <Text type='tertiary' size='small'>
          {window?.label || t('额度')}
        </Text>
        {remaining !== null ? (
          <Text type={tone.type} strong size='small'>
            {t('剩余')} {formatPercent(remaining)}
          </Text>
        ) : window?.remainingText ? (
          <Text size='small'>
            {t('剩余')} {window.remainingText}
          </Text>
        ) : (
          <Text type='tertiary' size='small'>
            --
          </Text>
        )}
      </div>
      {remaining !== null ? (
        <Progress
          percent={remaining}
          stroke={tone.stroke}
          showInfo={false}
          size='small'
        />
      ) : null}
      <div className='flex flex-wrap justify-between gap-x-3 gap-y-0.5'>
        {normalizedProgress !== null ? (
          <Text type='tertiary' size='small'>
            {t('已用')} {formatPercent(normalizedProgress)}
          </Text>
        ) : null}
        {window?.remainingText && remaining !== null ? (
          <Text type='tertiary' size='small'>
            {window.remainingText}
          </Text>
        ) : null}
        {window?.resetAt ? (
          <Text type='tertiary' size='small'>
            {t('重置')} {window.resetAt}
          </Text>
        ) : null}
      </div>
    </div>
  );
};

const AccountQuota = ({ account, t }) => {
  const windows = buildCRSUsageWindows(account);
  if (windows.length === 0) {
    return (
      <Text type='tertiary' size='small'>
        {t('暂无额度数据')}
      </Text>
    );
  }
  return (
    <div className='flex min-w-[220px] flex-col gap-2'>
      {windows.map((window) => (
        <UsageWindow key={window.key} window={window} t={t} />
      ))}
    </div>
  );
};

const DailyUsage = ({ account, t }) => {
  const requests = formatCRSRequestCount(account?.usage_daily_requests || 0);
  const tokens = formatCRSTokenCount(account?.usage_daily_tokens || 0);
  const cost = formatCRSDailyCost(account?.usage_daily_cost || 0);
  return (
    <div className='flex flex-col gap-0.5 font-mono text-xs tabular-nums'>
      <span>
        {requests} {t('次')}
      </span>
      <Text type='tertiary' size='small'>
        {tokens ? `${tokens} Tok` : '0 Tok'}
      </Text>
      {cost ? (
        <Text type='tertiary' size='small'>
          {cost}
        </Text>
      ) : null}
    </div>
  );
};

const SyncState = ({ account, t }) => {
  const health = getCRSAccountHealth(account);
  return (
    <div className='flex flex-col gap-1 text-xs'>
      {account?.rate_limited ? (
        <Text type='danger' size='small'>
          {account.rate_limit_minutes_remaining > 0
            ? t('{{minutes}} 分钟后恢复', {
                minutes: account.rate_limit_minutes_remaining,
              })
            : t('等待上游恢复')}
        </Text>
      ) : null}
      {account?.rate_limit_reset_at ? (
        <Text type='tertiary' size='small'>
          {account.rate_limit_reset_at}
        </Text>
      ) : null}
      <Text type={health.isStale ? 'warning' : 'tertiary'} size='small'>
        {t('同步')}:{' '}
        {account?.last_synced_at
          ? timestamp2string(account.last_synced_at)
          : t('尚未同步')}
      </Text>
      {account?.sync_error || account?.error_message ? (
        <Text type='danger' size='small' className='break-all'>
          {account.sync_error || account.error_message}
        </Text>
      ) : null}
    </div>
  );
};

const MobileAccountCard = ({ account, onOpenSite, t }) => (
  <Card bordered bodyStyle={{ padding: 12 }}>
    <div className='flex flex-col gap-3'>
      <div className='flex items-start justify-between gap-2'>
        <div className='min-w-0'>
          <div className='truncate font-semibold'>
            {account.name || account.remote_account_id}
          </div>
          <Button
            theme='borderless'
            size='small'
            className='-ml-3 mt-0.5'
            onClick={() => onOpenSite?.(account.site_id)}
          >
            {account.site_name || '-'}
          </Button>
        </div>
        <HealthTag account={account} t={t} />
      </div>
      <AccountStatus account={account} t={t} showHealth={false} />
      <AccountQuota account={account} t={t} />
      <div className='grid grid-cols-1 gap-3 border-t border-semi-color-border pt-3 sm:grid-cols-2'>
        <div>
          <Text type='tertiary' size='small'>
            {t('今日使用')}
          </Text>
          <DailyUsage account={account} t={t} />
        </div>
        <div>
          <Text type='tertiary' size='small'>
            {t('恢复 / 同步')}
          </Text>
          <SyncState account={account} t={t} />
        </div>
      </div>
    </div>
  </Card>
);

export default function CRSAccountUsageOverview({
  accounts,
  total,
  loading,
  sites,
  aggregate,
  observer,
  loadAccounts,
  onOpenSite,
  t,
}) {
  const isMobile = useIsMobile();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [siteId, setSiteId] = useState(0);
  const [platform, setPlatform] = useState('');
  const [healthState, setHealthState] = useState('');
  const [quotaState, setQuotaState] = useState('');
  const [keyword, setKeyword] = useState('');
  const [attentionFirst, setAttentionFirst] = useState(true);
  const [debouncedKeyword] = useDebounce(keyword, 300);

  const requestAccounts = useCallback(
    () =>
      loadAccounts({
        page,
        page_size: pageSize,
        site_id: siteId,
        platform,
        health_state: healthState,
        quota_state: quotaState,
        keyword: debouncedKeyword,
        attention_first: attentionFirst,
      }),
    [
      attentionFirst,
      debouncedKeyword,
      healthState,
      loadAccounts,
      page,
      pageSize,
      platform,
      quotaState,
      siteId,
    ],
  );

  useEffect(() => {
    requestAccounts();
  }, [requestAccounts]);

  useEffect(() => {
    const lastPage = Math.max(1, Math.ceil(Number(total || 0) / pageSize));
    if (page > lastPage) setPage(lastPage);
  }, [page, pageSize, total]);

  const siteOptions = useMemo(
    () => [
      { label: t('全部站点'), value: 0 },
      ...sites.map((site) => ({
        label: site.name || site.host,
        value: site.id,
      })),
    ],
    [sites, t],
  );

  const platformOptions = useMemo(() => {
    const values = new Set([
      ...Object.keys(aggregate?.overview?.accountsByPlatform || {}),
      ...accounts.map((account) => String(account?.platform || '').trim()),
    ]);
    return [
      { label: t('全部平台'), value: '' },
      ...Array.from(values)
        .filter(Boolean)
        .sort((left, right) => left.localeCompare(right, 'en'))
        .map((value) => ({ label: value, value })),
    ];
  }, [accounts, aggregate?.overview?.accountsByPlatform, t]);

  const healthOptions = useMemo(
    () => [
      { label: t('全部状态'), value: '' },
      { label: t('可用'), value: 'available' },
      { label: t('需关注'), value: 'attention' },
      { label: t('限速中'), value: 'rate_limited' },
      { label: t('未激活'), value: 'inactive' },
      { label: t('不可调度'), value: 'unschedulable' },
      { label: t('同步异常'), value: 'sync_error' },
      { label: t('数据过期'), value: 'stale' },
    ],
    [t],
  );

  const quotaOptions = useMemo(
    () => [
      { label: t('全部额度'), value: '' },
      { label: t('低额度'), value: 'low' },
      { label: t('空额度'), value: 'empty' },
      { label: t('不限额'), value: 'unlimited' },
    ],
    [t],
  );

  const displayedAccounts = useMemo(
    () =>
      attentionFirst ? sortCRSAccountsByAttention(accounts) : [...accounts],
    [accounts, attentionFirst],
  );

  const columns = [
    {
      title: t('账号 / 站点'),
      dataIndex: 'name',
      render: (_, record) => (
        <div className='min-w-0'>
          <div className='truncate font-medium'>
            {record.name || record.remote_account_id}
          </div>
          <div className='mt-0.5 break-all text-xs text-semi-color-text-2'>
            {record.remote_account_id}
          </div>
          <Button
            theme='borderless'
            size='small'
            className='-ml-3 mt-0.5'
            onClick={() => onOpenSite?.(record.site_id)}
          >
            {record.site_name || '-'}
          </Button>
        </div>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 210,
      render: (_, record) => <AccountStatus account={record} t={t} />,
    },
    {
      title: t('额度余量'),
      dataIndex: 'usage_windows',
      width: 320,
      render: (_, record) => <AccountQuota account={record} t={t} />,
    },
    {
      title: t('今日使用'),
      dataIndex: 'usage_daily_requests',
      width: 120,
      render: (_, record) => <DailyUsage account={record} t={t} />,
    },
    {
      title: t('恢复 / 同步'),
      dataIndex: 'last_synced_at',
      width: 190,
      render: (_, record) => <SyncState account={record} t={t} />,
    },
  ];

  const changeFilter = (setter, value) => {
    setter(value);
    setPage(1);
  };

  return (
    <Card
      bordered
      headerStyle={{ padding: '12px 16px' }}
      bodyStyle={{ padding: 12 }}
      title={
        <div className='flex items-center gap-2'>
          <WalletCards size={15} className='text-semi-color-primary' />
          <Title heading={6} style={{ margin: 0 }}>
            {t('CRS 账号使用情况')}
          </Title>
          <Tag color='blue' size='small'>
            {total ?? 0}
          </Tag>
        </div>
      }
      headerExtraContent={
        <Button
          theme='borderless'
          size='small'
          aria-label={t('刷新账号')}
          icon={<RefreshCw size={14} />}
          loading={loading}
          onClick={requestAccounts}
        >
          {isMobile ? null : t('刷新账号')}
        </Button>
      }
    >
      <div className='mb-3 flex flex-col gap-3'>
        <div className='flex flex-wrap items-center gap-2'>
          <Tag color='green'>
            {t('可调度')} {observer?.schedulable_count ?? 0}/
            {observer?.total_accounts ?? 0}
          </Tag>
          <Tag color={(observer?.rate_limited_count ?? 0) > 0 ? 'red' : 'grey'}>
            {t('限速中')} {observer?.rate_limited_count ?? 0}
          </Tag>
          <Tag color={(observer?.low_quota_count ?? 0) > 0 ? 'amber' : 'grey'}>
            {t('低额度')} {observer?.low_quota_count ?? 0}
          </Tag>
          <Tag color={(observer?.empty_quota_count ?? 0) > 0 ? 'red' : 'grey'}>
            {t('空额度')} {observer?.empty_quota_count ?? 0}
          </Tag>
          <Text
            type='tertiary'
            size='small'
            className={isMobile ? 'w-full break-words' : ''}
          >
            {t('账号快照每分钟同步；超过 {{minutes}} 分钟会标记为数据过期', {
              minutes: CRS_ACCOUNT_STALE_AFTER_SECONDS / 60,
            })}
          </Text>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          <Input
            value={keyword}
            prefix={<Search size={14} />}
            placeholder={t('搜索账号或远端 ID')}
            showClear
            style={{ width: isMobile ? '100%' : 220 }}
            onChange={(value) => changeFilter(setKeyword, value)}
          />
          <Select
            value={siteId}
            optionList={siteOptions}
            style={{ width: isMobile ? 'calc(50% - 4px)' : 150 }}
            onChange={(value) => changeFilter(setSiteId, Number(value) || 0)}
          />
          <Select
            value={platform}
            optionList={platformOptions}
            style={{ width: isMobile ? 'calc(50% - 4px)' : 160 }}
            onChange={(value) => changeFilter(setPlatform, value || '')}
          />
          <Select
            value={healthState}
            optionList={healthOptions}
            style={{ width: isMobile ? 'calc(50% - 4px)' : 135 }}
            onChange={(value) => changeFilter(setHealthState, value || '')}
          />
          <Select
            value={quotaState}
            optionList={quotaOptions}
            style={{ width: isMobile ? 'calc(50% - 4px)' : 125 }}
            onChange={(value) => changeFilter(setQuotaState, value || '')}
          />
          <Checkbox
            checked={attentionFirst}
            onChange={(event) =>
              changeFilter(setAttentionFirst, event.target.checked)
            }
          >
            {t('异常优先')}
          </Checkbox>
        </div>
      </div>

      <Spin spinning={loading}>
        {displayedAccounts.length === 0 && !loading ? (
          <Empty
            image={<WalletCards size={36} className='opacity-30' />}
            title={t('暂无匹配的 CRS 账号')}
            description={t('请调整筛选条件或刷新 CRS 账号快照')}
          />
        ) : isMobile ? (
          <div className='flex flex-col gap-3'>
            {displayedAccounts.map((account) => (
              <MobileAccountCard
                key={`${account.site_id}:${account.remote_account_id}`}
                account={account}
                onOpenSite={onOpenSite}
                t={t}
              />
            ))}
            <Pagination
              currentPage={page}
              pageSize={pageSize}
              total={total}
              showSizeChanger
              pageSizeOpts={[10, 20, 50]}
              onPageChange={setPage}
              onPageSizeChange={(size) => {
                setPageSize(size);
                setPage(1);
              }}
            />
          </div>
        ) : (
          <Table
            dataSource={displayedAccounts}
            columns={columns}
            rowKey='id'
            size='small'
            bordered
            pagination={{
              currentPage: page,
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOpts: [10, 20, 50],
              onPageChange: setPage,
              onPageSizeChange: (size) => {
                setPageSize(size);
                setPage(1);
              },
            }}
          />
        )}
      </Spin>
    </Card>
  );
}
