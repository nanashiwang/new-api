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
import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Select,
  SideSheet,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { RefreshCw, Search } from 'lucide-react';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import { timestamp2string } from '../../../helpers/date';
import {
  buildCRSUsageWindows,
  filterCRSAccounts,
  getCRSPlatformBadgeLabel,
  getCRSPlatformOptions,
} from './crsDashboard.utils';

const { Text, Title } = Typography;

const SITE_STATUS_MAP = {
  0: { color: 'grey', labelKey: '未同步' },
  1: { color: 'green', labelKey: '已同步' },
  2: { color: 'red', labelKey: '错误' },
};

const QUOTA_FILTER_OPTIONS = [
  { label: '全部额度', value: '' },
  { label: '低额度', value: 'low' },
  { label: '空额度', value: 'empty' },
  { label: '不限额', value: 'unlimited' },
];

const USAGE_WINDOW_TONE_CLASSES = {
  success: {
    card: 'border-emerald-200 bg-emerald-50/80',
    bar: 'bg-emerald-500',
    text: 'text-emerald-700',
  },
  info: {
    card: 'border-sky-200 bg-sky-50/80',
    bar: 'bg-sky-500',
    text: 'text-sky-700',
  },
  warning: {
    card: 'border-amber-200 bg-amber-50/90',
    bar: 'bg-amber-500',
    text: 'text-amber-700',
  },
  danger: {
    card: 'border-red-200 bg-red-50/90',
    bar: 'bg-red-500',
    text: 'text-red-700',
  },
  muted: {
    card: 'border-slate-200 bg-slate-50/80',
    bar: 'bg-slate-400',
    text: 'text-slate-600',
  },
};

const SummaryItem = ({ label, value, subText = '', tone = '' }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <div className={`mt-2 text-2xl font-semibold tabular-nums ${tone}`}>
      {value}
    </div>
    {subText ? (
      <Text type='tertiary' size='small' className='mt-1 block'>
        {subText}
      </Text>
    ) : null}
  </div>
);

const renderSiteStatusTag = (status, t) => {
  const meta = SITE_STATUS_MAP[status] || SITE_STATUS_MAP[0];
  return (
    <Tag color={meta.color} size='small'>
      {t(meta.labelKey)}
    </Tag>
  );
};

const formatUsageWindowProgress = (value) => {
  if (value === null || value === undefined || Number.isNaN(Number(value))) {
    return '--';
  }
  const normalized = Number(value);
  const displayValue = Number.isInteger(normalized)
    ? normalized
    : Number(normalized.toFixed(1));
  return `${displayValue}%`;
};

const UsageWindowCard = ({ window, t }) => {
  const toneClasses =
    USAGE_WINDOW_TONE_CLASSES[window?.tone] || USAGE_WINDOW_TONE_CLASSES.muted;
  const progress = Number.isFinite(window?.progress) ? window.progress : 0;
  const remainingText = window?.remainingText || window?.resetAt || t('待同步');

  return (
    <div
      className={`rounded-xl border px-3 py-2 ${toneClasses.card}`}
      key={window?.key}
    >
      <div className='flex items-center justify-between gap-2'>
        <span className='text-[11px] font-semibold uppercase tracking-wide text-semi-color-text-2'>
          {window?.label || '-'}
        </span>
        <span className={`text-xs font-semibold tabular-nums ${toneClasses.text}`}>
          {formatUsageWindowProgress(window?.progress)}
        </span>
      </div>
      <div className='mt-2 h-1.5 overflow-hidden rounded-full bg-white/70'>
        <div
          className={`h-full rounded-full transition-all ${toneClasses.bar}`}
          style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
        />
      </div>
      <div className='mt-2 text-xs text-semi-color-text-1'>{remainingText}</div>
      {window?.resetAt ? (
        <div className='mt-1 text-[11px] text-semi-color-text-2 break-all'>
          {window.resetAt}
        </div>
      ) : null}
    </div>
  );
};

const renderAccountSignals = (account, t) => {
  const tags = [];
  tags.push(
    <Tag key='platform_badge' color='cyan' size='small'>
      {getCRSPlatformBadgeLabel(account)}
    </Tag>,
  );
  if (account?.is_active) {
    tags.push(
      <Tag key='active' color='green' size='small'>
        {t('活跃')}
      </Tag>,
    );
  } else {
    tags.push(
      <Tag key='inactive' color='grey' size='small'>
        {t('未激活')}
      </Tag>,
    );
  }
  if (account?.schedulable) {
    tags.push(
      <Tag key='schedulable' color='blue' size='small'>
        {t('可调度')}
      </Tag>,
    );
  }
  if (account?.rate_limited) {
    tags.push(
      <Tag key='rate_limited' color='orange' size='small'>
        {t('限速中')}
      </Tag>,
    );
  }
  if (account?.status) {
    tags.push(
      <Tag key='status' color='white' size='small'>
        {account.status}
      </Tag>,
    );
  }
  return <div className='flex flex-wrap gap-1'>{tags}</div>;
};

const CRSSiteDetailSideSheet = ({
  visible,
  onClose,
  detail,
  loading,
  onRefresh,
  refreshing,
  t,
}) => {
  const isMobile = useIsMobile();
  const site = detail?.site;
  const observer = detail?.observer || {};
  const dashboard = detail?.dashboard || {};
  const accounts = detail?.accounts || [];
  const [keyword, setKeyword] = useState('');
  const [platform, setPlatform] = useState('');
  const [quotaState, setQuotaState] = useState('');

  useEffect(() => {
    setKeyword('');
    setPlatform('');
    setQuotaState('');
  }, [site?.id, visible]);

  const platformOptions = useMemo(
    () => [
      { label: t('全部平台'), value: '' },
      ...getCRSPlatformOptions(accounts),
    ],
    [accounts, t],
  );

  const filteredAccounts = useMemo(
    () =>
      filterCRSAccounts(accounts, {
        keyword,
        platform,
        quotaState,
      }),
    [accounts, keyword, platform, quotaState],
  );

  const columns = [
    {
      title: t('账号'),
      dataIndex: 'name',
      render: (_, record) => (
        <div className='min-w-0'>
          <div className='font-medium text-semi-color-text-0'>
            {record.name || record.remote_account_id}
          </div>
          <div className='mt-1 text-xs text-semi-color-text-2 break-all'>
            {record.remote_account_id}
          </div>
          {record.subscription_plan ? (
            <div className='mt-1 text-xs text-semi-color-text-2'>
              {t('计划')}: {record.subscription_plan}
            </div>
          ) : null}
          {record.sync_error || record.error_message ? (
            <div className='mt-1 text-xs text-red-500 break-all'>
              {record.sync_error || record.error_message}
            </div>
          ) : null}
        </div>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 280,
      render: (_, record) => renderAccountSignals(record, t),
    },
    {
      title: t('额度'),
      dataIndex: 'quota_remaining',
      width: 240,
      render: (_, record) => {
        const windows = buildCRSUsageWindows(record);
        if (windows.length === 0) {
          return <Text type='tertiary'>-</Text>;
        }
        return (
          <div className='space-y-2'>
            {windows.map((window) => (
              <UsageWindowCard key={window.key} window={window} t={t} />
            ))}
          </div>
        );
      },
    },
    {
      title: t('限速'),
      dataIndex: 'rate_limited',
      width: 160,
      render: (_, record) =>
        record.rate_limited ? (
          <div className='space-y-1 text-xs'>
            <Tag color='orange' size='small'>
              {t('限速中')}
            </Tag>
            {record.rate_limit_reset_at ? (
              <div className='text-semi-color-text-2'>
                {record.rate_limit_reset_at}
              </div>
            ) : null}
          </div>
        ) : (
          <Tag color='green' size='small'>
            {t('正常')}
          </Tag>
        ),
    },
    {
      title: t('最近同步'),
      dataIndex: 'last_synced_at',
      width: 170,
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
  ];

  return (
    <SideSheet
      visible={visible}
      onCancel={onClose}
      title={site?.name || site?.host || t('站点详情')}
      width={isMobile ? '100%' : 980}
      bodyStyle={{ padding: 16 }}
    >
      <Spin spinning={loading}>
        {site ? (
          <div className='space-y-4'>
            <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='flex flex-wrap items-start justify-between gap-3'>
                <div className='min-w-0 flex-1'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Title heading={6} style={{ margin: 0 }}>
                      {site.name || site.host}
                    </Title>
                    {renderSiteStatusTag(site.status, t)}
                    {site.group ? (
                      <Tag color='blue' size='small'>
                        {site.group}
                      </Tag>
                    ) : null}
                  </div>
                  <Text
                    type='tertiary'
                    size='small'
                    className='mt-1 block break-all'
                  >
                    {site.scheme}://{site.host}
                  </Text>
                  <div className='mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-semi-color-text-2'>
                    <span>
                      {t('用户名')}:{' '}
                      <span className='font-medium'>
                        {site.username || '-'}
                      </span>
                    </span>
                    <span>
                      {t('最近同步')}:{' '}
                      <span className='font-medium'>
                        {site.last_synced_at
                          ? timestamp2string(site.last_synced_at)
                          : '-'}
                      </span>
                    </span>
                  </div>
                  {site.last_sync_error ? (
                    <div className='mt-3 rounded-lg bg-red-500/5 px-3 py-2 text-xs text-red-500 break-all'>
                      {site.last_sync_error}
                    </div>
                  ) : null}
                </div>
                <Button
                  icon={<RefreshCw size={14} />}
                  loading={refreshing}
                  onClick={() => onRefresh?.(site.id)}
                  size='small'
                >
                  {t('刷新站点')}
                </Button>
              </div>
            </div>

            <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
              <SummaryItem
                label={t('观察账号')}
                value={observer.total_accounts ?? accounts.length}
              />
              <SummaryItem
                label={t('可调度')}
                value={observer.schedulable_count ?? 0}
                tone='text-blue-600 dark:text-blue-400'
                subText={`${t('活跃')} ${observer.active_accounts ?? 0}`}
              />
              <SummaryItem
                label={t('限速中')}
                value={observer.rate_limited_count ?? 0}
                tone='text-orange-500'
              />
              <SummaryItem
                label={t('低额度')}
                value={observer.low_quota_count ?? 0}
                tone='text-amber-500'
                subText={`${t('空额度')} ${observer.empty_quota_count ?? 0}`}
              />
            </div>

            {dashboard?.overview ? (
              <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
                <div className='mb-3'>
                  <Title heading={6} style={{ margin: 0 }}>
                    {t('Dashboard 概览')}
                  </Title>
                  <Text type='tertiary' size='small' className='mt-1 block'>
                    {t('这是远端 CRS /admin/dashboard 的缓存结果')}
                  </Text>
                </div>
                <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
                  <SummaryItem
                    label={t('总账号')}
                    value={dashboard.overview.totalAccounts ?? 0}
                  />
                  <SummaryItem
                    label={t('正常账号')}
                    value={dashboard.overview.normalAccounts ?? 0}
                    tone='text-green-600 dark:text-green-400'
                  />
                  <SummaryItem
                    label={t('API Keys')}
                    value={dashboard.overview.totalApiKeys ?? 0}
                  />
                  <SummaryItem
                    label={t('今日请求')}
                    value={dashboard.recentActivity?.requestsToday ?? 0}
                  />
                </div>
              </div>
            ) : null}

            <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='mb-3 flex flex-wrap items-start justify-between gap-3'>
                <div>
                  <Title heading={6} style={{ margin: 0 }}>
                    {t('站点账号明细')}
                  </Title>
                  <Text type='tertiary' size='small' className='mt-1 block'>
                    {t(
                      '来自远端 CRS 的标准化账号快照，可按平台、额度和关键词快速筛选',
                    )}
                  </Text>
                </div>
                <Text type='tertiary' size='small'>
                  {t('显示 {{filtered}} / {{total}}', {
                    filtered: filteredAccounts.length,
                    total: accounts.length,
                  })}
                </Text>
              </div>
              {accounts.length > 0 ? (
                <div className='mb-3 grid gap-2 lg:grid-cols-[minmax(0,1.2fr),220px,220px]'>
                  <Input
                    prefix={<Search size={14} />}
                    placeholder={t('搜索账号名、远端 ID、订阅计划')}
                    value={keyword}
                    onChange={setKeyword}
                    showClear
                  />
                  <Select
                    value={platform}
                    optionList={platformOptions}
                    onChange={(value) => setPlatform(value || '')}
                    placeholder={t('平台')}
                    showClear
                  />
                  <Select
                    value={quotaState}
                    optionList={QUOTA_FILTER_OPTIONS.map((item) => ({
                      ...item,
                      label: t(item.label),
                    }))}
                    onChange={(value) => setQuotaState(value || '')}
                    placeholder={t('额度状态')}
                  />
                </div>
              ) : null}
              {accounts.length > 0 ? (
                <Table
                  dataSource={filteredAccounts}
                  columns={columns}
                  rowKey={(record) =>
                    `${record.site_id}-${record.remote_account_id}`
                  }
                  pagination={false}
                  size='small'
                  scroll={{ x: 980 }}
                  empty={
                    <Empty
                      image={null}
                      description={t('筛选后没有匹配的账号快照')}
                    />
                  }
                />
              ) : (
                <Empty image={null} description={t('该站点暂时没有账号快照')} />
              )}
            </div>
          </div>
        ) : (
          <Empty image={null} description={t('请选择一个站点')} />
        )}
      </Spin>
    </SideSheet>
  );
};

export default CRSSiteDetailSideSheet;
