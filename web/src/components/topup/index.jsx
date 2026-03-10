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
  renderQuota,
  renderQuotaWithAmount,
  copy,
  getQuotaPerUnit,
  timestamp2string,
} from '../../helpers';
import { Modal, Select, Toast } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';

import RechargeCard from './RechargeCard';
import InvitationCard from './InvitationCard';
import TransferModal from './modals/TransferModal';
import PaymentConfirmModal from './modals/PaymentConfirmModal';
import TopupHistoryModal from './modals/TopupHistoryModal';

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
  const [priceRatio, setPriceRatio] = useState(statusState?.status?.price || 1);

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

  // 邀请状态
  const [affLink, setAffLink] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(0);

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
      // 统一兑换接口同时支持余额码与套餐码，前端不再根据入口拆两套逻辑。
      const res = await API.post('/api/user/redeem', payload);
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        if (data?.benefit_type === 'subscription') {
          // 套餐兑换后主要是刷新订阅态，而不是直接改本地余额。
          Modal.success({
            title: t('套餐兑换成功！'),
            content:
              data?.action_summary ||
              t('已成功兑换套餐：') + (data?.plan_title || '-'),
            centered: true,
          });
          await getSubscriptionSelf();
        } else {
          // 余额码仍然保留旧体验：弹出额度到账提示，并乐观更新本地余额展示。
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
        // 套餐码未指定兑换方式时，弹出选择框让用户选择叠加或续费。
        if (data?.code === 'redeem_select_purchase_mode') {
          setPurchaseModePlanTitle(data?.plan_title || '');
          setSelectedPurchaseMode('stack');
          setPurchaseModeModalOpen(true);
          return;
        }
        // 同套餐存在多条可续费订阅时，弹出选择框，由用户决定续到哪一条。
        if (data?.code === 'redeem_select_renew_target') {
          const options = (data?.options || []).map((item) => item?.subscription).filter(Boolean);
          setRedeemTargetOptions(options);
          setRedeemTargetPlanTitle(data?.plan_title || '');
          setSelectedRenewTargetId(Number(options?.[0]?.id || 0));
          setRedeemTargetModalOpen(true);
          return;
        }
        showError(message);
      }
    } catch (err) {
      showError(t('请求失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTopUpLink = () => {
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const preTopUp = async (payment) => {
    if (payment === 'stripe') {
      if (!enableStripeTopUp) {
        showError(t('管理员未开启Stripe充值！'));
        return;
      }
    } else {
      if (!enableOnlineTopUp) {
        showError(t('管理员未开启在线充值！'));
        return;
      }
    }

    setPayWay(payment);
    setPaymentLoading(true);
    try {
      if (payment === 'stripe') {
        await getStripeAmount();
      } else {
        await getAmount();
      }

      if (topUpCount < minTopUp) {
        showError(t('充值数量不能小于') + minTopUp);
        return;
      }
      setOpen(true);
    } catch (error) {
      showError(t('获取金额失败'));
    } finally {
      setPaymentLoading(false);
    }
  };

  const onlineTopUp = async () => {
    if (payWay === 'stripe') {
      // Stripe 支付分支
      if (amount === 0) {
        await getStripeAmount();
      }
    } else {
      // 常规支付分支
      if (amount === 0) {
        await getAmount();
      }
    }

    if (topUpCount < minTopUp) {
      showError('充值数量不能小于' + minTopUp);
      return;
    }
    setConfirmLoading(true);
    try {
      let res;
      if (payWay === 'stripe') {
        // Stripe 支付请求
        res = await API.post('/api/user/stripe/pay', {
          amount: parseInt(topUpCount),
          payment_method: 'stripe',
        });
      } else {
        // 常规支付请求
        res = await API.post('/api/user/pay', {
          amount: parseInt(topUpCount),
          payment_method: payWay,
        });
      }

      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          if (payWay === 'stripe') {
            // Stripe 回调处理
            window.open(data.pay_link, '_blank');
          } else {
            // 常规支付表单提交
            let params = data;
            let url = res.data.url;
            let form = document.createElement('form');
            form.action = url;
            form.method = 'POST';
            let isSafari =
              navigator.userAgent.indexOf('Safari') > -1 &&
              navigator.userAgent.indexOf('Chrome') < 1;
            if (!isSafari) {
              form.target = '_blank';
            }
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
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
      showError(t('支付请求失败'));
    } finally {
      setOpen(false);
      setConfirmLoading(false);
    }
  };

  const creemPreTopUp = async (product) => {
    if (!enableCreemTopUp) {
      showError(t('管理员未开启 Creem 充值！'));
      return;
    }
    setSelectedCreemProduct(product);
    setCreemOpen(true);
  };

  const onlineCreemTopUp = async () => {
    if (!selectedCreemProduct) {
      showError(t('请选择产品'));
      return;
    }
    // Validate product has required fields
    if (!selectedCreemProduct.productId) {
      showError(t('产品配置错误，请联系管理员'));
      return;
    }
    setConfirmLoading(true);
    try {
      const res = await API.post('/api/user/creem/pay', {
        product_id: selectedCreemProduct.productId,
        payment_method: 'creem',
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          processCreemCallback(data);
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
      showError(t('支付请求失败'));
    } finally {
      setCreemOpen(false);
      setConfirmLoading(false);
    }
  };

  const processCreemCallback = (data) => {
    // 与 Stripe 保持一致的行为模式。
    window.open(data.checkout_url, '_blank');
  };

  const getUserQuota = async () => {
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
    } else {
      showError(message);
    }
  };

  const getSubscriptionPlans = async () => {
    setSubscriptionLoading(true);
    try {
      const res = await API.get('/api/subscription/plans');
      if (res.data?.success) {
        setSubscriptionPlans(res.data.data || []);
      }
    } catch (e) {
      setSubscriptionPlans([]);
    } finally {
      setSubscriptionLoading(false);
    }
  };

  const getSubscriptionSelf = async () => {
    try {
      const res = await API.get('/api/subscription/self');
      if (res.data?.success) {
        setBillingPreference(
          res.data.data?.billing_preference || 'subscription_first',
        );
        // Active subscriptions
        const activeSubs = res.data.data?.subscriptions || [];
        setActiveSubscriptions(activeSubs);
        setActiveQuantityByPlan(res.data.data?.active_quantity_by_plan || {});
        // All subscriptions (including expired)
        const allSubs = res.data.data?.all_subscriptions || [];
        setAllSubscriptions(allSubs);
      }
    } catch (e) {
      // ignore
    }
  };

  const updateBillingPreference = async (pref) => {
    const previousPref = billingPreference;
    setBillingPreference(pref);
    try {
      const res = await API.put('/api/subscription/self/preference', {
        billing_preference: pref,
      });
      if (res.data?.success) {
        showSuccess(t('更新成功'));
        const normalizedPref =
          res.data?.data?.billing_preference || pref || previousPref;
        setBillingPreference(normalizedPref);
      } else {
        showError(res.data?.message || t('更新失败'));
        setBillingPreference(previousPref);
      }
    } catch (e) {
      showError(t('请求失败'));
      setBillingPreference(previousPref);
    }
  };

  // 拉取充值配置。
  const getTopupInfo = async () => {
    try {
      const res = await API.get('/api/user/topup/info');
      const { message, data, success } = res.data;
      if (success) {
        setTopupInfo({
          amount_options: data.amount_options || [],
          discount: data.discount || {},
        });

        // 标准化支付方式数据。
        let payMethods = data.pay_methods || [];
        try {
          if (typeof payMethods === 'string') {
            payMethods = JSON.parse(payMethods);
          }
          if (payMethods && payMethods.length > 0) {
            // 过滤缺少 name 或 type 的支付方式。
            payMethods = payMethods.filter((method) => {
              return method.name && method.type;
            });
            // 缺失颜色时补默认值。
            payMethods = payMethods.map((method) => {
              // 规范化最小充值值。
              const normalizedMinTopup = Number(method.min_topup);
              method.min_topup = Number.isFinite(normalizedMinTopup)
                ? normalizedMinTopup
                : 0;

              // 用后端字段回填 Stripe 最小充值值。
              if (
                method.type === 'stripe' &&
                (!method.min_topup || method.min_topup <= 0)
              ) {
                const stripeMin = Number(data.stripe_min_topup);
                if (Number.isFinite(stripeMin)) {
                  method.min_topup = stripeMin;
                }
              }

              if (!method.color) {
                if (method.type === 'alipay') {
                  method.color = 'rgba(var(--semi-blue-5), 1)';
                } else if (method.type === 'wxpay') {
                  method.color = 'rgba(var(--semi-green-5), 1)';
                } else if (method.type === 'stripe') {
                  method.color = 'rgba(var(--semi-purple-5), 1)';
                } else {
                  method.color = 'rgba(var(--semi-primary-5), 1)';
                }
              }
              return method;
            });
          } else {
            payMethods = [];
          }

          // 若启用 Stripe，则将其加入支付方式列表。
          // 该逻辑现由后端处理：启用时 Stripe 会出现在 pay_methods。

          setPayMethods(payMethods);
          const enableStripeTopUp = data.enable_stripe_topup || false;
          const enableOnlineTopUp = data.enable_online_topup || false;
          const enableCreemTopUp = data.enable_creem_topup || false;
          const minTopUpValue = enableOnlineTopUp
            ? data.min_topup
            : enableStripeTopUp
              ? data.stripe_min_topup
              : 1;
          setEnableOnlineTopUp(enableOnlineTopUp);
          setEnableStripeTopUp(enableStripeTopUp);
          setEnableCreemTopUp(enableCreemTopUp);
          setMinTopUp(minTopUpValue);
          setTopUpCount(minTopUpValue);

          // 设置 Creem 商品列表。
          try {
            console.log(' data is ?', data);
            console.log(' creem products is ?', data.creem_products);
            const products = JSON.parse(data.creem_products || '[]');
            setCreemProducts(products);
          } catch (e) {
            setCreemProducts([]);
          }

          // 当无自定义选项时，根据最小充值生成预设金额。
          if (topupInfo.amount_options.length === 0) {
            setPresetAmounts(generatePresetAmounts(minTopUpValue));
          }

          // 初始化应付金额预览。
          getAmount(minTopUpValue);
        } catch (e) {
          console.log('解析支付方式失败:', e);
          setPayMethods([]);
        }

        // 存在自定义金额选项时，替换默认预设。
        if (data.amount_options && data.amount_options.length > 0) {
          const customPresets = data.amount_options.map((amount) => ({
            value: amount,
            discount: data.discount[amount] || 1.0,
          }));
          setPresetAmounts(customPresets);
        }
      } else {
        console.error('获取充值配置失败:', data);
      }
    } catch (error) {
      console.error('获取充值配置异常:', error);
    }
  };

  // 获取邀请链接。
  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    const { success, message, data } = res.data;
    if (success) {
      let link = `${window.location.origin}/register?aff=${data}`;
      setAffLink(link);
    } else {
      showError(message);
    }
  };

  // 划转邀请额度。
  const transfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
      return;
    }
    const res = await API.post(`/api/user/aff_transfer`, {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      getUserQuota().then();
    } else {
      showError(message);
    }
  };

  // 复制邀请链接。
  const handleAffLinkClick = async () => {
    await copy(affLink);
    showSuccess(t('邀请链接已复制到剪切板'));
  };

  useEffect(() => {
    // 始终刷新用户数据，确保余额与统计准确。
    getUserQuota().then();
    setTransferAmount(getQuotaPerUnit());
  }, []);

  useEffect(() => {
    if (affFetchedRef.current) return;
    affFetchedRef.current = true;
    getAffLink().then();
  }, []);

  // statusState 可用后再拉取充值数据。
  useEffect(() => {
    getTopupInfo().then();
    getSubscriptionPlans().then();
    getSubscriptionSelf().then();
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      // const minTopUpValue = statusState.status.min_topup || 1;
      // setMinTopUp(minTopUpValue);
      // setTopUpCount(minTopUpValue);
      setTopUpLink(statusState.status.top_up_link || '');
      setPriceRatio(statusState.status.price || 1);

      setStatusLoading(false);
    }
  }, [statusState?.status]);

  const renderAmount = () => {
    return amount + ' ' + t('元');
  };

  const getAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/amount', {
        amount: parseFloat(value),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    }
    setAmountLoading(false);
  };

  const getStripeAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/stripe/amount', {
        amount: parseFloat(value),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    } finally {
      setAmountLoading(false);
    }
  };

  const handleCancel = () => {
    setOpen(false);
  };

  const handleTransferCancel = () => {
    setOpenTransfer(false);
  };

  const handleOpenHistory = () => {
    setOpenHistory(true);
  };

  const handleHistoryCancel = () => {
    setOpenHistory(false);
  };

  const handleCreemCancel = () => {
    setCreemOpen(false);
    setSelectedCreemProduct(null);
  };

  // 选择预设充值金额。
  const selectPresetAmount = (preset) => {
    setTopUpCount(preset.value);
    setSelectedPreset(preset.value);

    // 计算折扣后的实际应付金额。
    const discount = preset.discount || topupInfo.discount[preset.value] || 1.0;
    const discountedAmount = preset.value * priceRatio * discount;
    setAmount(discountedAmount);
  };

  // 格式化大数字展示。
  const formatLargeNumber = (num) => {
    return num.toString();
  };

  // 根据最小充值生成预设金额选项。
  const generatePresetAmounts = (minAmount) => {
    const multipliers = [1, 5, 10, 30, 50, 100, 300, 500];
    return multipliers.map((multiplier) => ({
      value: minAmount * multiplier,
    }));
  };

  return (
    <div className='w-full max-w-7xl mx-auto relative min-h-screen lg:min-h-0 mt-[60px] px-2'>
      {/* 划转弹窗 */}
      <TransferModal
        t={t}
        openTransfer={openTransfer}
        transfer={transfer}
        handleTransferCancel={handleTransferCancel}
        userState={userState}
        renderQuota={renderQuota}
        getQuotaPerUnit={getQuotaPerUnit}
        transferAmount={transferAmount}
        setTransferAmount={setTransferAmount}
      />

      {/* 充值确认弹窗 */}
      <PaymentConfirmModal
        t={t}
        open={open}
        onlineTopUp={onlineTopUp}
        handleCancel={handleCancel}
        confirmLoading={confirmLoading}
        topUpCount={topUpCount}
        renderQuotaWithAmount={renderQuotaWithAmount}
        amountLoading={amountLoading}
        renderAmount={renderAmount}
        payWay={payWay}
        payMethods={payMethods}
        amountNumber={amount}
        discountRate={topupInfo?.discount?.[topUpCount] || 1.0}
      />

      {/* 充值记录弹窗 */}
      <TopupHistoryModal
        visible={openHistory}
        onCancel={handleHistoryCancel}
        t={t}
      />

      {/* Creem 确认弹窗 */}
      <Modal
        title={t('确定要充值 $')}
        visible={creemOpen}
        onOk={onlineCreemTopUp}
        onCancel={handleCreemCancel}
        maskClosable={false}
        size='small'
        centered
        confirmLoading={confirmLoading}
      >
        {selectedCreemProduct && (
          <>
            <p>
              {t('产品名称')}：{selectedCreemProduct.name}
            </p>
            <p>
              {t('价格')}：{selectedCreemProduct.currency === 'EUR' ? '€' : '$'}
              {selectedCreemProduct.price}
            </p>
            <p>
              {t('充值额度')}：{selectedCreemProduct.quota}
            </p>
            <p>{t('是否确认充值？')}</p>
          </>
        )}
      </Modal>

      {/* 兑换码续费目标选择弹窗 */}
      <Modal
        title={t('选择续费目标')}
        visible={redeemTargetModalOpen}
        onOk={() => topUp(selectedRenewTargetId, selectedPurchaseMode || 'renew')}
        onCancel={() => {
          setRedeemTargetModalOpen(false);
          setRedeemTargetOptions([]);
          setRedeemTargetPlanTitle('');
          setSelectedRenewTargetId(0);
        }}
        size='small'
        centered
        confirmLoading={isSubmitting}
      >
        <p>
          {t('当前套餐存在多条可续费订阅，请选择要续费的目标')}：
          {redeemTargetPlanTitle || '-'}
        </p>
        <Select
          value={selectedRenewTargetId}
          onChange={(value) => setSelectedRenewTargetId(Number(value || 0))}
          style={{ width: '100%' }}
          optionList={redeemTargetOptions.map((sub) => ({
            label: `${t('订阅')} #${sub.id} · ${t('到期时间')} ${timestamp2string(sub.end_time)}`,
            value: sub.id,
          }))}
        />
      </Modal>

      {/* 套餐码兑换方式选择弹窗 */}
      <Modal
        title={t('选择兑换方式')}
        visible={purchaseModeModalOpen}
        onOk={() => {
          setPurchaseModeModalOpen(false);
          topUp(0, selectedPurchaseMode);
        }}
        onCancel={() => {
          setPurchaseModeModalOpen(false);
          setPurchaseModePlanTitle('');
          setSelectedPurchaseMode('stack');
        }}
        size='small'
        centered
        confirmLoading={isSubmitting}
      >
        <p>
          {t('您正在兑换套餐')}：{purchaseModePlanTitle || '-'}
        </p>
        <p>{t('请选择兑换方式')}：</p>
        <Select
          value={selectedPurchaseMode}
          onChange={(value) => setSelectedPurchaseMode(value)}
          style={{ width: '100%' }}
          optionList={[
            { label: t('叠加（新增一条订阅）'), value: 'stack' },
            { label: t('续费（延长现有订阅）'), value: 'renew' },
          ]}
        />
      </Modal>

      {/* 主布局区域 */}
      <div className='grid grid-cols-1 lg:grid-cols-2 gap-6'>
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
          priceRatio={priceRatio}
          topUpCount={topUpCount}
          minTopUp={minTopUp}
          renderQuotaWithAmount={renderQuotaWithAmount}
          getAmount={getAmount}
          setTopUpCount={setTopUpCount}
          setSelectedPreset={setSelectedPreset}
          renderAmount={renderAmount}
          amountLoading={amountLoading}
          payMethods={payMethods}
          preTopUp={preTopUp}
          paymentLoading={paymentLoading}
          payWay={payWay}
          redemptionCode={redemptionCode}
          setRedemptionCode={setRedemptionCode}
          topUp={topUp}
          isSubmitting={isSubmitting}
          topUpLink={topUpLink}
          openTopUpLink={openTopUpLink}
          userState={userState}
          renderQuota={renderQuota}
          statusLoading={statusLoading}
          topupInfo={topupInfo}
          onOpenHistory={handleOpenHistory}
          subscriptionLoading={subscriptionLoading}
          subscriptionPlans={subscriptionPlans}
          billingPreference={billingPreference}
          onChangeBillingPreference={updateBillingPreference}
          activeSubscriptions={activeSubscriptions}
          activeQuantityByPlan={activeQuantityByPlan}
          allSubscriptions={allSubscriptions}
          reloadSubscriptionSelf={getSubscriptionSelf}
        />
        <InvitationCard
          t={t}
          userState={userState}
          renderQuota={renderQuota}
          setOpenTransfer={setOpenTransfer}
          affLink={affLink}
          handleAffLinkClick={handleAffLinkClick}
        />
      </div>
    </div>
  );
};

export default TopUp;
