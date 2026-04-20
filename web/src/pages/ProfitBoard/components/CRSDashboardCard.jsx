import { useEffect, useMemo, useState } from 'react';
import {
  Badge,
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
  CheckCircle2,
  Eye,
  Gauge,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldAlert,
  Trash2,
  Users,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { timestamp2string } from '../../../helpers/date';
import { useCRSData } from '../hooks/useCRSData';
import CRSSiteDetailSideSheet from './CRSSiteDetailSideSheet';
import CRSSiteModal from './CRSSiteModal';
import {
  buildCRSGroupOptions,
  getCRSLatestSyncAt,
  getCRSPlatformOptions,
  getCRSQuotaState,
} from './crsDashboard.utils';

const { Title, Text } = Typography;

const DEFAULT_ACCOUNT_QUERY = {
  page: 1,
  page_size: 20,
  site_id: 0,
  platform: '',
  status: '',
  quota_state: '',
  keyword: '',
};

const KNOWN_PLATFORM_OPTIONS = [
  'azure_openai',
  'bedrock',
  'ccr',
  'claude',
  'claude-console',
  'droid',
  'gemini',
  'gemini-api',
  'openai',
  'openai-responses',
].map((platform) => ({
  label: platform,
  value: platform,
}));

const QUOTA_STATE_OPTIONS = [
  { label: '全部额度', value: '' },
  { label: '低额度', value: 'low' },
  { label: '空额度', value: 'empty' },
  { label: '不限额', value: 'unlimited' },
];

function formatBigNumber(n) {
  if (n == null || Number.isNaN(Number(n))) return '-';
  if (Number(n) >= 1e8) return `${(Number(n) / 1e8).toFixed(2)}亿`;
  if (Number(n) >= 1e4) return `${(Number(n) / 1e4).toFixed(1)}万`;
  return String(n);
}

function healthPercent(stat) {
  if (!stat || stat.total === 0) return null;
  return Math.round((stat.normal / stat.total) * 100);
}

function formatQuotaSummary(account, t) {
  const quotaState = getCRSQuotaState(account);
  if (quotaState === 'unlimited') {
    return {
      color: 'blue',
      label: t('不限额'),
      detail: account.quota_reset_at || '-',
    };
  }
  const total = Number(account.quota_total || 0);
  const remaining = Number(account.quota_remaining || 0);
  const detail = total > 0 ? `${remaining} / ${total}` : `${remaining || 0}`;
  if (quotaState === 'empty') {
    return {
      color: 'red',
      label: t('已耗尽'),
      detail,
    };
  }
  if (quotaState === 'low') {
    return {
      color: 'orange',
      label: t('低额度'),
      detail,
    };
  }
  return {
    color: 'green',
    label: t('正常'),
    detail,
  };
}

function renderAccountStatus(record, t) {
  const tags = [];
  if (record.is_active) {
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
  if (record.schedulable) {
    tags.push(
      <Tag key='schedulable' color='blue' size='small'>
        {t('可调度')}
      </Tag>,
    );
  }
  if (record.rate_limited) {
    tags.push(
      <Tag key='rate_limited' color='orange' size='small'>
        {t('限速中')}
      </Tag>,
    );
  }
  if (record.status) {
    tags.push(
      <Tag key='status' color='white' size='small'>
        {record.status}
      </Tag>,
    );
  }
  return <div className='flex flex-wrap gap-1'>{tags}</div>;
}

function SiteMetric({ label, value, tone = '' }) {
  return (
    <div className='rounded-lg bg-gray-50 dark:bg-gray-800 px-3 py-2'>
      <div className={`text-lg font-bold ${tone}`}>{value}</div>
      <div className='text-xs text-gray-500'>{label}</div>
    </div>
  );
}

function HintPill({ label, value, tone = '' }) {
  return (
    <div className='rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300'>
      <span>{label}</span>
      <span className={`ml-1 font-semibold ${tone}`}>{value}</span>
    </div>
  );
}

function StatCard({ icon, title, main, sub, accent }) {
  return (
    <div className='rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900'>
      <div className='flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400'>
        <span className={accent}>{icon}</span>
        {title}
      </div>
      <div className='mt-2 text-2xl font-bold leading-tight text-gray-900 dark:text-gray-50'>
        {main}
      </div>
      {sub ? (
        <div className='mt-1 text-xs text-gray-400 dark:text-gray-500'>
          {sub}
        </div>
      ) : null}
    </div>
  );
}

function PlatformTable({ accountsByPlatform }) {
  const { t } = useTranslation();

  const rows = useMemo(() => {
    if (!accountsByPlatform) return [];
    return Object.entries(accountsByPlatform)
      .filter(([, stat]) => stat.total > 0)
      .map(([platform, stat]) => ({
        platform,
        total: stat.total,
        normal: stat.normal,
        abnormal: stat.abnormal,
        paused: stat.paused,
        rateLimited: stat.rateLimited,
        health:
          stat.total > 0 ? Math.round((stat.normal / stat.total) * 100) : 0,
      }))
      .sort((left, right) => right.total - left.total);
  }, [accountsByPlatform]);

  if (!rows.length) return null;

  return (
    <Table
      dataSource={rows}
      columns={[
        {
          title: t('平台'),
          dataIndex: 'platform',
          render: (value) => (
            <span className='font-medium capitalize'>{value}</span>
          ),
        },
        { title: t('总账号'), dataIndex: 'total', align: 'right' },
        {
          title: t('正常'),
          dataIndex: 'normal',
          align: 'right',
          render: (value) => (
            <span className='font-semibold text-emerald-600 dark:text-emerald-400'>
              {value}
            </span>
          ),
        },
        {
          title: t('异常'),
          dataIndex: 'abnormal',
          align: 'right',
          render: (value) => (
            <span
              className={
                value > 0 ? 'font-semibold text-red-500' : 'text-gray-400'
              }
            >
              {value}
            </span>
          ),
        },
        {
          title: t('暂停'),
          dataIndex: 'paused',
          align: 'right',
          render: (value) => (
            <span className={value > 0 ? 'text-amber-500' : 'text-gray-400'}>
              {value}
            </span>
          ),
        },
        {
          title: t('限速'),
          dataIndex: 'rateLimited',
          align: 'right',
          render: (value) => (
            <span className={value > 0 ? 'text-orange-500' : 'text-gray-400'}>
              {value}
            </span>
          ),
        },
        {
          title: t('健康度'),
          dataIndex: 'health',
          align: 'right',
          render: (value) => {
            const color =
              value >= 90
                ? 'text-emerald-600 dark:text-emerald-400'
                : value >= 70
                  ? 'text-amber-500'
                  : 'text-red-500';
            return <span className={`font-bold ${color}`}>{value}%</span>;
          },
        },
      ]}
      rowKey='platform'
      size='small'
      pagination={false}
      bordered
    />
  );
}

function SiteCard({
  site,
  onDetail,
  onRefresh,
  onEdit,
  onDelete,
  refreshingSiteId,
  deletingSiteId,
  t,
}) {
  const isRefreshing = refreshingSiteId === site.id;
  const isDeleting = deletingSiteId === site.id;
  const health = site.dashboard?.overview
    ? healthPercent({
        total: site.dashboard.overview.totalAccounts,
        normal: site.dashboard.overview.normalAccounts,
      })
    : null;

  const statusTag = useMemo(() => {
    if (site.status === 1) {
      return (
        <Tag color='green' size='small'>
          {t('已同步')}
        </Tag>
      );
    }
    if (site.status === 2) {
      return (
        <Tag color='red' size='small'>
          {t('错误')}
        </Tag>
      );
    }
    return (
      <Tag color='grey' size='small'>
        {t('未同步')}
      </Tag>
    );
  }, [site.status, t]);

  const confirmDelete = () => {
    Modal.confirm({
      title: t('确认删除该 CRS 站点？'),
      content: t('删除后会同时移除该站点的观察快照。'),
      okText: t('删除'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger', loading: isDeleting },
      onOk: () => onDelete(site.id),
    });
  };

  const handleOpenDetail = () => {
    onDetail(site);
  };

  const stopEvent = (event) => {
    event?.stopPropagation?.();
  };

  return (
    <div
      className='flex cursor-pointer flex-col gap-3 rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-all hover:-translate-y-0.5 hover:border-sky-300 hover:shadow-md dark:border-gray-700 dark:bg-gray-900 dark:hover:border-sky-700'
      onClick={handleOpenDetail}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          handleOpenDetail();
        }
      }}
      role='button'
      tabIndex={0}
    >
      <div className='flex items-start justify-between gap-2'>
        <div className='min-w-0 flex-1'>
          <div className='flex flex-wrap items-center gap-2'>
            <Server size={15} className='shrink-0 text-sky-500' />
            <span className='break-all text-sm font-semibold text-gray-800 dark:text-gray-200'>
              {site.name || site.host}
            </span>
            {statusTag}
            {site.group ? (
              <Tag color='blue' size='small'>
                {site.group}
              </Tag>
            ) : null}
          </div>
          <div className='mt-1 text-xs text-gray-500 dark:text-gray-400'>
            {site.scheme}://{site.host}
          </div>
        </div>
        <div className='flex shrink-0 gap-1'>
          <Tooltip content={t('详情')}>
            <Button
              theme='borderless'
              size='small'
              icon={<Eye size={14} />}
              onClick={(event) => {
                stopEvent(event);
                onDetail(site);
              }}
            />
          </Tooltip>
          <Tooltip content={t('刷新')}>
            <Button
              theme='borderless'
              size='small'
              icon={<RefreshCw size={14} />}
              loading={isRefreshing}
              onClick={(event) => {
                stopEvent(event);
                onRefresh(site.id);
              }}
            />
          </Tooltip>
          <Tooltip content={t('编辑')}>
            <Button
              theme='borderless'
              size='small'
              icon={<Pencil size={14} />}
              onClick={(event) => {
                stopEvent(event);
                onEdit(site);
              }}
            />
          </Tooltip>
          <Tooltip content={t('删除')}>
            <Button
              theme='borderless'
              size='small'
              icon={<Trash2 size={14} />}
              loading={isDeleting}
              type='danger'
              onClick={(event) => {
                stopEvent(event);
                confirmDelete();
              }}
            />
          </Tooltip>
        </div>
      </div>

      <div className='grid grid-cols-2 gap-2 xl:grid-cols-4'>
        <SiteMetric label={t('观察账号')} value={site.account_count ?? 0} />
        <SiteMetric
          label={t('限速中')}
          value={site.rate_limited_count ?? 0}
          tone='text-orange-500'
        />
        <SiteMetric
          label={t('低额度')}
          value={site.low_quota_count ?? 0}
          tone='text-amber-500'
        />
        <SiteMetric
          label={t('Dashboard 健康')}
          value={health != null ? `${health}%` : '-'}
          tone={
            health != null && health < 90
              ? 'text-amber-500'
              : 'text-emerald-600 dark:text-emerald-400'
          }
        />
      </div>

      {site.last_synced_at > 0 ? (
        <div className='text-xs text-gray-400'>
          {t('最近同步')}: {timestamp2string(site.last_synced_at)}
        </div>
      ) : null}

      {site.status === 2 && site.last_sync_error ? (
        <div className='break-all rounded p-2 text-xs text-red-500 dark:bg-red-900/20 dark:text-red-400'>
          {site.last_sync_error}
        </div>
      ) : null}

      <div className='flex items-center justify-between border-t border-dashed border-gray-200 pt-2 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400'>
        <span>{t('点击卡片查看站点详情')}</span>
        <span className='font-medium text-sky-600 dark:text-sky-400'>
          {t('进入详情')}
        </span>
      </div>
    </div>
  );
}

export default function CRSDashboardCard({ t: tProp }) {
  const { t } = useTranslation();
  const tFn = tProp ?? t;
  const {
    sites,
    aggregate,
    observer,
    accounts,
    accountsTotal,
    loadingOverview,
    loadingAccounts,
    refreshingAll,
    refreshingSiteId,
    savingSite,
    deletingSiteId,
    siteDetail,
    loadingSiteDetail,
    loadOverview,
    loadAccounts,
    loadSiteAccounts,
    setSiteDetail,
    refreshSite,
    refreshAll,
    createSite,
    updateSite,
    deleteSite,
  } = useCRSData();

  const [modalVisible, setModalVisible] = useState(false);
  const [editingSite, setEditingSite] = useState(null);
  const [siteDetailVisible, setSiteDetailVisible] = useState(false);
  const [accountQuery, setAccountQuery] = useState(DEFAULT_ACCOUNT_QUERY);
  const [keywordDraft, setKeywordDraft] = useState('');

  useEffect(() => {
    loadAccounts(accountQuery);
  }, [
    accountQuery.keyword,
    accountQuery.page,
    accountQuery.page_size,
    accountQuery.platform,
    accountQuery.quota_state,
    accountQuery.site_id,
    accountQuery.status,
    loadAccounts,
  ]);

  const openCreate = () => {
    setEditingSite(null);
    setModalVisible(true);
  };

  const openEdit = (site) => {
    setEditingSite(site);
    setModalVisible(true);
  };

  const openSiteDetail = async (site) => {
    const fullSite = siteMap.get(site.id) || site;
    setSiteDetailVisible(true);
    setSiteDetail({
      site: fullSite,
      dashboard: fullSite.dashboard || null,
      observer: null,
      accounts: [],
    });
    await loadSiteAccounts(fullSite.id);
  };

  const closeSiteDetail = () => {
    setSiteDetailVisible(false);
    setSiteDetail(null);
  };

  const handleModalOk = async (payload) => {
    const ok = editingSite
      ? await updateSite(editingSite.id, payload)
      : await createSite(payload);
    if (ok) setModalVisible(false);
  };

  const handleModalCancel = () => {
    if (savingSite) return;
    setModalVisible(false);
  };

  const applyFilters = () => {
    setAccountQuery((current) => ({
      ...current,
      page: 1,
      keyword: keywordDraft.trim(),
    }));
  };

  const resetFilters = () => {
    setKeywordDraft('');
    setAccountQuery(DEFAULT_ACCOUNT_QUERY);
  };

  const groupOptions = useMemo(
    () => buildCRSGroupOptions(sites, editingSite?.group),
    [editingSite?.group, sites],
  );

  const siteOptions = useMemo(
    () =>
      sites.map((site) => ({
        label: site.name || site.host,
        value: site.id,
      })),
    [sites],
  );

  const platformOptions = useMemo(() => {
    const map = new Map();
    [...KNOWN_PLATFORM_OPTIONS, ...getCRSPlatformOptions(accounts)].forEach(
      (item) => {
        map.set(item.value, item);
      },
    );
    return [{ label: tFn('全部平台'), value: '' }, ...Array.from(map.values())];
  }, [accounts, tFn]);

  const statusOptions = useMemo(() => {
    const values = Array.from(
      new Set(
        accounts
          .map((account) => String(account.status || '').trim())
          .filter(Boolean),
      ),
    ).sort((left, right) => left.localeCompare(right, 'en'));
    return values.map((value) => ({ label: value, value }));
  }, [accounts]);

  const latestSyncAt = useMemo(() => getCRSLatestSyncAt(sites), [sites]);
  const siteMap = useMemo(
    () => new Map(sites.map((site) => [site.id, site])),
    [sites],
  );

  const activeFilterTags = useMemo(() => {
    const tags = [];
    if (accountQuery.site_id > 0) {
      const matchedSite = siteMap.get(accountQuery.site_id);
      tags.push({
        key: 'site_id',
        label: tFn('站点'),
        value:
          matchedSite?.name || matchedSite?.host || `#${accountQuery.site_id}`,
      });
    }
    if (accountQuery.platform) {
      tags.push({
        key: 'platform',
        label: tFn('平台'),
        value: accountQuery.platform,
      });
    }
    if (accountQuery.status) {
      tags.push({
        key: 'status',
        label: tFn('状态'),
        value: accountQuery.status,
      });
    }
    if (accountQuery.quota_state) {
      const quotaLabel = QUOTA_STATE_OPTIONS.find(
        (item) => item.value === accountQuery.quota_state,
      )?.label;
      tags.push({
        key: 'quota_state',
        label: tFn('额度'),
        value: tFn(quotaLabel || accountQuery.quota_state),
      });
    }
    if (accountQuery.keyword) {
      tags.push({
        key: 'keyword',
        label: tFn('关键词'),
        value: accountQuery.keyword,
      });
    }
    return tags;
  }, [
    accountQuery.keyword,
    accountQuery.platform,
    accountQuery.quota_state,
    accountQuery.site_id,
    accountQuery.status,
    siteMap,
    tFn,
  ]);

  const ov = aggregate?.overview ?? {};
  const ra = aggregate?.recentActivity ?? {};
  const totalHealth =
    ov.totalAccounts > 0
      ? Math.round((ov.normalAccounts / ov.totalAccounts) * 100)
      : null;

  const statCards = [
    {
      icon: <Server size={15} />,
      title: tFn('CRS 站点'),
      main: observer?.total_sites ?? sites.length,
      sub: `${tFn('已同步')} ${observer?.synced_sites ?? 0} · ${tFn('错误')} ${observer?.error_sites ?? 0}`,
      accent: 'text-sky-500',
    },
    {
      icon: <Users size={15} />,
      title: tFn('观察账号'),
      main: observer?.total_accounts ?? 0,
      sub: `${tFn('活跃')} ${observer?.active_accounts ?? 0}`,
      accent: 'text-emerald-500',
    },
    {
      icon: <Gauge size={15} />,
      title: tFn('可调度'),
      main: observer?.schedulable_count ?? 0,
      sub: `${tFn('限速')} ${observer?.rate_limited_count ?? 0}`,
      accent: 'text-blue-500',
    },
    {
      icon: <ShieldAlert size={15} />,
      title: tFn('低额度'),
      main: observer?.low_quota_count ?? 0,
      sub: `${tFn('空额度')} ${observer?.empty_quota_count ?? 0}`,
      accent: 'text-amber-500',
    },
    {
      icon: <Activity size={15} />,
      title: tFn('今日请求'),
      main: formatBigNumber(ra.requestsToday ?? 0),
      sub:
        ra.tokensToday != null
          ? `Token ${formatBigNumber(ra.tokensToday)}`
          : undefined,
      accent: 'text-teal-500',
    },
    {
      icon:
        totalHealth != null && totalHealth >= 90 ? (
          <CheckCircle2 size={15} />
        ) : (
          <ShieldAlert size={15} />
        ),
      title: tFn('Dashboard 健康'),
      main:
        totalHealth != null ? (
          <span
            className={
              totalHealth >= 90
                ? 'text-emerald-600 dark:text-emerald-400'
                : totalHealth >= 70
                  ? 'text-amber-500'
                  : 'text-red-500'
            }
          >
            {totalHealth}%
          </span>
        ) : (
          '-'
        ),
      sub: `${tFn('总账号')} ${ov.totalAccounts ?? 0}`,
      accent:
        totalHealth != null && totalHealth >= 90
          ? 'text-emerald-500'
          : 'text-red-500',
    },
  ];

  const accountColumns = useMemo(
    () => [
      {
        title: tFn('站点'),
        dataIndex: 'site_name',
        width: 180,
        render: (_, record) => (
          <div className='min-w-0'>
            <div className='font-medium text-semi-color-text-0'>
              {record.site_name || '-'}
            </div>
            <div className='mt-1 text-xs text-semi-color-text-2'>
              #{record.site_id}
            </div>
          </div>
        ),
      },
      {
        title: tFn('平台'),
        dataIndex: 'platform',
        width: 130,
        render: (value) => (
          <Tag color='cyan' size='small'>
            {value || '-'}
          </Tag>
        ),
      },
      {
        title: tFn('账号'),
        dataIndex: 'name',
        render: (_, record) => (
          <div className='min-w-0'>
            <div className='font-medium text-semi-color-text-0'>
              {record.name || record.remote_account_id}
            </div>
            <div className='mt-1 break-all text-xs text-semi-color-text-2'>
              {record.remote_account_id}
            </div>
            {record.subscription_plan ? (
              <div className='mt-1 text-xs text-semi-color-text-2'>
                {tFn('计划')}: {record.subscription_plan}
              </div>
            ) : null}
          </div>
        ),
      },
      {
        title: tFn('状态'),
        dataIndex: 'status',
        width: 220,
        render: (_, record) => renderAccountStatus(record, tFn),
      },
      {
        title: tFn('额度'),
        dataIndex: 'quota_remaining',
        width: 160,
        render: (_, record) => {
          const quota = formatQuotaSummary(record, tFn);
          return (
            <div className='space-y-1'>
              <Tag color={quota.color} size='small'>
                {quota.label}
              </Tag>
              <div className='text-xs text-semi-color-text-1'>
                {quota.detail}
              </div>
            </div>
          );
        },
      },
      {
        title: tFn('最近同步'),
        dataIndex: 'last_synced_at',
        width: 170,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: tFn('操作'),
        dataIndex: 'actions',
        width: 120,
        fixed: 'right',
        render: (_, record) => (
          <Button
            type='tertiary'
            size='small'
            onClick={() =>
              openSiteDetail({ id: record.site_id, name: record.site_name })
            }
          >
            {tFn('站点详情')}
          </Button>
        ),
      },
    ],
    [openSiteDetail, tFn],
  );

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div>
          <Title heading={5} className='!mb-0'>
            {tFn('CRS 账号概览')}
          </Title>
          <Text type='tertiary' size='small'>
            {tFn('先看统一账号面，再按站点下钻排查额度、限速和异常')}
          </Text>
        </div>
        <div className='flex items-center gap-2'>
          <Button
            icon={<RefreshCw size={14} />}
            loading={loadingOverview}
            onClick={loadOverview}
            size='small'
            theme='borderless'
          >
            {tFn('刷新概览')}
          </Button>
          <Button
            icon={<RefreshCw size={14} />}
            loading={refreshingAll}
            onClick={refreshAll}
            size='small'
          >
            {tFn('刷新全部 CRS')}
          </Button>
          <Button
            icon={<Plus size={14} />}
            type='primary'
            size='small'
            onClick={openCreate}
          >
            {tFn('新增站点')}
          </Button>
        </div>
      </div>

      <div className='flex flex-wrap items-center gap-2 rounded-xl border border-gray-200 bg-gradient-to-r from-sky-50 via-white to-white px-3 py-2 dark:border-gray-700 dark:from-sky-950/20 dark:via-gray-900 dark:to-gray-900'>
        <HintPill
          label={tFn('模式')}
          value={tFn('只读观察')}
          tone='text-sky-600 dark:text-sky-400'
        />
        <HintPill
          label={tFn('最近同步')}
          value={
            latestSyncAt ? timestamp2string(latestSyncAt) : tFn('尚未同步')
          }
        />
        <HintPill
          label={tFn('限速中')}
          value={observer?.rate_limited_count ?? 0}
          tone='text-orange-500'
        />
        <HintPill
          label={tFn('低额度')}
          value={observer?.low_quota_count ?? 0}
          tone='text-amber-500'
        />
      </div>

      <Spin spinning={loadingOverview && !aggregate && !observer}>
        <div className='grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6'>
          {statCards.map((card) => (
            <StatCard key={card.title} {...card} />
          ))}
        </div>
      </Spin>

      <Card
        bordered={false}
        className='rounded-xl'
        title={
          <span className='text-sm font-semibold'>{tFn('跨站点账号总览')}</span>
        }
        headerExtraContent={
          <Text type='tertiary' size='small'>
            {tFn('共 {{count}} 个账号', { count: accountsTotal })}
          </Text>
        }
      >
        <div className='space-y-3'>
          <Text type='tertiary' size='small'>
            {tFn(
              '这里是统一筛选入口；刷新会重新拉取远端 CRS，再更新本地观察快照。',
            )}
          </Text>
          <div className='grid gap-2 lg:grid-cols-[minmax(0,1.2fr),repeat(4,minmax(0,180px)),auto,auto]'>
            <Input
              prefix={<Search size={14} />}
              placeholder={tFn('搜索站点名、账号名、远端 ID')}
              value={keywordDraft}
              onChange={setKeywordDraft}
              onEnterPress={applyFilters}
              showClear
            />
            <Select
              value={accountQuery.site_id || ''}
              optionList={[
                { label: tFn('全部站点'), value: '' },
                ...siteOptions,
              ]}
              onChange={(value) =>
                setAccountQuery((current) => ({
                  ...current,
                  page: 1,
                  site_id: Number(value || 0),
                }))
              }
              placeholder={tFn('站点')}
              showClear
            />
            <Select
              value={accountQuery.platform}
              optionList={platformOptions}
              onChange={(value) =>
                setAccountQuery((current) => ({
                  ...current,
                  page: 1,
                  platform: value || '',
                }))
              }
              placeholder={tFn('平台')}
              showClear
            />
            <Select
              value={accountQuery.status}
              optionList={statusOptions}
              allowCreate
              filter
              onChange={(value) =>
                setAccountQuery((current) => ({
                  ...current,
                  page: 1,
                  status: value || '',
                }))
              }
              placeholder={tFn('状态')}
              showClear
            />
            <Select
              value={accountQuery.quota_state}
              optionList={QUOTA_STATE_OPTIONS}
              onChange={(value) =>
                setAccountQuery((current) => ({
                  ...current,
                  page: 1,
                  quota_state: value || '',
                }))
              }
              placeholder={tFn('额度状态')}
            />
            <Button onClick={applyFilters}>{tFn('搜索')}</Button>
            <Button theme='borderless' onClick={resetFilters}>
              {tFn('重置')}
            </Button>
          </div>

          {activeFilterTags.length > 0 ? (
            <div className='flex flex-wrap gap-2'>
              {activeFilterTags.map((item) => (
                <Tag
                  key={item.key}
                  closable
                  color='blue'
                  onClose={() => {
                    setAccountQuery((current) => ({
                      ...current,
                      page: 1,
                      [item.key]: item.key === 'site_id' ? 0 : '',
                    }));
                    if (item.key === 'keyword') {
                      setKeywordDraft('');
                    }
                  }}
                >
                  {item.label}: {item.value}
                </Tag>
              ))}
            </div>
          ) : null}

          <Table
            dataSource={accounts}
            columns={accountColumns}
            rowKey={(record) =>
              record.id || `${record.site_id}-${record.remote_account_id}`
            }
            loading={loadingAccounts}
            size='small'
            scroll={{ x: 1180 }}
            empty={
              <Empty
                image={null}
                description={
                  accountsTotal > 0
                    ? tFn('当前页没有数据')
                    : tFn('暂无跨站点账号快照')
                }
              />
            }
            pagination={{
              currentPage: accountQuery.page,
              pageSize: accountQuery.page_size,
              total: accountsTotal,
              pageSizeOpts: [20, 50, 100],
              onPageChange: (page) =>
                setAccountQuery((current) => ({
                  ...current,
                  page,
                })),
              onPageSizeChange: (pageSize) =>
                setAccountQuery((current) => ({
                  ...current,
                  page: 1,
                  page_size: pageSize,
                })),
            }}
          />
        </div>
      </Card>

      {ov.accountsByPlatform &&
      Object.keys(ov.accountsByPlatform).length > 0 ? (
        <Card
          bordered={false}
          title={
            <span className='text-sm font-semibold text-gray-700 dark:text-gray-300'>
              {tFn('远端 Dashboard 平台分布')}
            </span>
          }
          bodyStyle={{ padding: '0 8px 8px' }}
        >
          <PlatformTable accountsByPlatform={ov.accountsByPlatform} />
        </Card>
      ) : null}

      <div>
        <div className='mb-2 flex flex-wrap items-center gap-1.5 text-sm font-semibold text-gray-600 dark:text-gray-400'>
          <Server size={14} />
          {tFn('按站点下钻')}
          <Badge count={sites.length} overflowCount={99} className='ml-1' />
          <Text type='tertiary' size='small'>
            {tFn('适合逐站排查同步错误、额度耗尽和限速情况')}
          </Text>
        </div>

        {sites.length === 0 && !loadingOverview ? (
          <div className='flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-gray-300 py-12 text-gray-400 dark:border-gray-600'>
            <Server size={32} className='opacity-40' />
            <div className='text-sm'>
              {tFn('暂无 CRS 站点，点击“新增站点”开始配置')}
            </div>
            <Button
              icon={<Plus size={14} />}
              type='primary'
              onClick={openCreate}
            >
              {tFn('新增站点')}
            </Button>
          </div>
        ) : (
          <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3'>
            {sites.map((site) => (
              <SiteCard
                key={site.id}
                site={site}
                onDetail={openSiteDetail}
                onRefresh={refreshSite}
                onEdit={openEdit}
                onDelete={deleteSite}
                refreshingSiteId={refreshingSiteId}
                deletingSiteId={deletingSiteId}
                t={tFn}
              />
            ))}
          </div>
        )}
      </div>

      <CRSSiteModal
        visible={modalVisible}
        site={editingSite}
        onOk={handleModalOk}
        onCancel={handleModalCancel}
        saving={savingSite}
        groupOptions={groupOptions}
      />

      <CRSSiteDetailSideSheet
        visible={siteDetailVisible}
        onClose={closeSiteDetail}
        detail={siteDetail}
        loading={loadingSiteDetail}
        onRefresh={refreshSite}
        refreshing={refreshingSiteId === siteDetail?.site?.id}
        t={tFn}
      />
    </div>
  );
}
