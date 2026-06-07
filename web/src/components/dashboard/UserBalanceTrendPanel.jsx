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

import React, {
  Suspense,
  lazy,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Search, Settings, TrendingUp, WalletCards } from 'lucide-react';
import { API } from '../../helpers/apiCore';
import { showError, showSuccess } from '../../helpers/toast';
import { renderNumber, renderQuota } from '../../helpers/dashboardFormat';

const { Text } = Typography;

const VChart = lazy(() =>
  import('@visactor/react-vchart').then((module) => ({
    default: module.VChart,
  })),
);

const formatPercent = (value) => `${(Number(value || 0) * 100).toFixed(1)}%`;

const getQuotaPerUnit = () => {
  const raw = parseFloat(localStorage.getItem('quota_per_unit') || '1');
  return Number.isFinite(raw) && raw > 0 ? raw : 1;
};

const getCurrencyConfig = () => {
  const type = localStorage.getItem('quota_display_type') || 'USD';
  const statusStr = localStorage.getItem('status');
  let symbol = '$';
  let rate = 1;

  if (type === 'CNY') {
    symbol = '¥';
    try {
      if (statusStr) {
        const status = JSON.parse(statusStr);
        rate = status?.usd_exchange_rate || 1;
      }
    } catch (e) {}
  } else if (type === 'CUSTOM') {
    symbol = '¤';
    try {
      if (statusStr) {
        const status = JSON.parse(statusStr);
        symbol = status?.custom_currency_symbol || symbol;
        rate = status?.custom_currency_exchange_rate || rate;
      }
    } catch (e) {}
  }

  return { type, symbol, rate };
};

const quotaToDisplayAmount = (quota) => {
  const value = Number(quota || 0);
  if (!Number.isFinite(value)) return 0;
  const { type, rate } = getCurrencyConfig();
  if (type === 'TOKENS') return value;
  return (value / getQuotaPerUnit()) * (type === 'USD' ? 1 : rate || 1);
};

const renderDisplayAmount = (amount, digits = 2) => {
  const value = Number(amount || 0);
  const { type, symbol } = getCurrencyConfig();
  if (type === 'TOKENS') {
    return renderNumber(value);
  }
  return `${symbol}${value.toFixed(digits)}`;
};

const formatSnapshotTime = (timestamp) => {
  if (!timestamp) return '-';
  const date = new Date(Number(timestamp) * 1000);
  if (Number.isNaN(date.getTime())) return '-';
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(
    2,
    '0',
  )}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

const buildMetricCards = (report, t) => {
  const latest = report?.current || report?.latest || {};
  const deltaQuota = Number(
    report?.current
      ? report?.current_delta_quota || 0
      : report?.delta_quota || 0,
  );
  return [
    {
      label: t('当前总余额'),
      value: renderQuota(latest.total_quota || 0, 2),
      color: 'blue',
    },
    {
      label: report?.current ? t('较 04:00 统计') : t('较上次统计'),
      value: renderQuota(deltaQuota, 2),
      color: deltaQuota >= 0 ? 'green' : 'red',
    },
    {
      label: t('有余额用户'),
      value: renderNumber(latest.positive_user_count || 0),
      color: 'cyan',
    },
    {
      label: t('Top10 占比'),
      value: formatPercent(latest.top10_share || 0),
      color: 'orange',
    },
    {
      label: t('负余额用户'),
      value: renderNumber(latest.negative_user_count || 0),
      color: latest.negative_user_count > 0 ? 'red' : 'grey',
    },
  ];
};

const RANGE_OPTIONS = [
  { label: '近7天', value: 7 },
  { label: '近30天', value: 30 },
];

const UserBalanceTrendUsersModal = ({ visible, onCancel, onChanged, t }) => {
  const [keyword, setKeyword] = useState('');
  const [appliedKeyword, setAppliedKeyword] = useState('');
  const [includeStatus, setIncludeStatus] = useState('all');
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({
    currentPage: 1,
    pageSize: 10,
    total: 0,
  });

  const loadUsers = useCallback(
    async (page = 1, pageSize = pagination.pageSize) => {
      if (!visible) return;
      setLoading(true);
      try {
        const res = await API.get('/api/data/user-balance-trend/users', {
          params: {
            p: page,
            page_size: pageSize,
            keyword: appliedKeyword,
            include_status: includeStatus,
          },
        });
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('加载失败'));
          return;
        }
        setUsers(data?.items || []);
        setPagination({
          currentPage: data?.page || page,
          pageSize: data?.page_size || pageSize,
          total: data?.total || 0,
        });
      } catch (err) {
        showError(err);
      } finally {
        setLoading(false);
      }
    },
    [appliedKeyword, includeStatus, pagination.pageSize, t, visible],
  );

  useEffect(() => {
    if (visible) {
      loadUsers(1);
    }
  }, [appliedKeyword, includeStatus, loadUsers, visible]);

  const applySearch = () => {
    const nextKeyword = keyword.trim();
    if (nextKeyword === appliedKeyword) {
      loadUsers(1);
      return;
    }
    setAppliedKeyword(nextKeyword);
  };

  const updateUserIncluded = async (record, included) => {
    try {
      const res = await API.put(`/api/data/user-balance-trend/users/${record.id}`, {
        balance_trend_disabled: !included,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('更新失败'));
        return;
      }
      if (includeStatus === 'all') {
        setUsers((prev) =>
          prev.map((item) =>
            item.id === record.id ? { ...item, ...data } : item,
          ),
        );
      } else {
        loadUsers(pagination.currentPage);
      }
      await onChanged?.();
      showSuccess(t('统计设置已更新，当前汇总已刷新'));
    } catch (err) {
      showError(err);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    {
      title: t('用户'),
      dataIndex: 'username',
      render: (text, record) => (
        <div className='min-w-0'>
          <div className='truncate font-medium'>{record.display_name || text || '-'}</div>
          <div className='truncate text-xs text-gray-500'>{text || '-'}</div>
        </div>
      ),
    },
    {
      title: t('当前余额'),
      dataIndex: 'quota',
      align: 'right',
      render: (value) => renderQuota(value || 0, 2),
    },
    {
      title: t('计入走势'),
      dataIndex: 'balance_trend_disabled',
      align: 'center',
      render: (disabled, record) => (
        <Switch
          checked={!disabled}
          onChange={(checked) => updateUserIncluded(record, checked)}
        />
      ),
    },
  ];

  return (
    <Modal
      title={t('余额走势统计用户设置')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={760}
    >
      <div className='space-y-3'>
        <div className='flex flex-col md:flex-row gap-2 md:items-center md:justify-between'>
          <Space wrap>
            <Input
              value={keyword}
              onChange={setKeyword}
              onEnterPress={applySearch}
              placeholder={t('搜索 ID / 用户名 / 邮箱 / 备注')}
              prefix={<Search size={14} />}
              showClear
              style={{ width: 260 }}
            />
            <Select
              value={includeStatus}
              onChange={setIncludeStatus}
              optionList={[
                { label: t('全部用户'), value: 'all' },
                { label: t('已计入'), value: 'included' },
                { label: t('已排除'), value: 'excluded' },
              ]}
              style={{ width: 120 }}
            />
            <Button onClick={applySearch}>{t('搜索')}</Button>
          </Space>
          <Text type='tertiary' size='small'>
            {t('关闭后不会影响用户正常余额，只影响每日余额走势统计')}
          </Text>
        </div>
        <Table
          columns={columns}
          dataSource={users}
          rowKey='id'
          loading={loading}
          size='small'
          pagination={{
            currentPage: pagination.currentPage,
            pageSize: pagination.pageSize,
            total: pagination.total,
            pageSizeOpts: [10, 20, 50],
            showSizeChanger: true,
            onPageChange: (page) => loadUsers(page, pagination.pageSize),
            onPageSizeChange: (pageSize) => loadUsers(1, pageSize),
          }}
        />
      </div>
    </Modal>
  );
};

const UserBalanceTrendPanel = ({
  report,
  loading,
  days = 7,
  onDaysChange,
  onUsersChanged,
  embedded = false,
  CARD_PROPS,
  CHART_CONFIG,
  t,
}) => {
  const [settingsVisible, setSettingsVisible] = useState(false);
  const snapshots = report?.snapshots || [];
  const metricCards = useMemo(() => buildMetricCards(report, t), [report, t]);
  const trendSpec = useMemo(
    () => ({
      type: 'area',
      data: [
        {
          id: 'userBalanceTrend',
          values: snapshots.map((item) => ({
            date: item.snapshot_date,
            amount: quotaToDisplayAmount(item.total_quota),
            positiveUsers: item.positive_user_count,
          })),
        },
      ],
      xField: 'date',
      yField: 'amount',
      title: {
        visible: true,
        text: t('全站用户余额走势（金额）'),
        subtext: `${t('每日 04:00 自动统计')} · ${t('当前汇总')}：${formatSnapshotTime(
          report?.current?.snapshot_at || report?.latest?.snapshot_at,
        )}`,
      },
      line: { style: { stroke: '#1664FF', lineWidth: 2 } },
      area: {
        style: {
          fill: {
            gradient: 'linear',
            x0: 0,
            y0: 0,
            x1: 0,
            y1: 1,
            stops: [
              { offset: 0, color: 'rgba(22, 100, 255, 0.28)' },
              { offset: 1, color: 'rgba(22, 100, 255, 0.02)' },
            ],
          },
        },
      },
      point: { visible: true, style: { size: 4 } },
      axes: [
        { orient: 'bottom', type: 'band' },
        {
          orient: 'left',
          type: 'linear',
          label: { formatMethod: (value) => renderDisplayAmount(value, 0) },
        },
      ],
      tooltip: {
        mark: {
          content: [
            { key: t('统计日期'), value: (datum) => datum.date },
            {
              key: t('总余额'),
              value: (datum) => renderDisplayAmount(datum.amount || 0, 2),
            },
            {
              key: t('有余额用户'),
              value: (datum) => renderNumber(datum.positiveUsers || 0),
            },
          ],
        },
      },
    }),
    [report?.current?.snapshot_at, report?.latest?.snapshot_at, snapshots, t],
  );

  const topUserColumns = [
    {
      title: t('用户'),
      dataIndex: 'username',
      render: (text, record) => record.display_name || text || '-',
    },
    {
      title: t('余额'),
      dataIndex: 'quota',
      align: 'right',
      render: (value) => renderQuota(value || 0, 2),
    },
  ];

  const body = (
    <>
      <Spin spinning={loading}>
        <div className='space-y-4'>
          <div className='flex flex-col gap-3 md:flex-row md:items-center md:justify-between'>
            <Space wrap>
              {RANGE_OPTIONS.map((item) => (
                <Button
                  key={item.value}
                  size='small'
                  theme={days === item.value ? 'solid' : 'light'}
                  type='primary'
                  onClick={() => onDaysChange?.(item.value)}
                >
                  {t(item.label)}
                </Button>
              ))}
            </Space>
            <Button
              size='small'
              icon={<Settings size={14} />}
              onClick={() => setSettingsVisible(true)}
            >
              {t('统计用户设置')}
            </Button>
          </div>
          {snapshots.length > 0 ? (
            <>
              <div className='grid grid-cols-2 md:grid-cols-5 gap-3'>
                {metricCards.map((item) => (
                  <div key={item.label} className='rounded-xl bg-gray-50 p-3'>
                    <div className='mb-2 flex items-center justify-between gap-2'>
                      <span className='text-xs text-gray-500'>{item.label}</span>
                      <Tag color={item.color} shape='circle' size='small'>
                        <TrendingUp size={12} />
                      </Tag>
                    </div>
                    <div className='text-lg font-semibold text-gray-900'>
                      {item.value}
                    </div>
                  </div>
                ))}
              </div>

              <div className='grid grid-cols-1 xl:grid-cols-3 gap-4'>
                <div className='h-80 xl:col-span-2 rounded-xl border border-gray-100 p-2'>
                  <Suspense
                    fallback={
                      <div className='h-full flex items-center justify-center text-gray-400 text-sm'>
                        {t('加载中...')}
                      </div>
                    }
                  >
                    <VChart spec={trendSpec} option={CHART_CONFIG} />
                  </Suspense>
                </div>
                <div className='rounded-xl border border-gray-100 p-3'>
                  <div className='mb-3 text-sm font-semibold text-gray-700'>
                    {t('高余额用户 Top10')}
                  </div>
                  <Table
                    columns={topUserColumns}
                    dataSource={
                      report?.current_top_users || report?.top_users || []
                    }
                    rowKey='id'
                    pagination={false}
                    size='small'
                  />
                  {(
                    report?.current_negative_users ||
                    report?.negative_users ||
                    []
                  ).length > 0 ? (
                    <div className='mt-3 rounded-lg bg-red-50 p-2 text-xs text-red-600'>
                      {t('发现负余额用户')}：
                      {(
                        report?.current_negative_users ||
                        report?.negative_users ||
                        []
                      )
                        .slice(0, 3)
                        .map((item) => item.display_name || item.username)
                        .join('、')}
                    </div>
                  ) : null}
                </div>
              </div>
            </>
          ) : (
            <div className='py-10'>
              <Empty
                title={t('暂无余额快照')}
                description={t('系统会在每天 04:00 自动统计用户余额')}
              />
            </div>
          )}
        </div>
      </Spin>
      <UserBalanceTrendUsersModal
        visible={settingsVisible}
        onCancel={() => setSettingsVisible(false)}
        onChanged={onUsersChanged}
        t={t}
      />
    </>
  );

  if (embedded) {
    return body;
  }

  return (
    <Card
      {...CARD_PROPS}
      className='!rounded-2xl mb-4'
      title={
        <div className='flex items-center gap-2'>
          <WalletCards size={16} />
          {t('用户余额走势')}
        </div>
      }
    >
      {body}
    </Card>
  );
};

export default UserBalanceTrendPanel;
