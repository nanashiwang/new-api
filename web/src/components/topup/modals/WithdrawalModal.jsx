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

import React, { useMemo } from 'react';
import { Input, InputNumber, Modal, Typography } from '@douyinfe/semi-ui';
import { Wallet } from 'lucide-react';
import { getCurrencyConfig } from '../../../helpers';
import { getPaymentCurrencySymbol } from '../../../helpers/render';
import {
  displayAmountToQuota,
  quotaToDisplayAmount,
} from '../../../helpers/quota';

const roundCurrencyAmountUp = (amount) => {
  const numericAmount = Number(amount || 0);
  return Math.ceil(numericAmount * 100 - 1e-8) / 100;
};

const roundCurrencyAmountDown = (amount) => {
  const numericAmount = Number(amount || 0);
  return Math.floor(numericAmount * 100 + 1e-8) / 100;
};

const getTopUpPrice = () => {
  try {
    const status = JSON.parse(localStorage.getItem('status') || '{}');
    const price = Number(status?.price || 0);
    return Number.isFinite(price) && price > 0 ? price : 0;
  } catch (_) {
    return 0;
  }
};

const formatPaymentAmount = (value) =>
  `${getPaymentCurrencySymbol()}${Number(value || 0).toFixed(2)}`;

const WithdrawalModal = ({
  t,
  visible,
  onOk,
  onCancel,
  confirmLoading,
  userState,
  renderQuota,
  getQuotaPerUnit,
  withdrawalAmount,
  setWithdrawalAmount,
  alipayAccount,
  setAlipayAccount,
  alipayName,
  setAlipayName,
}) => {
  const currencyConfig = getCurrencyConfig();
  const isTokenDisplay = currencyConfig.type === 'TOKENS';
  const minWithdrawalAmount = isTokenDisplay
    ? getQuotaPerUnit()
    : roundCurrencyAmountUp(quotaToDisplayAmount(getQuotaPerUnit()));
  const maxWithdrawalAmount = isTokenDisplay
    ? userState?.user?.aff_quota || 0
    : roundCurrencyAmountDown(
        quotaToDisplayAmount(userState?.user?.aff_quota || 0),
      );

  const estimatedQuota = useMemo(
    () => displayAmountToQuota(withdrawalAmount),
    [withdrawalAmount],
  );
  const estimatedPaymentAmount = useMemo(() => {
    const price = getTopUpPrice();
    const quotaPerUnit = Number(getQuotaPerUnit() || 0);
    if (!price || !quotaPerUnit || estimatedQuota <= 0) return 0;
    return (estimatedQuota / quotaPerUnit) * price;
  }, [estimatedQuota, getQuotaPerUnit]);

  return (
    <Modal
      title={
        <div className='flex items-center'>
          <Wallet className='mr-2' size={18} />
          {t('提现邀请收益')}
        </div>
      }
      visible={visible}
      onOk={onOk}
      onCancel={onCancel}
      maskClosable={false}
      centered
      confirmLoading={confirmLoading}
      okText={t('提交提现')}
    >
      <div className='space-y-4'>
        <div>
          <Typography.Text strong className='block mb-2'>
            {t('可提现收益')}
          </Typography.Text>
          <Input
            value={renderQuota(userState?.user?.aff_quota || 0)}
            disabled
            className='!rounded-lg'
          />
        </div>
        <div>
          <Typography.Text strong className='block mb-2'>
            {(isTokenDisplay ? t('提现额度') : t('提现金额')) +
              ' · ' +
              t('最低') +
              renderQuota(getQuotaPerUnit())}
          </Typography.Text>
          <InputNumber
            min={minWithdrawalAmount}
            max={maxWithdrawalAmount}
            value={withdrawalAmount}
            onChange={(value) => setWithdrawalAmount(value)}
            placeholder={isTokenDisplay ? undefined : t('输入提现额度')}
            prefix={isTokenDisplay ? undefined : currencyConfig.symbol}
            precision={isTokenDisplay ? undefined : 2}
            step={isTokenDisplay ? undefined : 0.01}
            className='w-full !rounded-lg'
          />
          <div className='mt-2 text-sm text-semi-color-text-2'>
            {t('预计到账金额')}：{formatPaymentAmount(estimatedPaymentAmount)}
          </div>
        </div>
        <div>
          <Typography.Text strong className='block mb-2'>
            {t('支付宝账号')}
          </Typography.Text>
          <Input
            value={alipayAccount}
            onChange={setAlipayAccount}
            placeholder={t('请输入支付宝账号')}
            maxLength={128}
            showClear
            className='!rounded-lg'
          />
          <div className='mt-1 text-xs text-right text-semi-color-text-2'>
            {alipayAccount?.length || 0}/128
          </div>
        </div>
        <div>
          <Typography.Text strong className='block mb-2'>
            {t('支付宝姓名')}
          </Typography.Text>
          <Input
            value={alipayName}
            onChange={setAlipayName}
            placeholder={t('请输入支付宝实名姓名')}
            maxLength={64}
            showClear
            className='!rounded-lg'
          />
          <div className='mt-1 text-xs text-right text-semi-color-text-2'>
            {alipayName?.length || 0}/64
          </div>
        </div>
        <div className='text-xs text-semi-color-text-2'>
          {t('提交后将冻结对应待使用收益，审核驳回后会自动退回。')}
        </div>
      </div>
    </Modal>
  );
};

export default WithdrawalModal;
