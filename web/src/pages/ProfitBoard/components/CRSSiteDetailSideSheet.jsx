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
  Card,
  Divider,
  Empty,
  Input,
  Progress,
  Select,
  SideSheet,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { RefreshCw, Search, AlertTriangle } from 'lucide-react';
import { useIsMobile } from '@/hooks/common/useIsMobile';
import { timestamp2string } from '../../../helpers/date';
import {
  buildCRSUsageWindows,
  filterCRSAccounts,
  formatCRSDailyCost,
  formatCRSRequestCount,
  formatCRSTokenCount,
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

const TONE_TO_SEMI_COLOR = {
  success: 'green',
  info: 'blue',
  warning: 'amber',
  danger: 'red',
  muted: 'grey',
};

const TONE_TO_TEXT_TYPE = {
  success: 'success',
  info: 'primary',
  warning: 'warning',
  danger: 'danger',
  muted: 'tertiary',
};

const toneToSemiColor = (tone) => TONE_TO_SEMI_COLOR[tone] || 'grey';
const toneToTextType = (tone) => TONE_TO_TEXT_TYPE[tone] || 'tertiary';

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

const renderSiteStatusTag = (status, t) => {
  const meta = SITE_STATUS_MAP[status] || SITE_STATUS_MAP[0];
  return (
    <Tag color={meta.color} size='small'>
      {t(meta.labelKey)}
    </Tag>
  );
};

const SummaryStat = ({ label, value, hint = '', tone = 'default' }) => {
  const valueType =
    tone === 'danger'
      ? 'danger'
      : tone === 'warning'
        ? 'warning'
        : tone === 'primary'
          ? 'primary'
          : undefined;
  return (
    <Card bordered bodyStyle={{ padding: 12 }}>
      <Text type='tertiary' size='small'>
        {label}
      </Text>
      <div className='mt-1 text-2xl font-semibold tabular-nums'>
        <Text type={valueType}>{value}</Text>
      </div>
      {hint ? (
        <Text type='tertiary' size='small' className='mt-1 block'>
          {hint}
        </Text>
      ) : null}
    </Card>
  );
};

const UsageWindowRow = ({ window, t }) => {
  const tone = window?.tone || 'muted';
  const progress = Number.isFinite(window?.progress) ? window.progress : 0;
  const hint = window?.remainingText || window?.resetAt || '';
  return (
    <div className='flex items-center gap-2 text-xs' key={window?.key}>
      <span className='w-12 shrink-0 font-medium uppercase tracking-wide text-semi-color-text-2'>
        {window?.label || '-'}
      </span>
      <div className='min-w-0 flex-1'>
        <Progress
          percent={Math.min(100, Math.max(0, progress))}
          stroke={toneToSemiColor(tone)}
          showInfo={false}
          size='small'
        />
      </div>
      <span className='w-12 shrink-0 text-right font-mono tabular-nums'>
        <Text type={toneToTextType(tone)} size='small'>
          {formatUsageWindowProgress(window?.progress)}
        </Text>
      </span>
      {hint ? (
        <span
          className='min-w-0 shrink-0 truncate text-semi-color-text-2'
          style={{ maxWidth: 110 }}
          title={hint}
        >
          {hint}
        </span>
      ) : null}
    </div>
  );
};

const DailyUsageLine = ({ account, t }) => {
  const requests = Number(account?.usage_daily_requests || 0);
  const tokens = Number(account?.usage_daily_tokens || 0);
  const cost = Number(account?.usage_daily_cost || 0);
  const segments = [];
  if (requests > 0) {
    segments.push(`${formatCRSRequestCount(requests)} ${t('次')}`);
  }
  const tokensText = formatCRSTokenCount(tokens);
  if (tokensText) segments.push(`${tokensText} Tok`);
  const costText = formatCRSDailyCost(cost);
  if (costText) segments.push(costText);
  if (segments.length === 0) {
    return (
      <Text type='tertiary' size='small'>
        {t('今日暂无数据')}
      </Text>
    );
  }
  return (
    <div className='flex items-center gap-2 text-xs font-mono tabular-nums text-semi-color-text-1'>
      <Text type='tertiary' size='small'>
        {t('今日')}
      </Text>
      <span>{segments.join(' · ')}</span>
    </div>
  );
};

const AccountStatusTags = ({ account, t }) => (
  <div className='flex flex-wrap gap-1'>
    <Tag color='cyan' size='small'>
      {getCRSPlatformBadgeLabel(account)}
    </Tag>
    {account?.is_active ? (
      <Tag color='green' size='small'>
        {t('活跃')}
      </Tag>
    ) : (
      <Tag color='grey' size='small'>
        {t('未激活')}
      </Tag>
    )}
    {account?.schedulable ? (
      <Tag color='blue' size='small'>
        {t('可调度')}
      </Tag>
    ) : null}
    {account?.rate_limited ? (
      <Tag color='orange' size='small'>
        {t('限速中')}
      </Tag>
    ) : null}
    {account?.status && account.status !== 'active' ? (
      <Tag color='white' size='small'>
        {account.status}
      </Tag>
    ) : null}
  </div>
);

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
  const [sortProblematic, setSortProblematic] = useState(false);

  useEffect(() => {
    setKeyword('');
    setPlatform('');
    setQuotaState('');
    setSortProblematic(false);
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

  const problemScore = (account) => {
    if (account?.rate_limited) return 3;
    if (account?.sync_error || account?.error_message) return 2;
    if (!account?.is_active) return 1;
    return 0;
  };

  const displayAccounts = useMemo(() => {
    if (!sortProblematic) return filteredAccounts;
    return [...filteredAccounts].sort(
      (a, b) => problemScore(b) - problemScore(a),
    );
  }, [filteredAccounts, sortProblematic]);

  const columns = [
    {
      title: t('账号'),
      dataIndex: 'name',
      render: (_, record) => (
        <div className='min-w-0'>
          <div className='truncate font-medium text-semi-color-text-0'>
            {record.name || record.remote_account_id}
          </div>
          <div className='mt-0.5 break-all text-xs text-semi-color-text-2'>
            {record.remote_account_id}
          </div>
          {record.subscription_plan ? (
            <div className='mt-0.5 text-xs'>
              <Text type='tertiary' size='small'>
                {t('计划')}: {record.subscription_plan}
              </Text>
            </div>
          ) : null}
          {record.last_synced_at ? (
            <div className='mt-0.5 text-xs'>
              <Text type='tertiary' size='small'>
                {t('同步')} {timestamp2string(record.last_synced_at)}
              </Text>
            </div>
          ) : null}
          {record.sync_error || record.error_message ? (
            <div className='mt-1 break-all text-xs'>
              <Text type='danger' size='small'>
                {record.sync_error || record.error_message}
              </Text>
            </div>
          ) : null}
        </div>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 180,
      render: (_, record) => <AccountStatusTags account={record} t={t} />,
    },
    {
      title: t('会话窗口'),
      dataIndex: 'usage_windows',
      render: (_, record) => {
        const windows = buildCRSUsageWindows(record);
        const hasWindows = windows.length > 0;
        return (
          <div className='flex flex-col gap-1.5'>
            {hasWindows ? (
              windows.map((window) => (
                <UsageWindowRow key={window.key} window={window} t={t} />
              ))
            ) : (
              <Text type='tertiary' size='small'>
                {t('暂无额度数据')}
              </Text>
            )}
            <Divider margin='4px' />
            <DailyUsageLine account={record} t={t} />
          </div>
        );
      },
    },
  ];

  return (
    <SideSheet
      visible={visible}
      onCancel={onClose}
      title={site?.name || site?.host || t('站点详情')}
      width={isMobile ? '100%' : 1040}
      bodyStyle={{ padding: 16 }}
    >
      <Spin spinning={loading}>
        {site ? (
          <div className='flex flex-col gap-4'>
            <Card bordered bodyStyle={{ padding: 16 }}>
              <div className='flex flex-wrap items-start justify-between gap-3'>
                <div className='min-w-0 flex-1'>
                  <Space spacing={8} wrap>
                    <Title heading={6} style={{ margin: 0 }}>
                      {site.name || site.host}
                    </Title>
                    {renderSiteStatusTag(site.status, t)}
                    {site.group ? (
                      <Tag color='blue' size='small'>
                        {site.group}
                      </Tag>
                    ) : null}
                  </Space>
                  <div className='mt-1'>
                    <Text
                      type='tertiary'
                      size='small'
                      className='break-all'
                    >
                      {site.scheme}://{site.host}
                    </Text>
                  </div>
                  <div className='mt-2 flex flex-wrap gap-x-4 gap-y-1'>
                    <Text type='tertiary' size='small'>
                      {t('用户名')}: {site.username || '-'}
                    </Text>
                    <Text type='tertiary' size='small'>
                      {t('最近同步')}:{' '}
                      {site.last_synced_at
                        ? timestamp2string(site.last_synced_at)
                        : '-'}
                    </Text>
                  </div>
                  {site.last_sync_error ? (
                    <div className='mt-2'>
                      <Text type='danger' size='small' className='break-all'>
                        {site.last_sync_error}
                      </Text>
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
            </Card>

            <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
              <SummaryStat
                label={t('观察账号')}
                value={observer.total_accounts ?? accounts.length}
              />
              <SummaryStat
                label={t('可调度')}
                tone='primary'
                value={observer.schedulable_count ?? 0}
                hint={`${t('活跃')} ${observer.active_accounts ?? 0}`}
              />
              <SummaryStat
                label={t('限速中')}
                tone='danger'
                value={observer.rate_limited_count ?? 0}
              />
              <SummaryStat
                label={t('低额度')}
                tone='warning'
                value={observer.low_quota_count ?? 0}
                hint={`${t('空额度')} ${observer.empty_quota_count ?? 0}`}
              />
            </div>

            {dashboard?.overview ? (
              <Card
                bordered
                title={t('Dashboard 概览')}
                bodyStyle={{ padding: 12 }}
                headerStyle={{ padding: '10px 16px' }}
              >
                <div className='mb-2'>
                  <Text type='tertiary' size='small'>
                    {t('这是远端 CRS /admin/dashboard 的缓存结果')}
                  </Text>
                </div>
                <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
                  <SummaryStat
                    label={t('总账号')}
                    value={dashboard.overview.totalAccounts ?? 0}
                  />
                  <SummaryStat
                    label={t('正常账号')}
                    tone='primary'
                    value={dashboard.overview.normalAccounts ?? 0}
                  />
                  <SummaryStat
                    label={t('API Keys')}
                    value={dashboard.overview.totalApiKeys ?? 0}
                  />
                  <SummaryStat
                    label={t('今日请求')}
                    value={dashboard.recentActivity?.requestsToday ?? 0}
                  />
                </div>
              </Card>
            ) : null}

            <Card
              bordered
              bodyStyle={{ padding: 12 }}
              headerStyle={{ padding: '10px 16px' }}
              title={
                <div className='flex items-center justify-between'>
                  <div>
                    <Title heading={6} style={{ margin: 0 }}>
                      {t('站点账号明细')}
                    </Title>
                    <Text type='tertiary' size='small' className='mt-1 block'>
                      {t('来自远端 CRS 的标准化账号快照')}
                    </Text>
                  </div>
                  <Text type='tertiary' size='small'>
                    {t('显示 {{filtered}} / {{total}}', {
                      filtered: displayAccounts.length,
                      total: accounts.length,
                    })}
                  </Text>
                </div>
              }
            >
              {accounts.length > 0 ? (
                <div className='mb-3 flex flex-col gap-2'>
                  <div className='grid gap-2 lg:grid-cols-[minmax(0,1.2fr),200px,200px]'>
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
                    <Tooltip
                      content={t(
                        '额度过滤仅对有 credit 配额的平台（如 OpenAI API）有效；Claude 会话窗口账号不适用',
                      )}
                    >
                      <Select
                        value={quotaState}
                        optionList={QUOTA_FILTER_OPTIONS.map((item) => ({
                          ...item,
                          label: t(item.label),
                        }))}
                        onChange={(value) => setQuotaState(value || '')}
                        placeholder={t('额度状态')}
                      />
                    </Tooltip>
                  </div>
                  <div className='flex items-center justify-end'>
                    <Button
                      size='small'
                      theme={sortProblematic ? 'solid' : 'borderless'}
                      type={sortProblematic ? 'warning' : 'tertiary'}
                      icon={<AlertTriangle size={13} />}
                      onClick={() => setSortProblematic((prev) => !prev)}
                    >
                      {t('问题优先')}
                    </Button>
                  </div>
                </div>
              ) : null}
              {accounts.length > 0 ? (
                <Table
                  dataSource={displayAccounts}
                  columns={columns}
                  rowKey={(record) =>
                    `${record.site_id}-${record.remote_account_id}`
                  }
                  pagination={false}
                  size='small'
                  scroll={isMobile ? { x: 820 } : undefined}
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
            </Card>
          </div>
        ) : (
          <Empty image={null} description={t('请选择一个站点')} />
        )}
      </Spin>
    </SideSheet>
  );
};

export default CRSSiteDetailSideSheet;
