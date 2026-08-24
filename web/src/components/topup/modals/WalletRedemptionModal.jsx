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

import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  InputNumber,
  Modal,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Copy, Gift, RefreshCw } from 'lucide-react';
import {
  API,
  getCurrencyConfig,
  getQuotaPerUnit,
  renderQuota,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../helpers';
import {
  displayAmountToQuota,
  quotaToDisplayAmount,
} from '../../../helpers/quota';
import { copy } from '../../../helpers/clipboard';

const { Text } = Typography;

const newRequestID = () => {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return `wallet-${Date.now()}-${Math.random().toString(36).slice(2, 14)}`;
};

const WalletRedemptionModal = ({
  visible,
  onCancel,
  walletQuota,
  transferableQuota,
  isAdmin,
  onQuotaChanged,
  t,
}) => {
  const currencyConfig = getCurrencyConfig();
  const isTokenDisplay = currencyConfig.type === 'TOKENS';
  const [amount, setAmount] = useState(0);
  const [creating, setCreating] = useState(false);
  const [loadingList, setLoadingList] = useState(false);
  const [redemptions, setRedemptions] = useState([]);
  const [createdCode, setCreatedCode] = useState('');
  const [uncertainRequest, setUncertainRequest] = useState(false);
  const [usageSummary, setUsageSummary] = useState(null);
  const pendingRequestRef = useRef(null);
  const minimumQuota = isAdmin
    ? 1
    : Number(usageSummary?.minimum_quota || 10 * getQuotaPerUnit());

  const maxAmount = isTokenDisplay
    ? Number(transferableQuota || 0)
    : Math.floor(quotaToDisplayAmount(transferableQuota || 0) * 100 + 1e-8) /
      100;
  const minimumAmount = isTokenDisplay
    ? minimumQuota
    : isAdmin
      ? 0.01
      : quotaToDisplayAmount(minimumQuota);

  const loadRedemptions = async () => {
    setLoadingList(true);
    try {
      const response = await API.get(
        '/api/user/redemptions/self?p=1&page_size=20',
      );
      if (response.data?.success) {
        setRedemptions(response.data?.data?.items || []);
      } else {
        showError(response.data?.message || t('请求失败，请重试'));
      }
    } catch {
      showError(t('请求失败，请重试'));
    } finally {
      setLoadingList(false);
    }
  };

  const loadUsageSummary = async () => {
    try {
      const response = await API.get('/api/user/redemptions/self/summary');
      if (response.data?.success) {
        setUsageSummary(response.data?.data || null);
      }
    } catch {
      setUsageSummary(null);
    }
  };

  useEffect(() => {
    if (visible) {
      loadRedemptions();
      loadUsageSummary();
    }
  }, [visible]);

  const createRedemption = async () => {
    const quota = displayAmountToQuota(amount);
    if (quota <= 0) {
      showError(t('兑换码额度必须大于 0'));
      return;
    }
    if (!isAdmin && quota < minimumQuota) {
      showError(
        t('兑换码额度不能低于 {{amount}}', {
          amount: renderQuota(minimumQuota),
        }),
      );
      return;
    }
    if (quota > Number(walletQuota || 0)) {
      showError(t('钱包余额不足'));
      return;
    }
    if (quota > Number(transferableQuota || 0)) {
      showError(t('可创建兑换码额度不足'));
      return;
    }

    if (!pendingRequestRef.current) {
      pendingRequestRef.current = { requestID: newRequestID(), quota };
    }
    if (pendingRequestRef.current.quota !== quota) {
      showError(t('上次请求结果未知，请保持原额度并重试'));
      return;
    }

    setCreating(true);
    try {
      const response = await API.post(
        '/api/user/redemptions',
        {
          quota,
          request_id: pendingRequestRef.current.requestID,
        },
        { skipErrorHandler: true },
      );
      if (!response.data?.success) {
        pendingRequestRef.current = null;
        setUncertainRequest(false);
        showError(response.data?.message || t('请求失败，请重试'));
        return;
      }

      const data = response.data?.data || {};
      const redemption = data.redemption;
      if (!redemption?.key) {
        throw new Error('missing redemption key');
      }
      pendingRequestRef.current = null;
      setUncertainRequest(false);
      setCreatedCode(redemption.key);
      setAmount(0);
      setRedemptions((items) => [
        redemption,
        ...items.filter((item) => item.id !== redemption.id),
      ]);
      onQuotaChanged?.(
        Number(data.remaining_quota || 0),
        Number(data.remaining_transferable_quota || 0),
      );
      await loadUsageSummary();
      showSuccess(t('创建成功'));
    } catch {
      // 请求可能已经在服务端成功，保留同一个 request_id，让重试只返回
      // 原兑换码而不会再次扣款。
      setUncertainRequest(true);
      showError(t('请求结果未知，请保持原额度并重试'));
    } finally {
      setCreating(false);
    }
  };

  const copyCode = async (code) => {
    if (await copy(code)) {
      showSuccess(t('复制成功'));
    } else {
      showError(t('复制失败'));
    }
  };

  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          <Gift size={18} />
          {t('创建兑换码')}
        </div>
      }
      visible={visible}
      onCancel={onCancel}
      footer={null}
      maskClosable={false}
      centered
      width={640}
    >
      <div className='space-y-4'>
        <div className='rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900'>
          <div>{t('创建兑换码会立即从钱包扣除对应额度')}</div>
          <div className='mt-1'>
            {t('余额均可创建兑换码；仅其他用户兑换码转入的额度不可再次转赠')}
          </div>
          <div className='mt-1'>
            {t('自己兑换自己的码只会返还额度，不会建立邀请关系或产生邀请奖励')}
          </div>
          <div className='mt-1'>
            {t(
              '其他未绑定上游的用户兑换后，会按邀请阈值与概率尝试绑定你为上游；兑换码本身不产生返佣',
            )}
          </div>
          <div className='mt-1'>
            {t('转赠给他人后的额度不能再次创建兑换码')}
          </div>
        </div>

        <div className='rounded-lg border border-slate-200 p-3'>
          <div className='mb-2 flex items-center justify-between gap-3'>
            <Text strong>{t('兑换码额度')}</Text>
            <Text type='tertiary'>
              {t('当前钱包余额')}：{renderQuota(walletQuota || 0)}
            </Text>
          </div>
          <div className='mb-2'>
            <Text type='tertiary'>
              {t('可创建兑换码额度')}：{renderQuota(transferableQuota || 0)}
            </Text>
          </div>
          <Space align='end' className='w-full'>
            <InputNumber
              value={amount}
              onChange={(value) => setAmount(Number(value || 0))}
              min={minimumAmount}
              max={maxAmount}
              precision={isTokenDisplay ? 0 : 2}
              disabled={uncertainRequest}
              prefix={currencyConfig.symbol}
              className='min-w-[260px]'
            />
            <Button
              type='primary'
              theme='solid'
              onClick={createRedemption}
              loading={creating}
            >
              {uncertainRequest ? t('重试并查询原请求') : t('创建并扣除额度')}
            </Button>
          </Space>
          {uncertainRequest && (
            <Text type='warning' size='small' className='mt-2 block'>
              {t('上次请求结果未知，请保持原额度并重试')}
            </Text>
          )}
        </div>

        {usageSummary && (
          <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4'>
            <div className='rounded-lg border border-slate-200 p-3'>
              <Text type='tertiary'>{t('今日已创建')}</Text>
              <div className='mt-1 text-lg font-semibold'>
                {usageSummary.daily_created_count} /{' '}
                {usageSummary.daily_create_limit || t('不限')}
              </div>
            </div>
            <div className='rounded-lg border border-slate-200 p-3'>
              <Text type='tertiary'>{t('今日已转赠')}</Text>
              <div className='mt-1 text-lg font-semibold'>
                {renderQuota(usageSummary.daily_created_quota || 0)} /{' '}
                {usageSummary.daily_quota_limit
                  ? renderQuota(usageSummary.daily_quota_limit)
                  : t('不限')}
              </div>
            </div>
            <div className='rounded-lg border border-slate-200 p-3'>
              <Text type='tertiary'>{t('当前未使用')}</Text>
              <div className='mt-1 text-lg font-semibold'>
                {usageSummary.active_count} /{' '}
                {usageSummary.active_limit || t('不限')}
              </div>
            </div>
            <div className='rounded-lg border border-slate-200 p-3'>
              <Text type='tertiary'>{t('北京时间重置')}</Text>
              <div className='mt-1 text-lg font-semibold'>
                {timestamp2string(usageSummary.reset_at)}
              </div>
            </div>
          </div>
        )}

        {createdCode && (
          <div className='rounded-lg border border-green-200 bg-green-50 p-3'>
            <Text strong>{t('创建成功')}</Text>
            <div className='mt-2 flex items-center gap-2'>
              <code className='min-w-0 flex-1 break-all rounded bg-white px-3 py-2 text-sm'>
                {createdCode}
              </code>
              <Button
                icon={<Copy size={15} />}
                onClick={() => copyCode(createdCode)}
              >
                {t('复制兑换码')}
              </Button>
            </div>
          </div>
        )}

        <div>
          <div className='mb-2 flex items-center justify-between'>
            <Text strong>{t('我创建的兑换码')}</Text>
            <Button
              type='tertiary'
              theme='borderless'
              icon={<RefreshCw size={14} />}
              loading={loadingList}
              onClick={loadRedemptions}
            >
              {t('刷新')}
            </Button>
          </div>
          <div className='max-h-64 space-y-2 overflow-y-auto'>
            {redemptions.length === 0 && !loadingList ? (
              <Text type='tertiary'>{t('暂无兑换码')}</Text>
            ) : (
              redemptions.map((item) => (
                <div
                  key={item.id}
                  className='rounded-lg border border-slate-200 px-3 py-2'
                >
                  <div className='flex items-center justify-between gap-3'>
                    <code className='min-w-0 flex-1 truncate text-xs'>
                      {item.key}
                    </code>
                    <Tag color={item.status === 1 ? 'green' : 'grey'}>
                      {item.status === 1 ? t('未使用') : t('已使用')}
                    </Tag>
                    <Button
                      type='tertiary'
                      theme='borderless'
                      icon={<Copy size={14} />}
                      onClick={() => copyCode(item.key)}
                    />
                  </div>
                  <div className='mt-1 flex items-center justify-between gap-3'>
                    <Text type='secondary' size='small'>
                      {renderQuota(item.quota || 0)}
                    </Text>
                    <Text type='tertiary' size='small'>
                      {timestamp2string(item.created_time)}
                    </Text>
                  </div>
                </div>
              ))
            )}
          </div>
          <Text type='tertiary' size='small' className='mt-2 block'>
            {t('为保护隐私，创建者不会看到兑换者的用户信息')}
          </Text>
        </div>
      </div>
    </Modal>
  );
};

export default WalletRedemptionModal;
