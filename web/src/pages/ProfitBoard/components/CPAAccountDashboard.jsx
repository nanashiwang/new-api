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
  CircleAlert,
  Clock3,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  ServerCog,
  ShieldCheck,
  Trash2,
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
  available: { color: 'green', label: '可用' },
  limited: { color: 'orange', label: '冷却 / 限速' },
  disabled: { color: 'grey', label: '已停用' },
  abnormal: { color: 'red', label: '异常' },
  unknown: { color: 'light-blue', label: '未知' },
};

const formatRemoteTime = (value, fallback = '-') => {
  const timestamp = Date.parse(value ?? '');
  return Number.isFinite(timestamp)
    ? timestamp2string(Math.floor(timestamp / 1000))
    : fallback;
};

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

function SummaryCard({ label, value, hint, icon: Icon, tone }) {
  const tones = {
    blue: 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300',
    green:
      'bg-green-50 text-green-600 dark:bg-green-900/30 dark:text-green-300',
    orange:
      'bg-orange-50 text-orange-600 dark:bg-orange-900/30 dark:text-orange-300',
    red: 'bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-300',
  };
  return (
    <Card bordered bodyStyle={{ padding: 14 }}>
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

function CPASiteCard({
  site,
  t,
  onEdit,
  onRefresh,
  onDelete,
  refreshing,
  deleting,
}) {
  const status =
    site.status === 1
      ? { color: 'green', label: '已同步' }
      : site.status === 2
        ? { color: 'red', label: '同步失败' }
        : { color: 'grey', label: '尚未同步' };
  return (
    <Card bordered bodyStyle={{ padding: 14 }}>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0 flex-1'>
          <div className='flex flex-wrap items-center gap-2'>
            <Text strong>{site.name || site.host}</Text>
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
        <div className='flex shrink-0 gap-1'>
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
      <div className='mt-3 grid grid-cols-3 gap-2 sm:grid-cols-6'>
        {[
          ['账号', site.account_count ?? 0, undefined],
          ['可用', site.available_count ?? 0, 'success'],
          ['冷却', site.limited_count ?? 0, 'warning'],
          ['异常', site.abnormal_count ?? 0, 'danger'],
          ['已停用', site.disabled_count ?? 0, 'tertiary'],
          ['未知', site.unknown_count ?? 0, 'tertiary'],
        ].map(([label, value, type]) => (
          <div
            key={label}
            className='rounded-lg border border-solid border-[var(--semi-color-border)] px-2 py-2 text-center'
          >
            <Text type='tertiary' size='small'>
              {t(label)}
            </Text>
            <div className='mt-0.5 font-semibold tabular-nums'>
              <Text type={type}>{value}</Text>
            </div>
          </div>
        ))}
      </div>
      {site.last_synced_at > 0 ? (
        <Text type='tertiary' size='small' className='mt-3 block'>
          {t('最近同步')}: {timestamp2string(site.last_synced_at)}
        </Text>
      ) : null}
      {site.last_sync_error ? (
        <Text type='danger' size='small' className='mt-2 block break-all'>
          {site.last_sync_error}
        </Text>
      ) : null}
    </Card>
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

  const summary = useMemo(() => summarizeCPAAccounts(accounts), [accounts]);
  const filteredAccounts = useMemo(
    () => filterCPAAccounts(accounts, { keyword, siteId, state }),
    [accounts, keyword, siteId, state],
  );
  const latestSyncAt = useMemo(
    () => Math.max(0, ...sites.map((site) => Number(site.last_synced_at || 0))),
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
    { title: tFn('来源 / 服务'), dataIndex: 'site_name' },
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
              <Text type='tertiary' size='small' className='mt-2 block'>
                {tFn('最近同步')}:{' '}
                {latestSyncAt
                  ? timestamp2string(latestSyncAt)
                  : tFn('尚未同步')}
              </Text>
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
          />
          <SummaryCard
            label={tFn('可用账号')}
            value={summary.available}
            icon={ShieldCheck}
            tone='green'
          />
          <SummaryCard
            label={tFn('冷却 / 限速')}
            value={summary.limited}
            hint={`${tFn('已停用')} ${summary.disabled}`}
            icon={Clock3}
            tone='orange'
          />
          <SummaryCard
            label={tFn('异常账号')}
            value={summary.abnormal}
            hint={`${tFn('未知')} ${summary.unknown}`}
            icon={CircleAlert}
            tone='red'
          />
        </div>
      </Spin>

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
          <div className='grid grid-cols-1 gap-3 xl:grid-cols-2'>
            {sites.map((site) => (
              <CPASiteCard
                key={site.id}
                site={site}
                t={tFn}
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

      <Card bordered bodyStyle={{ padding: 16 }}>
        <div className='mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
          <div>
            <Title heading={5}>{tFn('CPA 账号使用情况')}</Title>
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
        <div className='mb-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-[minmax(240px,1.5fr)_minmax(160px,0.8fr)_minmax(160px,0.8fr)]'>
          <Input
            prefix={<Search size={15} />}
            value={keyword}
            onChange={setKeyword}
            placeholder={tFn('搜索账号、平台或服务')}
            showClear
          />
          <Select
            value={siteId}
            onChange={setSiteId}
            className='w-full'
            optionList={[
              { value: 0, label: tFn('全部服务') },
              ...sites.map((site) => ({
                value: site.id,
                label: site.name || site.host,
              })),
            ]}
          />
          <Select
            value={state}
            onChange={setState}
            className='w-full'
            optionList={[
              { value: '', label: tFn('全部状态') },
              ...Object.entries(STATE_META).map(([value, meta]) => ({
                value,
                label: tFn(meta.label),
              })),
            ]}
          />
        </div>

        {filteredAccounts.length === 0 ? (
          <Empty
            title={tFn(
              sites.length === 0 ? '尚未连接 CPA 服务' : '暂无 CPA 账号数据',
            )}
            description={tFn(
              sites.length === 0
                ? '请先使用“人工接入 CPA”添加服务。'
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
