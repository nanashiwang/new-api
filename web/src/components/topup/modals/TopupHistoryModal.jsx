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
  TextArea,
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

const EMPTY_WITHDRAWAL_FILTERS = {
  username: '',
  status: '',
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

const WITHDRAWAL_STATUS_CONFIG = {
  pending: { color: 'orange', label: '待审核' },
  approved: { color: 'green', label: '已通过' },
  rejected: { color: 'red', label: '已驳回' },
};

const WITHDRAWAL_STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '待审核', value: 'pending' },
  { label: '已通过', value: 'approved' },
  { label: '已驳回', value: 'rejected' },
];

const INVOICE_STATUS_CONFIG = {
  pending: { color: 'orange', label: '申请中' },
  invoiced: { color: 'green', label: '已开票' },
  rejected: { color: 'red', label: '已驳回' },
};

const INVOICE_STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '待审核', value: 'pending' },
  { label: '已开票', value: 'invoiced' },
  { label: '已驳回', value: 'rejected' },
];

const EMPTY_INVOICE_FILTERS = {
  username: '',
  status: '',
};

const EMPTY_INVOICE_FORM = {
  titleType: 'company',
  title: '',
  taxNumber: '',
  email: '',
  phone: '',
  remark: '',
};

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

function formatAmountCents(cents) {
  return `¥${(Number(cents || 0) / 100).toFixed(2)}`;
}

function getInvoiceOrderKey(record) {
  if (!record?.id) {
    return '';
  }
  return `${resolveOrderType(record)}-${record.id}`;
}

function getInvoiceItemKey(item) {
  if (!item?.order_id) {
    return '';
  }
  return `${item.order_type}-${item.order_id}`;
}

function getInvoiceOrderTypeLabel(orderType) {
  return RECORD_TYPE_MAP[orderType] || orderType || '-';
}

function getInvoiceStatusLabel(status) {
  return INVOICE_STATUS_CONFIG[status]?.label || status || '-';
}

function getInvoicePaymentLabel(paymentMethod) {
  return PAYMENT_METHOD_MAP[paymentMethod] || paymentMethod || '-';
}

function escapeHtml(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function displayValue(value) {
  return value === undefined || value === null || value === '' ? '-' : value;
}

function formatInvoiceTime(value) {
  return value ? timestamp2string(value) : '-';
}

function formatInvoiceCode(item) {
  return (
    item?.trade_no || `${item?.order_type || '-'}-${item?.order_id || '-'}`
  );
}

function formatInvoiceUser(invoice) {
  if (invoice?.username) {
    const displayName =
      invoice.display_name && invoice.display_name !== invoice.username
        ? ` / ${invoice.display_name}`
        : '';
    return `${invoice.username}${displayName}（ID: ${invoice.user_id || '-'}）`;
  }
  return invoice?.user_id ? `ID: ${invoice.user_id}` : '-';
}

function formatInvoiceReviewer(invoice) {
  if (invoice?.reviewer_username) {
    const displayName =
      invoice.reviewer_display_name &&
      invoice.reviewer_display_name !== invoice.reviewer_username
        ? ` / ${invoice.reviewer_display_name}`
        : '';
    return `${invoice.reviewer_username}${displayName}（ID: ${invoice.reviewer_user_id || '-'}）`;
  }
  return invoice?.reviewer_user_id ? `ID: ${invoice.reviewer_user_id}` : '-';
}

function buildInvoicePrintHtml(invoice) {
  const items = invoice?.items || [];
  const cell = (value) => escapeHtml(displayValue(value));
  const infoSection = (title, rows) => `
    <section>
      <h2>${cell(title)}</h2>
      <div class="info-grid">
        ${rows
          .map(
            ([label, value]) => `
              <div class="info-item">
                <div class="label">${cell(label)}</div>
                <div class="value">${cell(value)}</div>
              </div>
            `,
          )
          .join('')}
      </div>
    </section>
  `;
  const orderRows =
    items.length > 0
      ? items
          .map(
            (item, index) => `
              <tr>
                <td>${index + 1}</td>
                <td>${cell(getInvoiceOrderTypeLabel(item?.order_type))}</td>
                <td>${cell(item?.order_id)}</td>
                <td>${cell(formatInvoiceCode(item))}</td>
                <td>${cell(item?.product_name)}</td>
                <td>${cell(getInvoicePaymentLabel(item?.payment_method))}</td>
                <td>${cell(formatMoney(item?.money))}</td>
                <td>${cell(Number(item?.amount || 0) > 0 ? renderQuota(item.amount) : '-')}</td>
                <td>${cell(formatInvoiceTime(item?.complete_time || item?.create_time))}</td>
              </tr>
            `,
          )
          .join('')
      : '<tr><td colspan="9" class="empty">暂无订单明细</td></tr>';

  return `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>发票申请单 #${cell(invoice?.id)}</title>
  <style>
    body { margin: 0; padding: 28px; color: #111827; font: 13px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    h1 { margin: 0 0 6px; font-size: 24px; }
    h2 { margin: 24px 0 10px; font-size: 16px; border-bottom: 1px solid #e5e7eb; padding-bottom: 6px; }
    .muted { color: #6b7280; }
    .info-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
    .info-item { border: 1px solid #e5e7eb; border-radius: 8px; padding: 8px 10px; min-height: 42px; }
    .label { color: #6b7280; font-size: 12px; }
    .value { margin-top: 2px; word-break: break-all; }
    table { width: 100%; border-collapse: collapse; margin-top: 8px; }
    th, td { border: 1px solid #e5e7eb; padding: 7px 8px; text-align: left; vertical-align: top; word-break: break-all; }
    th { background: #f9fafb; }
    .empty { text-align: center; color: #6b7280; }
    @media print { body { padding: 0; } .no-print { display: none; } }
  </style>
</head>
<body>
  <h1>发票申请单 #${cell(invoice?.id)}</h1>
  <div class="muted">打印时间：${cell(timestamp2string(Math.floor(Date.now() / 1000)))}</div>
  ${infoSection('申请信息', [
    ['申请编号', `#${invoice?.id || '-'}`],
    ['状态', getInvoiceStatusLabel(invoice?.status)],
    ['申请时间', formatInvoiceTime(invoice?.created_at)],
    ['申请用户', formatInvoiceUser(invoice)],
    ['订单数量', `${items.length} 笔`],
    ['合计支付金额', formatMoney(invoice?.total_money)],
    [
      '合计额度',
      Number(invoice?.total_quota || 0) > 0
        ? renderQuota(invoice.total_quota)
        : '-',
    ],
  ])}
  ${infoSection('发票抬头与接收信息', [
    ['抬头类型', invoice?.title_type === 'company' ? '企业' : '个人'],
    ['抬头名称', invoice?.title],
    ['税号', invoice?.tax_number],
    ['接收邮箱', invoice?.email],
    ['手机号', invoice?.phone],
    ['用户备注', invoice?.remark],
  ])}
  <section>
    <h2>订单明细</h2>
    <table>
      <thead>
        <tr>
          <th>#</th>
          <th>订单类型</th>
          <th>平台订单ID</th>
          <th>订单编码/交易号</th>
          <th>商品/套餐</th>
          <th>支付渠道</th>
          <th>支付金额</th>
          <th>额度</th>
          <th>支付时间</th>
        </tr>
      </thead>
      <tbody>${orderRows}</tbody>
    </table>
  </section>
  ${infoSection('审核与开票信息', [
    ['审核人', formatInvoiceReviewer(invoice)],
    ['审核时间', formatInvoiceTime(invoice?.reviewed_at)],
    ['发票号/代码', invoice?.invoice_no],
    ['发票链接', invoice?.invoice_url],
    ['管理员备注/驳回原因', invoice?.admin_remark],
  ])}
</body>
</html>`;
}

function pickInvoiceStatus(current, next) {
  const priority = { invoiced: 3, pending: 2, rejected: 1 };
  if (!current) return next || '';
  if (!next) return current;
  return (priority[next] || 0) > (priority[current] || 0) ? next : current;
}

function maskAlipayAccount(account) {
  const text = String(account || '');
  if (text.length <= 4) return text ? '****' : '-';
  return `${text.slice(0, 2)}****${text.slice(-2)}`;
}

function buildTableEmpty(t, description) {
  return (
    <Empty
      image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
      darkModeImage={
        <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
      }
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

const TopupHistoryModal = ({
  visible,
  onCancel,
  t,
  initialTab = 'records',
}) => {
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
  const [dashboardDateRange, setDashboardDateRange] = useState(() =>
    createDashboardDateRange('today'),
  );
  const [dashboardRankLimit, setDashboardRankLimit] = useState(10);

  const [riskLoading, setRiskLoading] = useState(false);
  const [riskCases, setRiskCases] = useState([]);
  const [riskTotal, setRiskTotal] = useState(0);
  const [riskPage, setRiskPage] = useState(1);
  const [riskPageSize, setRiskPageSize] = useState(10);
  const [riskFilters, setRiskFilters] = useState(EMPTY_RISK_FILTERS);
  const [riskAppliedFilters, setRiskAppliedFilters] =
    useState(EMPTY_RISK_FILTERS);
  const [riskDetailVisible, setRiskDetailVisible] = useState(false);
  const [selectedRiskCaseId, setSelectedRiskCaseId] = useState(0);
  const [selectedRiskCaseSeed, setSelectedRiskCaseSeed] = useState(null);

  const [withdrawalLoading, setWithdrawalLoading] = useState(false);
  const [withdrawals, setWithdrawals] = useState([]);
  const [withdrawalTotal, setWithdrawalTotal] = useState(0);
  const [withdrawalPage, setWithdrawalPage] = useState(1);
  const [withdrawalPageSize, setWithdrawalPageSize] = useState(10);
  const [withdrawalFilters, setWithdrawalFilters] = useState(
    EMPTY_WITHDRAWAL_FILTERS,
  );
  const [withdrawalAppliedFilters, setWithdrawalAppliedFilters] = useState(
    EMPTY_WITHDRAWAL_FILTERS,
  );
  const [reviewState, setReviewState] = useState({
    visible: false,
    action: null,
    record: null,
  });
  const [reviewRemark, setReviewRemark] = useState('');
  const [reviewSubmitting, setReviewSubmitting] = useState(false);

  const [invoiceLoading, setInvoiceLoading] = useState(false);
  const [invoices, setInvoices] = useState([]);
  const [invoiceTotal, setInvoiceTotal] = useState(0);
  const [invoicePage, setInvoicePage] = useState(1);
  const [invoicePageSize, setInvoicePageSize] = useState(10);
  const [invoiceFilters, setInvoiceFilters] = useState(EMPTY_INVOICE_FILTERS);
  const [invoiceAppliedFilters, setInvoiceAppliedFilters] = useState(
    EMPTY_INVOICE_FILTERS,
  );
  const [invoiceStatusMap, setInvoiceStatusMap] = useState({});
  const [invoiceApplyVisible, setInvoiceApplyVisible] = useState(false);
  const [eligibleInvoiceOrders, setEligibleInvoiceOrders] = useState([]);
  const [eligibleInvoiceLoading, setEligibleInvoiceLoading] = useState(false);
  const [selectedInvoiceOrderKeys, setSelectedInvoiceOrderKeys] = useState([]);
  const [invoiceForm, setInvoiceForm] = useState(EMPTY_INVOICE_FORM);
  const [invoiceSubmitting, setInvoiceSubmitting] = useState(false);
  const [invoiceReviewState, setInvoiceReviewState] = useState({
    visible: false,
    action: null,
    record: null,
  });
  const [invoiceReviewForm, setInvoiceReviewForm] = useState({
    invoiceNo: '',
    invoiceUrl: '',
    adminRemark: '',
  });
  const [invoiceReviewSubmitting, setInvoiceReviewSubmitting] = useState(false);
  const [invoiceDetailVisible, setInvoiceDetailVisible] = useState(false);
  const [invoiceDetailLoading, setInvoiceDetailLoading] = useState(false);
  const [invoiceDetail, setInvoiceDetail] = useState(null);

  const isMobile = useIsMobile();
  const userIsAdmin = useMemo(() => isAdmin(), []);

  const loadTopups = async (currentPage, currentPageSize, currentFilters) => {
    setLoading(true);
    try {
      const base = userIsAdmin
        ? '/api/user/payment-records'
        : '/api/user/payment-records/self';
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

  const loadRiskCases = async (
    currentPage,
    currentPageSize,
    currentFilters,
  ) => {
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

      const res = await API.get(
        `/api/user/payment-risk-cases?${searchParams.toString()}`,
      );
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

  const loadWithdrawals = async (
    currentPage,
    currentPageSize,
    currentFilters,
    viewMode = 'self',
  ) => {
    setWithdrawalLoading(true);
    try {
      const isAdminView = viewMode === 'admin' && userIsAdmin;
      const base = isAdminView
        ? '/api/user/aff-withdrawals'
        : '/api/user/aff-withdrawals/self';
      const searchParams = new URLSearchParams({
        p: String(currentPage),
        page_size: String(currentPageSize),
      });
      if (currentFilters.status) {
        searchParams.set('status', currentFilters.status);
      }
      if (isAdminView && currentFilters.username) {
        searchParams.set('username', currentFilters.username.trim());
      }

      const res = await API.get(`${base}?${searchParams.toString()}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载提现记录失败') });
        return;
      }
      setWithdrawals(data?.items || []);
      setWithdrawalTotal(data?.total || 0);
    } catch (error) {
      Toast.error({ content: t('加载提现记录失败') });
    } finally {
      setWithdrawalLoading(false);
    }
  };

  const loadInvoices = async (
    currentPage,
    currentPageSize,
    currentFilters,
    viewMode = 'self',
  ) => {
    setInvoiceLoading(true);
    try {
      const isAdminView = viewMode === 'admin' && userIsAdmin;
      const base = isAdminView
        ? '/api/user/invoices'
        : '/api/user/invoices/self';
      const searchParams = new URLSearchParams({
        p: String(currentPage),
        page_size: String(currentPageSize),
      });
      if (currentFilters.status) {
        searchParams.set('status', currentFilters.status);
      }
      if (isAdminView && currentFilters.username) {
        searchParams.set('username', currentFilters.username.trim());
      }

      const res = await API.get(`${base}?${searchParams.toString()}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载发票申请失败') });
        return;
      }
      setInvoices(data?.items || []);
      setInvoiceTotal(data?.total || 0);
    } catch (error) {
      Toast.error({ content: t('加载发票申请失败') });
    } finally {
      setInvoiceLoading(false);
    }
  };

  const loadInvoiceStatusMap = async () => {
    try {
      const base = userIsAdmin
        ? '/api/user/invoices'
        : '/api/user/invoices/self';
      const res = await API.get(`${base}?p=1&page_size=100`);
      const { success, data } = res.data || {};
      if (!success) {
        return;
      }
      const nextMap = {};
      (data?.items || []).forEach((request) => {
        (request?.items || []).forEach((item) => {
          const key = getInvoiceItemKey(item);
          if (!key) return;
          nextMap[key] = pickInvoiceStatus(nextMap[key], request.status);
        });
      });
      setInvoiceStatusMap(nextMap);
    } catch (error) {
      // 状态展示失败不阻塞支付记录。
    }
  };

  const loadEligibleInvoiceOrders = async (prefillRecord = null) => {
    setEligibleInvoiceLoading(true);
    try {
      const res = await API.get(
        '/api/user/invoices/eligible-orders?p=1&page_size=100',
      );
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载可开票订单失败') });
        return;
      }
      const items = data?.items || [];
      setEligibleInvoiceOrders(items);
      if (prefillRecord) {
        const key = getInvoiceOrderKey(prefillRecord);
        setSelectedInvoiceOrderKeys(
          items.some((item) => getInvoiceOrderKey(item) === key) ? [key] : [],
        );
      }
    } catch (error) {
      Toast.error({ content: t('加载可开票订单失败') });
    } finally {
      setEligibleInvoiceLoading(false);
    }
  };

  const loadDashboard = async (
    dateRange = dashboardDateRange,
    limit = dashboardRankLimit,
  ) => {
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
        Toast.error({
          content: t(rankingsPayload.message || '加载充值榜单失败'),
        });
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

  const refreshWithdrawals = async () => {
    const mode = activeTab === 'withdrawals' ? 'admin' : 'self';
    await loadWithdrawals(
      withdrawalPage,
      withdrawalPageSize,
      withdrawalAppliedFilters,
      mode,
    );
  };

  const refreshInvoices = async () => {
    const mode = activeTab === 'invoices' && userIsAdmin ? 'admin' : 'self';
    await loadInvoices(
      invoicePage,
      invoicePageSize,
      invoiceAppliedFilters,
      mode,
    );
    await loadInvoiceStatusMap();
  };

  useEffect(() => {
    if (visible && initialTab) {
      const safeTab =
        !userIsAdmin && initialTab === 'withdrawals'
          ? 'my-withdrawals'
          : initialTab;
      setActiveTab(safeTab);
    }
  }, [visible, initialTab, userIsAdmin]);

  useEffect(() => {
    if (!visible) {
      return;
    }
    if (activeTab === 'records') {
      loadTopups(page, pageSize, appliedFilters);
      loadInvoiceStatusMap();
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
  }, [
    visible,
    activeTab,
    riskPage,
    riskPageSize,
    riskAppliedFilters,
    userIsAdmin,
  ]);

  useEffect(() => {
    if (!visible) {
      return;
    }
    if (activeTab !== 'withdrawals' && activeTab !== 'my-withdrawals') {
      return;
    }
    const mode = activeTab === 'withdrawals' ? 'admin' : 'self';
    loadWithdrawals(
      withdrawalPage,
      withdrawalPageSize,
      withdrawalAppliedFilters,
      mode,
    );
  }, [
    visible,
    activeTab,
    withdrawalPage,
    withdrawalPageSize,
    withdrawalAppliedFilters,
    userIsAdmin,
  ]);

  useEffect(() => {
    if (!visible || activeTab !== 'invoices') {
      return;
    }
    const mode = userIsAdmin ? 'admin' : 'self';
    loadInvoices(invoicePage, invoicePageSize, invoiceAppliedFilters, mode);
  }, [
    visible,
    activeTab,
    invoicePage,
    invoicePageSize,
    invoiceAppliedFilters,
    userIsAdmin,
  ]);

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

  const handleWithdrawalFilterChange = (key, value) => {
    setWithdrawalFilters((prev) => ({
      ...prev,
      [key]: value || '',
    }));
  };

  const applyWithdrawalFilters = (nextFilters = withdrawalFilters) => {
    setWithdrawalPage(1);
    setWithdrawalAppliedFilters({
      ...nextFilters,
      username: nextFilters.username.trim(),
    });
  };

  const resetWithdrawalFilters = () => {
    setWithdrawalPage(1);
    setWithdrawalFilters(EMPTY_WITHDRAWAL_FILTERS);
    setWithdrawalAppliedFilters(EMPTY_WITHDRAWAL_FILTERS);
  };

  const handleInvoiceFilterChange = (key, value) => {
    setInvoiceFilters((prev) => ({
      ...prev,
      [key]: value || '',
    }));
  };

  const applyInvoiceFilters = (nextFilters = invoiceFilters) => {
    setInvoicePage(1);
    setInvoiceAppliedFilters({
      ...nextFilters,
      username: nextFilters.username.trim(),
    });
  };

  const resetInvoiceFilters = () => {
    setInvoicePage(1);
    setInvoiceFilters(EMPTY_INVOICE_FILTERS);
    setInvoiceAppliedFilters(EMPTY_INVOICE_FILTERS);
  };

  const activeFilterTags = useMemo(() => {
    const tags = [];
    if (appliedFilters.username) {
      tags.push({
        key: 'username',
        label: `ID/用户名: ${appliedFilters.username}`,
      });
    }
    if (appliedFilters.status) {
      const found = STATUS_OPTIONS.find(
        (option) => option.value === appliedFilters.status,
      );
      tags.push({
        key: 'status',
        label: `状态: ${found ? found.label : appliedFilters.status}`,
      });
    }
    if (appliedFilters.paymentMethod) {
      const found = PAYMENT_OPTIONS.find(
        (option) => option.value === appliedFilters.paymentMethod,
      );
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
    if (activeTab === 'withdrawals') {
      const nextFilters = { ...withdrawalFilters, username };
      setWithdrawalFilters(nextFilters);
      applyWithdrawalFilters(nextFilters);
      return;
    }
    if (activeTab === 'invoices') {
      const nextFilters = { ...invoiceFilters, username };
      setInvoiceFilters(nextFilters);
      applyInvoiceFilters(nextFilters);
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

  const handleReviewWithdrawal = async (record, action, remark = '') => {
    const id = Number(record?.id || 0);
    if (!id) return false;
    try {
      const res = await API.post(`/api/user/aff-withdrawals/${id}/${action}`, {
        admin_remark: remark,
      });
      const { success, message } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '审核提现失败') });
        return false;
      }
      Toast.success({
        content: t(action === 'approve' ? '提现已通过' : '提现已驳回'),
      });
      await refreshWithdrawals();
      return true;
    } catch (error) {
      const msg =
        error?.response?.data?.message || error?.message || '审核提现失败';
      Toast.error({ content: t(msg) });
      return false;
    }
  };

  const confirmApproveWithdrawal = (record) => {
    setReviewRemark('');
    setReviewState({ visible: true, action: 'approve', record });
  };

  const confirmRejectWithdrawal = (record) => {
    setReviewRemark('');
    setReviewState({ visible: true, action: 'reject', record });
  };

  const closeReviewModal = () => {
    setReviewState({ visible: false, action: null, record: null });
    setReviewRemark('');
  };

  const submitReviewModal = async () => {
    const { record, action } = reviewState;
    if (!record || !action) return;
    setReviewSubmitting(true);
    try {
      const ok = await handleReviewWithdrawal(
        record,
        action,
        reviewRemark.trim(),
      );
      if (ok) closeReviewModal();
    } finally {
      setReviewSubmitting(false);
    }
  };

  const openInvoiceApplyModal = async (record = null) => {
    setInvoiceForm(EMPTY_INVOICE_FORM);
    setSelectedInvoiceOrderKeys(record ? [getInvoiceOrderKey(record)] : []);
    setInvoiceApplyVisible(true);
    await loadEligibleInvoiceOrders(record);
  };

  const closeInvoiceApplyModal = () => {
    setInvoiceApplyVisible(false);
    setSelectedInvoiceOrderKeys([]);
    setEligibleInvoiceOrders([]);
    setInvoiceForm(EMPTY_INVOICE_FORM);
  };

  const handleInvoiceFormChange = (key, value) => {
    setInvoiceForm((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const selectedInvoiceOrders = useMemo(() => {
    const selected = new Set(selectedInvoiceOrderKeys);
    return eligibleInvoiceOrders.filter((item) =>
      selected.has(getInvoiceOrderKey(item)),
    );
  }, [eligibleInvoiceOrders, selectedInvoiceOrderKeys]);

  const selectedInvoiceSummary = useMemo(() => {
    return selectedInvoiceOrders.reduce(
      (summary, item) => ({
        money: summary.money + Number(item?.money || 0),
        quota: summary.quota + Number(item?.amount || 0),
      }),
      { money: 0, quota: 0 },
    );
  }, [selectedInvoiceOrders]);

  const submitInvoiceRequest = async () => {
    const form = {
      titleType: invoiceForm.titleType,
      title: invoiceForm.title.trim(),
      taxNumber: invoiceForm.taxNumber.trim(),
      email: invoiceForm.email.trim(),
      phone: invoiceForm.phone.trim(),
      remark: invoiceForm.remark.trim(),
    };
    if (selectedInvoiceOrders.length === 0) {
      Toast.error({ content: t('请选择需要开票的订单') });
      return;
    }
    if (!form.title) {
      Toast.error({ content: t('发票抬头不能为空') });
      return;
    }
    if (form.titleType === 'company' && !form.taxNumber) {
      Toast.error({ content: t('企业抬头需要填写税号') });
      return;
    }
    if (!form.email) {
      Toast.error({ content: t('接收邮箱不能为空') });
      return;
    }

    setInvoiceSubmitting(true);
    try {
      const res = await API.post('/api/user/invoices', {
        title_type: form.titleType,
        title: form.title,
        tax_number: form.taxNumber,
        email: form.email,
        phone: form.phone,
        remark: form.remark,
        orders: selectedInvoiceOrders.map((item) => ({
          order_type: resolveOrderType(item),
          id: item.id,
        })),
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '提交发票申请失败') });
        return;
      }
      Toast.success({ content: t('发票申请已提交') });
      closeInvoiceApplyModal();
      if (data?.id) {
        setInvoiceDetail(data);
        setInvoiceDetailVisible(true);
      }
      await Promise.all([refreshInvoices(), refreshRecords()]);
    } catch (error) {
      Toast.error({ content: t('提交发票申请失败') });
    } finally {
      setInvoiceSubmitting(false);
    }
  };

  const openInvoiceReviewModal = (record, action) => {
    setInvoiceReviewState({ visible: true, action, record });
    setInvoiceReviewForm({
      invoiceNo: record?.invoice_no || '',
      invoiceUrl: record?.invoice_url || '',
      adminRemark: '',
    });
  };

  const closeInvoiceReviewModal = () => {
    setInvoiceReviewState({ visible: false, action: null, record: null });
    setInvoiceReviewForm({ invoiceNo: '', invoiceUrl: '', adminRemark: '' });
  };

  const submitInvoiceReview = async () => {
    const { record, action } = invoiceReviewState;
    const id = Number(record?.id || 0);
    if (!id || !action) return;
    setInvoiceReviewSubmitting(true);
    try {
      const payload =
        action === 'approve'
          ? {
              invoice_no: invoiceReviewForm.invoiceNo.trim(),
              invoice_url: invoiceReviewForm.invoiceUrl.trim(),
              admin_remark: invoiceReviewForm.adminRemark.trim(),
            }
          : {
              admin_remark: invoiceReviewForm.adminRemark.trim(),
            };
      const res = await API.post(`/api/user/invoices/${id}/${action}`, payload);
      const { success, message } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '审核发票失败') });
        return;
      }
      Toast.success({
        content: t(action === 'approve' ? '发票已通过' : '发票已驳回'),
      });
      closeInvoiceReviewModal();
      await refreshInvoices();
    } catch (error) {
      Toast.error({ content: t('审核发票失败') });
    } finally {
      setInvoiceReviewSubmitting(false);
    }
  };

  const openInvoiceDetail = async (record) => {
    const id = Number(record?.id || 0);
    if (!id) {
      return;
    }

    setInvoiceDetail(record || null);
    setInvoiceDetailVisible(true);
    setInvoiceDetailLoading(true);
    try {
      const endpoint = userIsAdmin
        ? `/api/user/invoices/${id}`
        : `/api/user/invoices/self/${id}`;
      const res = await API.get(endpoint);
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载发票详情失败') });
        return;
      }
      setInvoiceDetail(data || record || null);
    } catch (error) {
      Toast.error({ content: t('加载发票详情失败') });
    } finally {
      setInvoiceDetailLoading(false);
    }
  };

  const closeInvoiceDetail = () => {
    setInvoiceDetailVisible(false);
    setInvoiceDetail(null);
  };

  const printInvoiceDetail = () => {
    if (!invoiceDetail) {
      return;
    }
    const printWindow = window.open('', '_blank', 'width=960,height=720');
    if (!printWindow) {
      Toast.error({ content: t('浏览器阻止了打印窗口') });
      return;
    }
    printWindow.opener = null;
    printWindow.document.open();
    printWindow.document.write(buildInvoicePrintHtml(invoiceDetail));
    printWindow.document.close();
    printWindow.focus();
    setTimeout(() => {
      printWindow.print();
    }, 200);
  };

  const renderStatusBadge = (status) => {
    const config = STATUS_CONFIG[status] || {
      type: 'primary',
      label: status || '-',
    };
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
    const config = RISK_STATUS_CONFIG[status] || {
      color: 'grey',
      label: status || '-',
    };
    return (
      <Tag color={config.color} shape='circle' size='small'>
        {t(config.label)}
      </Tag>
    );
  };

  const renderRiskReason = (reason) =>
    t(RISK_REASON_MAP[reason] || reason || '-');

  const renderWithdrawalStatusTag = (status) => {
    const config = WITHDRAWAL_STATUS_CONFIG[status] || {
      color: 'grey',
      label: status || '-',
    };
    return (
      <Tag color={config.color} shape='circle' size='small'>
        {t(config.label)}
      </Tag>
    );
  };

  const renderInvoiceStatusTag = (status) => {
    const config = INVOICE_STATUS_CONFIG[status] || {
      color: 'grey',
      label: status || '-',
    };
    return (
      <Tag color={config.color} shape='circle' size='small'>
        {t(config.label)}
      </Tag>
    );
  };

  const renderInvoiceRecordStatus = (record) => {
    if (record?.status !== 'success') {
      return <Text type='tertiary'>-</Text>;
    }
    const status = invoiceStatusMap[getInvoiceOrderKey(record)];
    if (!status) {
      if (userIsAdmin) {
        return <Text type='tertiary'>-</Text>;
      }
      return (
        <Button
          size='small'
          type='primary'
          theme='outline'
          onClick={() => openInvoiceApplyModal(record)}
        >
          {t('申请发票')}
        </Button>
      );
    }
    return (
      <Space wrap>
        {renderInvoiceStatusTag(status)}
        {!userIsAdmin && status === 'rejected' ? (
          <Button
            size='small'
            theme='outline'
            onClick={() => openInvoiceApplyModal(record)}
          >
            {t('重新申请')}
          </Button>
        ) : null}
      </Space>
    );
  };

  const isSellableTokenPurchase = (record) =>
    record?.record_type === 'sellable_token_purchase';

  const isSubscriptionTopup = (record) =>
    resolveOrderType(record) === 'subscription';

  const renderRecordType = (record) => {
    if (isSellableTokenPurchase(record)) {
      return (
        <div className='min-w-0'>
          <Tag color='blue' shape='circle' size='small'>
            {t('钱包购买')}
          </Tag>
          <div className='mt-1'>
            <Text
              ellipsis={{ showTooltip: true }}
              style={{ maxWidth: 180, display: 'inline-block' }}
            >
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
        ellipsis={{
          showTooltip: { opts: { style: { wordBreak: 'break-all' } } },
        }}
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
                {displayName ? (
                  <Text type='tertiary'>{displayName}</Text>
                ) : null}
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
            return (
              <Text type='danger'>{renderQuota(record?.amount ?? 0)}</Text>
            );
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
        title: t('发票'),
        key: 'invoice',
        width: 130,
        render: (_, record) => renderInvoiceRecordStatus(record),
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

          if (
            record?.record_type === 'topup' &&
            record?.status === 'pending' &&
            !record?.risk_case_id
          ) {
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
                    record_type:
                      resolveRiskRecordType(record) || resolveOrderType(record),
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
  }, [userIsAdmin, filters, riskFilters, invoiceStatusMap]);

  const dashboardData = useMemo(
    () => normalizeDashboardStats(dashboardStats),
    [dashboardStats],
  );

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
    const items = Object.entries(dashboardData.payment_methods || {}).map(
      ([method, stats]) => ({
        method,
        money: Number(stats?.money || 0),
        orderCount: Number(stats?.order_count || 0),
      }),
    );
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
                {record.display_name ? (
                  <Text type='tertiary'>{record.display_name}</Text>
                ) : null}
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
        render: (_, record) => (
          <Text type='success'>{formatMoney(record?.success_money)}</Text>
        ),
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
        render: (_, record) => (
          <Text type='warning'>{formatMoney(record?.pending_money)}</Text>
        ),
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
                {t(
                  RECORD_TYPE_MAP[record.record_type] ||
                    record.record_type ||
                    '-',
                )}
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
                {record.display_name ? (
                  <Text type='tertiary'>{record.display_name}</Text>
                ) : null}
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
              {t('订单状态')}:{' '}
              {t(
                STATUS_CONFIG[record.order_status]?.label ||
                  record.order_status ||
                  '-',
              )}
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
          <Button
            size='small'
            theme='outline'
            onClick={() => openRiskCaseDetail(record.id, record)}
          >
            {t('查看详情')}
          </Button>
        ),
      },
    ];
  }, [riskFilters]);

  const withdrawalColumns = useMemo(() => {
    const isReviewMode = userIsAdmin && activeTab === 'withdrawals';
    const columns = [
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: renderWithdrawalStatusTag,
      },
    ];

    if (isReviewMode) {
      columns.push({
        title: t('用户'),
        key: 'username',
        width: 180,
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
                <Text type='tertiary' size='small'>
                  ID: {record.user_id}
                </Text>
                <Text
                  link
                  size='small'
                  style={{ cursor: 'pointer' }}
                  onClick={() => filterByUsername(record.username)}
                >
                  {record.username}
                </Text>
                {record.display_name ? (
                  <Text type='tertiary'>{record.display_name}</Text>
                ) : null}
              </div>
            </Space>
          );
        },
      });
    }

    columns.push(
      {
        title: t('提现额度'),
        dataIndex: 'quota',
        key: 'quota',
        width: 130,
        render: (quota) => <Text strong>{renderQuota(quota || 0)}</Text>,
      },
      {
        title: t('预计到账'),
        dataIndex: 'amount_cents',
        key: 'amount_cents',
        width: 120,
        render: (amountCents) => (
          <Text type='danger'>{formatAmountCents(amountCents)}</Text>
        ),
      },
      {
        title: t('支付宝信息'),
        key: 'alipay',
        width: 220,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text copyable={userIsAdmin}>
              {userIsAdmin
                ? record?.alipay_account || '-'
                : maskAlipayAccount(record?.alipay_account)}
            </Text>
            <Text type='tertiary' size='small'>
              {record?.alipay_name || '-'}
            </Text>
          </div>
        ),
      },
      {
        title: t('提交时间'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 150,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('审核时间'),
        dataIndex: 'reviewed_at',
        key: 'reviewed_at',
        width: 150,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('备注'),
        dataIndex: 'admin_remark',
        key: 'admin_remark',
        render: (value) => value || <Text type='tertiary'>-</Text>,
      },
    );

    if (isReviewMode) {
      columns.push({
        title: t('操作'),
        key: 'action',
        width: 150,
        render: (_, record) => {
          if (record?.status !== 'pending') {
            return <Text type='tertiary'>-</Text>;
          }
          return (
            <Space wrap>
              <Button
                size='small'
                type='primary'
                theme='outline'
                onClick={() => confirmApproveWithdrawal(record)}
              >
                {t('通过')}
              </Button>
              <Button
                size='small'
                type='danger'
                theme='outline'
                onClick={() => confirmRejectWithdrawal(record)}
              >
                {t('驳回')}
              </Button>
            </Space>
          );
        },
      });
    }

    return columns;
  }, [
    userIsAdmin,
    activeTab,
    withdrawalFilters,
    withdrawalPage,
    withdrawalPageSize,
    withdrawalAppliedFilters,
    t,
  ]);

  const invoiceOrderColumns = useMemo(
    () => [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
        key: 'trade_no',
        width: 220,
        render: (_, record) => renderRecordNo(record),
      },
      {
        title: t('类型 / 商品'),
        key: 'record_type',
        render: (_, record) => renderRecordType(record),
      },
      {
        title: t('支付金额'),
        key: 'money',
        width: 140,
        render: (_, record) =>
          isSellableTokenPurchase(record) ? (
            <Text type='danger'>{renderQuota(record?.amount || 0)}</Text>
          ) : (
            <Text type='danger'>{formatMoney(record?.money)}</Text>
          ),
      },
      {
        title: t('支付时间'),
        dataIndex: 'complete_time',
        key: 'complete_time',
        width: 160,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
    ],
    [t],
  );

  const invoiceDetailItemColumns = useMemo(
    () => [
      {
        title: t('订单类型'),
        dataIndex: 'order_type',
        key: 'order_type',
        width: 110,
        render: (value) => (
          <Tag shape='circle' size='small' color='blue'>
            {t(getInvoiceOrderTypeLabel(value))}
          </Tag>
        ),
      },
      {
        title: t('平台订单ID'),
        dataIndex: 'order_id',
        key: 'order_id',
        width: 100,
      },
      {
        title: t('订单编码/交易号'),
        key: 'trade_no',
        width: 220,
        render: (_, item) => (
          <Text copyable ellipsis style={{ maxWidth: 190 }}>
            {formatInvoiceCode(item)}
          </Text>
        ),
      },
      {
        title: t('商品/套餐'),
        dataIndex: 'product_name',
        key: 'product_name',
        width: 160,
        render: (value) => value || <Text type='tertiary'>-</Text>,
      },
      {
        title: t('支付渠道'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        width: 120,
        render: renderPaymentMethod,
      },
      {
        title: t('支付金额'),
        dataIndex: 'money',
        key: 'money',
        width: 120,
        render: (value) => <Text type='danger'>{formatMoney(value)}</Text>,
      },
      {
        title: t('额度'),
        dataIndex: 'amount',
        key: 'amount',
        width: 120,
        render: (value) =>
          Number(value || 0) > 0 ? (
            <Text>{renderQuota(value)}</Text>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
      {
        title: t('支付时间'),
        key: 'complete_time',
        width: 160,
        render: (_, item) =>
          formatInvoiceTime(item?.complete_time || item?.create_time),
      },
    ],
    [t],
  );

  const invoiceColumns = useMemo(() => {
    const isReviewMode = userIsAdmin && activeTab === 'invoices';
    const columns = [
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: renderInvoiceStatusTag,
      },
    ];

    if (isReviewMode) {
      columns.push({
        title: t('用户'),
        key: 'username',
        width: 180,
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
                <Text type='tertiary' size='small'>
                  ID: {record.user_id}
                </Text>
                <Text
                  link
                  size='small'
                  style={{ cursor: 'pointer' }}
                  onClick={() => filterByUsername(record.username)}
                >
                  {record.username}
                </Text>
                {record.display_name ? (
                  <Text type='tertiary'>{record.display_name}</Text>
                ) : null}
              </div>
            </Space>
          );
        },
      });
    }

    columns.push(
      {
        title: t('发票抬头'),
        key: 'title',
        width: 220,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Space wrap>
              <Tag shape='circle' size='small' color='blue'>
                {t(record?.title_type === 'company' ? '企业' : '个人')}
              </Tag>
              <Text strong>{record?.title || '-'}</Text>
            </Space>
            {record?.tax_number ? (
              <Text type='tertiary' size='small' copyable>
                {record.tax_number}
              </Text>
            ) : null}
          </div>
        ),
      },
      {
        title: t('申请金额'),
        key: 'total',
        width: 140,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text type='danger'>{formatMoney(record?.total_money)}</Text>
            {Number(record?.total_quota || 0) > 0 ? (
              <Text type='tertiary' size='small'>
                {renderQuota(record.total_quota)}
              </Text>
            ) : null}
          </div>
        ),
      },
      {
        title: t('订单'),
        key: 'items',
        width: 260,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text type='tertiary' size='small'>
              {(record?.items || []).length} {t('笔订单')}
            </Text>
            {(record?.items || []).slice(0, 2).map((item) => (
              <Text key={item.id} copyable ellipsis style={{ maxWidth: 220 }}>
                {item.trade_no || `${item.order_type}-${item.order_id}`}
              </Text>
            ))}
          </div>
        ),
      },
      {
        title: t('接收方式'),
        key: 'contact',
        width: 180,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text copyable>{record?.email || '-'}</Text>
            {record?.phone ? (
              <Text type='tertiary' size='small'>
                {record.phone}
              </Text>
            ) : null}
          </div>
        ),
      },
      {
        title: t('发票信息'),
        key: 'invoice_info',
        width: 220,
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            {record?.invoice_no ? (
              <Text copyable>{record.invoice_no}</Text>
            ) : (
              <Text type='tertiary'>-</Text>
            )}
            {record?.invoice_url ? (
              <a href={record.invoice_url} target='_blank' rel='noreferrer'>
                {t('查看发票')}
              </a>
            ) : null}
          </div>
        ),
      },
      {
        title: t('提交时间'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 150,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('备注'),
        key: 'remark',
        render: (_, record) => record?.admin_remark || record?.remark || '-',
      },
    );

    columns.push({
      title: t('操作'),
      key: 'action',
      width: isReviewMode ? 230 : 110,
      render: (_, record) => (
        <Space wrap>
          <Button
            size='small'
            theme='outline'
            onClick={() => openInvoiceDetail(record)}
          >
            {t('详情/打印')}
          </Button>
          {isReviewMode && record?.status === 'pending' ? (
            <>
              <Button
                size='small'
                type='primary'
                theme='outline'
                onClick={() => openInvoiceReviewModal(record, 'approve')}
              >
                {t('通过')}
              </Button>
              <Button
                size='small'
                type='danger'
                theme='outline'
                onClick={() => openInvoiceReviewModal(record, 'reject')}
              >
                {t('驳回')}
              </Button>
            </>
          ) : null}
        </Space>
      ),
    });

    return columns;
  }, [userIsAdmin, activeTab, t]);

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
              <div
                className='text-xs mb-1'
                style={{ color: 'var(--semi-color-text-2)' }}
              >
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
            <div
              className='text-xs mb-1'
              style={{ color: 'var(--semi-color-text-2)' }}
            >
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
            <div
              className='text-xs mb-1'
              style={{ color: 'var(--semi-color-text-2)' }}
            >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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

  const renderWithdrawalFilterPanel = () => (
    <div className='mb-3'>
      <div
        className='rounded-lg p-3 flex flex-wrap items-end gap-3'
        style={{
          background: 'var(--semi-color-fill-0)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        {userIsAdmin && activeTab === 'withdrawals' ? (
          <div style={{ minWidth: 180, flex: 1 }}>
            <div
              className='text-xs mb-1'
              style={{ color: 'var(--semi-color-text-2)' }}
            >
              ID/用户名
            </div>
            <Input
              placeholder={t('ID/用户名')}
              value={withdrawalFilters.username}
              onChange={(value) =>
                handleWithdrawalFilterChange('username', value)
              }
              onEnterPress={() => applyWithdrawalFilters()}
              showClear
              size='small'
            />
          </div>
        ) : null}
        <div style={{ minWidth: 140, flex: 1 }}>
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
            {t('状态')}
          </div>
          <Select
            value={withdrawalFilters.status}
            optionList={WITHDRAWAL_STATUS_OPTIONS.map((item) => ({
              ...item,
              label: t(item.label),
            }))}
            onChange={(value) => handleWithdrawalFilterChange('status', value)}
            size='small'
            style={{ width: '100%' }}
          />
        </div>
        <Space>
          <Button type='primary' onClick={() => applyWithdrawalFilters()}>
            {t('搜索')}
          </Button>
          <Button
            theme='borderless'
            type='tertiary'
            onClick={resetWithdrawalFilters}
          >
            {t('重置')}
          </Button>
        </Space>
      </div>
    </div>
  );

  const renderInvoiceFilterPanel = () => (
    <div className='mb-3 flex flex-wrap items-end gap-3'>
      {userIsAdmin ? (
        <div style={{ minWidth: 180, flex: 1 }}>
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
            ID/用户名
          </div>
          <Input
            placeholder={t('ID/用户名')}
            value={invoiceFilters.username}
            onChange={(value) => handleInvoiceFilterChange('username', value)}
            onEnterPress={() => applyInvoiceFilters()}
            showClear
            size='small'
          />
        </div>
      ) : null}
      <div style={{ minWidth: 140, flex: 1 }}>
        <div
          className='text-xs mb-1'
          style={{ color: 'var(--semi-color-text-2)' }}
        >
          {t('状态')}
        </div>
        <Select
          value={invoiceFilters.status}
          optionList={INVOICE_STATUS_OPTIONS.map((item) => ({
            ...item,
            label: t(item.label),
          }))}
          onChange={(value) => handleInvoiceFilterChange('status', value)}
          size='small'
          style={{ width: '100%' }}
        />
      </div>
      <Space>
        <Button type='primary' onClick={() => applyInvoiceFilters()}>
          {t('搜索')}
        </Button>
        <Button
          theme='borderless'
          type='tertiary'
          onClick={resetInvoiceFilters}
        >
          {t('重置')}
        </Button>
        {!userIsAdmin ? (
          <Button
            type='primary'
            theme='outline'
            onClick={() => openInvoiceApplyModal()}
          >
            {t('申请发票')}
          </Button>
        ) : null}
      </Space>
    </div>
  );

  const renderRecordsTable = () => (
    <>
      {renderRecordFilterPanel()}
      <Table
        columns={recordColumns}
        dataSource={topups}
        loading={loading}
        rowKey={(record) =>
          `${record?.record_type || 'topup'}-${record?.id || '0'}`
        }
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
          <div
            className='text-xs mb-1'
            style={{ color: 'var(--semi-color-text-2)' }}
          >
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
        <Button
          type='primary'
          loading={dashboardLoading}
          onClick={refreshDashboard}
        >
          {t('刷新')}
        </Button>
      </div>

      <div
        className='grid gap-3'
        style={{
          gridTemplateColumns: isMobile
            ? '1fr'
            : 'repeat(auto-fit, minmax(180px, 1fr))',
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
          gridTemplateColumns: isMobile
            ? '1fr'
            : 'minmax(0, 1.6fr) minmax(280px, 1fr)',
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
            rowKey={(record) =>
              String(record?.user_id || record?.username || '')
            }
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
                <div
                  key={item.method}
                  className='flex items-center justify-between gap-3'
                >
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

  const renderWithdrawalTable = () => (
    <>
      {renderWithdrawalFilterPanel()}
      <Table
        columns={withdrawalColumns}
        dataSource={withdrawals}
        loading={withdrawalLoading}
        rowKey={(record) => String(record?.id || 0)}
        size='small'
        pagination={{
          currentPage: withdrawalPage,
          pageSize: withdrawalPageSize,
          total: withdrawalTotal,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: (currentPage) => setWithdrawalPage(currentPage),
          onPageSizeChange: (currentPageSize) => {
            setWithdrawalPageSize(currentPageSize);
            setWithdrawalPage(1);
          },
        }}
        scroll={{ x: '100%' }}
        empty={buildTableEmpty(
          t,
          activeTab === 'withdrawals' ? '暂无提现审核记录' : '暂无提现订单',
        )}
      />
    </>
  );

  const renderInvoiceDetailValue = (label, content, copyable = false) => {
    const empty = content === undefined || content === null || content === '';
    return (
      <div
        className='rounded-lg p-3'
        style={{
          background: 'var(--semi-color-fill-0)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        <Text type='tertiary' size='small'>
          {t(label)}
        </Text>
        <div className='mt-1 break-all'>
          {React.isValidElement(content) ? (
            content
          ) : (
            <Text copyable={copyable && !empty}>{displayValue(content)}</Text>
          )}
        </div>
      </div>
    );
  };

  const renderInvoiceDetailModalContent = () => {
    const detail = invoiceDetail;
    if (!detail && !invoiceDetailLoading) {
      return buildTableEmpty(t, '暂无发票详情');
    }

    const items = detail?.items || [];
    const infoGridStyle = {
      gridTemplateColumns: isMobile ? '1fr' : 'repeat(3, minmax(0, 1fr))',
    };

    return (
      <div className='space-y-4'>
        <Card
          title={t('申请信息')}
          bordered={false}
          bodyStyle={{ padding: 12 }}
          style={{ border: '1px solid var(--semi-color-border)' }}
        >
          <div className='grid gap-3' style={infoGridStyle}>
            {renderInvoiceDetailValue(
              '申请编号',
              `#${detail?.id || '-'}`,
              true,
            )}
            {renderInvoiceDetailValue(
              '状态',
              renderInvoiceStatusTag(detail?.status),
            )}
            {renderInvoiceDetailValue(
              '申请时间',
              formatInvoiceTime(detail?.created_at),
            )}
            {renderInvoiceDetailValue(
              '申请用户',
              formatInvoiceUser(detail),
              true,
            )}
            {renderInvoiceDetailValue(
              '订单数量',
              `${items.length} ${t('笔订单')}`,
            )}
            {renderInvoiceDetailValue(
              '合计支付金额',
              <Text type='danger'>{formatMoney(detail?.total_money)}</Text>,
            )}
            {renderInvoiceDetailValue(
              '合计额度',
              Number(detail?.total_quota || 0) > 0
                ? renderQuota(detail.total_quota)
                : '-',
            )}
          </div>
        </Card>

        <Card
          title={t('发票抬头与接收信息')}
          bordered={false}
          bodyStyle={{ padding: 12 }}
          style={{ border: '1px solid var(--semi-color-border)' }}
        >
          <div className='grid gap-3' style={infoGridStyle}>
            {renderInvoiceDetailValue(
              '抬头类型',
              t(detail?.title_type === 'company' ? '企业' : '个人'),
            )}
            {renderInvoiceDetailValue('抬头名称', detail?.title, true)}
            {renderInvoiceDetailValue('税号', detail?.tax_number, true)}
            {renderInvoiceDetailValue('接收邮箱', detail?.email, true)}
            {renderInvoiceDetailValue('手机号', detail?.phone, true)}
            {renderInvoiceDetailValue('用户备注', detail?.remark)}
          </div>
        </Card>

        <Card
          title={t('订单明细')}
          bordered={false}
          bodyStyle={{ padding: 0 }}
          style={{ border: '1px solid var(--semi-color-border)' }}
        >
          <Table
            columns={invoiceDetailItemColumns}
            dataSource={items}
            loading={invoiceDetailLoading}
            rowKey={(item) => String(item?.id || getInvoiceItemKey(item))}
            size='small'
            pagination={false}
            scroll={{ x: '100%' }}
            empty={buildTableEmpty(t, '暂无订单明细')}
          />
        </Card>

        <Card
          title={t('审核与开票信息')}
          bordered={false}
          bodyStyle={{ padding: 12 }}
          style={{ border: '1px solid var(--semi-color-border)' }}
        >
          <div className='grid gap-3' style={infoGridStyle}>
            {renderInvoiceDetailValue(
              '审核人',
              formatInvoiceReviewer(detail),
              true,
            )}
            {renderInvoiceDetailValue(
              '审核时间',
              formatInvoiceTime(detail?.reviewed_at),
            )}
            {renderInvoiceDetailValue('发票号/代码', detail?.invoice_no, true)}
            {renderInvoiceDetailValue('发票链接', detail?.invoice_url, true)}
            {renderInvoiceDetailValue(
              '管理员备注/驳回原因',
              detail?.admin_remark,
            )}
          </div>
        </Card>
      </div>
    );
  };

  const renderInvoiceTable = () => (
    <>
      {renderInvoiceFilterPanel()}
      <Table
        columns={invoiceColumns}
        dataSource={invoices}
        loading={invoiceLoading}
        rowKey={(record) => String(record?.id || 0)}
        size='small'
        pagination={{
          currentPage: invoicePage,
          pageSize: invoicePageSize,
          total: invoiceTotal,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: (currentPage) => setInvoicePage(currentPage),
          onPageSizeChange: (currentPageSize) => {
            setInvoicePageSize(currentPageSize);
            setInvoicePage(1);
          },
        }}
        scroll={{ x: '100%' }}
        empty={buildTableEmpty(t, '暂无发票申请')}
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
        style={isMobile ? undefined : { width: '1400px', maxWidth: '95vw' }}
      >
        {userIsAdmin ? (
          <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
            <Tabs.TabPane tab={t('支付记录')} itemKey='records'>
              {renderRecordsTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('发票')} itemKey='invoices'>
              {renderInvoiceTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('提现订单')} itemKey='my-withdrawals'>
              {renderWithdrawalTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('提现审核')} itemKey='withdrawals'>
              {renderWithdrawalTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('对账看板')} itemKey='dashboard'>
              {renderDashboardBoard()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('异常单')} itemKey='risk'>
              {renderRiskCaseTable()}
            </Tabs.TabPane>
          </Tabs>
        ) : (
          <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
            <Tabs.TabPane tab={t('支付记录')} itemKey='records'>
              {renderRecordsTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('发票')} itemKey='invoices'>
              {renderInvoiceTable()}
            </Tabs.TabPane>
            <Tabs.TabPane tab={t('提现订单')} itemKey='my-withdrawals'>
              {renderWithdrawalTable()}
            </Tabs.TabPane>
          </Tabs>
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

      <Modal
        title={t('发票申请详情')}
        visible={invoiceDetailVisible}
        onCancel={closeInvoiceDetail}
        footer={
          <Space>
            <Button onClick={closeInvoiceDetail}>{t('关闭')}</Button>
            <Button
              type='primary'
              loading={invoiceDetailLoading}
              disabled={!invoiceDetail}
              onClick={printInvoiceDetail}
            >
              {t('打印')}
            </Button>
          </Space>
        }
        size={isMobile ? 'full-width' : 'large'}
        style={isMobile ? undefined : { width: '1100px', maxWidth: '95vw' }}
      >
        {renderInvoiceDetailModalContent()}
      </Modal>

      <Modal
        title={t('申请发票')}
        visible={invoiceApplyVisible}
        onOk={submitInvoiceRequest}
        onCancel={closeInvoiceApplyModal}
        confirmLoading={invoiceSubmitting}
        maskClosable={false}
        size={isMobile ? 'full-width' : 'large'}
        style={isMobile ? undefined : { width: '1000px', maxWidth: '95vw' }}
      >
        <div className='space-y-4'>
          <div
            className='grid gap-3'
            style={{
              gridTemplateColumns: isMobile
                ? '1fr'
                : 'repeat(2, minmax(0, 1fr))',
            }}
          >
            <div>
              <div className='text-xs mb-1'>{t('抬头类型')}</div>
              <Select
                value={invoiceForm.titleType}
                optionList={[
                  { label: t('企业'), value: 'company' },
                  { label: t('个人'), value: 'personal' },
                ]}
                onChange={(value) =>
                  handleInvoiceFormChange('titleType', value)
                }
                style={{ width: '100%' }}
              />
            </div>
            <div>
              <div className='text-xs mb-1'>{t('发票抬头')}</div>
              <Input
                value={invoiceForm.title}
                onChange={(value) => handleInvoiceFormChange('title', value)}
                placeholder={t('请输入发票抬头')}
                maxLength={128}
                showClear
              />
            </div>
            <div>
              <div className='text-xs mb-1'>{t('税号')}</div>
              <Input
                value={invoiceForm.taxNumber}
                onChange={(value) =>
                  handleInvoiceFormChange('taxNumber', value)
                }
                placeholder={t('企业抬头必填')}
                maxLength={64}
                showClear
              />
            </div>
            <div>
              <div className='text-xs mb-1'>{t('接收邮箱')}</div>
              <Input
                value={invoiceForm.email}
                onChange={(value) => handleInvoiceFormChange('email', value)}
                placeholder={t('用于接收电子发票')}
                maxLength={128}
                showClear
              />
            </div>
            <div>
              <div className='text-xs mb-1'>{t('手机号')}</div>
              <Input
                value={invoiceForm.phone}
                onChange={(value) => handleInvoiceFormChange('phone', value)}
                placeholder={t('可选')}
                maxLength={32}
                showClear
              />
            </div>
            <div>
              <div className='text-xs mb-1'>{t('备注')}</div>
              <Input
                value={invoiceForm.remark}
                onChange={(value) => handleInvoiceFormChange('remark', value)}
                placeholder={t('可选')}
                maxLength={1000}
                showClear
              />
            </div>
          </div>

          <Card
            bordered={false}
            bodyStyle={{ padding: 12 }}
            style={{
              background: 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
            }}
          >
            <Space wrap>
              <Text strong>
                {t('已选')} {selectedInvoiceOrders.length} {t('笔订单')}
              </Text>
              <Text type='danger'>
                {t('金额')} {formatMoney(selectedInvoiceSummary.money)}
              </Text>
              {selectedInvoiceSummary.quota > 0 ? (
                <Text type='tertiary'>
                  {t('额度')} {renderQuota(selectedInvoiceSummary.quota)}
                </Text>
              ) : null}
            </Space>
          </Card>

          <Table
            columns={invoiceOrderColumns}
            dataSource={eligibleInvoiceOrders}
            loading={eligibleInvoiceLoading}
            rowKey={(record) => getInvoiceOrderKey(record)}
            size='small'
            pagination={false}
            scroll={{ x: '100%' }}
            rowSelection={{
              selectedRowKeys: selectedInvoiceOrderKeys,
              onChange: (keys) => setSelectedInvoiceOrderKeys(keys),
            }}
            empty={buildTableEmpty(t, '暂无可开票订单')}
          />
        </div>
      </Modal>

      <Modal
        title={t(
          invoiceReviewState.action === 'approve'
            ? '通过发票申请'
            : '驳回发票申请',
        )}
        visible={invoiceReviewState.visible}
        onOk={submitInvoiceReview}
        onCancel={closeInvoiceReviewModal}
        confirmLoading={invoiceReviewSubmitting}
        maskClosable={false}
        okButtonProps={
          invoiceReviewState.action === 'reject'
            ? { type: 'danger' }
            : undefined
        }
      >
        <div className='space-y-3'>
          {invoiceReviewState.action === 'approve' ? (
            <>
              <Input
                placeholder={t('发票号或发票代码')}
                value={invoiceReviewForm.invoiceNo}
                onChange={(value) =>
                  setInvoiceReviewForm((prev) => ({
                    ...prev,
                    invoiceNo: value,
                  }))
                }
                maxLength={128}
                showClear
              />
              <Input
                placeholder={t('发票链接')}
                value={invoiceReviewForm.invoiceUrl}
                onChange={(value) =>
                  setInvoiceReviewForm((prev) => ({
                    ...prev,
                    invoiceUrl: value,
                  }))
                }
                showClear
              />
            </>
          ) : null}
          <TextArea
            placeholder={t(
              invoiceReviewState.action === 'approve'
                ? '审核备注，可选'
                : '驳回原因，必填',
            )}
            value={invoiceReviewForm.adminRemark}
            onChange={(value) =>
              setInvoiceReviewForm((prev) => ({
                ...prev,
                adminRemark: value,
              }))
            }
            maxCount={1000}
            autosize
          />
        </div>
      </Modal>

      <Modal
        title={t(
          reviewState.action === 'approve' ? '通过提现申请' : '驳回提现申请',
        )}
        visible={reviewState.visible}
        onOk={submitReviewModal}
        onCancel={closeReviewModal}
        confirmLoading={reviewSubmitting}
        maskClosable={false}
        okButtonProps={
          reviewState.action === 'reject' ? { type: 'danger' } : undefined
        }
      >
        <div className='space-y-3'>
          <div>
            {t(
              reviewState.action === 'approve'
                ? '请确认已通过支付宝完成转账后再点击通过。'
                : '驳回后会把冻结的待使用收益退回给用户。',
            )}
          </div>
          <Input
            placeholder={t('审核备注，可选')}
            value={reviewRemark}
            onChange={setReviewRemark}
            maxLength={255}
            showClear
          />
        </div>
      </Modal>
    </>
  );
};

export default TopupHistoryModal;
