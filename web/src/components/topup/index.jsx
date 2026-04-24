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

import React, { useEffect, useState, useContext, useRef } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  getCurrencyConfig,
  renderQuota,
  renderQuotaWithAmount,
  handleCopyUrl,
  getQuotaPerUnit,
  setUserData,
  timestamp2string,
} from '../../helpers';
import {
  displayAmountToQuota,
  quotaToDisplayAmount,
} from '../../helpers/quota';
import { getPaymentCurrencySymbol, formatWindowLimitShort, formatConcurrencyLabel } from '../../helpers/render';
import { Button, Card, Form, Modal, Select, Space, Tag, Toast, Typography, Divider } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useDebouncedCallback } from 'use-debounce';
import { ArrowRight, Clock3, Coins, Gauge, ShieldCheck, Zap, Sparkles, BellRing } from 'lucide-react';
import { IconGift } from '@douyinfe/semi-icons';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';

import RechargeCard from './RechargeCard';
import InvitationCard from './InvitationCard';
import TransferModal from './modals/TransferModal';
import PaymentConfirmModal from './modals/PaymentConfirmModal';
import TopupHistoryModal from './modals/TopupHistoryModal';
import SellableTokenIssuanceModal from '../sellable-tokens/SellableTokenIssuanceModal';
import SubscriptionIssuanceModal from '../subscriptions/SubscriptionIssuanceModal';
import MyTokensCard from './MyTokensCard';

const { Text, Title } = Typography;

const roundCurrencyAmountUp = (amount) => {
  const numericAmount = Number(amount || 0);
  const cents = Math.round(numericAmount * 100);
  return Math.ceil(cents - 0.01) / 100;
};

const PERIOD_LABELS = {
  hourly: '每小时',
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
  custom: '自定义',
};

const renderPeriodLabel = (t, period) => t(PERIOD_LABELS[period] || PERIOD_LABELS.custom);
const renderValidityLabel = (t, validitySeconds) =>
  Number(validitySeconds || 0) > 0
    ? `${Number(validitySeconds || 0)}s`
    : t('长期有效');

const TopUp = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const [redemptionCode, setRedemptionCode] = useState('');
  const [amount, setAmount] = useState(0.0);
  const [minTopUp, setMinTopUp] = useState(statusState?.status?.min_topup || 1);
  const [topUpCount, setTopUpCount] = useState(
    statusState?.status?.min_topup || 1,
  );
  const [topUpLink, setTopUpLink] = useState(
    statusState?.status?.top_up_link || '',
  );
  const [enableOnlineTopUp, setEnableOnlineTopUp] = useState(
    statusState?.status?.enable_online_topup || false,
  );

  const [enableStripeTopUp, setEnableStripeTopUp] = useState(
    statusState?.status?.enable_stripe_topup || false,
  );
  const [statusLoading, setStatusLoading] = useState(true);

  // Creem 状态
  const [creemProducts, setCreemProducts] = useState([]);
  const [enableCreemTopUp, setEnableCreemTopUp] = useState(false);
  const [creemOpen, setCreemOpen] = useState(false);
  const [selectedCreemProduct, setSelectedCreemProduct] = useState(null);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [payWay, setPayWay] = useState('');
  const [amountLoading, setAmountLoading] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [payMethods, setPayMethods] = useState([]);

  const affFetchedRef = useRef(false);
  const amountRequestRef = useRef(0);

  // 邀请状态
  const [affLink, setAffLink] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(() => {
    const minTransferAmount = quotaToDisplayAmount(getQuotaPerUnit());
    return getCurrencyConfig().type === 'TOKENS'
      ? minTransferAmount
      : roundCurrencyAmountUp(minTransferAmount);
  });

  // 计费弹窗状态
  const [openHistory, setOpenHistory] = useState(false);
  const [redeemTargetModalOpen, setRedeemTargetModalOpen] = useState(false);
  const [redeemTargetOptions, setRedeemTargetOptions] = useState([]);
  const [redeemTargetPlanTitle, setRedeemTargetPlanTitle] = useState('');
  const [selectedRenewTargetId, setSelectedRenewTargetId] = useState(0);

  // 套餐兑换方式选择弹窗状态
  const [purchaseModeModalOpen, setPurchaseModeModalOpen] = useState(false);
  const [purchaseModePlanTitle, setPurchaseModePlanTitle] = useState('');
  const [selectedPurchaseMode, setSelectedPurchaseMode] = useState('stack');
  const [sellableTokenProducts, setSellableTokenProducts] = useState([]);
  const [sellableTokenLoading, setSellableTokenLoading] = useState(false);
  const [sellableTokenIssuanceId, setSellableTokenIssuanceId] = useState(0);
  const [sellableTokenIssuanceVisible, setSellableTokenIssuanceVisible] =
    useState(false);
  const [pendingSellableIssuances, setPendingSellableIssuances] = useState([]);
  const [activeSellableTokens, setActiveSellableTokens] = useState([]);
  const [pendingSubscriptionIssuances, setPendingSubscriptionIssuances] =
    useState([]);
  const [subscriptionIssuanceId, setSubscriptionIssuanceId] = useState(0);
  const [subscriptionIssuanceVisible, setSubscriptionIssuanceVisible] =
    useState(false);

  // 订阅状态
  const [subscriptionPlans, setSubscriptionPlans] = useState([]);
  const [subscriptionLoading, setSubscriptionLoading] = useState(true);
  const [billingPreference, setBillingPreference] =
    useState('subscription_first');
  const [activeSubscriptions, setActiveSubscriptions] = useState([]);
  const [activeQuantityByPlan, setActiveQuantityByPlan] = useState({});
  const [allSubscriptions, setAllSubscriptions] = useState([]);

  // 预设充值金额选项
  const [presetAmounts, setPresetAmounts] = useState([]);
  const [selectedPreset, setSelectedPreset] = useState(null);

  // 充值配置数据
  const [topupInfo, setTopupInfo] = useState({
    amount_options: [],
    discount: {},
  });

  const isPayMethodEnabled = (paymentMethod) => {
    if (!paymentMethod) return false;
    if (paymentMethod === 'stripe') return enableStripeTopUp;
    return enableOnlineTopUp;
  };

  const canUsePayMethodForAmount = (paymentMethod, amountValue) => {
    if (!paymentMethod || !isPayMethodEnabled(paymentMethod)) return false;
    const payMethod = payMethods.find((item) => item.type === paymentMethod);
    if (!payMethod) return false;
    const minTopupVal = Number(payMethod.min_topup) || 0;
    return Number(amountValue || 0) >= minTopupVal;
  };

  const pickAvailablePayMethod = (amountValue) =>
    payMethods.find((item) => canUsePayMethodForAmount(item.type, amountValue));

  const topUp = async (renewTargetSubscriptionId = 0, purchaseMode = '') => {
    if (redemptionCode === '') {
      showInfo(t('请输入兑换码！'));
      return;
    }
    setIsSubmitting(true);
    try {
      const payload = {
        key: redemptionCode,
        renew_target_subscription_id: renewTargetSubscriptionId || 0,
      };
      if (purchaseMode) {
        payload.purchase_mode = purchaseMode;
      }
      const res = await API.post('/api/user/redeem', payload, {
        skipErrorHandler: true,
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        if (data?.benefit_type === 'sellable_token') {
          setSellableTokenIssuanceId(Number(data?.issuance_id || 0));
          setSellableTokenIssuanceVisible(true);
          loadPendingSellableIssuances();
          Modal.success({
            title: t('可售令牌待发放'),
            content:
              t('已创建待发放记录，请继续完成令牌命名和分组设置：') +
              (data?.product_name || '-'),
            centered: true,
          });
        } else if (data?.benefit_type === 'subscription') {
          setSubscriptionIssuanceId(Number(data?.issuance_id || 0));
          setSubscriptionIssuanceVisible(true);
          Modal.success({
            title: t('套餐待发放已创建'),
            content:
              data?.action_summary ||
              t('已生成套餐待发放记录，请继续选择叠加或续费方式。'),
            centered: true,
          });
          await getSubscriptionSelf();
        } else {
          Modal.success({
            title: t('兑换成功！'),
            content: t('成功兑换额度：') + renderQuota(data?.quota_added || 0),
            centered: true,
          });
          if (userState.user) {
            const updatedUser = {
              ...userState.user,
              quota: userState.user.quota + (data?.quota_added || 0),
            };
            userDispatch({ type: 'login', payload: updatedUser });
          }
        }
        setRedemptionCode('');
        setRedeemTargetModalOpen(false);
        setRedeemTargetOptions([]);
        setRedeemTargetPlanTitle('');
        setSelectedRenewTargetId(0);
        setPurchaseModeModalOpen(false);
        setPurchaseModePlanTitle('');
        setSelectedPurchaseMode('stack');
      } else {
        if (data?.code === 'redeem_select_purchase_mode') {
          setPurchaseModePlanTitle(data?.plan_title || '');
          setSelectedPurchaseMode('stack');
          setPurchaseModeModalOpen(true);
          return;
        }
        if (data?.code === 'redeem_select_renew_target') {
          const options = (data?.options || [])
            .map((item) => item?.subscription)
            .filter(Boolean);
          setRedeemTargetOptions(options);
          setRedeemTargetPlanTitle(data?.plan_title || '');
          setSelectedRenewTargetId(Number(options?.[0]?.id || 0));
          setRedeemTargetModalOpen(true);
          return;
        }
        showError(message);
      }
    } catch (err) {
      const status = err?.response?.status;
      const body = err?.response?.data;
      if (status === 429 || body?.error?.code === 'rate_limited') {
        const retry = Number(body?.error?.retry_after) || 60;
        const scope = body?.error?.scope;
        if (scope === 'RDF') {
          showError(
            t('兑换失败次数过多，{{sec}} 秒后再试', { sec: retry }),
          );
        } else {
          showError(
            t('兑换次数过于频繁，{{sec}} 秒后再试', { sec: retry }),
          );
        }
      } else {
        showError(t('请求失败'));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const loadSellableTokenProducts = async () => {
    setSellableTokenLoading(true);
    try {
      const res = await API.get('/api/user/sellable-token/products');
      if (res.data?.success) setSellableTokenProducts(res.data.data || []);
    } finally {
      setSellableTokenLoading(false);
    }
  };

  const loadPendingSellableIssuances = async () => {
    try {
      const res = await API.get('/api/user/sellable-token/issuances', {
        params: { status: 'pending' },
      });
      if (res.data?.success) setPendingSellableIssuances(Array.isArray(res.data.data) ? res.data.data : []);
    } catch (error) {}
  };

  const loadActiveSellableTokens = async () => {
    try {
      const res = await API.get('/api/token/', {
        params: { p: 1, size: 100 },
      });
      if (res.data?.success) {
        const items = Array.isArray(res.data.data?.items)
          ? res.data.data.items
          : Array.isArray(res.data.data)
            ? res.data.data
          : Array.isArray(res.data.data?.data)
            ? res.data.data.data
            : [];
        const isActiveSellableToken = (token) => {
          const statusEnabled = Number(token?.status || 0) === 1;
          if (!statusEnabled) return false;
          return (
            token?.source_type === 'sellable_token' ||
            Number(token?.sellable_token_product_id || 0) > 0 ||
            Number(token?.sellable_token_issuance_id || 0) > 0
          );
        };
        setActiveSellableTokens(
          items.filter(isActiveSellableToken),
        );
      }
    } catch (_) {}
  };

  const purchaseSellableToken = async (productId) => {
    try {
      const res = await API.post('/api/user/sellable-token/purchase', { product_id: productId });
      if (res.data?.success) {
        showSuccess(t('购买成功，请继续完成令牌发放'));
        getUserQuota();
        setSellableTokenIssuanceId(Number(res.data.data?.issuance_id || 0));
        setSellableTokenIssuanceVisible(true);
        loadPendingSellableIssuances();
      } else {
        showError(res.data?.message);
      }
    } catch (error) {}
  };

  const confirmPurchaseSellableToken = (product) => {
    const productId = Number(product?.id || 0);
    const priceQuota = Number(product?.price_quota || 0);
    const currentQuota = Number(userState?.user?.quota || 0);
    if (productId <= 0 || priceQuota <= 0) return;

    if (currentQuota < priceQuota) {
      Modal.warning({
        title: t('余额不足'),
        centered: true,
        content: (
          <div className='space-y-2 text-sm'>
            <div>{t('商品')}: {product?.name || '-'}</div>
            <div>{t('需扣除余额')}: {renderQuota(priceQuota)}</div>
            <div>{t('当前余额')}: {renderQuota(currentQuota)}</div>
            <div className='text-red-500'>{t('余额不足，还差')} {renderQuota(priceQuota - currentQuota)}</div>
          </div>
        ),
      });
      return;
    }

    Modal.confirm({
      title: t('确认购买可售令牌'),
      centered: true,
      okText: t('确认购买'),
      cancelText: t('取消'),
      content: (
        <div className='space-y-2 text-sm'>
          <div>{t('商品')}: {product?.name || '-'}</div>
          <div>{t('需扣除余额')}: {renderQuota(priceQuota)}</div>
          <div>{t('当前余额')}: {renderQuota(currentQuota)}</div>
          <div>{t('购买后剩余')}: {renderQuota(currentQuota - priceQuota)}</div>
          <div className='text-gray-500'>{t('确认后将立即扣除钱包余额，并进入令牌发放设置。')}</div>
        </div>
      ),
      onOk: async () => purchaseSellableToken(productId),
    });
  };

  const openTopUpLink = () => {
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const fetchQuotedAmount = async (value, paymentMethod = payWay, options = {}) => {
    const { showErrorToast = false } = options;
    const normalizedValue = Number(value ?? topUpCount);
    if (!paymentMethod) {
      setAmount(0);
      return 0;
    }

    const requestId = amountRequestRef.current + 1;
    amountRequestRef.current = requestId;
    setAmountLoading(true);

    try {
      const endpoint = paymentMethod === 'stripe' ? '/api/user/stripe/amount' : '/api/user/amount';
      const res = await API.post(endpoint, { amount: parseFloat(normalizedValue) });
      if (requestId !== amountRequestRef.current) return null;
      if (res?.data?.message === 'success') {
        const quotedAmount = parseFloat(res.data.data);
        setAmount(quotedAmount);
        return quotedAmount;
      }
      setAmount(0);
      if (showErrorToast) showError(res?.data?.data || t('获取金额失败'));
      return 0;
    } catch (err) {
      if (requestId === amountRequestRef.current) setAmount(0);
      return null;
    } finally {
      if (requestId === amountRequestRef.current) setAmountLoading(false);
    }
  };

  const debouncedGetAmount = useDebouncedCallback((value, options = {}) => {
    fetchQuotedAmount(value, options.paymentMethod || payWay, options);
  }, 400);

  const selectPayMethod = (payment) => {
    if (!payment || !canUsePayMethodForAmount(payment, topUpCount)) return;
    setPayWay(payment);
  };

  const openPaymentConfirm = async () => {
    if (!payWay) {
      showError(t('请选择支付方式'));
      return;
    }
    if (topUpCount < minTopUp) {
      showError(t('充值数量不能小于') + minTopUp);
      return;
    }
    if (!canUsePayMethodForAmount(payWay, topUpCount)) {
      const payMethod = payMethods.find((item) => item.type === payWay);
      const minTopupVal = Number(payMethod?.min_topup) || 0;
      if (minTopupVal > Number(topUpCount || 0)) {
        showError(t('此支付方式最低充值金额为') + ' ' + minTopupVal);
      } else {
        showError(t('请选择支付方式'));
      }
      return;
    }

    setPaymentLoading(true);
    try {
      debouncedGetAmount.cancel();
      const quotedAmount = await fetchQuotedAmount(topUpCount, payWay, {
        showErrorToast: true,
      });
      if (quotedAmount !== null && quotedAmount > 0) setOpen(true);
    } catch (error) {
    } finally {
      setPaymentLoading(false);
    }
  };

  const creemPreTopUp = (product) => {
    if (!product?.productId) {
      showError(t('产品配置错误'));
      return;
    }
    setSelectedCreemProduct(product);
    setCreemOpen(true);
  };

  const onlineTopUp = async () => {
    if (!payWay) return;
    setConfirmLoading(true);
    try {
      const endpoint = payWay === 'stripe' ? '/api/user/stripe/pay' : '/api/user/pay';
      const res = await API.post(endpoint, { amount: parseInt(topUpCount), payment_method: payWay });
      if (res?.data?.message === 'success') {
        if (payWay === 'stripe') {
          window.open(res.data.data.pay_link, '_blank');
        } else {
          let params = res.data.data;
          let url = res.data.url;
          let form = document.createElement('form');
          form.action = url;
          form.method = 'POST';
          form.target = '_blank';
          for (let key in params) {
            let input = document.createElement('input');
            input.type = 'hidden';
            input.name = key;
            input.value = params[key];
            form.appendChild(input);
          }
          document.body.appendChild(form);
          form.submit();
          document.body.removeChild(form);
        }
      } else {
        showError(res?.data?.message || t('支付失败'));
      }
    } catch (err) {} finally {
      setOpen(false);
      setConfirmLoading(false);
    }
  };

  const onlineCreemTopUp = async () => {
    if (!selectedCreemProduct?.productId) return;
    setConfirmLoading(true);
    try {
      const res = await API.post('/api/user/creem/pay', {
        product_id: selectedCreemProduct.productId,
        payment_method: 'creem',
      });
      if (res?.data?.message === 'success') {
        window.open(res.data.data.checkout_url, '_blank');
      } else {
        showError(res?.data?.message || t('支付失败'));
      }
    } catch (err) {} finally {
      setCreemOpen(false);
      setConfirmLoading(false);
    }
  };

  const getUserQuota = async () => {
    let res = await API.get(`/api/user/self`);
    if (res.data?.success) {
      userDispatch({ type: 'login', payload: res.data.data });
      setUserData(res.data.data);
    }
  };

  const getSubscriptionPlans = async () => {
    setSubscriptionLoading(true);
    try {
      const res = await API.get('/api/subscription/plans');
      if (res.data?.success) setSubscriptionPlans(res.data.data || []);
    } finally {
      setSubscriptionLoading(false);
    }
  };

  const getSubscriptionSelf = async () => {
    try {
      const res = await API.get('/api/subscription/self');
      if (res.data?.success) {
        const d = res.data.data;
        setBillingPreference(d?.billing_preference || 'subscription_first');
        setActiveSubscriptions(d?.subscriptions || []);
        setActiveQuantityByPlan(d?.active_quantity_by_plan || {});
        setAllSubscriptions(d?.all_subscriptions || []);
        setPendingSubscriptionIssuances(d?.pending_issuances || []);
      }
    } catch (e) {}
  };

  const updateBillingPreference = async (pref) => {
    const previousPref = billingPreference;
    setBillingPreference(pref);
    try {
      const res = await API.put('/api/subscription/self/preference', { billing_preference: pref });
      if (!res.data?.success) setBillingPreference(previousPref);
    } catch (e) {
      setBillingPreference(previousPref);
    }
  };

  const getTopupInfo = async () => {
    try {
      const res = await API.get('/api/user/topup/info');
      if (res.data?.success) {
        const data = res.data.data;
        setTopupInfo({ amount_options: data.amount_options || [], discount: data.discount || {} });
        let pMethods = Array.isArray(data.pay_methods) ? data.pay_methods : JSON.parse(data.pay_methods || '[]');
        setPayMethods(pMethods.filter(m => m.name && m.type));
        setEnableStripeTopUp(!!data.enable_stripe_topup);
        setEnableOnlineTopUp(!!data.enable_online_topup);
        setEnableCreemTopUp(!!data.enable_creem_topup);
        const mTopUp = data.enable_online_topup ? data.min_topup : (data.enable_stripe_topup ? data.stripe_min_topup : 1);
        setMinTopUp(mTopUp);
        setTopUpCount(mTopUp);
        setCreemProducts(JSON.parse(data.creem_products || '[]'));
        if (data.amount_options?.length > 0) {
          setPresetAmounts(data.amount_options.map(v => ({ value: v, discount: data.discount[v] || 1.0 })));
        } else {
          setPresetAmounts([1, 5, 10, 30, 50, 100, 300, 500].map(m => ({ value: mTopUp * m })));
        }
      }
    } catch (e) {}
  };

  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    if (res.data?.success) setAffLink(`${window.location.origin}/register?aff=${res.data.data}`);
  };

  const handleAffLinkClick = async () => {
    if (!affLink) {
      showError(t('查询失败，请稍后重试'));
      return;
    }
    await handleCopyUrl(affLink, t);
  };

  const transfer = async () => {
    const transferQuota = displayAmountToQuota(transferAmount);
    if (transferQuota < getQuotaPerUnit()) return;
    const res = await API.post(`/api/user/aff_transfer`, { quota: transferQuota });
    if (res.data?.success) {
      getUserQuota();
      setOpenTransfer(false);
    }
  };

  const selectPresetAmount = (preset) => {
    setTopUpCount(preset.value);
    setSelectedPreset(preset.value);
  };

  const formatLargeNumber = (num) => num.toString();

  useEffect(() => {
    getUserQuota();
    loadSellableTokenProducts();
    loadPendingSellableIssuances();
    loadActiveSellableTokens();
    getAffLink();
    getTopupInfo();
    getSubscriptionPlans();
    getSubscriptionSelf();
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      setTopUpLink(statusState.status.top_up_link || '');
      setStatusLoading(false);
    }
  }, [statusState?.status]);

  useEffect(() => {
    if (!payMethods.length) {
      if (payWay) setPayWay('');
      setAmount(0);
      return;
    }

    if (payWay && canUsePayMethodForAmount(payWay, topUpCount)) return;

    const nextPayMethod = pickAvailablePayMethod(topUpCount);
    if (nextPayMethod) {
      if (nextPayMethod.type !== payWay) setPayWay(nextPayMethod.type);
      return;
    }

    if (payWay) setPayWay('');
    setAmount(0);
  }, [payMethods, payWay, topUpCount, enableOnlineTopUp, enableStripeTopUp]);

  useEffect(() => {
    debouncedGetAmount.cancel();
    if (!payWay || !canUsePayMethodForAmount(payWay, topUpCount)) {
      setAmount(0);
      return;
    }

    debouncedGetAmount(topUpCount, { paymentMethod: payWay });
    return () => debouncedGetAmount.cancel();
  }, [payWay, topUpCount]);

  const renderAmount = () => `${getPaymentCurrencySymbol()}${Number(amount || 0).toFixed(2)}`;
  const showSellableTokenTab =
    sellableTokenLoading ||
    sellableTokenProducts.length > 0 ||
    activeSellableTokens.length > 0 ||
    pendingSellableIssuances.length > 0;

  return (
    <div className='w-full max-w-7xl mx-auto relative min-h-screen lg:min-h-0 mt-[60px] px-2'>
      <TransferModal t={t} openTransfer={openTransfer} transfer={transfer} handleTransferCancel={() => setOpenTransfer(false)} userState={userState} renderQuota={renderQuota} getQuotaPerUnit={getQuotaPerUnit} transferAmount={transferAmount} setTransferAmount={setTransferAmount} />
      <PaymentConfirmModal t={t} open={open} onlineTopUp={onlineTopUp} handleCancel={() => setOpen(false)} confirmLoading={confirmLoading} topUpCount={topUpCount} renderQuotaWithAmount={renderQuotaWithAmount} amountLoading={amountLoading} renderAmount={renderAmount} payWay={payWay} payMethods={payMethods} amountNumber={amount} discountRate={topupInfo?.discount?.[topUpCount] || 1.0} />
      <TopupHistoryModal visible={openHistory} onCancel={() => setOpenHistory(false)} t={t} />
      <SellableTokenIssuanceModal visible={sellableTokenIssuanceVisible} issuanceId={sellableTokenIssuanceId} onCancel={() => { setSellableTokenIssuanceVisible(false); setSellableTokenIssuanceId(0); }} onSuccess={() => { setSellableTokenIssuanceVisible(false); setSellableTokenIssuanceId(0); loadPendingSellableIssuances(); loadActiveSellableTokens(); }} />
      <SubscriptionIssuanceModal visible={subscriptionIssuanceVisible} issuanceId={subscriptionIssuanceId} onCancel={() => { setSubscriptionIssuanceVisible(false); setSubscriptionIssuanceId(0); }} onSuccess={() => { setSubscriptionIssuanceVisible(false); setSubscriptionIssuanceId(0); getSubscriptionSelf(); }} />

      <div className='grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.22fr)_minmax(380px,1fr)]'>
        <div>
          <RechargeCard
            t={t}
            enableOnlineTopUp={enableOnlineTopUp}
            enableStripeTopUp={enableStripeTopUp}
            enableCreemTopUp={enableCreemTopUp}
            creemProducts={creemProducts}
            creemPreTopUp={creemPreTopUp}
            presetAmounts={presetAmounts}
            selectedPreset={selectedPreset}
            selectPresetAmount={selectPresetAmount}
            formatLargeNumber={formatLargeNumber}
            topUpCount={topUpCount}
            minTopUp={minTopUp}
            renderQuotaWithAmount={renderQuotaWithAmount}
            setTopUpCount={setTopUpCount}
            setSelectedPreset={setSelectedPreset}
            renderAmount={renderAmount}
            amountLoading={amountLoading}
            payMethods={payMethods}
            selectPayMethod={selectPayMethod}
            openPaymentConfirm={openPaymentConfirm}
            paymentLoading={paymentLoading}
            payWay={payWay}
            userState={userState}
            renderQuota={renderQuota}
            statusLoading={statusLoading}
            topupInfo={topupInfo}
            onOpenHistory={() => setOpenHistory(true)}
            subscriptionLoading={subscriptionLoading}
            subscriptionPlans={subscriptionPlans}
            billingPreference={billingPreference}
            onChangeBillingPreference={updateBillingPreference}
            activeSubscriptions={activeSubscriptions}
            activeQuantityByPlan={activeQuantityByPlan}
            allSubscriptions={allSubscriptions}
            reloadSubscriptionSelf={getSubscriptionSelf}
            showSellableTokenTab
            sellableTokenContent={
              <div className='space-y-6'>
                <MyTokensCard
                  t={t}
                  activeSellableTokens={activeSellableTokens}
                  onRefresh={loadActiveSellableTokens}
                />

                {sellableTokenProducts.length === 0 ? (<Card className='!rounded-xl border border-dashed border-slate-200 bg-slate-50/50'><div className='py-12 text-center'><div className='text-base font-medium text-slate-600'>{sellableTokenLoading ? t('加载中...') : t('当前暂无可售令牌商品')}</div><div className='mt-2 text-sm text-slate-400'>{t('管理员上架后，会在这里展示可直接购买的令牌商品。')}</div></div></Card>) : (
                  <div className='grid grid-cols-1 gap-6 px-1 xl:grid-cols-2'>
                    {sellableTokenProducts.map((item, index) => {
                      const product = item?.product || {};
                      const isRecommended =
                        index === 0 && sellableTokenProducts.length > 1;
                      const productPriceQuota = Number(product?.price_quota || 0);
                      const productHighlights = [
                        { icon: <Coins size={14} />, label: t('钱包售价'), value: renderQuota(productPriceQuota) },
                        { icon: <ShieldCheck size={14} />, label: t('总额度'), value: Number(product?.total_quota || 0) === 0 ? t('不限') : renderQuota(product.total_quota) },
                        { icon: <Clock3 size={14} />, label: t('有效期'), value: renderValidityLabel(t, product?.validity_seconds) },
                        ...(product.package_enabled ? [{ icon: <Gauge size={14} />, label: t('周期额度'), value: `${renderQuota(product.package_limit_quota || 0)} / ${renderPeriodLabel(t, product.package_period)}` }] : []),
                        { icon: <Zap size={14} />, label: t('速率限制'), value: (() => { const hasConcurrency = Number(product.max_concurrency || 0) > 0; const hasWindow = Number(product.window_request_limit || 0) > 0 && Number(product.window_seconds || 0) > 0; if (!hasConcurrency && !hasWindow) return t('不限'); const parts = []; if (hasConcurrency) parts.push(formatConcurrencyLabel(product.max_concurrency, t)); if (hasWindow) parts.push(formatWindowLimitShort(product.window_seconds, product.window_request_limit, t)); return parts.join(' · '); })() },
                      ];
                      return (
                        <Card
                          key={product.id}
                          className={`!rounded-xl border border-slate-200 flex flex-col h-full transition-all hover:shadow-lg ${
                            isRecommended ? 'ring-2 ring-purple-500' : ''
                          }`}
                          bodyStyle={{ padding: 20, display: 'flex', flexDirection: 'column', flex: 1 }}
                        >
                          {isRecommended && (
                            <div className='mb-3'>
                              <Tag color='purple' shape='circle' size='small'>
                                <Sparkles size={10} className='mr-1' />
                                {t('推荐')}
                              </Tag>
                            </div>
                          )}

                          <div className='flex items-start justify-between gap-4 mb-4'>
                            <div className='min-w-0 flex-1'>
                              <Title heading={5}>{product.name}</Title>
                              {product.subtitle ? (
                                <Text
                                  type='tertiary'
                                  size='small'
                                  className='block mt-1'
                                  ellipsis={{ rows: 1, showTooltip: true }}
                                >
                                  {product.subtitle}
                                </Text>
                              ) : null}
                            </div>
                          </div>

                          {/* 中间：参数矩阵 */}
                          <div className='grid grid-cols-2 gap-2 mb-6'>
                            {productHighlights.map((highlight) => (
                              <div key={highlight.label} className='rounded-lg border border-slate-100 bg-slate-50/30 p-3'>
                                <div className='flex items-center gap-1.5 text-slate-400 mb-1'>
                                  {highlight.icon}
                                  <Text type='tertiary' size='extra-small' className='uppercase'>{highlight.label}</Text>
                                </div>
                                <Text strong size='small' className='text-slate-800'>{highlight.value}</Text>
                              </div>
                            ))}
                          </div>

                          {/* 底部：固定位置的分割线和按钮 */}
                          <div className='mt-auto'>
                            <Divider margin={12} />
                            <div className='mt-2 flex justify-end'>
                              <Button
                                theme='solid'
                                type='primary'
                                className='!rounded-lg'
                                icon={<ArrowRight size={16} />}
                                onClick={() => confirmPurchaseSellableToken(product)}
                              >
                                {t('立即购买')}
                              </Button>
                            </div>
                          </div>
                        </Card>
                      );
                    })}
                  </div>
                )}
              </div>
            }
          />
        </div>
        <div className='space-y-6'>
          {/* 待发放动作卡片 - 移至侧边栏顶部 */}
          {(pendingSellableIssuances.length > 0 || pendingSubscriptionIssuances.length > 0) && (
            <Card className='!rounded-xl border border-orange-100 bg-orange-50/10' bodyStyle={{ padding: '12px' }}>
              <div className='flex items-center gap-2 mb-3'>
                <BellRing size={16} className='text-orange-500' />
                <Text strong size='small'>{t('待完成动作')}</Text>
              </div>
              <div className='space-y-2'>
                {pendingSellableIssuances.length > 0 && (
                  <div className='flex items-center justify-between gap-3 p-2 rounded-lg bg-white border border-orange-100'>
                    <div className='flex items-center gap-2 overflow-hidden'>
                      <div className='flex h-6 w-6 items-center justify-center rounded-full bg-orange-50 text-orange-600 flex-shrink-0'><Coins size={12} /></div>
                      <Text size='small' className='truncate'>{t('令牌待发放')} ({pendingSellableIssuances.length})</Text>
                    </div>
                    <Space spacing={4}>
                      <Button theme='solid' type='primary' size='extra-small' className='!rounded-full' onClick={() => { const nextId = Number(pendingSellableIssuances?.[0]?.issuance?.id || pendingSellableIssuances?.[0]?.id || 0); setSellableTokenIssuanceId(nextId); setSellableTokenIssuanceVisible(true); }}>{t('继续')}</Button>
                      <Button theme='light' type='warning' size='extra-small' className='!rounded-full' onClick={() => {
                        if (pendingSellableIssuances.length === 1) {
                          const item = pendingSellableIssuances[0];
                          const itemId = Number(item?.issuance?.id || item?.id || 0);
                          const productName = item?.product?.name || item?.issuance?.product?.name || '-';
                          const sourceType = item?.issuance?.source_type || item?.source_type || '';
                          Modal.confirm({
                            title: t('确认取消'),
                            content: (
                              <div className='space-y-1 text-sm'>
                                <div>{t('商品')}: {productName}</div>
                                {sourceType === 'wallet' && <div className='text-orange-600'>{t('钱包购买的将退还额度')}</div>}
                                {sourceType !== 'wallet' && <div className='text-gray-500'>{t('兑换码/管理员发放不退还额度')}</div>}
                                <div className='text-red-500 mt-2'>{t('此操作不可恢复')}</div>
                              </div>
                            ),
                            centered: true,
                            okType: 'danger',
                            okText: t('确认取消'),
                            onOk: async () => {
                              try {
                                const res = await API.post(`/api/user/sellable-token/issuances/${itemId}/cancel`);
                                if (res.data?.success) {
                                  showSuccess(t('已取消'));
                                  loadPendingSellableIssuances();
                                  getUserQuota();
                                } else {
                                  showError(res.data?.message || t('取消失败'));
                                }
                              } catch (e) { showError(t('请求失败')); }
                            },
                          });
                        } else {
                          Modal.confirm({
                            title: t('取消待发放令牌'),
                            content: (
                              <div className='space-y-2 text-sm'>
                                <div>{t('当前有 {{count}} 条待发放记录', { count: pendingSellableIssuances.length })}</div>
                                <div className='text-orange-600'>{t('钱包购买的将退还额度，兑换码/管理员发放不退还')}</div>
                                <div className='text-red-500'>{t('此操作不可恢复')}</div>
                              </div>
                            ),
                            centered: true,
                            okType: 'danger',
                            okText: t('全部取消'),
                            cancelText: t('逐个取消'),
                            onOk: async () => {
                              try {
                                const res = await API.post('/api/user/sellable-token/issuances/cancel-all');
                                if (res.data?.success) {
                                  const d = res.data.data || {};
                                  showSuccess(t('已取消 {{count}} 条', { count: d.cancelled_count || 0 }));
                                  loadPendingSellableIssuances();
                                  getUserQuota();
                                } else {
                                  showError(res.data?.message || t('取消失败'));
                                }
                              } catch (e) { showError(t('请求失败')); }
                            },
                            onCancel: () => {
                              Modal.info({
                                title: t('选择要取消的待发放'),
                                content: (
                                  <div className='space-y-2 max-h-64 overflow-y-auto'>
                                    {pendingSellableIssuances.map((item) => {
                                      const itemId = Number(item?.issuance?.id || item?.id || 0);
                                      const productName = item?.product?.name || item?.issuance?.product?.name || '-';
                                      const sourceType = item?.issuance?.source_type || item?.source_type || '';
                                      return (
                                        <div key={itemId} className='flex items-center justify-between p-2 rounded-lg border border-slate-100'>
                                          <div className='min-w-0'>
                                            <div className='text-sm font-medium truncate'>{productName}</div>
                                            <div className='text-xs text-gray-500'>{sourceType === 'wallet' ? t('钱包购买·可退') : t('不退还额度')}</div>
                                          </div>
                                          <Button size='small' type='danger' onClick={async () => {
                                            try {
                                              const res = await API.post(`/api/user/sellable-token/issuances/${itemId}/cancel`);
                                              if (res.data?.success) {
                                                showSuccess(t('已取消'));
                                                loadPendingSellableIssuances();
                                                getUserQuota();
                                              } else {
                                                showError(res.data?.message || t('取消失败'));
                                              }
                                            } catch (e) { showError(t('请求失败')); }
                                          }}>{t('取消')}</Button>
                                        </div>
                                      );
                                    })}
                                  </div>
                                ),
                                centered: true,
                                okText: t('关闭'),
                              });
                            },
                          });
                        }
                      }}>{t('取消')}</Button>
                    </Space>
                  </div>
                )}
                {pendingSubscriptionIssuances.length > 0 && (
                  <div className='flex items-center justify-between gap-3 p-2 rounded-lg bg-white border border-purple-100'>
                    <div className='flex items-center gap-2 overflow-hidden'>
                      <div className='flex h-6 w-6 items-center justify-center rounded-full bg-purple-50 text-purple-600 flex-shrink-0'><Sparkles size={12} /></div>
                      <Text size='small' className='truncate'>{t('套餐待发放')} ({pendingSubscriptionIssuances.length})</Text>
                    </div>
                    <Button theme='solid' type='primary' size='extra-small' className='!rounded-full' onClick={() => { const nextId = Number(pendingSubscriptionIssuances?.[0]?.id || 0); setSubscriptionIssuanceId(nextId); setSubscriptionIssuanceVisible(true); }}>{t('继续')}</Button>
                  </div>
                )}
              </div>
            </Card>
          )}

          <InvitationCard t={t} userState={userState} renderQuota={renderQuota} setOpenTransfer={setOpenTransfer} affLink={affLink} handleAffLinkClick={handleAffLinkClick} />
          <Card className='!rounded-xl shadow-sm border border-slate-200' title={<Text type='tertiary' strong size='small'>{t('兑换码充值')}</Text>} bodyStyle={{ padding: '12px' }}>
            <Form initValues={{ redemptionCode }}>
              <Form.Input field='redemptionCode' noLabel placeholder={t('请输入兑换码')} value={redemptionCode} onChange={setRedemptionCode} prefix={<IconGift />} suffix={<Button type='primary' theme='solid' size='small' onClick={() => topUp()} loading={isSubmitting}>{t('立即兑换')}</Button>} showClear style={{ width: '100%' }} extraText={topUpLink && (<Text type='tertiary' size='extra-small'>{t('在找兑换码？')}<Text type='secondary' underline className='cursor-pointer' onClick={openTopUpLink}>{t('购买兑换码')}</Text></Text>)} />
            </Form>
          </Card>
        </div>
      </div>

      <Modal title={t('选择续费目标')} visible={redeemTargetModalOpen} onOk={() => topUp(selectedRenewTargetId, selectedPurchaseMode || 'renew')} onCancel={() => { setRedeemTargetModalOpen(false); setSelectedRenewTargetId(0); }} size='small' centered confirmLoading={isSubmitting}><p>{t('当前套餐存在多条可续费订阅，请选择要续费的目标')}：{redeemTargetPlanTitle || '-'}</p><Select value={selectedRenewTargetId} onChange={v => setSelectedRenewTargetId(Number(v || 0))} style={{ width: '100%' }} optionList={redeemTargetOptions.map(sub => ({ label: `${t('订阅')} #${sub.id} · ${t('到期时间')} ${timestamp2string(sub.end_time)}`, value: sub.id }))} /></Modal>
      <Modal title={t('选择兑换方式')} visible={purchaseModeModalOpen} onOk={() => { setPurchaseModeModalOpen(false); topUp(0, selectedPurchaseMode); }} onCancel={() => setPurchaseModeModalOpen(false)} size='small' centered confirmLoading={isSubmitting}><p>{t('您正在兑换套餐')}：{purchaseModePlanTitle || '-'}</p><p>{t('请选择兑换方式')}：</p><Select value={selectedPurchaseMode} onChange={setSelectedPurchaseMode} style={{ width: '100%' }} optionList={[{ label: t('叠加（新增一条订阅）'), value: 'stack' }, { label: t('续费（延长现有订阅）'), value: 'renew' }]} /></Modal>
      <Modal title={t('确定要充值 $')} visible={creemOpen} onOk={onlineCreemTopUp} onCancel={() => setCreemOpen(false)} maskClosable={false} size='small' centered confirmLoading={confirmLoading}>{selectedCreemProduct && (<div className='space-y-2'><p>{t('产品名称')}：{selectedCreemProduct.name}</p><p>{t('价格')}：{selectedCreemProduct.currency === 'EUR' ? '€' : '$'}{selectedCreemProduct.price}</p><p>{t('充值额度')}：{selectedCreemProduct.quota}</p><p>{t('是否确认充值？')}</p></div>)}</Modal>
    </div>
  );
};

export default TopUp;
