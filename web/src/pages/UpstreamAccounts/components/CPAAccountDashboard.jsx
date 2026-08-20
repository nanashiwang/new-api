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
import { useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Select,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  Activity,
  ArrowUpRight,
  CircleAlert,
  CircleCheck,
  Clock3,
  FilterX,
  Layers3,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  ServerCog,
  ShieldCheck,
  Trash2,
  TriangleAlert,
  WalletCards,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { timestamp2string } from '../../../helpers';
import { useCPAData } from '../hooks/useCPAData';
import CPASiteModal from './CPASiteModal';
import {
  filterCPAAccounts,
  getCPAAccountDisplayName,
  getCPAAccountState,
  getCPAProviderName,
  summarizeCPAAccounts,
} from './cpaDashboard.utils';

const { Text, Title } = Typography;

const STATE_META = {
  available: { color: 'green', label: '可用', tone: 'bg-green-500' },
  limited: { color: 'orange', label: '冷却 / 限速', tone: 'bg-orange-400' },
  disabled: { color: 'grey', label: '已停用', tone: 'bg-slate-400' },
  abnormal: { color: 'red', label: '异常', tone: 'bg-red-500' },
  unknown: { color: 'light-blue', label: '未知', tone: 'bg-sky-400' },
};

const SITE_STATUS_META = {
  0: { color: 'grey', label: '尚未同步' },
  1: { color: 'green', label: '已同步' },
  2: { color: 'red', label: '同步失败' },
};

const PROVIDER_TONES = [
  'bg-emerald-500',
  'bg-orange-400',
  'bg-blue-500',
  'bg-violet-500',
  'bg-pink-500',
  'bg-slate-500',
];

const parseRemoteTimestamp = (value) => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value > 1e12 ? value : value * 1000;
  }
  const text = String(value ?? '').trim();
  if (!text) return NaN;
  if (/^\d+(\.\d+)?$/.test(text)) {
    const numeric = Number(text);
    return numeric > 1e12 ? numeric : numeric * 1000;
  }
  return Date.parse(text);
};

const formatRemoteTime = (value, fallback = '-') => {
  const timestamp = parseRemoteTimestamp(value);
  return Number.isFinite(timestamp)
    ? timestamp2string(Math.floor(timestamp / 1000))
    : fallback;
};

const formatPercent = (value) =>
  Number.isFinite(value) ? `${Math.round(value)}%` : '-';

const stopCardClick = (event) => event.stopPropagation();

function AccountStateTag({ account, t }) {
  const state = getCPAAccountState(account);
  const meta = STATE_META[state];
  return (
    <Tooltip content={account?.status_message || undefined}>
      <Tag color={meta.color} size='small'>
        {t(meta.label)}
      </Tag>
    </Tooltip>
  );
}

function SummaryCard({ label, value, hint, icon: Icon, tone, onClick }) {
  const tones = {
    blue: 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300',
    green:
      'bg-green-50 text-green-600 dark:bg-green-900/30 dark:text-green-300',
    orange:
      'bg-orange-50 text-orange-600 dark:bg-orange-900/30 dark:text-orange-300',
    red: 'bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-300',
  };
  return (
    <Card
      bordered
      bodyStyle={{ padding: 14 }}
      className={
        onClick ? 'cursor-pointer transition-shadow hover:shadow-sm' : ''
      }
      onClick={onClick}
    >
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <Text type='tertiary' size='small'>
            {label}
          </Text>
          <div className='mt-1 text-2xl font-semibold tabular-nums'>
            {value}
          </div>
          {hint ? (
            <Text type='tertiary' size='small' className='mt-1 block'>
              {hint}
            </Text>
          ) : null}
        </div>
        <span
          className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl ${tones[tone]}`}
          aria-hidden='true'
        >
          <Icon size={18} />
        </span>
      </div>
    </Card>
  );
}

function SiteStatusBar({ site }) {
  const total = Math.max(Number(site.account_count || 0), 1);
  const segments = [
    ['available_count', 'bg-green-500'],
    ['limited_count', 'bg-orange-400'],
    ['abnormal_count', 'bg-red-500'],
    ['disabled_count', 'bg-slate-400'],
    ['unknown_count', 'bg-sky-400'],
  ];
  return (
    <div className='mt-3 flex h-1.5 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800'>
      {segments.map(([key, tone]) => {
        const value = Number(site[key] || 0);
        if (!value) return null;
        return (
          <span
            key={key}
            className={tone}
            style={{ width: `${Math.min((value / total) * 100, 100)}%` }}
          />
        );
      })}
    </div>
  );
}

function CPASiteCard({
  site,
  t,
  selected,
  onSelect,
  onEdit,
  onRefresh,
  onDelete,
  refreshing,
  deleting,
}) {
  const status = SITE_STATUS_META[site.status] || SITE_STATUS_META[0];
  const total = Number(site.account_count || 0);
  const available = Number(site.available_count || 0);
  const availability = total ? (available / total) * 100 : 0;

  return (
    <div
      role='button'
      tabIndex={0}
      onClick={() => onSelect(site.id)}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onSelect(site.id);
        }
      }}
      className='min-w-0 outline-none'
    >
      <Card
        bordered
        bodyStyle={{ padding: 14 }}
        className={`h-full cursor-pointer transition-all hover:shadow-sm ${selected ? 'border-blue-500 bg-blue-50/30 shadow-sm dark:bg-blue-900/10' : ''}`}
      >
        <div className='flex items-start justify-between gap-3'>
          <div className='min-w-0 flex-1'>
            <div className='flex flex-wrap items-center gap-2'>
              <Text strong className='truncate'>
                {site.name || site.host}
              </Text>
              <Tag color={status.color} size='small'>
                {t(status.label)}
              </Tag>
            </div>
            <Text type='tertiary' size='small' className='mt-1 block break-all'>
              {site.scheme}://{site.host}
            </Text>
            <Text type='tertiary' size='small' className='mt-1 block'>
              Management Key: {site.management_key_masked || '****'}
            </Text>
          </div>
          <div
            className='flex shrink-0 gap-1'
            onClick={stopCardClick}
            onKeyDown={stopCardClick}
          >
            <Tooltip content={t('刷新')}>
              <Button
                theme='borderless'
                size='small'
                icon={<RefreshCw size={14} />}
                loading={refreshing}
                onClick={() => onRefresh(site.id)}
              />
            </Tooltip>
            <Tooltip content={t('编辑')}>
              <Button
                theme='borderless'
                size='small'
                icon={<Pencil size={14} />}
                onClick={() => onEdit(site)}
              />
            </Tooltip>
            <Tooltip content={t('删除')}>
              <Button
                theme='borderless'
                type='danger'
                size='small'
                icon={<Trash2 size={14} />}
                loading={deleting}
                onClick={() => onDelete(site)}
              />
            </Tooltip>
          </div>
        </div>

        <SiteStatusBar site={site} />
        <div className='mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-[var(--semi-color-text-2)]'>
          <span>
            <Text strong className='tabular-nums'>
              {available}
            </Text>{' '}
            {t('可用')} / {total}
          </span>
          <span>
            {t('可用率')} {formatPercent(availability)}
          </span>
          <span className='ml-auto flex items-center gap-1'>
            <Activity size={12} />
            {site.last_synced_at > 0
              ? timestamp2string(site.last_synced_at)
              : t('尚未同步')}
          </span>
        </div>

        {site.last_sync_error ? (
          <div className='mt-3 rounded-md border border-red-200 bg-red-50 px-2.5 py-2 text-xs text-red-600 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300'>
            <div className='flex items-start gap-1.5'>
              <CircleAlert size={13} className='mt-0.5 shrink-0' />
              <span className='break-all'>{site.last_sync_error}</span>
            </div>
          </div>
        ) : null}
      </Card>
    </div>
  );
}

function ProviderDistribution({ providers, t }) {
  return (
    <Card bordered bodyStyle={{ padding: 14 }}>
      <div className='mb-3 flex items-center justify-between gap-2'>
        <div className='flex items-center gap-2'>
          <Layers3 size={16} className='text-blue-500' />
          <Title heading={6}>{t('平台分布')}</Title>
        </div>
        <Text type='tertiary' size='small'>
          {t('可用 / 总数')}
        </Text>
      </div>
      {providers.length === 0 ? (
        <Text type='tertiary' size='small'>
          {t('暂无 CPA 账号数据')}
        </Text>
      ) : (
        <div className='space-y-3'>
          {providers.map((provider, index) => (
            <div key={provider.name}>
              <div className='mb-1 flex items-center gap-2 text-xs'>
                <span
                  className={`h-2 w-2 rounded-sm ${PROVIDER_TONES[index % PROVIDER_TONES.length]}`}
                />
                <span className='min-w-0 flex-1 truncate'>{provider.name}</span>
                <span className='tabular-nums text-[var(--semi-color-text-2)]'>
                  {provider.available} / {provider.total}
                </span>
              </div>
              <div className='h-1.5 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800'>
                <span
                  className={`block h-full rounded-full ${PROVIDER_TONES[index % PROVIDER_TONES.length]}`}
                  style={{
                    width: `${provider.total ? (provider.available / provider.total) * 100 : 0}%`,
                  }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

function RiskPanel({ alerts, t }) {
  return (
    <Card bordered bodyStyle={{ padding: 14 }}>
      <div className='mb-3 flex items-center justify-between gap-2'>
        <div className='flex items-center gap-2'>
          <TriangleAlert size={16} className='text-orange-500' />
          <Title heading={6}>{t('风险提醒')}</Title>
        </div>
        <Text type='tertiary' size='small'>
          {alerts.length} {t('项待关注')}
        </Text>
      </div>
      {alerts.length === 0 ? (
        <div className='flex items-center gap-2 rounded-md bg-green-50 px-3 py-2 text-xs text-green-600 dark:bg-green-950/20 dark:text-green-300'>
          <CircleCheck size={14} />
          {t('当前没有需要关注的 CPA 状态')}
        </div>
      ) : (
        <div className='space-y-2'>
          {alerts.map((alert) => (
            <div
              key={alert.id}
              className={`rounded-md border-l-2 px-3 py-2 ${
                alert.level === 'high'
                  ? 'border-red-500 bg-red-50 dark:bg-red-950/20'
                  : alert.level === 'mid'
                    ? 'border-orange-400 bg-orange-50 dark:bg-orange-950/20'
                    : 'border-slate-300 bg-slate-50 dark:bg-slate-900/60'
              }`}
            >
              <div className='flex items-start gap-2'>
                <div className='min-w-0 flex-1'>
                  <Text strong size='small' className='block'>
                    {alert.title}
                  </Text>
                  <Text type='tertiary' size='small' className='mt-1 block'>
                    {alert.description}
                  </Text>
                </div>
                {alert.onClick ? (
                  <Button
                    theme='borderless'
                    size='small'
                    icon={<ArrowUpRight size={14} />}
                    onClick={alert.onClick}
                  />
                ) : null}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

function AccountDetail({ account, t }) {
  const total = Number(account.success || 0) + Number(account.failed || 0);
  const failureRate = total ? (Number(account.failed || 0) / total) * 100 : 0;
  return (
    <div className='grid gap-3 bg-[var(--semi-color-fill-0)] p-3 sm:grid-cols-3'>
      <div className='rounded-md border border-[var(--semi-color-border)] bg-white p-3 dark:bg-gray-900'>
        <div className='mb-2 flex items-center justify-between text-xs text-[var(--semi-color-text-2)]'>
          <span>{t('调用统计')}</span>
          <span className='tabular-nums'>
            {formatPercent(failureRate)} {t('失败率')}
          </span>
        </div>
        <div className='text-sm tabular-nums'>
          <Text type='success'>{account.success ?? 0}</Text>
          <span className='mx-1 text-[var(--semi-color-text-3)]'>/</span>
          <Text type={(account.failed ?? 0) > 0 ? 'danger' : 'tertiary'}>
            {account.failed ?? 0}
          </Text>
        </div>
      </div>
      <div className='rounded-md border border-[var(--semi-color-border)] bg-white p-3 dark:bg-gray-900'>
        <div className='mb-2 text-xs text-[var(--semi-color-text-2)]'>
          {t('账号信息')}
        </div>
        <div className='space-y-1 text-xs text-[var(--semi-color-text-2)]'>
          <div>
            {t('类型')}: {account.account_type || account.type || '-'}
          </div>
          <div>
            {t('优先级')}: {account.priority ?? '-'}
          </div>
          <div>
            {t('项目')}: {account.project_id || '-'}
          </div>
        </div>
      </div>
      <div className='rounded-md border border-[var(--semi-color-border)] bg-white p-3 dark:bg-gray-900'>
        <div className='mb-2 text-xs text-[var(--semi-color-text-2)]'>
          {t('最近事件')}
        </div>
        <div className='space-y-1 text-xs text-[var(--semi-color-text-2)]'>
          <div>
            {t('最近更新')}:{' '}
            {formatRemoteTime(account.updated_at || account.last_refresh)}
          </div>
          <div>
            {t('下次重试')}: {formatRemoteTime(account.next_retry_after)}
          </div>
          {account.status_message ? (
            <div className='break-all text-red-500'>
              {account.status_message}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

export default function CPAAccountDashboard({ t: tProp }) {
  const { t } = useTranslation();
  const tFn = tProp ?? t;
  const {
    sites,
    accounts,
    loading,
    saving,
    testing,
    refreshingSiteId,
    deletingSiteId,
    loadOverview,
    testConnection,
    saveSite,
    refreshSite,
    deleteSite,
  } = useCPAData();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSite, setEditingSite] = useState(null);
  const [keyword, setKeyword] = useState('');
  const [siteId, setSiteId] = useState(0);
  const [state, setState] = useState('');
  const [provider, setProvider] = useState('');

  const summary = useMemo(() => summarizeCPAAccounts(accounts), [accounts]);
  const providerOptions = useMemo(() => {
    const values = new Set(
      accounts.map((account) => getCPAProviderName(account)).filter(Boolean),
    );
    return [...values].sort((left, right) => left.localeCompare(right));
  }, [accounts]);
  const providerSummary = useMemo(() => {
    const grouped = new Map();
    accounts.forEach((account) => {
      const name = getCPAProviderName(account);
      const entry = grouped.get(name) || { name, total: 0, available: 0 };
      entry.total += 1;
      if (getCPAAccountState(account) === 'available') entry.available += 1;
      grouped.set(name, entry);
    });
    return [...grouped.values()].sort(
      (left, right) => right.total - left.total,
    );
  }, [accounts]);
  const filteredAccounts = useMemo(
    () => filterCPAAccounts(accounts, { keyword, siteId, state, provider }),
    [accounts, keyword, siteId, state, provider],
  );
  const latestSyncAt = useMemo(
    () => Math.max(0, ...sites.map((site) => Number(site.last_synced_at || 0))),
    [sites],
  );
  const activeFilterCount =
    Number(Boolean(keyword)) +
    Number(siteId > 0) +
    Number(Boolean(state)) +
    Number(Boolean(provider));

  const focusAccounts = (nextState = state, nextSiteId = siteId) => {
    setState(nextState);
    setSiteId(nextSiteId);
    window.setTimeout(() => {
      document
        .getElementById('cpa-account-details')
        ?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }, 0);
  };

  const alerts = useMemo(() => {
    const items = [];
    const failedSites = sites.filter((site) => site.status === 2);
    if (failedSites.length) {
      items.push({
        id: 'sites-failed',
        level: 'high',
        title: `${failedSites.length} ${tFn('个 CPA 服务同步失败')}`,
        description: failedSites
          .map((site) => site.name || site.host)
          .join('、'),
        onClick: () => focusAccounts('', failedSites[0].id),
      });
    }
    if (summary.abnormal > 0) {
      items.push({
        id: 'accounts-abnormal',
        level: 'high',
        title: `${summary.abnormal} ${tFn('个账号异常')}`,
        description: tFn('建议优先检查状态信息和最近失败记录'),
        onClick: () => focusAccounts('abnormal', 0),
      });
    }
    if (summary.limited > 0) {
      items.push({
        id: 'accounts-limited',
        level: 'mid',
        title: `${summary.limited} ${tFn('个账号冷却 / 限速')}`,
        description: tFn('账号将在下次重试时间后重新评估'),
        onClick: () => focusAccounts('limited', 0),
      });
    }
    if (summary.unknown > 0) {
      items.push({
        id: 'accounts-unknown',
        level: 'low',
        title: `${summary.unknown} ${tFn('个账号状态未知')}`,
        description: tFn('CPA 返回了未识别的状态值'),
        onClick: () => focusAccounts('unknown', 0),
      });
    }
    return items;
  }, [sites, summary, tFn]);

  const clearFilters = () => {
    setKeyword('');
    setSiteId(0);
    setState('');
    setProvider('');
  };
  const openCreate = () => {
    setEditingSite(null);
    setModalVisible(true);
  };
  const openEdit = (site) => {
    setEditingSite(site);
    setModalVisible(true);
  };
  const handleSave = async (payload) => {
    if (await saveSite(editingSite, payload)) setModalVisible(false);
  };
  const handleDelete = (site) => {
    Modal.confirm({
      title: tFn('确认删除该 CPA 服务？'),
      content: tFn('删除后会移除该服务的本地账号快照，不会修改 CPA 服务器。'),
      okText: tFn('删除'),
      cancelText: tFn('取消'),
      okButtonProps: { type: 'danger' },
      onOk: () => deleteSite(site.id),
    });
  };

  const columns = [
    {
      title: tFn('账号'),
      render: (_, account) => (
        <div className='min-w-0'>
          <Text strong className='block truncate'>
            {getCPAAccountDisplayName(account)}
          </Text>
          <Text type='tertiary' size='small' className='block truncate'>
            {account.auth_index || account.id || '-'}
          </Text>
        </div>
      ),
    },
    {
      title: tFn('来源 / 服务'),
      dataIndex: 'site_name',
      ellipsis: true,
    },
    {
      title: tFn('平台'),
      render: (_, account) => (
        <Tag color='blue' size='small'>
          {getCPAProviderName(account)}
        </Tag>
      ),
    },
    {
      title: tFn('状态'),
      render: (_, account) => <AccountStateTag account={account} t={tFn} />,
    },
    {
      title: tFn('成功 / 失败'),
      sorter: (left, right) =>
        Number(left.success || 0) +
        Number(left.failed || 0) -
        (Number(right.success || 0) + Number(right.failed || 0)),
      render: (_, account) => (
        <span className='tabular-nums'>
          <Text type='success'>{account.success ?? 0}</Text>
          {' / '}
          <Text type={(account.failed ?? 0) > 0 ? 'danger' : 'tertiary'}>
            {account.failed ?? 0}
          </Text>
        </span>
      ),
    },
    {
      title: tFn('下次重试'),
      render: (_, account) => formatRemoteTime(account.next_retry_after),
    },
    {
      title: tFn('最近更新'),
      sorter: (left, right) =>
        parseRemoteTimestamp(left.updated_at || left.last_refresh) -
        parseRemoteTimestamp(right.updated_at || right.last_refresh),
      render: (_, account) =>
        formatRemoteTime(account.updated_at || account.last_refresh),
    },
  ];

  return (
    <div className='space-y-4'>
      <Card bordered bodyStyle={{ padding: 0 }} className='overflow-hidden'>
        <div className='relative overflow-hidden bg-gradient-to-br from-sky-50 via-white to-amber-50 px-4 py-5 dark:from-sky-950/35 dark:via-gray-900 dark:to-amber-950/25 sm:px-5'>
          <div className='relative flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
            <div>
              <div className='flex flex-wrap items-center gap-2'>
                <span className='flex h-10 w-10 items-center justify-center rounded-xl bg-sky-600 text-white shadow-sm'>
                  <ServerCog size={21} />
                </span>
                <div>
                  <Title heading={5}>{tFn('CPA 账号概览')}</Title>
                  <Text type='tertiary' size='small'>
                    {tFn('通过 Management API 只读同步真实账号状态')}
                  </Text>
                </div>
              </div>
              <div className='mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-[var(--semi-color-text-2)]'>
                <span>
                  {tFn('最近同步')}:{' '}
                  {latestSyncAt
                    ? timestamp2string(latestSyncAt)
                    : tFn('尚未同步')}
                </span>
                <span className='text-[var(--semi-color-text-3)]'>·</span>
                <span>
                  {sites.length} {tFn('个 CPA 服务')}
                </span>
              </div>
            </div>
            <div className='flex flex-wrap gap-2'>
              <Button
                icon={<RefreshCw size={15} />}
                loading={loading}
                onClick={loadOverview}
              >
                {tFn('刷新概览')}
              </Button>
              <Button
                type='primary'
                icon={<Plus size={15} />}
                onClick={openCreate}
              >
                {tFn('人工接入 CPA')}
              </Button>
            </div>
          </div>
        </div>
      </Card>

      <Spin spinning={loading && sites.length === 0}>
        <div className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
          <SummaryCard
            label={tFn('账号总数')}
            value={summary.total}
            hint={`${tFn('CPA 服务')} ${sites.length}`}
            icon={WalletCards}
            tone='blue'
            onClick={() => focusAccounts('', 0)}
          />
          <SummaryCard
            label={tFn('可用账号')}
            value={summary.available}
            icon={ShieldCheck}
            tone='green'
            onClick={() => focusAccounts('available', 0)}
          />
          <SummaryCard
            label={tFn('冷却 / 限速')}
            value={summary.limited}
            hint={`${tFn('已停用')} ${summary.disabled}`}
            icon={Clock3}
            tone='orange'
            onClick={() => focusAccounts('limited', 0)}
          />
          <SummaryCard
            label={tFn('异常账号')}
            value={summary.abnormal}
            hint={`${tFn('未知')} ${summary.unknown}`}
            icon={CircleAlert}
            tone='red'
            onClick={() => focusAccounts('abnormal', 0)}
          />
        </div>
      </Spin>

      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.5fr)_minmax(280px,0.7fr)]'>
        <Card
          bordered
          title={tFn('CPA 服务列表')}
          headerStyle={{ padding: '10px 16px' }}
          bodyStyle={{ padding: 12 }}
        >
          {sites.length === 0 && !loading ? (
            <div className='flex flex-col items-center justify-center gap-3 py-10'>
              <ServerCog size={36} className='opacity-30' />
              <Text type='tertiary' size='small' className='text-center'>
                {tFn('暂无 CPA 服务，点击“人工接入 CPA”开始配置')}
              </Text>
              <Button
                type='primary'
                icon={<Plus size={14} />}
                onClick={openCreate}
              >
                {tFn('人工接入 CPA')}
              </Button>
            </div>
          ) : (
            <div className='grid grid-cols-1 gap-3 lg:grid-cols-2'>
              {sites.map((site) => (
                <CPASiteCard
                  key={site.id}
                  site={site}
                  t={tFn}
                  selected={siteId === site.id}
                  onSelect={(id) => setSiteId(siteId === id ? 0 : id)}
                  onEdit={openEdit}
                  onRefresh={refreshSite}
                  onDelete={handleDelete}
                  refreshing={refreshingSiteId === site.id}
                  deleting={deletingSiteId === site.id}
                />
              ))}
            </div>
          )}
        </Card>

        <div className='space-y-4'>
          <ProviderDistribution providers={providerSummary} t={tFn} />
          <RiskPanel alerts={alerts} t={tFn} />
        </div>
      </div>

      <Card bordered bodyStyle={{ padding: 16 }} id='cpa-account-details'>
        <div className='mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
          <div>
            <div className='flex items-center gap-2'>
              <Activity size={17} className='text-blue-500' />
              <Title heading={5}>{tFn('CPA 账号使用情况')}</Title>
            </div>
            <Text type='tertiary' size='small'>
              {tFn('账号状态和成功 / 失败计数均来自 CPA Management API')}
            </Text>
          </div>
          <Text type='tertiary' size='small'>
            {tFn('显示 {{shown}} / {{total}} 个账号', {
              shown: filteredAccounts.length,
              total: accounts.length,
            })}
          </Text>
        </div>
        <Banner
          className='mb-4'
          type='warning'
          closeIcon={null}
          description={tFn(
            '当前 CLIProxyAPI Management API 不提供额度窗口，面板只展示真实账号状态与请求计数。',
          )}
        />

        <div className='mb-3 rounded-md border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3'>
          <div className='mb-2 flex items-center gap-2 text-xs text-[var(--semi-color-text-2)]'>
            <Search size={13} />
            <span>{tFn('搜索')}</span>
          </div>
          <div className='flex flex-col gap-2 lg:flex-row'>
            <Input
              prefix={<Search size={15} />}
              value={keyword}
              onChange={setKeyword}
              placeholder={tFn('搜索账号、平台或服务')}
              showClear
              className='min-w-0 flex-1'
            />
            <Select
              value={siteId}
              onChange={setSiteId}
              className='w-full lg:w-52'
              optionList={[
                { value: 0, label: tFn('全部服务') },
                ...sites.map((site) => ({
                  value: site.id,
                  label: site.name || site.host,
                })),
              ]}
            />
          </div>
        </div>

        <div className='mb-3 flex flex-wrap items-center gap-2'>
          <span className='mr-1 text-xs text-[var(--semi-color-text-2)]'>
            {tFn('平台')}
          </span>
          <button
            type='button'
            className={`rounded-md border px-2.5 py-1 text-xs transition-colors ${!provider ? 'border-blue-300 bg-blue-50 text-blue-600 dark:bg-blue-950/30' : 'border-[var(--semi-color-border)] text-[var(--semi-color-text-2)] hover:border-blue-300'}`}
            onClick={() => setProvider('')}
          >
            {tFn('全部平台')}
          </button>
          {providerOptions.map((option) => (
            <button
              key={option}
              type='button'
              className={`rounded-md border px-2.5 py-1 text-xs transition-colors ${provider === option ? 'border-blue-300 bg-blue-50 text-blue-600 dark:bg-blue-950/30' : 'border-[var(--semi-color-border)] text-[var(--semi-color-text-2)] hover:border-blue-300'}`}
              onClick={() => setProvider(provider === option ? '' : option)}
            >
              {option}
            </button>
          ))}
        </div>

        <div className='mb-4 flex flex-wrap items-center gap-2'>
          <span className='mr-1 text-xs text-[var(--semi-color-text-2)]'>
            {tFn('状态')}
          </span>
          <button
            type='button'
            className={`rounded-md border px-2.5 py-1 text-xs transition-colors ${!state ? 'border-blue-300 bg-blue-50 text-blue-600 dark:bg-blue-950/30' : 'border-[var(--semi-color-border)] text-[var(--semi-color-text-2)] hover:border-blue-300'}`}
            onClick={() => setState('')}
          >
            {tFn('全部状态')}
          </button>
          {Object.entries(STATE_META).map(([value, meta]) => (
            <button
              key={value}
              type='button'
              className={`inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors ${state === value ? 'border-blue-300 bg-blue-50 text-blue-600 dark:bg-blue-950/30' : 'border-[var(--semi-color-border)] text-[var(--semi-color-text-2)] hover:border-blue-300'}`}
              onClick={() => setState(state === value ? '' : value)}
            >
              <span className={`h-1.5 w-1.5 rounded-full ${meta.tone}`} />
              {tFn(meta.label)}
              <span className='tabular-nums text-[var(--semi-color-text-3)]'>
                {summary[value]}
              </span>
            </button>
          ))}
          {activeFilterCount > 0 ? (
            <Button
              theme='borderless'
              size='small'
              icon={<FilterX size={14} />}
              onClick={clearFilters}
            >
              {tFn('清空')}
            </Button>
          ) : null}
        </div>

        {filteredAccounts.length === 0 ? (
          <Empty
            title={tFn(
              sites.length === 0 ? '尚未连接 CPA 服务' : '暂无 CPA 账号数据',
            )}
            description={tFn(
              sites.length === 0
                ? '请先使用“人工接入 CPA”添加服务。'
                : activeFilterCount > 0
                  ? '没有符合当前筛选条件的账号，请清空筛选。'
                  : '请刷新服务或调整筛选条件。',
            )}
          />
        ) : (
          <>
            <div className='hidden lg:block'>
              <Table
                dataSource={filteredAccounts}
                columns={columns}
                rowKey={(account) =>
                  `${account.site_id}:${account.auth_index || account.id || account.name}`
                }
                pagination={{ pageSize: 20 }}
                expandedRowRender={(account) => (
                  <AccountDetail account={account} t={tFn} />
                )}
                rowExpandable={(account) =>
                  Boolean(
                    account.status_message ||
                      account.note ||
                      account.updated_at ||
                      account.last_refresh,
                  )
                }
                expandRowByClick
                size='small'
              />
            </div>
            <div className='grid gap-3 lg:hidden'>
              {filteredAccounts.map((account) => (
                <Card
                  key={`${account.site_id}:${account.auth_index || account.id || account.name}`}
                  bordered
                  bodyStyle={{ padding: 12 }}
                >
                  <div className='flex items-start justify-between gap-3'>
                    <div className='min-w-0'>
                      <Text strong className='block truncate'>
                        {getCPAAccountDisplayName(account)}
                      </Text>
                      <Text type='tertiary' size='small'>
                        {account.site_name} · {getCPAProviderName(account)}
                      </Text>
                    </div>
                    <AccountStateTag account={account} t={tFn} />
                  </div>
                  <div className='mt-3 grid grid-cols-2 gap-2 text-sm'>
                    <div>
                      <Text type='tertiary' size='small'>
                        {tFn('成功 / 失败')}
                      </Text>
                      <div className='tabular-nums'>
                        {account.success ?? 0} / {account.failed ?? 0}
                      </div>
                    </div>
                    <div>
                      <Text type='tertiary' size='small'>
                        {tFn('下次重试')}
                      </Text>
                      <div>{formatRemoteTime(account.next_retry_after)}</div>
                    </div>
                  </div>
                  {account.status_message ? (
                    <div className='mt-3 rounded-md bg-red-50 px-2.5 py-2 text-xs text-red-600 dark:bg-red-950/20 dark:text-red-300'>
                      {account.status_message}
                    </div>
                  ) : null}
                </Card>
              ))}
            </div>
          </>
        )}
      </Card>

      <CPASiteModal
        visible={modalVisible}
        site={editingSite}
        onSave={handleSave}
        onTest={testConnection}
        onCancel={() => {
          if (!saving && !testing) setModalVisible(false);
        }}
        saving={saving}
        testing={testing}
      />
    </div>
  );
}
