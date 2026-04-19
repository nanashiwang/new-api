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
  Collapse,
  Descriptions,
  Empty,
  Modal,
  Space,
  Spin,
  Tag,
  TextArea,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { API, timestamp2string } from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const RISK_STATUS_CONFIG = {
  open: { color: 'red', label: '待处理' },
  confirmed: { color: 'green', label: '已确认' },
  reversed: { color: 'orange', label: '已回退' },
  voided: { color: 'grey', label: '已作废' },
};

const RISK_REASON_LABELS = {
  manual_review: '人工标记',
  order_not_found: '订单不存在',
  order_status_invalid: '订单状态异常',
  payment_method_mismatch: '支付方式不匹配',
  amount_mismatch: '支付金额不匹配',
  unsupported_order_type: '订单类型不支持',
};

const RECORD_TYPE_LABELS = {
  topup: '充值订单',
  subscription: '订阅订单',
};

const ORDER_STATUS_LABELS = {
  pending: '待支付',
  success: '成功',
  expired: '已过期',
  cancelled: '已取消',
};

const ACTION_LABELS = {
  confirm: '确认放行',
  reverse: '回退额度',
  void: '作废订单',
};

const ACTION_DESCRIPTIONS = {
  confirm: '确认该回调有效，并按订单类型补发或放行本地订单状态。',
  reverse: '撤销这笔异常充值已经发放的额度，适合已到账异常单。',
  void: '将异常单直接作废，不再允许后续补单。',
};

const PAYMENT_METHOD_LABELS = {
  stripe: 'Stripe',
  creem: 'Creem',
  alipay: '支付宝',
  wxpay: '微信',
  wallet: '钱包',
};

function formatMoney(value, currency) {
  if (value === null || value === undefined || value === '') {
    return '-';
  }
  const amount = Number(value);
  if (!Number.isFinite(amount)) {
    return String(value);
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

function formatPayload(payload) {
  const raw = String(payload || '').trim();
  if (!raw) {
    return '';
  }
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function renderStatusTag(status, t) {
  const config = RISK_STATUS_CONFIG[status] || {
    color: 'grey',
    label: status || '-',
  };
  return <Tag color={config.color}>{t(config.label)}</Tag>;
}

function renderMethod(method, t) {
  const label = PAYMENT_METHOD_LABELS[method] || method || '-';
  return <Tag color='grey'>{t(label)}</Tag>;
}

const PaymentRiskCaseDetailModal = ({
  visible,
  riskCaseId,
  initialRiskCase,
  onCancel,
  onResolved,
  t,
}) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [detailData, setDetailData] = useState(null);
  const [note, setNote] = useState('');

  const loadRiskCaseDetail = async () => {
    if (!riskCaseId) {
      setDetailData(null);
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(`/api/user/payment-risk-cases/${riskCaseId}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '加载异常单详情失败') });
        return;
      }
      setDetailData(data || null);
    } catch (error) {
      Toast.error({ content: t('加载异常单详情失败') });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) {
      setDetailData(null);
      setNote('');
      return;
    }
    loadRiskCaseDetail();
  }, [visible, riskCaseId]);

  const riskCase = detailData?.risk_case || initialRiskCase || null;
  const availableActions = detailData?.available_actions || [];

  const descriptionRows = useMemo(() => {
    if (!riskCase) {
      return [];
    }
    return [
      {
        key: t('异常单 ID'),
        value: riskCase.id || '-',
      },
      {
        key: t('订单类型'),
        value: t(RECORD_TYPE_LABELS[riskCase.record_type] || riskCase.record_type || '-'),
      },
      {
        key: t('订单号'),
        value: riskCase.trade_no ? <Text copyable>{riskCase.trade_no}</Text> : '-',
      },
      {
        key: t('状态'),
        value: renderStatusTag(riskCase.status, t),
      },
      {
        key: t('异常原因'),
        value: t(RISK_REASON_LABELS[riskCase.reason] || riskCase.reason || '-'),
      },
      {
        key: t('订单状态'),
        value: t(ORDER_STATUS_LABELS[riskCase.order_status] || riskCase.order_status || '-'),
      },
      {
        key: t('用户'),
        value: riskCase.username ? (
          <span>
            <Text>{riskCase.username}</Text>
            {riskCase.user_id ? (
              <Text type='tertiary' size='small'>
                {' '}
                (ID: {riskCase.user_id})
              </Text>
            ) : null}
          </span>
        ) : (
          '-'
        ),
      },
      {
        key: t('本地支付方式'),
        value: renderMethod(riskCase.payment_method, t),
      },
      {
        key: t('回调支付方式'),
        value: renderMethod(riskCase.provider_payment_method, t),
      },
      {
        key: t('预期额度'),
        value: riskCase.expected_amount || '-',
      },
      {
        key: t('预期金额'),
        value: formatMoney(riskCase.expected_money, riskCase.currency),
      },
      {
        key: t('回调金额'),
        value: formatMoney(riskCase.received_money, riskCase.currency),
      },
      {
        key: t('来源'),
        value: riskCase.source || '-',
      },
      {
        key: t('创建时间'),
        value: riskCase.created_at ? timestamp2string(riskCase.created_at) : '-',
      },
      {
        key: t('处理时间'),
        value: riskCase.resolved_at ? timestamp2string(riskCase.resolved_at) : '-',
      },
      {
        key: t('处理管理员'),
        value: riskCase.handler_admin_id || '-',
      },
      {
        key: t('额度变更'),
        value: riskCase.applied_quota_delta || '-',
      },
      {
        key: t('处理备注'),
        value: riskCase.handler_note || '-',
      },
    ];
  }, [riskCase, t]);

  const submitResolve = async (action) => {
    if (!riskCaseId) {
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post(`/api/user/payment-risk-cases/${riskCaseId}/resolve`, {
        action,
        note: note.trim(),
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        Toast.error({ content: t(message || '处理异常单失败') });
        return;
      }
      Toast.success({ content: t('异常单处理成功') });
      setDetailData(data || null);
      if (typeof onResolved === 'function') {
        onResolved(data?.risk_case || null);
      }
    } catch (error) {
      Toast.error({ content: t('处理异常单失败') });
    } finally {
      setSubmitting(false);
    }
  };

  const confirmResolve = (action) => {
    const label = ACTION_LABELS[action] || action;
    Modal.confirm({
      title: t(label),
      content: t(ACTION_DESCRIPTIONS[action] || '请确认是否继续执行该操作'),
      okText: t('确认'),
      cancelText: t('取消'),
      okButtonProps: action === 'reverse' ? { type: 'danger' } : undefined,
      onOk: () => submitResolve(action),
    });
  };

  const payloadText = formatPayload(riskCase?.provider_payload);

  return (
    <Modal
      title={t('支付异常单详情')}
      visible={visible}
      onCancel={onCancel}
      footer={
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <Text type='tertiary' size='small'>
            {riskCase ? `${t('可执行动作')}：${availableActions.length}` : ''}
          </Text>
          <Space wrap>
            <Button onClick={onCancel}>{t('关闭')}</Button>
            {availableActions.map((action) => (
              <Button
                key={action}
                type={action === 'reverse' ? 'danger' : 'primary'}
                theme={action === 'confirm' ? 'solid' : 'outline'}
                loading={submitting}
                onClick={() => confirmResolve(action)}
              >
                {t(ACTION_LABELS[action] || action)}
              </Button>
            ))}
          </Space>
        </div>
      }
      size={isMobile ? 'full-width' : 'medium'}
      style={isMobile ? undefined : { width: '820px', maxWidth: '95vw' }}
    >
      <Spin spinning={loading}>
        {!riskCase ? (
          <Empty description={t('暂无异常单详情')} style={{ padding: 40 }} />
        ) : (
          <div className='space-y-4'>
            <Descriptions data={descriptionRows} />

            <div>
              <Text strong>{t('处理备注')}</Text>
              <TextArea
                className='mt-2'
                value={note}
                rows={3}
                maxCount={500}
                placeholder={t('可选，用于记录人工审核结论')}
                onChange={setNote}
              />
            </div>

            <Collapse>
              <Collapse.Panel header={t('回调原始数据')} itemKey='provider_payload'>
                {payloadText ? (
                  <pre
                    style={{
                      margin: 0,
                      padding: 12,
                      borderRadius: 8,
                      background: 'var(--semi-color-fill-0)',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-all',
                      maxHeight: 320,
                      overflow: 'auto',
                    }}
                  >
                    {payloadText}
                  </pre>
                ) : (
                  <Text type='tertiary'>{t('暂无回调原始数据')}</Text>
                )}
              </Collapse.Panel>
            </Collapse>
          </div>
        )}
      </Spin>
    </Modal>
  );
};

export default PaymentRiskCaseDetailModal;
