import { useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Card,
  Popconfirm,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  Activity,
  CheckCircle2,
  Key,
  Plus,
  RefreshCw,
  Server,
  ShieldAlert,
  Trash2,
  Users,
  Zap,
  Pencil,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { timestamp2string } from '../../../helpers/date';
import { useCRSData } from '../hooks/useCRSData';
import CRSSiteModal from './CRSSiteModal';

const { Title, Text } = Typography;

// ---------- 工具函数 ----------

function formatBigNumber(n) {
  if (n == null || isNaN(n)) return '–';
  if (n >= 1e8) return `${(n / 1e8).toFixed(2)}亿`;
  if (n >= 1e4) return `${(n / 1e4).toFixed(1)}万`;
  return String(n);
}

function formatUptime(seconds) {
  if (!seconds) return '–';
  const h = Math.floor(seconds / 3600);
  if (h < 24) return `${h}h`;
  return `${Math.floor(h / 24)}d ${h % 24}h`;
}

function healthPercent(stat) {
  if (!stat || stat.total === 0) return null;
  return Math.round((stat.normal / stat.total) * 100);
}

// ---------- 概览统计卡片 ----------

function StatCard({ icon, title, main, sub, accent }) {
  return (
    <div
      className={
        'rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4 flex flex-col gap-1 shadow-sm'
      }
    >
      <div className='flex items-center gap-2 text-gray-500 dark:text-gray-400 text-xs font-medium uppercase tracking-wide'>
        <span className={accent}>{icon}</span>
        {title}
      </div>
      <div className='text-2xl font-bold text-gray-900 dark:text-gray-50 leading-tight'>
        {main}
      </div>
      {sub && (
        <div className='text-xs text-gray-400 dark:text-gray-500'>{sub}</div>
      )}
    </div>
  );
}

// ---------- 平台分布表格 ----------

function PlatformTable({ accountsByPlatform }) {
  const { t } = useTranslation();

  const rows = useMemo(() => {
    if (!accountsByPlatform) return [];
    return Object.entries(accountsByPlatform)
      .filter(([, s]) => s.total > 0)
      .map(([platform, s]) => ({
        platform,
        total: s.total,
        normal: s.normal,
        abnormal: s.abnormal,
        paused: s.paused,
        rateLimited: s.rateLimited,
        health: s.total > 0 ? Math.round((s.normal / s.total) * 100) : 0,
      }))
      .sort((a, b) => b.total - a.total);
  }, [accountsByPlatform]);

  if (!rows.length) return null;

  const columns = [
    {
      title: t('平台'),
      dataIndex: 'platform',
      render: (v) => <span className='font-medium capitalize'>{v}</span>,
    },
    { title: t('总账号'), dataIndex: 'total', align: 'right' },
    {
      title: t('正常'),
      dataIndex: 'normal',
      align: 'right',
      render: (v) => (
        <span className='font-semibold text-emerald-600 dark:text-emerald-400'>
          {v}
        </span>
      ),
    },
    {
      title: t('异常'),
      dataIndex: 'abnormal',
      align: 'right',
      render: (v) => (
        <span className={v > 0 ? 'font-semibold text-red-500 dark:text-red-400' : 'text-gray-400'}>
          {v}
        </span>
      ),
    },
    {
      title: t('暂停'),
      dataIndex: 'paused',
      align: 'right',
      render: (v) => (
        <span className={v > 0 ? 'text-amber-500 dark:text-amber-400' : 'text-gray-400'}>
          {v}
        </span>
      ),
    },
    {
      title: t('限速'),
      dataIndex: 'rateLimited',
      align: 'right',
      render: (v) => (
        <span className={v > 0 ? 'text-orange-500 dark:text-orange-400' : 'text-gray-400'}>
          {v}
        </span>
      ),
    },
    {
      title: t('健康度'),
      dataIndex: 'health',
      align: 'right',
      render: (v) => {
        const color =
          v >= 90
            ? 'text-emerald-600 dark:text-emerald-400'
            : v >= 70
              ? 'text-amber-500'
              : 'text-red-500';
        return <span className={`font-bold ${color}`}>{v}%</span>;
      },
    },
  ];

  return (
    <Table
      dataSource={rows}
      columns={columns}
      rowKey='platform'
      size='small'
      pagination={false}
      bordered
    />
  );
}

// ---------- 单个站点卡片 ----------

function SiteCard({
  site,
  onRefresh,
  onEdit,
  onDelete,
  refreshingSiteId,
  deletingSiteId,
  t,
}) {
  const isRefreshing = refreshingSiteId === site.id;
  const isDeleting = deletingSiteId === site.id;

  const statusTag = useMemo(() => {
    if (site.status === 1)
      return (
        <Tag color='green' size='small'>
          {t('已同步')}
        </Tag>
      );
    if (site.status === 2)
      return (
        <Tag color='red' size='small'>
          {t('错误')}
        </Tag>
      );
    return (
      <Tag color='grey' size='small'>
        {t('未同步')}
      </Tag>
    );
  }, [site.status, t]);

  const hp = site.dashboard?.overview
    ? healthPercent({
        total: site.dashboard.overview.totalAccounts,
        normal: site.dashboard.overview.normalAccounts,
      })
    : null;

  return (
    <div className='rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4 shadow-sm flex flex-col gap-3'>
      {/* 站点标题行 */}
      <div className='flex items-start justify-between gap-2'>
        <div className='flex-1 min-w-0'>
          <div className='flex items-center gap-2 flex-wrap'>
            <Server size={15} className='text-sky-500 flex-shrink-0' />
            <span className='font-semibold text-sm text-gray-800 dark:text-gray-200 break-all'>
              {site.name || site.host}
            </span>
            {statusTag}
          </div>
          <div className='text-xs text-gray-500 mt-0.5 ml-5 space-x-2'>
            {site.group && <span className='bg-sky-50 dark:bg-sky-900/30 text-sky-600 dark:text-sky-300 px-1.5 py-0.5 rounded font-medium'>{site.group}</span>}
            <span className='opacity-70'>{site.scheme}://{site.host}</span>
          </div>
        </div>
        <div className='flex gap-1 flex-shrink-0'>
          <Tooltip content={t('刷新')}>
            <Button
              theme='borderless'
              size='small'
              icon={<RefreshCw size={14} />}
              loading={isRefreshing}
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
          <Popconfirm
            title={t('确认删除该 CRS 站点？')}
            onConfirm={() => onDelete(site.id)}
            okText={t('删除')}
            cancelText={t('取消')}
            okButtonProps={{ type: 'danger' }}
          >
            <Tooltip content={t('删除')}>
              <Button
                theme='borderless'
                size='small'
                icon={<Trash2 size={14} />}
                loading={isDeleting}
                type='danger'
              />
            </Tooltip>
          </Popconfirm>
        </div>
      </div>

      {/* 关键数据行 */}
      {site.dashboard && (
        <div className='grid grid-cols-3 gap-2 text-center'>
          <div className='bg-gray-50 dark:bg-gray-800 rounded-lg py-1.5 px-2'>
            <div className='text-lg font-bold text-gray-800 dark:text-gray-100'>
              {site.dashboard.overview?.totalAccounts ?? '–'}
            </div>
            <div className='text-xs text-gray-500'>{t('账号')}</div>
          </div>
          <div className='bg-gray-50 dark:bg-gray-800 rounded-lg py-1.5 px-2'>
            <div className='text-lg font-bold text-emerald-600 dark:text-emerald-400'>
              {site.dashboard.overview?.normalAccounts ?? '–'}
            </div>
            <div className='text-xs text-gray-500'>{t('正常')}</div>
          </div>
          <div className='bg-gray-50 dark:bg-gray-800 rounded-lg py-1.5 px-2'>
            <div
              className={`text-lg font-bold ${
                hp != null && hp < 90
                  ? 'text-amber-500'
                  : 'text-sky-600 dark:text-sky-400'
              }`}
            >
              {hp != null ? `${hp}%` : '–'}
            </div>
            <div className='text-xs text-gray-500'>{t('健康度')}</div>
          </div>
        </div>
      )}

      {/* 最后同步时间 */}
      {site.last_synced_at > 0 && (
        <div className='text-xs text-gray-400'>
          {t('最近同步')}: {timestamp2string(site.last_synced_at)}
        </div>
      )}

      {/* 错误提示 */}
      {site.status === 2 && site.last_sync_error && (
        <div className='text-xs text-red-500 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded p-2 break-all'>
          {site.last_sync_error}
        </div>
      )}
    </div>
  );
}

// ---------- 主组件 ----------

export default function CRSDashboardCard({ t: tProp }) {
  const { t } = useTranslation();
  const tFn = tProp ?? t;

  const {
    sites,
    aggregate,
    loadingOverview,
    refreshingAll,
    refreshingSiteId,
    savingSite,
    deletingSiteId,
    loadOverview,
    refreshSite,
    refreshAll,
    createSite,
    updateSite,
    deleteSite,
  } = useCRSData();

  const [modalVisible, setModalVisible] = useState(false);
  const [editingSite, setEditingSite] = useState(null);

  const openCreate = () => {
    setEditingSite(null);
    setModalVisible(true);
  };

  const openEdit = (site) => {
    setEditingSite(site);
    setModalVisible(true);
  };

  const handleModalOk = async (payload) => {
    let ok;
    if (editingSite) {
      ok = await updateSite(editingSite.id, payload);
    } else {
      ok = await createSite(payload);
    }
    if (ok) setModalVisible(false);
  };

  const handleModalCancel = () => {
    if (savingSite) return;
    setModalVisible(false);
  };

  // 汇总 overview 数据
  const ov = aggregate?.overview ?? {};
  const ra = aggregate?.recentActivity ?? {};
  const rt = aggregate?.realtimeMetrics ?? {};

  const totalAccounts = ov.totalAccounts ?? 0;
  const normalAccounts = ov.normalAccounts ?? 0;
  const abnormalAccounts = ov.abnormalAccounts ?? 0;
  const totalHealth =
    totalAccounts > 0 ? Math.round((normalAccounts / totalAccounts) * 100) : null;

  const statCards = [
    {
      icon: <Users size={15} />,
      title: tFn('总账号数'),
      main: totalAccounts,
      sub: normalAccounts
        ? `${tFn('正常')} ${normalAccounts} · ${tFn('异常')} ${abnormalAccounts}`
        : undefined,
      accent: 'text-sky-500',
    },
    {
      icon: <Key size={15} />,
      title: 'API Keys',
      main: ov.totalApiKeys ?? 0,
      sub: ov.activeApiKeys != null ? `${tFn('活跃')} ${ov.activeApiKeys}` : undefined,
      accent: 'text-violet-500',
    },
    {
      icon: <Zap size={15} />,
      title: tFn('实时指标'),
      main:
        rt.rpm != null
          ? `${Number(rt.rpm).toFixed(1)} RPM`
          : '–',
      sub:
        rt.windowMinutes
          ? `${tFn('窗口')} ${rt.windowMinutes}min`
          : undefined,
      accent: 'text-amber-500',
    },
    {
      icon: <Activity size={15} />,
      title: tFn('今日请求'),
      main: formatBigNumber(ra.requestsToday ?? 0),
      sub: ra.tokensToday != null
        ? `Token: ${formatBigNumber(ra.tokensToday)}`
        : undefined,
      accent: 'text-emerald-500',
    },
    {
      icon: <Server size={15} />,
      title: tFn('累计 Token'),
      main: formatBigNumber(ov.totalTokensUsed ?? 0),
      sub:
        ov.totalInputTokensUsed != null
          ? `${tFn('输入')} ${formatBigNumber(ov.totalInputTokensUsed)} · ${tFn('输出')} ${formatBigNumber(ov.totalOutputTokensUsed)}`
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
      title: tFn('整体健康度'),
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
          '–'
        ),
      sub: sites.length ? `${sites.length} ${tFn('个站点')}` : undefined,
      accent:
        totalHealth == null || totalHealth >= 90
          ? 'text-emerald-500'
          : 'text-red-500',
    },
  ];

  return (
    <div className='space-y-5'>
      {/* 顶部操作栏 */}
      <div className='flex items-center justify-between flex-wrap gap-2'>
        <div>
          <Title heading={5} className='!mb-0'>
            {tFn('CRS 账号概览')}
          </Title>
          <Text type='tertiary' size='small'>
            {tFn('Claude Relay Service 站点账号状态汇总')}
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

      {/* 统计卡片 */}
      <Spin spinning={loadingOverview && !aggregate}>
        <div className='grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3'>
          {statCards.map((card, i) => (
            <StatCard key={i} {...card} />
          ))}
        </div>
      </Spin>

      {/* 平台分布 */}
      {ov.accountsByPlatform &&
        Object.keys(ov.accountsByPlatform).length > 0 && (
          <Card
            bordered={false}
            title={
              <span className='text-sm font-semibold text-gray-700 dark:text-gray-300'>
                {tFn('平台账号分布')}
              </span>
            }
            bodyStyle={{ padding: '0 8px 8px' }}
          >
            <PlatformTable accountsByPlatform={ov.accountsByPlatform} />
          </Card>
        )}

      {/* 站点列表 */}
      <div>
        <div className='text-sm font-semibold text-gray-600 dark:text-gray-400 mb-2 flex items-center gap-1.5'>
          <Server size={14} />
          {tFn('站点列表')}
          <Badge count={sites.length} overflowCount={99} className='ml-1' />
        </div>

        {sites.length === 0 && !loadingOverview ? (
          <div className='rounded-xl border border-dashed border-gray-300 dark:border-gray-600 flex flex-col items-center justify-center py-12 gap-3 text-gray-400'>
            <Server size={32} className='opacity-40' />
            <div className='text-sm'>{tFn('暂无 CRS 站点，点击"新增站点"开始配置')}</div>
            <Button
              icon={<Plus size={14} />}
              type='primary'
              onClick={openCreate}
            >
              {tFn('新增站点')}
            </Button>
          </div>
        ) : (
          <div className='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3'>
            {sites.map((site) => (
              <SiteCard
                key={site.id}
                site={site}
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

      {/* 新增/编辑弹窗 */}
      <CRSSiteModal
        visible={modalVisible}
        site={editingSite}
        onOk={handleModalOk}
        onCancel={handleModalCancel}
        saving={savingSite}
      />
    </div>
  );
}
