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
  Empty,
  Modal,
  Select,
  SideSheet,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlusCircle } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, showError, showSuccess } from '../../../../helpers';
import { convertUSDToCurrency } from '../../../../helpers/render';
import {
  buildPurchaseQuantityOptions,
  getPlanPurchaseQuantityConfig,
  getRenewableSubscriptionsByPlan,
} from '../../../../helpers/subscriptionPurchase';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import CardTable from '../../../common/ui/CardTable';

const { Text } = Typography;

function formatTs(ts) {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString();
}

function getSubscriptionRuntimeState(sub) {
  const nowUnix = Date.now() / 1000;
  const endTime = Number(sub?.end_time || 0);
  const status = sub?.status || '';
  const isExpiredByTime = endTime > 0 && endTime <= nowUnix;
  const isActive = status === 'active' && !isExpiredByTime;
  const isCancelled = status === 'cancelled';
  return {
    isActive,
    isCancelled,
    isExpiredByTime,
    canEnable: isCancelled && !isExpiredByTime,
    canDisable: isActive,
  };
}

function renderStatusTag(sub, t) {
  const state = getSubscriptionRuntimeState(sub);
  if (state.isActive) {
    return (
      <Tag color='green' shape='circle' size='small'>
        {t('生效')}
      </Tag>
    );
  }
  if (state.isCancelled) {
    return (
      <Tag color='grey' shape='circle' size='small'>
        {t('已禁用')}
      </Tag>
    );
  }
  return (
    <Tag color='grey' shape='circle' size='small'>
      {t('已过期')}
    </Tag>
  );
}

const UserSubscriptionsModal = ({ visible, onCancel, user, t, onSuccess }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [plansLoading, setPlansLoading] = useState(false);
  const [singleActionLoading, setSingleActionLoading] = useState({
    id: 0,
    action: '',
  });
  const [batchLoading, setBatchLoading] = useState(false);

  const [plans, setPlans] = useState([]);
  const [selectedPlanId, setSelectedPlanId] = useState(null);
  const [purchaseMode, setPurchaseMode] = useState('stack');
  const [purchaseQuantity, setPurchaseQuantity] = useState(1);
  const [renewTargetSubscriptionId, setRenewTargetSubscriptionId] = useState(0);
  const [selectedSubscriptionIds, setSelectedSubscriptionIds] = useState([]);
  const [activeQuantityByPlan, setActiveQuantityByPlan] = useState({});

  const [subs, setSubs] = useState([]);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;

  const selectedPlan = useMemo(() => {
    const option = (plans || []).find(
      (item) => Number(item?.plan?.id || 0) === Number(selectedPlanId || 0),
    );
    return option?.plan || null;
  }, [plans, selectedPlanId]);

  const activeQuantityByPlanMap = useMemo(() => {
    const map = new Map();
    Object.entries(activeQuantityByPlan || {}).forEach(([planId, quantity]) => {
      const id = Number(planId);
      if (id <= 0) return;
      map.set(id, Math.max(0, Number(quantity || 0)));
    });
    return map;
  }, [activeQuantityByPlan]);

  const selectedPlanQuantityConfig = useMemo(() => {
    const planId = Number(selectedPlan?.id || 0);
    const activeQuantity = Number(activeQuantityByPlanMap.get(planId) || 0);
    return getPlanPurchaseQuantityConfig(selectedPlan, activeQuantity);
  }, [selectedPlan, activeQuantityByPlanMap]);

  const minQuantity = selectedPlanQuantityConfig.min;
  const maxQuantity = selectedPlanQuantityConfig.max;
  const hasPurchasableQuantity = maxQuantity >= minQuantity;

  const purchaseQuantityOptions = useMemo(() => {
    return buildPurchaseQuantityOptions(minQuantity, maxQuantity);
  }, [minQuantity, maxQuantity]);

  const renewableSubscriptions = useMemo(() => {
    const targetPlanId = Number(selectedPlanId || 0);
    if (targetPlanId <= 0) {
      return [];
    }
    return getRenewableSubscriptionsByPlan(subs, targetPlanId);
  }, [selectedPlanId, subs]);

  const renewTargetOptions = useMemo(() => {
    return renewableSubscriptions.map((sub) => {
      const endText = formatTs(sub?.end_time);
      return {
        value: Number(sub?.id || 0),
        label: `${t('订阅')} #${sub?.id} · ${t('至')} ${endText}`,
      };
    });
  }, [renewableSubscriptions, t]);

  const canRenew = renewTargetOptions.length > 0;
  const allowRenewExtend = !canRenew && Number(purchaseQuantity || 0) > 1;

  const purchaseModeOptions = useMemo(() => {
    return [
      { value: 'stack', label: t('叠加新增') },
      canRenew
        ? { value: 'renew', label: t('续费已有订阅') }
        : allowRenewExtend
          ? {
              value: 'renew_extend',
              label: t('续费式购买（无可续费订阅时自动顺延）'),
            }
          : {
              value: 'renew',
              label: `${t('续费已有订阅')} (${t('当前无可续费订阅')})`,
              disabled: true,
            },
    ];
  }, [allowRenewExtend, canRenew, t]);

  const planTitleMap = useMemo(() => {
    const map = new Map();
    (plans || []).forEach((item) => {
      const id = item?.plan?.id;
      const title = item?.plan?.title;
      if (id) map.set(id, title || `#${id}`);
    });
    return map;
  }, [plans]);

  const pagedSubs = useMemo(() => {
    const start = Math.max(0, (Number(currentPage || 1) - 1) * pageSize);
    const end = start + pageSize;
    return (subs || []).slice(start, end);
  }, [subs, currentPage]);

  const planOptions = useMemo(() => {
    return (plans || []).map((item) => ({
      label: `${item?.plan?.title || ''} (${convertUSDToCurrency(
        Number(item?.plan?.price_amount || 0),
        2,
      )})`,
      value: item?.plan?.id,
    }));
  }, [plans]);

  const rowSelection = useMemo(() => {
    return {
      selectedRowKeys: selectedSubscriptionIds,
      onChange: (selectedRowKeys) => {
        setSelectedSubscriptionIds(
          (selectedRowKeys || [])
            .map((key) => Number(key || 0))
            .filter((id) => id > 0),
        );
      },
    };
  }, [selectedSubscriptionIds]);

  const loadPlans = async () => {
    setPlansLoading(true);
    try {
      const res = await API.get('/api/subscription/admin/plans');
      if (res.data?.success) {
        setPlans(res.data.data || []);
      } else {
        showError(res.data?.message || t('加载失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setPlansLoading(false);
    }
  };

  const loadUserSubscriptions = async () => {
    if (!user?.id) return;
    setLoading(true);
    try {
      const res = await API.get(
        `/api/subscription/admin/users/${user.id}/subscriptions`,
      );
      if (res.data?.success) {
        const raw = res.data.data;
        if (Array.isArray(raw)) {
          // 向后兼容：旧版响应可能直接返回订阅数组。
          setSubs(raw);
          setActiveQuantityByPlan({});
        } else {
          setSubs(Array.isArray(raw?.subscriptions) ? raw.subscriptions : []);
          setActiveQuantityByPlan(raw?.active_quantity_by_plan || {});
        }
        setCurrentPage(1);
      } else {
        showError(res.data?.message || t('加载失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) return;
    setSelectedPlanId(null);
    setPurchaseMode('stack');
    setPurchaseQuantity(1);
    setRenewTargetSubscriptionId(0);
    setCurrentPage(1);
    setSelectedSubscriptionIds([]);
    setActiveQuantityByPlan({});
    loadPlans();
    loadUserSubscriptions();
  }, [visible]);

  useEffect(() => {
    // 切换套餐时重置创建表单，避免带入过期的续费目标。
    setPurchaseMode('stack');
    setPurchaseQuantity(hasPurchasableQuantity ? minQuantity : 0);
    setRenewTargetSubscriptionId(0);
  }, [selectedPlanId, hasPurchasableQuantity, minQuantity]);

  useEffect(() => {
    setPurchaseQuantity((prev) => {
      const quantity = Number(prev || minQuantity);
      if (maxQuantity <= 0) return 0;
      if (quantity < minQuantity) return minQuantity;
      if (quantity > maxQuantity) return maxQuantity;
      return quantity;
    });
  }, [maxQuantity, minQuantity]);

  useEffect(() => {
    // "renew_extend" 仅在数量 > 1 时有效；数量回到 1 时切回 stack。
    if (purchaseMode === 'renew_extend' && Number(purchaseQuantity || 0) <= 1) {
      setPurchaseMode('stack');
    }
  }, [purchaseMode, purchaseQuantity]);

  useEffect(() => {
    if (purchaseMode !== 'renew') {
      setRenewTargetSubscriptionId(0);
      return;
    }
    if (renewTargetOptions.length === 1) {
      setRenewTargetSubscriptionId(Number(renewTargetOptions[0].value || 0));
      return;
    }
    const exists = renewTargetOptions.some(
      (option) => Number(option.value) === Number(renewTargetSubscriptionId || 0),
    );
    if (!exists) {
      setRenewTargetSubscriptionId(0);
    }
  }, [purchaseMode, renewTargetOptions, renewTargetSubscriptionId]);

  useEffect(() => {
    // 刷新后清理陈旧选择，避免批量请求带上无效 ID。
    const exists = new Set(
      (subs || [])
        .map((item) => Number(item?.subscription?.id || 0))
        .filter((id) => id > 0),
    );
    setSelectedSubscriptionIds((prev) => prev.filter((id) => exists.has(id)));
  }, [subs]);

  const handlePageChange = (page) => {
    setCurrentPage(page);
  };

  const createSubscription = async () => {
    if (!user?.id) {
      showError(t('用户信息缺失'));
      return;
    }
    if (!selectedPlanId) {
      showError(t('请选择订阅套餐'));
      return;
    }
    if (purchaseMode === 'renew' && !canRenew) {
      showError(t('当前无可续费订阅'));
      return;
    }
    if (
      purchaseMode === 'renew' &&
      renewTargetOptions.length > 1 &&
      Number(renewTargetSubscriptionId || 0) <= 0
    ) {
      showError(t('请选择续费目标订阅'));
      return;
    }
    if (!hasPurchasableQuantity) {
      showError(t('当前可购买数量为 0，请等待部分订阅到期后再试'));
      return;
    }
    setCreating(true);
    try {
      const payload = {
        plan_id: Number(selectedPlanId),
        purchase_mode: purchaseMode,
        purchase_quantity: Number(purchaseQuantity || minQuantity),
        renew_target_subscription_id:
          purchaseMode === 'renew'
            ? Number(renewTargetSubscriptionId || 0)
            : 0,
      };
      const res = await API.post(
        `/api/subscription/admin/users/${user.id}/subscriptions`,
        payload,
      );
      if (res.data?.success) {
        const msg = res.data?.data?.message || t('新增成功');
        showSuccess(msg);
        setSelectedPlanId(null);
        await loadUserSubscriptions();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('新增失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setCreating(false);
    }
  };

  const confirmCreateSubscription = () => {
    Modal.confirm({
      title: t('确认操作'),
      content: t('是否确认新增订阅？'),
      centered: true,
      onOk: async () => {
        await createSubscription();
      },
    });
  };

  const actionLabelMap = useMemo(
    () => ({
      enable: t('启用'),
      disable: t('禁用'),
      delete: t('删除'),
    }),
    [t],
  );

  const manageSubscription = async (subscriptionId, action) => {
    if (!user?.id || !subscriptionId || !action) return;
    setSingleActionLoading({ id: Number(subscriptionId), action });
    try {
      const res = await API.post(
        `/api/subscription/admin/users/${user.id}/subscriptions/manage`,
        { id: Number(subscriptionId), action },
      );
      if (res.data?.success) {
        const msg =
          res.data?.data?.message ||
          t('操作成功：{{action}}', { action: actionLabelMap[action] || action });
        showSuccess(msg);
        await loadUserSubscriptions();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('操作失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setSingleActionLoading({ id: 0, action: '' });
    }
  };

  const confirmManageSubscription = (subscriptionId, action) => {
    const isDelete = action === 'delete';
    const title =
      action === 'enable'
        ? t('确认启用')
        : action === 'disable'
        ? t('确认禁用')
        : t('确认删除');
    const content =
      action === 'enable'
        ? t('仅未过期的已禁用订阅可启用。是否继续？')
        : action === 'disable'
        ? t('禁用后不会改动结束时间，可在未过期前重新启用。是否继续？')
        : t('删除会彻底移除该订阅记录（含权益明细）。是否继续？');
    Modal.confirm({
      title,
      content,
      centered: true,
      okType: isDelete ? 'danger' : 'primary',
      onOk: async () => {
        await manageSubscription(subscriptionId, action);
      },
    });
  };

  const batchManageSubscriptions = async (action) => {
    if (!user?.id || selectedSubscriptionIds.length === 0) return;
    setBatchLoading(true);
    try {
      const res = await API.post(
        `/api/subscription/admin/users/${user.id}/subscriptions/manage/batch`,
        {
          ids: selectedSubscriptionIds,
          action,
        },
      );
      if (res.data?.success) {
        const result = res.data?.data || {};
        const successCount = Number(result?.success_count || 0);
        const failedCount = Number(result?.failed_count || 0);
        if (failedCount > 0) {
          const firstFailedMessage = result?.failed?.[0]?.message;
          showError(
            t('批量{{action}}完成：成功 {{success}} 条，失败 {{failed}} 条', {
              action: actionLabelMap[action] || action,
              success: successCount,
              failed: failedCount,
            }) + (firstFailedMessage ? `；${firstFailedMessage}` : ''),
          );
        } else {
          showSuccess(
            t('批量{{action}}成功：{{count}} 条', {
              action: actionLabelMap[action] || action,
              count: successCount,
            }),
          );
        }
        setSelectedSubscriptionIds([]);
        await loadUserSubscriptions();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('批量操作失败'));
      }
    } catch (e) {
      showError(t('请求失败'));
    } finally {
      setBatchLoading(false);
    }
  };

  const confirmBatchDelete = () => {
    if (selectedSubscriptionIds.length === 0) return;
    Modal.confirm({
      title: t('确认批量删除'),
      content: t('确定要删除所选的 {{count}} 条订阅吗？', {
        count: selectedSubscriptionIds.length,
      }),
      centered: true,
      okType: 'danger',
      onOk: async () => {
        await batchManageSubscriptions('delete');
      },
    });
  };

  const columns = useMemo(() => {
    return [
      {
        title: 'ID',
        dataIndex: ['subscription', 'id'],
        key: 'id',
        width: 70,
      },
      {
        title: t('套餐'),
        key: 'plan',
        width: 180,
        render: (_, record) => {
          const sub = record?.subscription;
          const planId = sub?.plan_id;
          const title =
            planTitleMap.get(planId) || (planId ? `#${planId}` : '-');
          return (
            <div className='min-w-0'>
              <div className='font-medium truncate'>{title}</div>
              <div className='text-xs text-gray-500'>
                {t('来源')}: {sub?.source || '-'}
              </div>
            </div>
          );
        },
      },
      {
        title: t('状态'),
        key: 'status',
        width: 90,
        render: (_, record) => renderStatusTag(record?.subscription, t),
      },
      {
        title: t('有效期'),
        key: 'validity',
        width: 200,
        render: (_, record) => {
          const sub = record?.subscription;
          return (
            <div className='text-xs text-gray-600'>
              <div>
                {t('开始')}: {formatTs(sub?.start_time)}
              </div>
              <div>
                {t('结束')}: {formatTs(sub?.end_time)}
              </div>
            </div>
          );
        },
      },
      {
        title: t('总额度'),
        key: 'total',
        width: 120,
        render: (_, record) => {
          const sub = record?.subscription;
          const total = Number(sub?.amount_total || 0);
          const used = Number(sub?.amount_used || 0);
          return (
            <Text type={total > 0 ? 'secondary' : 'tertiary'}>
              {total > 0 ? `${used}/${total}` : t('不限')}
            </Text>
          );
        },
      },
      {
        title: '',
        key: 'operate',
        width: 230,
        fixed: 'right',
        render: (_, record) => {
          const sub = record?.subscription;
          const state = getSubscriptionRuntimeState(sub);
          const loadingId = Number(singleActionLoading.id || 0);
          const loadingAction = singleActionLoading.action;
          const currentSubId = Number(sub?.id || 0);
          const busy = batchLoading || creating || loading;
          return (
            <Space>
              <Button
                size='small'
                theme='light'
                type='tertiary'
                disabled={!state.canEnable || busy}
                loading={loadingId === currentSubId && loadingAction === 'enable'}
                onClick={() => confirmManageSubscription(currentSubId, 'enable')}
              >
                {t('启用')}
              </Button>
              <Button
                size='small'
                theme='light'
                type='warning'
                disabled={!state.canDisable || busy}
                loading={loadingId === currentSubId && loadingAction === 'disable'}
                onClick={() => confirmManageSubscription(currentSubId, 'disable')}
              >
                {t('禁用')}
              </Button>
              <Button
                size='small'
                type='danger'
                theme='light'
                disabled={busy}
                loading={loadingId === currentSubId && loadingAction === 'delete'}
                onClick={() => confirmManageSubscription(currentSubId, 'delete')}
              >
                {t('删除')}
              </Button>
            </Space>
          );
        },
      },
    ];
  }, [
    t,
    planTitleMap,
    singleActionLoading,
    batchLoading,
    creating,
    loading,
  ]);

  return (
    <SideSheet
      visible={visible}
      placement='right'
      width={isMobile ? '100%' : 980}
      bodyStyle={{ padding: 0 }}
      onCancel={onCancel}
      title={
        <Space>
          <Tag color='blue' shape='circle'>
            {t('管理')}
          </Tag>
          <Typography.Title heading={4} className='m-0'>
            {t('用户订阅管理')}
          </Typography.Title>
          <Text type='tertiary' className='ml-2'>
            {user?.username || '-'} (ID: {user?.id || '-'})
          </Text>
        </Space>
      }
    >
      <div className='p-4'>
        {/* 新增订阅控制项（mode/quantity/target）集中在一个区块，减少误操作。 */}
        <div className='mb-4 rounded-lg border border-solid border-[var(--semi-color-border)] p-3'>
          <div className='flex flex-col gap-3'>
            <div className='flex flex-col lg:flex-row gap-2'>
              <Select
                placeholder={t('选择订阅套餐')}
                optionList={planOptions}
                value={selectedPlanId}
                onChange={setSelectedPlanId}
                loading={plansLoading}
                filter
                style={{ minWidth: isMobile ? undefined : 280, flex: 1 }}
              />
              <Select
                placeholder={t('购买方式')}
                optionList={purchaseModeOptions}
                value={purchaseMode}
                onChange={(value) => setPurchaseMode(value || 'stack')}
                style={{ minWidth: isMobile ? undefined : 180 }}
                disabled={!selectedPlanId}
              />
              <Select
                placeholder={t('购买数量')}
                optionList={purchaseQuantityOptions}
                value={purchaseQuantity}
                onChange={(value) => setPurchaseQuantity(Number(value || minQuantity))}
                style={{ minWidth: isMobile ? undefined : 120 }}
                disabled={!selectedPlanId || !hasPurchasableQuantity}
              />
              <Button
                type='primary'
                theme='solid'
                icon={<IconPlusCircle />}
                loading={creating}
                disabled={!selectedPlanId || !hasPurchasableQuantity}
                onClick={confirmCreateSubscription}
              >
                {t('新增订阅')}
              </Button>
            </div>

            {purchaseMode === 'renew' && renewTargetOptions.length > 1 && (
              <Select
                placeholder={t('选择续费目标')}
                optionList={renewTargetOptions}
                value={renewTargetSubscriptionId || undefined}
                onChange={(value) =>
                  setRenewTargetSubscriptionId(Number(value || 0))
                }
                style={{ minWidth: isMobile ? undefined : 420 }}
              />
            )}
            {purchaseMode === 'renew' && renewTargetOptions.length === 1 && (
              <Text type='tertiary'>
                {t('续费目标')}: {renewTargetOptions[0].label}
              </Text>
            )}
            {!selectedPlanId ? null : (
              <Text type='tertiary'>
                {t('购买数量范围')}:{' '}
                {hasPurchasableQuantity ? `${minQuantity} - ${maxQuantity}` : '0'}
              </Text>
            )}
            {selectedPlanId && !hasPurchasableQuantity && (
              <Text type='tertiary'>
                {t('当前可购买数量为 0，请等待部分订阅到期后再试')}
              </Text>
            )}
          </div>
        </div>

        {/* 批量操作区：与用户管理保持一致（enable/disable/delete）。 */}
        <div className='flex flex-col md:flex-row md:items-center md:justify-between gap-2 mb-3'>
          <Text type='tertiary'>
            {t('已选择 {{count}} 条订阅', {
              count: selectedSubscriptionIds.length,
            })}
          </Text>
          <Space wrap>
            <Button
              size='small'
              type='tertiary'
              disabled={selectedSubscriptionIds.length === 0 || batchLoading}
              loading={batchLoading}
              onClick={() => batchManageSubscriptions('enable')}
            >
              {t('批量启用')}
            </Button>
            <Button
              size='small'
              type='tertiary'
              disabled={selectedSubscriptionIds.length === 0 || batchLoading}
              loading={batchLoading}
              onClick={() => batchManageSubscriptions('disable')}
            >
              {t('批量禁用')}
            </Button>
            <Button
              size='small'
              type='danger'
              disabled={selectedSubscriptionIds.length === 0 || batchLoading}
              loading={batchLoading}
              onClick={confirmBatchDelete}
            >
              {t('批量删除')}
            </Button>
          </Space>
        </div>

        {/* 订阅列表 */}
        <CardTable
          columns={columns}
          dataSource={pagedSubs}
          rowKey={(row) => Number(row?.subscription?.id || 0)}
          rowSelection={!isMobile ? rowSelection : undefined}
          loading={loading}
          scroll={{ x: 'max-content' }}
          hidePagination={false}
          pagination={{
            currentPage,
            pageSize,
            total: subs.length,
            pageSizeOpts: [10, 20, 50],
            showSizeChanger: false,
            onPageChange: handlePageChange,
          }}
          empty={
            <Empty
              image={
                <IllustrationNoResult style={{ width: 150, height: 150 }} />
              }
              darkModeImage={
                <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
              }
              description={t('暂无订阅记录')}
              style={{ padding: 30 }}
            />
          }
          size='middle'
        />
      </div>
    </SideSheet>
  );
};

export default UserSubscriptionsModal;
