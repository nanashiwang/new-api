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
  Badge,
  Button,
  Card,
  Modal,
  Progress,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  Eye,
  Pencil,
  Plus,
  RefreshCw,
  Server,
  Trash2,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { timestamp2string } from '../../../helpers/date';
import { useCRSData } from '../hooks/useCRSData';
import CRSSiteDetailSideSheet from './CRSSiteDetailSideSheet';
import CRSSiteModal from './CRSSiteModal';
import { buildCRSGroupOptions, getCRSLatestSyncAt } from './crsDashboard.utils';

const { Title, Text } = Typography;

function formatBigNumber(n) {
  if (n == null || Number.isNaN(Number(n))) return '-';
  const num = Number(n);
  if (num >= 1e8) return `${(num / 1e8).toFixed(2)}亿`;
  if (num >= 1e4) return `${(num / 1e4).toFixed(1)}万`;
  return String(n);
}

function healthPercent(stat) {
  if (!stat || stat.total === 0) return null;
  return Math.round((stat.normal / stat.total) * 100);
}

function healthTone(value) {
  if (value == null) return { textType: 'tertiary', stroke: 'grey' };
  if (value >= 90) return { textType: 'success', stroke: 'green' };
  if (value >= 30) return { textType: 'warning', stroke: 'amber' };
  return { textType: 'danger', stroke: 'red' };
}

function StatBlock({ label, value, hint, tone = 'default' }) {
  const valueType =
    tone === 'danger'
      ? 'danger'
      : tone === 'warning'
        ? 'warning'
        : tone === 'primary'
          ? 'primary'
          : tone === 'success'
            ? 'success'
            : undefined;
  return (
    <Card bordered bodyStyle={{ padding: 12 }}>
      <Text type='tertiary' size='small'>
        {label}
      </Text>
      <div className='mt-1 text-2xl font-semibold tabular-nums leading-tight'>
        <Text type={valueType}>{value}</Text>
      </div>
      {hint ? (
        <Text type='tertiary' size='small' className='mt-1 block'>
          {hint}
        </Text>
      ) : null}
    </Card>
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

  const renderCount = (value, valueType) => (
    <Text type={value > 0 ? valueType : 'tertiary'}>{value}</Text>
  );

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
          render: (value) => renderCount(value, 'success'),
        },
        {
          title: t('异常'),
          dataIndex: 'abnormal',
          align: 'right',
          render: (value) => renderCount(value, 'danger'),
        },
        {
          title: t('暂停'),
          dataIndex: 'paused',
          align: 'right',
          render: (value) => renderCount(value, 'warning'),
        },
        {
          title: t('限速'),
          dataIndex: 'rateLimited',
          align: 'right',
          render: (value) => renderCount(value, 'danger'),
        },
        {
          title: t('健康度'),
          dataIndex: 'health',
          align: 'right',
          render: (value) => {
            const { textType } = healthTone(value);
            return (
              <Text type={textType} strong>
                {value}%
              </Text>
            );
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
  const healthColors = healthTone(health);

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
    <Card
      bordered
      shadows='hover'
      onClick={handleOpenDetail}
      bodyStyle={{ padding: 14 }}
      className='cursor-pointer transition-all hover:-translate-y-0.5'
    >
      <div
        role='button'
        tabIndex={0}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            handleOpenDetail();
          }
        }}
        className='flex flex-col gap-3'
      >
        <div className='flex items-start justify-between gap-2'>
          <div className='min-w-0 flex-1'>
            <div className='flex flex-wrap items-center gap-2'>
              <span className='truncate text-sm font-semibold'>
                {site.name || site.host}
              </span>
              {statusTag}
              {site.group ? (
                <Tag color='blue' size='small'>
                  {site.group}
                </Tag>
              ) : null}
            </div>
            <div className='mt-1'>
              <Text type='tertiary' size='small' className='break-all'>
                {site.scheme}://{site.host}
              </Text>
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
          <div className='rounded-lg border border-semi-color-border px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('观察账号')}
            </Text>
            <div className='mt-0.5 text-base font-semibold tabular-nums'>
              {site.account_count ?? 0}
            </div>
          </div>
          <div className='rounded-lg border border-semi-color-border px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('限速中')}
            </Text>
            <div className='mt-0.5 text-base font-semibold tabular-nums'>
              <Text
                type={(site.rate_limited_count ?? 0) > 0 ? 'danger' : undefined}
              >
                {site.rate_limited_count ?? 0}
              </Text>
            </div>
          </div>
          <div className='rounded-lg border border-semi-color-border px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('低额度')}
            </Text>
            <div className='mt-0.5 text-base font-semibold tabular-nums'>
              <Text
                type={(site.low_quota_count ?? 0) > 0 ? 'warning' : undefined}
              >
                {site.low_quota_count ?? 0}
              </Text>
            </div>
          </div>
          <div className='rounded-lg border border-semi-color-border px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('健康度')}
            </Text>
            <div className='mt-0.5 flex items-center gap-2'>
              <span className='text-base font-semibold tabular-nums'>
                <Text type={healthColors.textType}>
                  {health != null ? `${health}%` : '-'}
                </Text>
              </span>
              {health != null ? (
                <div className='flex-1 min-w-0'>
                  <Progress
                    percent={health}
                    stroke={healthColors.stroke}
                    showInfo={false}
                    size='small'
                  />
                </div>
              ) : null}
            </div>
          </div>
        </div>

        {site.last_synced_at > 0 ? (
          <Text type='tertiary' size='small'>
            {t('最近同步')}: {timestamp2string(site.last_synced_at)}
          </Text>
        ) : null}

        {site.status === 2 && site.last_sync_error ? (
          <Text type='danger' size='small' className='break-all'>
            {site.last_sync_error}
          </Text>
        ) : null}
      </div>
    </Card>
  );
}

export default function CRSDashboardCard({ t: tProp }) {
  const { t } = useTranslation();
  const tFn = tProp ?? t;
  const {
    sites,
    aggregate,
    observer,
    loadingOverview,
    refreshingAll,
    refreshingSiteId,
    savingSite,
    deletingSiteId,
    siteDetail,
    loadingSiteDetail,
    loadOverview,
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

  const siteMap = useMemo(
    () => new Map(sites.map((site) => [site.id, site])),
    [sites],
  );

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

  const groupOptions = useMemo(
    () => buildCRSGroupOptions(sites, editingSite?.group),
    [editingSite?.group, sites],
  );

  const latestSyncAt = useMemo(() => getCRSLatestSyncAt(sites), [sites]);

  const ov = aggregate?.overview ?? {};
  const ra = aggregate?.recentActivity ?? {};
  const totalHealth =
    ov.totalAccounts > 0
      ? Math.round((ov.normalAccounts / ov.totalAccounts) * 100)
      : null;
  const totalHealthColors = healthTone(totalHealth);

  const statBlocks = [
    {
      label: tFn('CRS 站点'),
      value: observer?.total_sites ?? sites.length,
      hint: `${tFn('已同步')} ${observer?.synced_sites ?? 0} · ${tFn('错误')} ${observer?.error_sites ?? 0}`,
    },
    {
      label: tFn('观察账号'),
      value: observer?.total_accounts ?? 0,
      hint: `${tFn('活跃')} ${observer?.active_accounts ?? 0} · ${tFn('可调度')} ${observer?.schedulable_count ?? 0}`,
    },
    {
      label: tFn('限速中'),
      value: observer?.rate_limited_count ?? 0,
      tone: (observer?.rate_limited_count ?? 0) > 0 ? 'danger' : 'default',
    },
    {
      label: tFn('低额度'),
      value: observer?.low_quota_count ?? 0,
      tone: (observer?.low_quota_count ?? 0) > 0 ? 'warning' : 'default',
      hint: `${tFn('空额度')} ${observer?.empty_quota_count ?? 0}`,
    },
    {
      label: tFn('今日请求'),
      value: formatBigNumber(ra.requestsToday ?? 0),
      hint:
        ra.tokensToday != null
          ? `Token ${formatBigNumber(ra.tokensToday)}`
          : undefined,
    },
    {
      label: tFn('Dashboard 健康'),
      value: totalHealth != null ? `${totalHealth}%` : '-',
      tone:
        totalHealth == null
          ? 'default'
          : totalHealth >= 90
            ? 'success'
            : totalHealth >= 70
              ? 'warning'
              : 'danger',
      hint: `${tFn('总账号')} ${ov.totalAccounts ?? 0}`,
    },
  ];

  return (
    <div className='flex flex-col gap-4'>
      <Card bordered bodyStyle={{ padding: 16 }}>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='min-w-0 flex-1'>
            <Title heading={5} style={{ margin: 0 }}>
              {tFn('CRS 账号概览')}
            </Title>
            <Text type='tertiary' size='small' className='mt-1 block'>
              {tFn('汇总所有 CRS 站点的账号状态,按站点查看额度、限速与异常')}
            </Text>
            <Text type='tertiary' size='small' className='mt-1 block'>
              {tFn('最近同步')}:{' '}
              {latestSyncAt ? timestamp2string(latestSyncAt) : tFn('尚未同步')}
            </Text>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
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
      </Card>

      <Spin spinning={loadingOverview && !aggregate && !observer}>
        <div className='grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6'>
          {statBlocks.map((block) => (
            <StatBlock key={block.label} {...block} />
          ))}
        </div>
      </Spin>

      {ov.accountsByPlatform &&
      Object.keys(ov.accountsByPlatform).length > 0 ? (
        <Card
          bordered
          title={tFn('远端 Dashboard 平台分布')}
          headerStyle={{ padding: '10px 16px' }}
          bodyStyle={{ padding: 12 }}
        >
          <PlatformTable accountsByPlatform={ov.accountsByPlatform} />
        </Card>
      ) : null}

      <Card
        bordered
        headerStyle={{ padding: '10px 16px' }}
        bodyStyle={{ padding: 12 }}
        title={
          <div className='flex items-center gap-2'>
            <Server size={14} className='text-semi-color-text-2' />
            <span className='text-sm font-semibold'>
              {tFn('CRS 站点列表')}
            </span>
            <Badge count={sites.length} overflowCount={99} />
          </div>
        }
      >
        {sites.length === 0 && !loadingOverview ? (
          <div className='flex flex-col items-center justify-center gap-3 py-12'>
            <Server size={32} className='opacity-30' />
            <Text type='tertiary' size='small'>
              {tFn('暂无 CRS 站点,点击"新增站点"开始配置')}
            </Text>
            <Button
              icon={<Plus size={14} />}
              type='primary'
              size='small'
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
      </Card>

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
