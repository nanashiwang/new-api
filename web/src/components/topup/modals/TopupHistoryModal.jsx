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
  Avatar,
  Badge,
  Button,
  Card,
  Collapsible,
  DatePicker,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { IconFilter, IconSearch } from '@douyinfe/semi-icons';
import { Coins } from 'lucide-react';
import {
  API,
  renderQuota,
  renderQuotaWithAmount,
  stringToColor,
  timestamp2string,
} from '../../../helpers';
import { isAdmin } from '../../../helpers/utils';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import PaymentRiskCaseDetailModal from './PaymentRiskCaseDetailModal';

const { Text } = Typography;

const STATUS_CONFIG = {
  success: { type: 'success', label: '成功' },
  pending: { type: 'warning', label: '待支付' },
  expired: { type: 'danger', label: '已过期' },
  cancelled: { type: 'tertiary', label: '已取消' },
};

const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  alipay: '支付宝',
  wxpay: '微信',
  wallet: '钱包',
};

const RISK_STATUS_CONFIG = {
  open: { color: 'red', label: '待处理' },
  confirmed: { color: 'green', label: '已确认' },
  reversed: { color: 'orange', label: '已回退' },
  voided: { color: 'grey', label: '已作废' },
};

const RISK_REASON_MAP = {
  manual_review: '人工标记',
  order_not_found: '订单不存在',
  order_status_invalid: '订单状态异常',
  payment_method_mismatch: '支付方式不匹配',
  amount_mismatch: '支付金额不匹配',
  unsupported_order_type: '订单类型不支持',
};

const RECORD_TYPE_MAP = {
  topup: '在线充值',
  subscription: '订阅套餐',
  sellable_token_purchase: '钱包购买',
};

const EMPTY_FILTERS = {
  keyword: '',
  username: '',
  status: '',
  paymentMethod: '',
};

const EMPTY_RISK_FILTERS = {
  keyword: '',
  username: '',
  status: 'open',
  recordType: '',
  reason: '',
};

const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '待支付', value: 'pending' },
  { label: '成功', value: 'success' },
  { label: '已过期', value: 'expired' },
  { label: '已取消', value: 'cancelled' },
];

const PAYMENT_OPTIONS = [
  { label: '全部支付方式', value: '' },
  { label: '钱包', value: 'wallet' },
  { label: '微信', value: 'wxpay' },
  { label: '支付宝', value: 'alipay' },
  { label: 'Stripe', value: 'stripe' },
  { label: 'Creem', value: 'creem' },
];

const DASHBOARD_PRESET_OPTIONS = [
  { key: 'today', label: '今天' },
  { key: 'week', label: '近7天' },
  { key: 'month', label: '近30天' },
];

const DASHBOARD_RANK_LIMIT_OPTIONS = [
  { label: 'Top 10', value: 10 },
  { label: 'Top 20', value: 20 },
  { label: 'Top 50', value: 50 },
];

const RISK_STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '待处理', value: 'open' },
  { label: '已确认', value: 'confirmed' },
  { label: '已回退', value: 'reversed' },
  { label: '已作废', value: 'voided' },
];

const RISK_RECORD_TYPE_OPTIONS = [
  { label: '全部订单类型', value: '' },
  { label: '充值订单', value: 'topup' },
  { label: '订阅订单', value: 'subscription' },
];

const RISK_REASON_OPTIONS = [
  { label: '全部原因', value: '' },
  { label: '人工标记', value: 'manual_review' },
  { label: '订单不存在', value: 'order_not_found' },
  { label: '订单状态异常', value: 'order_status_invalid' },
  { label: '支付方式不匹配', value: 'payment_method_mismatch' },
  { label: '支付金额不匹配', value: 'amount_mismatch' },
];

function resolveOrderType(record) {
  if (!record) {
    return '';
  }
  if (record.order_type) {
    return record.order_type;
  }
  const tradeNo = String(record.trade_no || '').toLowerCase();
  if (Number(record.amount || 0) === 0 && tradeNo.startsWith('sub')) {
    return 'subscription';
  }
  return record.record_type || 'topup';
}

function formatMoney(value, currency = 'CNY') {
  const amount = Number(value || 0);
  if (!Number.isFinite(amount)) {
    return '-';
  }
  const upperCurrency = String(currency || '').toUpperCase();
  const symbolMap = {
    CNY: '¥',
    RMB: '¥',
    USD: '$',
    EUR: '€',
    GBP: '£',
  };
  const symbol = symbolMap[upperCurrency] || '';
  return `${symbol}${amount.toFixed(2)}${upperCurrency && !symbol ? ` ${upperCurrency}` : ''}`;
}

function buildTableEmpty(t, description) {
  return (
    <Empty
      image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
      darkModeImage={<IllustrationNoResultDark style={{ width: 150, height: 150 }} />}
      description={t(description)}
      style={{ padding: 30 }}
    />
  );
}

function createDashboardDateRange(preset) {
  const now = new Date();
  const start = new Date(now);
  start.setMilliseconds(0);

  switch (preset) {
    case 'month':
      start.setDate(start.getDate() - 29);
      start.setHours(0, 0, 0, 0);
      return [start, now];
    case 'week':
      start.setDate(start.getDate() - 6);
      start.setHours(0, 0, 0, 0);
      return [start, now];
    case 'today':
    default:
      start.setHours(0, 0, 0, 0);
      return [start, now];
  }
}

function toTimestampSeconds(value) {
  if (!value) {
    return 0;
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time) || time <= 0) {
    return 0;
  }
  return Math.floor(time / 1000);
}

function formatCount(value) {
  const count = Number(value || 0);
  if (!Number.isFinite(count)) {
    return '0';
  }
  return count.toLocaleString();
}

function normalizeDashboardStats(stats) {
  return {
    totals: stats?.totals || { money: 0, order_count: 0 },
    statuses: stats?.statuses || {},
    payment_methods: stats?.payment_methods || {},
  };
}

const TopupHistoryModal = ({ visible, onCancel, t }) => {
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState(EMPTY_FILTERS);
  const [showFilters, setShowFilters] = useState(false);
  const [activeTab, setActiveTab] = useState('records');

  const [dashboardLoading, setDashboardLoading] = useState(false);
  const [dashboardStats, setDashboardStats] = useState(null);
  const [dashboardRankings, setDashboardRankings] = useState([]);
  const [dashboardPreset, setDashboardPreset] = useState('today');
  const [dashboardDateRange, setDashboardDateRange] = useState(() => createDashboardDateRange('today'));
  const [dashboardRankLimit, setDashboardRankLimit] = useState(10);

  const [riskLoading, setRiskLoading] = useState(false);
  const [riskCases, setRiskCases] = useState([]);
  const [riskTotal, setRiskTotal] = useState(0);
  const [riskPage, setRiskPage] = useState(1);
  const [riskPageSize, setRiskPageSize] = useState(10);
  const [riskFilters, setRiskFilters] = useState(EMPTY_RISK_FILTERS);
  const [riskAppliedFilters, setRiskAppliedFilters] = useState(EMPTY_RISK_FILTERS);
  const [riskDetailVisible, setRiskDetailVisible] = useState(false);
  const [selectedRiskCaseId, setSelectedRiskCaseId] = useState(0);
  const [selectedRiskCaseSeed, setSelectedRiskCaseSeed] = useState(null);

  const isMobile = useIsMobile();
  const userIsAdmin = useMemo(() => isAdmin(), []);

  const loadTopups = async (currentPage, currentPageSize, currentFilters) => {
    setLoading(true);
    try {
      const base = userIsAdmin ? '/api/user/payment-records' : '/api/user/payment-records/self';
      const searchParams = new URLSearchParams({
        p: String(currentPage),
        page_size: String(currentPageSize),
      });

      if (currentFilters.keyword) {
        searchParams.set('keyword', currentFilters.keyword.trim());
      }
      if (currentFilters.status) {
        searchParams.set('status', currentFilters.status);
      }
      if (currentFilters.paymentMethod) {
        searchParams.set('payment_method', currentFilters.paymentMethod);
      }
      if (userIsAdmin && currentFilters.username) {
        searchParams.set('username', currentFilters.username.trim());
      }

      const res = await API.get(`${base}?${searchParams.toString()}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载支付记录失败') });
        return;
      }

      setTopups(data?.items || []);
      setTotal(data?.total || 0);
    } catch (error) {
      Toast.error({ content: t('加载支付记录失败') });
    } finally {
      setLoading(false);
    }
  };

  const loadRiskCases = async (currentPage, currentPageSize, currentFilters) => {
    if (!userIsAdmin) {
      return;
    }
    setRiskLoading(true);
    try {
      const searchParams = new URLSearchParams({
        p: String(currentPage),
        page_size: String(currentPageSize),
      });

      if (currentFilters.keyword) {
        searchParams.set('keyword', currentFilters.keyword.trim());
      }
      if (currentFilters.username) {
        searchParams.set('username', currentFilters.username.trim());
      }
      if (currentFilters.status) {
        searchParams.set('status', currentFilters.status);
      }
      if (currentFilters.recordType) {
        searchParams.set('record_type', currentFilters.recordType);
      }
      if (currentFilters.reason) {
        searchParams.set('reason', currentFilters.reason);
      }

      const res = await API.get(`/api/user/payment-risk-cases?${searchParams.toString()}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载异常单失败') });
        return;
      }

      setRiskCases(data?.items || []);
      setRiskTotal(data?.total || 0);
    } catch (error) {
      Toast.error({ content: t('加载异常单失败') });
    } finally {
      setRiskLoading(false);
    }
  };

  const loadDashboard = async (dateRange = dashboardDateRange, limit = dashboardRankLimit) => {
    if (!userIsAdmin) {
      return;
    }

    setDashboardLoading(true);
    try {
      const [start, end] = dateRange || [];
      const params = {
        start_timestamp: toTimestampSeconds(start),
        end_timestamp: toTimestampSeconds(end),
      };

      const [statsRes, rankingsRes] = await Promise.all([
        API.get('/api/user/payment-records/stats', { params }),
        API.get('/api/user/payment-records/rankings', {
          params: {
            ...params,
            limit,
          },
        }),
      ]);

      const statsPayload = statsRes?.data || {};
      const rankingsPayload = rankingsRes?.data || {};

      if (!statsPayload.success) {
        Toast.error({ content: t(statsPayload.message || '加载对账统计失败') });
        return;
      }
      if (!rankingsPayload.success) {
        Toast.error({ content: t(rankingsPayload.message || '加载充值榜单失败') });
        return;
      }

      setDashboardStats(normalizeDashboardStats(statsPayload.data));
      setDashboardRankings(rankingsPayload.data?.items || []);
    } catch (error) {
      Toast.error({ content: t('加载对账看板失败') });
    } finally {
      setDashboardLoading(false);
    }
  };

  const refreshRecords = async () => {
    await loadTopups(page, pageSize, appliedFilters);
  };

  const refreshDashboard = async () => {
    await loadDashboard(dashboardDateRange, dashboardRankLimit);
  };

  const refreshRiskCases = async () => {
    if (!userIsAdmin) {
      return;
    }
    await loadRiskCases(riskPage, riskPageSize, riskAppliedFilters);
  };

  useEffect(() => {
    if (!visible) {
      return;
    }
    if (activeTab === 'records') {
      loadTopups(page, pageSize, appliedFilters);
    }
  }, [visible, activeTab, page, pageSize, appliedFilters, userIsAdmin]);

  useEffect(() => {
    if (!visible || !userIsAdmin) {
      return;
    }
    if (activeTab === 'dashboard') {
      loadDashboard(dashboardDateRange, dashboardRankLimit);
    }
  }, [visible, activeTab, userIsAdmin, dashboardDateRange, dashboardRankLimit]);

  useEffect(() => {
    if (!visible || !userIsAdmin) {
      return;
    }
    if (activeTab === 'risk') {
      loadRiskCases(riskPage, riskPageSize, riskAppliedFilters);
    }
  }, [visible, activeTab, riskPage, riskPageSize, riskAppliedFilters, userIsAdmin]);

  const handleFilterChange = (key, value) => {
    setFilters((prev) => ({
      ...prev,
      [key]: value || '',
    }));
  };

  const applyFilters = (nextFilters = filters) => {
    setPage(1);
    setAppliedFilters({
      ...nextFilters,
      keyword: nextFilters.keyword.trim(),
      username: nextFilters.username.trim(),
    });
  };

  const resetFilters = () => {
    setPage(1);
    setFilters(EMPTY_FILTERS);
    setAppliedFilters(EMPTY_FILTERS);
    setShowFilters(false);
  };

  const handleRiskFilterChange = (key, value) => {
    setRiskFilters((prev) => ({
      ...prev,
      [key]: value || '',
    }));
  };

  const applyRiskFilters = (nextFilters = riskFilters) => {
    setRiskPage(1);
    setRiskAppliedFilters({
      ...nextFilters,
      keyword: nextFilters.keyword.trim(),
      username: nextFilters.username.trim(),
    });
  };

  const resetRiskFilters = () => {
    setRiskPage(1);
    setRiskFilters(EMPTY_RISK_FILTERS);
    setRiskAppliedFilters(EMPTY_RISK_FILTERS);
  };

  const activeFilterTags = useMemo(() => {
    const tags = [];
    if (appliedFilters.username) {
      tags.push({ key: 'username', label: `ID/用户名: ${appliedFilters.username}` });
    }
    if (appliedFilters.status) {
      const found = STATUS_OPTIONS.find((option) => option.value === appliedFilters.status);
      tags.push({ key: 'status', label: `状态: ${found ? found.label : appliedFilters.status}` });
    }
    if (appliedFilters.paymentMethod) {
      const found = PAYMENT_OPTIONS.find((option) => option.value === appliedFilters.paymentMethod);
      tags.push({
        key: 'paymentMethod',
        label: `支付方式: ${found ? found.label : appliedFilters.paymentMethod}`,
      });
    }
    return tags;
  }, [appliedFilters]);

  const removeFilterTag = (key) => {
    const nextFilters = { ...filters, [key]: '' };
    setFilters(nextFilters);
    applyFilters(nextFilters);
  };

  const filterByUsername = (username) => {
    if (!username) {
      return;
    }
    if (activeTab === 'risk') {
      const nextFilters = { ...riskFilters, username };
      setRiskFilters(nextFilters);
      applyRiskFilters(nextFilters);
      return;
    }
    const nextFilters = { ...filters, username };
    setFilters(nextFilters);
    applyFilters(nextFilters);
  };

  const openRecordTabForUsername = (username) => {
    if (!username) {
      return;
    }
    const nextFilters = { ...filters, username };
    setFilters(nextFilters);
    applyFilters(nextFilters);
    setActiveTab('records');
  };

  const handleDashboardPresetChange = (preset) => {
    setDashboardPreset(preset);
    if (preset === 'custom') {
      return;
    }
    setDashboardDateRange(createDashboardDateRange(preset));
  };

  const handleDashboardDateRangeChange = (value) => {
    setDashboardPreset('custom');
    setDashboardDateRange(value || []);
  };

  const handleAdminComplete = async (tradeNo) => {
    try {
      const res = await API.post('/api/user/topup/complete', {
        trade_no: tradeNo,
      });
      const { success, message } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '补单失败') });
        return;
      }
      Toast.success({ content: t('补单成功') });
      await refreshRecords();
    } catch (error) {
      Toast.error({ content: t('补单失败') });
    }
  };

  const confirmAdminComplete = (tradeNo) => {
    Modal.confirm({
      title: t('确认补单'),
      content: t('是否将该订单标记为成功并为用户入账？'),
      onOk: () => handleAdminComplete(tradeNo),
    });
  };

  const openRiskCaseDetail = (riskCaseId, riskCase) => {
    setSelectedRiskCaseId(Number(riskCaseId || 0));
    setSelectedRiskCaseSeed(riskCase || null);
    setRiskDetailVisible(true);
  };

  const resolveRiskRecordType = (record) => {
    const orderType = resolveOrderType(record);
    if (orderType === 'subscription') {
      return 'subscription';
    }
    if (record?.record_type === 'topup') {
      return 'topup';
    }
    return '';
  };

  const canCreateRiskCase = (record) => {
    if (!userIsAdmin || record?.risk_case_id) {
      return false;
    }
    return resolveRiskRecordType(record) !== '';
  };

  const handleCreateRiskCase = async (record) => {
    const recordType = resolveRiskRecordType(record);
    if (!recordType || !record?.trade_no) {
      Toast.error({ content: t('当前记录不支持标记异常') });
      return;
    }
    try {
      const res = await API.post('/api/user/payment-risk-cases', {
        record_type: recordType,
        trade_no: record.trade_no,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '标记异常失败') });
        return;
      }
      Toast.success({ content: t('已加入异常审核队列') });
      await Promise.all([refreshRecords(), refreshRiskCases()]);
      openRiskCaseDetail(data?.risk_case?.id, data?.risk_case || null);
    } catch (error) {
      Toast.error({ content: t('标记异常失败') });
    }
  };

  const confirmCreateRiskCase = (record) => {
    Modal.confirm({
      title: t('标记异常'),
      content: t('确认将这笔订单加入人工审核队列吗？'),
      onOk: () => handleCreateRiskCase(record),
    });
  };

  const handleRiskCaseResolved = async (updatedRiskCase) => {
    if (updatedRiskCase?.id) {
      setSelectedRiskCaseSeed(updatedRiskCase);
    }
    await Promise.all([refreshRecords(), refreshRiskCases()]);
  };

  const renderStatusBadge = (status) => {
    const config = STATUS_CONFIG[status] || { type: 'primary', label: status || '-' };
    return (
      <span className='flex items-center gap-2'>
        <Badge dot type={config.type} />
        <span>{t(config.label)}</span>
      </span>
    );
  };

  const renderPaymentMethod = (paymentMethod) => {
    const displayName = PAYMENT_METHOD_MAP[paymentMethod];
    return (
      <Tag shape='circle' color={paymentMethod === 'wallet' ? 'blue' : 'grey'}>
        {t(displayName || paymentMethod || '-')}
      </Tag>
    );
  };

  const renderRiskStatusTag = (status) => {
    const config = RISK_STATUS_CONFIG[status] || { color: 'grey', label: status || '-' };
    return (
      <Tag color={config.color} shape='circle' size='small'>
        {t(config.label)}
      </Tag>
    );
  };

  const renderRiskReason = (reason) => t(RISK_REASON_MAP[reason] || reason || '-');

  const isSellableTokenPurchase = (record) => record?.record_type === 'sellable_token_purchase';

  const isSubscriptionTopup = (record) => resolveOrderType(record) === 'subscription';

  const renderRecordType = (record) => {
    if (isSellableTokenPurchase(record)) {
      return (
        <div className='min-w-0'>
          <Tag color='blue' shape='circle' size='small'>
            {t('钱包购买')}
          </Tag>
          <div className='mt-1'>
            <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 180, display: 'inline-block' }}>
              {record?.product_name || t('可售令牌')}
            </Text>
          </div>
        </div>
      );
    }

    if (isSubscriptionTopup(record)) {
      return (
        <Tag color='purple' shape='circle' size='small'>
          {t('订阅套餐')}
        </Tag>
      );
    }

    return (
      <Tag color='green' shape='circle' size='small'>
        {t('在线充值')}
      </Tag>
    );
  };

  const renderRecordNo = (record) => {
    const text = record?.trade_no || '-';
    return (
      <Text
        copyable={text !== '-'}
        ellipsis={{ showTooltip: { opts: { style: { wordBreak: 'break-all' } } } }}
        style={{ width: 170, display: 'inline-block' }}
      >
        {text}
      </Text>
    );
  };

  const renderRiskSummary = (record) => {
    if (!record?.risk_case_id) {
      return <Text type='tertiary'>-</Text>;
    }
    return (
      <div className='flex flex-col gap-1'>
        {renderRiskStatusTag(record.risk_status)}
        <Text type='tertiary' size='small'>
          {renderRiskReason(record.risk_reason)}
        </Text>
      </div>
    );
  };

  const recordColumns = useMemo(() => {
    const columns = [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
        key: 'trade_no',
        width: 200,
        render: (_, record) => renderRecordNo(record),
      },
      {
        title: t('类型 / 商品'),
        key: 'record_type',
        width: 200,
        render: (_, record) => renderRecordType(record),
      },
    ];

    if (userIsAdmin) {
      columns.push({
        title: t('用户名'),
        dataIndex: 'username',
        key: 'username',
        render: (_, record) => {
          const username = record?.username || '';
          const displayName = record?.display_name || '';
          if (!username) {
            return <Text type='tertiary'>-</Text>;
          }

          return (
            <Space spacing={8} align='center'>
              <Avatar size='extra-small' color={stringToColor(username)}>
                {username.slice(0, 1).toUpperCase()}
              </Avatar>
              <div className='flex flex-col leading-5'>
                {record?.user_id > 0 ? (
                  <Text type='tertiary' size='small'>
                    ID: {record.user_id}
                  </Text>
                ) : null}
                <Text
                  link
                  size='small'
                  style={{ cursor: 'pointer' }}
                  onClick={() => filterByUsername(username)}
                >
                  {username}
                </Text>
                {displayName ? <Text type='tertiary'>{displayName}</Text> : null}
              </div>
            </Space>
          );
        },
      });
    }

    columns.push(
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        render: renderPaymentMethod,
      },
      {
        title: t('充值额度'),
        dataIndex: 'amount',
        key: 'amount',
        render: (amount, record) => {
          if (isSellableTokenPurchase(record)) {
            return <Text type='tertiary'>-</Text>;
          }
          if (isSubscriptionTopup(record)) {
            return (
              <Tag color='purple' shape='circle' size='small'>
                SUB
              </Tag>
            );
          }
          return (
            <span className='flex items-center gap-1'>
              <Coins size={16} />
              <Text>{renderQuotaWithAmount(amount ?? 0)}</Text>
            </span>
          );
        },
      },
      {
        title: t('支付金额'),
        dataIndex: 'money',
        key: 'money',
        render: (money, record) => {
          if (isSellableTokenPurchase(record)) {
            return <Text type='danger'>{renderQuota(record?.amount ?? 0)}</Text>;
          }
          return <Text type='danger'>{formatMoney(money)}</Text>;
        },
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        render: renderStatusBadge,
      },
      {
        title: t('创建时间'),
        dataIndex: 'create_time',
        key: 'create_time',
        render: (time) => (time ? timestamp2string(time) : '-'),
      },
    );

    if (userIsAdmin) {
      columns.push({
        title: t('风控'),
        key: 'risk',
        width: 150,
        render: (_, record) => renderRiskSummary(record),
      });
      columns.push({
        title: t('操作'),
        key: 'action',
        width: 220,
        render: (_, record) => {
          const actionButtons = [];

          if (record?.record_type === 'topup' && record?.status === 'pending' && !record?.risk_case_id) {
            actionButtons.push(
              <Button
                key='complete'
                size='small'
                type='primary'
                theme='outline'
                onClick={() => confirmAdminComplete(record.trade_no)}
              >
                {t('补单')}
              </Button>,
            );
          }

          if (record?.risk_case_id) {
            actionButtons.push(
              <Button
                key='detail'
                size='small'
                theme='outline'
                onClick={() =>
                  openRiskCaseDetail(record.risk_case_id, {
                    id: record.risk_case_id,
                    trade_no: record.trade_no,
                    record_type: resolveRiskRecordType(record) || resolveOrderType(record),
                    status: record.risk_status,
                    reason: record.risk_reason,
                    user_id: record.user_id,
                    username: record.username,
                    display_name: record.display_name,
                    payment_method: record.payment_method,
                    expected_amount: record.amount,
                    expected_money: record.money,
                    order_status: record.status,
                  })
                }
              >
                {t('查看异常')}
              </Button>,
            );
          } else if (canCreateRiskCase(record)) {
            actionButtons.push(
              <Button
                key='mark-risk'
                size='small'
                theme='outline'
                type='danger'
                onClick={() => confirmCreateRiskCase(record)}
              >
                {t('标记异常')}
              </Button>,
            );
          }

          if (actionButtons.length === 0) {
            return null;
          }
          return <Space wrap>{actionButtons}</Space>;
        },
      });
    }

    return columns;
  }, [userIsAdmin, filters, riskFilters]);

  const dashboardData = useMemo(() => normalizeDashboardStats(dashboardStats), [dashboardStats]);

  const dashboardSummaryCards = useMemo(() => {
    const totals = dashboardData.totals;
    const statuses = dashboardData.statuses || {};
    return [
      {
        key: 'total-money',
        label: '总支付金额',
        value: formatMoney(totals.money),
        helper: `${t('总订单数')} ${formatCount(totals.order_count)}`,
      },
      {
        key: 'success-money',
        label: '成功支付金额',
        value: formatMoney(statuses.success?.money),
        helper: `${t('成功订单')} ${formatCount(statuses.success?.order_count)}`,
      },
      {
        key: 'pending-money',
        label: '待支付金额',
        value: formatMoney(statuses.pending?.money),
        helper: `${t('待支付订单')} ${formatCount(statuses.pending?.order_count)}`,
      },
      {
        key: 'expired-money',
        label: '失效金额',
        value: formatMoney(statuses.expired?.money),
        helper: `${t('失效订单')} ${formatCount(statuses.expired?.order_count)}`,
      },
      {
        key: 'cancelled-money',
        label: '已取消金额',
        value: formatMoney(statuses.cancelled?.money),
        helper: `${t('已取消订单')} ${formatCount(statuses.cancelled?.order_count)}`,
      },
    ];
  }, [dashboardData, t]);

  const dashboardPaymentMethods = useMemo(() => {
    const items = Object.entries(dashboardData.payment_methods || {}).map(([method, stats]) => ({
      method,
      money: Number(stats?.money || 0),
      orderCount: Number(stats?.order_count || 0),
    }));
    items.sort((left, right) => {
      if (left.money !== right.money) {
        return right.money - left.money;
      }
      return right.orderCount - left.orderCount;
    });
    return items;
  }, [dashboardData]);

  const dashboardRankingColumns = useMemo(
    () => [
      {
        title: t('排名'),
        key: 'rank',
        width: 72,
        render: (_, __, index) => (
          <Tag color={index < 3 ? 'orange' : 'grey'} shape='circle'>
            #{index + 1}
          </Tag>
        ),
      },
      {
        title: t('用户'),
        key: 'username',
        render: (_, record) => {
          if (!record?.username) {
            return <Text type='tertiary'>-</Text>;
          }
          return (
            <Space spacing={8} align='center'>
              <Avatar size='extra-small' color={stringToColor(record.username)}>
                {record.username.slice(0, 1).toUpperCase()}
              </Avatar>
              <div className='flex flex-col leading-5'>
                <Text
                  link
                  size='small'
                  style={{ cursor: 'pointer' }}
                  onClick={() => openRecordTabForUsername(record.username)}
                >
                  {record.username}
                </Text>
                {record.user_id ? (
                  <Text type='tertiary' size='small'>
                    ID: {record.user_id}
                  </Text>
                ) : null}
                {record.display_name ? <Text type='tertiary'>{record.display_name}</Text> : null}
              </div>
            </Space>
          );
        },
      },
      {
        title: t('充值金额'),
        key: 'money',
        width: 120,
        render: (_, record) => <Text strong>{formatMoney(record?.money)}</Text>,
      },
      {
        title: t('成功金额'),
        key: 'success_money',
        width: 120,
        render: (_, record) => <Text type='success'>{formatMoney(record?.success_money)}</Text>,
      },
      {
        title: t('订单数'),
        key: 'order_count',
        width: 100,
        render: (_, record) => formatCount(record?.order_count),
      },
      {
        title: t('待支付金额'),
        key: 'pending_money',
        width: 120,
        render: (_, record) => <Text type='warning'>{formatMoney(record?.pending_money)}</Text>,
      },
    ],
    [t, openRecordTabForUsername],
  );

  const riskCaseColumns = useMemo(() => {
    return [
      {
        title: t('异常单'),
        key: 'trade',
        width: 260,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Space wrap>
              <Tag shape='circle' color='grey' size='small'>
                {t(RECORD_TYPE_MAP[record.record_type] || record.record_type || '-')}
              </Tag>
              {renderRiskStatusTag(record.status)}
            </Space>
            <Text copyable>{record.trade_no || '-'}</Text>
          </div>
        ),
      },
      {
        title: t('用户'),
        key: 'username',
        render: (_, record) => {
          if (!record?.username) {
            return <Text type='tertiary'>-</Text>;
          }
          return (
            <Space spacing={8} align='center'>
              <Avatar size='extra-small' color={stringToColor(record.username)}>
                {record.username.slice(0, 1).toUpperCase()}
              </Avatar>
              <div className='flex flex-col leading-5'>
                <Text
                  link
                  size='small'
                  style={{ cursor: 'pointer' }}
                  onClick={() => filterByUsername(record.username)}
                >
                  {record.username}
                </Text>
                {record.user_id ? (
                  <Text type='tertiary' size='small'>
                    ID: {record.user_id}
                  </Text>
                ) : null}
                {record.display_name ? <Text type='tertiary'>{record.display_name}</Text> : null}
              </div>
            </Space>
          );
        },
      },
      {
        title: t('异常原因'),
        key: 'reason',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text>{renderRiskReason(record.reason)}</Text>
            <Text type='tertiary' size='small'>
              {t('订单状态')}: {t(STATUS_CONFIG[record.order_status]?.label || record.order_status || '-')}
            </Text>
          </div>
        ),
      },
      {
        title: t('金额对比'),
        key: 'money',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text type='tertiary' size='small'>
              {t('预期')}: {formatMoney(record.expected_money, record.currency)}
            </Text>
            <Text size='small'>
              {t('回调')}: {formatMoney(record.received_money, record.currency)}
            </Text>
          </div>
        ),
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_at',
        key: 'created_at',
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('操作'),
        key: 'action',
        render: (_, record) => (
          <Button size='small' theme='outline' onClick={() => openRiskCaseDetail(record.id, record)}>
            {t('查看详情')}
          </Button>
        ),
      },
    ];
  }, [riskFilters]);

  const renderRecordFilterPanel = () => (
    <div className='mb-3'>
      <div className='flex items-center gap-2'>
        <Input
          prefix={<IconSearch />}
          placeholder={t('订单号 / 商品名')}
          value={filters.keyword}
          onChange={(value) => handleFilterChange('keyword', value)}
          onEnterPress={() => applyFilters()}
          showClear
          style={{ flex: 1 }}
        />
        <Button
          icon={<IconFilter />}
          theme={showFilters ? 'solid' : 'light'}
          type={activeFilterTags.length > 0 ? 'primary' : 'tertiary'}
          onClick={() => setShowFilters((current) => !current)}
        >
          {t('筛选')}
          {activeFilterTags.length > 0 ? ` (${activeFilterTags.length})` : ''}
        </Button>
        <Button type='primary' onClick={() => applyFilters()}>
          {t('搜索')}
        </Button>
        {activeFilterTags.length > 0 ? (
          <Button theme='borderless' type='tertiary' onClick={resetFilters}>
            {t('重置')}
          </Button>
        ) : null}
      </div>

      {!showFilters && activeFilterTags.length > 0 ? (
        <div className='flex flex-wrap items-center gap-1 mt-2'>
          {activeFilterTags.map((tag) => (
            <Tag
              key={tag.key}
              closable
              size='small'
              color='blue'
              shape='circle'
              onClose={() => removeFilterTag(tag.key)}
            >
              {tag.label}
            </Tag>
          ))}
        </div>
      ) : null}

      <Collapsible isOpen={showFilters} keepDOM>
        <div
          className='mt-2 rounded-lg p-3 flex flex-wrap items-end gap-3'
          style={{
            background: 'var(--semi-color-fill-0)',
            border: '1px solid var(--semi-color-border)',
          }}
        >
          {userIsAdmin ? (
            <div style={{ minWidth: 160, flex: 1 }}>
              <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
                ID/用户名
              </div>
              <Input
                placeholder={t('ID/用户名')}
                value={filters.username}
                onChange={(value) => handleFilterChange('username', value)}
                onEnterPress={() => applyFilters()}
                showClear
                size='small'
              />
            </div>
          ) : null}
          <div style={{ minWidth: 120, flex: 1 }}>
            <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
              {t('状态')}
            </div>
            <Select
              value={filters.status}
              optionList={STATUS_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              onChange={(value) => handleFilterChange('status', value)}
              size='small'
              style={{ width: '100%' }}
            />
          </div>
          <div style={{ minWidth: 130, flex: 1 }}>
            <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
              {t('支付方式')}
            </div>
            <Select
              value={filters.paymentMethod}
              optionList={PAYMENT_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              onChange={(value) => handleFilterChange('paymentMethod', value)}
              size='small'
              style={{ width: '100%' }}
            />
          </div>
        </div>
      </Collapsible>
    </div>
  );

  const renderRiskFilterPanel = () => (
    <div className='mb-3'>
      <div
        className='rounded-lg p-3 flex flex-wrap items-end gap-3'
        style={{
          background: 'var(--semi-color-fill-0)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        <div style={{ minWidth: 220, flex: 2 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('订单号')}
          </div>
          <Input
            prefix={<IconSearch />}
            placeholder={t('订单号')}
            value={riskFilters.keyword}
            onChange={(value) => handleRiskFilterChange('keyword', value)}
            onEnterPress={() => applyRiskFilters()}
            showClear
            size='small'
          />
        </div>
        <div style={{ minWidth: 160, flex: 1 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            ID/用户名
          </div>
          <Input
            placeholder={t('ID/用户名')}
            value={riskFilters.username}
            onChange={(value) => handleRiskFilterChange('username', value)}
            onEnterPress={() => applyRiskFilters()}
            showClear
            size='small'
          />
        </div>
        <div style={{ minWidth: 120, flex: 1 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('状态')}
          </div>
          <Select
            value={riskFilters.status}
            optionList={RISK_STATUS_OPTIONS.map((item) => ({
              ...item,
              label: t(item.label),
            }))}
            onChange={(value) => handleRiskFilterChange('status', value)}
            size='small'
            style={{ width: '100%' }}
          />
        </div>
        <div style={{ minWidth: 130, flex: 1 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('订单类型')}
          </div>
          <Select
            value={riskFilters.recordType}
            optionList={RISK_RECORD_TYPE_OPTIONS.map((item) => ({
              ...item,
              label: t(item.label),
            }))}
            onChange={(value) => handleRiskFilterChange('recordType', value)}
            size='small'
            style={{ width: '100%' }}
          />
        </div>
        <div style={{ minWidth: 160, flex: 1 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('异常原因')}
          </div>
          <Select
            value={riskFilters.reason}
            optionList={RISK_REASON_OPTIONS.map((item) => ({
              ...item,
              label: t(item.label),
            }))}
            onChange={(value) => handleRiskFilterChange('reason', value)}
            size='small'
            style={{ width: '100%' }}
          />
        </div>
        <Space>
          <Button type='primary' onClick={() => applyRiskFilters()}>
            {t('搜索')}
          </Button>
          <Button theme='borderless' type='tertiary' onClick={resetRiskFilters}>
            {t('重置')}
          </Button>
        </Space>
      </div>
    </div>
  );

  const renderRecordsTable = () => (
    <>
      {renderRecordFilterPanel()}
      <Table
        columns={recordColumns}
        dataSource={topups}
        loading={loading}
        rowKey={(record) => `${record?.record_type || 'topup'}-${record?.id || '0'}`}
        size='small'
        pagination={{
          currentPage: page,
          pageSize,
          total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: (currentPage) => setPage(currentPage),
          onPageSizeChange: (currentPageSize) => {
            setPageSize(currentPageSize);
            setPage(1);
          },
        }}
        scroll={{ x: '100%' }}
        empty={buildTableEmpty(t, '暂无支付记录')}
      />
    </>
  );

  const renderDashboardBoard = () => (
    <div className='space-y-4'>
      <div
        className='rounded-lg p-3 flex flex-wrap items-end gap-3'
        style={{
          background: 'var(--semi-color-fill-0)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        <div style={{ minWidth: 260, flex: 2 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('时间范围')}
          </div>
          <Space wrap>
            {DASHBOARD_PRESET_OPTIONS.map((item) => (
              <Button
                key={item.key}
                size='small'
                type={dashboardPreset === item.key ? 'primary' : 'tertiary'}
                theme={dashboardPreset === item.key ? 'solid' : 'outline'}
                onClick={() => handleDashboardPresetChange(item.key)}
              >
                {t(item.label)}
              </Button>
            ))}
          </Space>
        </div>
        <div style={{ minWidth: 280, flex: 2 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('自定义时间')}
          </div>
          <DatePicker
            type='dateTimeRange'
            value={dashboardDateRange}
            onChange={handleDashboardDateRangeChange}
            style={{ width: '100%' }}
          />
        </div>
        <div style={{ minWidth: 120 }}>
          <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
            {t('榜单条数')}
          </div>
          <Select
            value={dashboardRankLimit}
            optionList={DASHBOARD_RANK_LIMIT_OPTIONS.map((item) => ({
              ...item,
              label: item.label,
            }))}
            onChange={(value) => setDashboardRankLimit(Number(value || 10))}
            size='small'
            style={{ width: '100%' }}
          />
        </div>
        <Button type='primary' loading={dashboardLoading} onClick={refreshDashboard}>
          {t('刷新')}
        </Button>
      </div>

      <div
        className='grid gap-3'
        style={{
          gridTemplateColumns: isMobile ? '1fr' : 'repeat(auto-fit, minmax(180px, 1fr))',
        }}
      >
        {dashboardSummaryCards.map((item) => (
          <Card
            key={item.key}
            bordered={false}
            bodyStyle={{ padding: 16 }}
            style={{
              background: 'var(--semi-color-bg-1)',
              border: '1px solid var(--semi-color-border)',
            }}
          >
            <Text type='tertiary' size='small'>
              {t(item.label)}
            </Text>
            <div className='mt-2 text-xl font-semibold'>{item.value}</div>
            <Text type='tertiary' size='small'>
              {item.helper}
            </Text>
          </Card>
        ))}
      </div>

      <div
        className='grid gap-4'
        style={{
          gridTemplateColumns: isMobile ? '1fr' : 'minmax(0, 1.6fr) minmax(280px, 1fr)',
        }}
      >
        <Card
          bordered={false}
          bodyStyle={{ padding: 0 }}
          style={{
            background: 'var(--semi-color-bg-1)',
            border: '1px solid var(--semi-color-border)',
          }}
          title={t('充值金额榜单')}
        >
          <Table
            columns={dashboardRankingColumns}
            dataSource={dashboardRankings}
            loading={dashboardLoading}
            rowKey={(record) => String(record?.user_id || record?.username || '')}
            size='small'
            pagination={false}
            scroll={{ x: '100%' }}
            empty={buildTableEmpty(t, '暂无充值榜单数据')}
          />
        </Card>

        <Card
          bordered={false}
          bodyStyle={{ padding: 16 }}
          style={{
            background: 'var(--semi-color-bg-1)',
            border: '1px solid var(--semi-color-border)',
          }}
          title={t('支付方式分布')}
        >
          {dashboardPaymentMethods.length === 0 ? (
            buildTableEmpty(t, '暂无统计数据')
          ) : (
            <div className='space-y-3'>
              {dashboardPaymentMethods.map((item) => (
                <div key={item.method} className='flex items-center justify-between gap-3'>
                  <div className='flex items-center gap-2'>
                    {renderPaymentMethod(item.method)}
                    <Text type='tertiary' size='small'>
                      {formatCount(item.orderCount)} {t('单')}
                    </Text>
                  </div>
                  <Text strong>{formatMoney(item.money)}</Text>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>
    </div>
  );

  const renderRiskCaseTable = () => (
    <>
      {renderRiskFilterPanel()}
      <Table
        columns={riskCaseColumns}
        dataSource={riskCases}
        loading={riskLoading}
        rowKey={(record) => String(record?.id || 0)}
        size='small'
        pagination={{
          currentPage: riskPage,
          pageSize: riskPageSize,
          total: riskTotal,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: (currentPage) => setRiskPage(currentPage),
          onPageSizeChange: (currentPageSize) => {
            setRiskPageSize(currentPageSize);
            setRiskPage(1);
          },
        }}
        scroll={{ x: '100%' }}
        empty={buildTableEmpty(t, '暂无异常单')}
      />
    </>
  );

  return (
    <>
      <Modal
        title={t('支付记录')}
        visible={visible}
        onCancel={onCancel}
        footer={null}
        size={isMobile ? 'full-width' : 'large'}
        style={isMobile ? undefined : { width: '1180px', maxWidth: '95vw' }}
      >
        {userIsAdmin ? (
          <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
            <Tabs.TabPane tab={t('支付记录')} itemKey='records'>
              {renderRecordsTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('对账看板')} itemKey='dashboard'>
              {renderDashboardBoard()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('异常单')} itemKey='risk'>
              {renderRiskCaseTable()}
            </Tabs.TabPane>
          </Tabs>
        ) : (
          renderRecordsTable()
        )}
      </Modal>

      <PaymentRiskCaseDetailModal
        visible={riskDetailVisible}
        riskCaseId={selectedRiskCaseId}
        initialRiskCase={selectedRiskCaseSeed}
        onCancel={() => setRiskDetailVisible(false)}
        onResolved={handleRiskCaseResolved}
        t={t}
      />
    </>
  );
};

export default TopupHistoryModal;
