import React, { useEffect, useMemo, useState } from 'react';
import { Button, Modal, Select, Space, Spin, Tag } from '@douyinfe/semi-ui';
import { API, showError, showSuccess, timestamp2string, renderQuota } from '../../helpers';
import { useTranslation } from 'react-i18next';

const SubscriptionIssuanceModal = ({
  visible,
  issuanceId,
  onCancel,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState(null);
  const [purchaseMode, setPurchaseMode] = useState('stack');
  const [renewTargetSubscriptionId, setRenewTargetSubscriptionId] = useState(0);

  const issuance = detail?.issuance || null;
  const plan = issuance?.plan || null;
  const renewableTargets = issuance?.renewable_targets || [];

  useEffect(() => {
    if (!visible || !issuanceId) {
      setDetail(null);
      setPurchaseMode('stack');
      setRenewTargetSubscriptionId(0);
      return;
    }
    const load = async () => {
      setLoading(true);
      try {
        const res = await API.get(`/api/subscription/self/issuances/${issuanceId}`);
        const { success, message, data } = res.data || {};
        if (success) {
          setDetail(data || null);
        } else {
          showError(message);
        }
      } catch (error) {
        showError(error.message || t('加载套餐待发放记录失败'));
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [visible, issuanceId]);

  useEffect(() => {
    const nextMode =
      issuance?.purchase_mode === 'renew' && renewableTargets.length > 0
        ? 'renew'
        : 'stack';
    setPurchaseMode(nextMode);
    setRenewTargetSubscriptionId(
      Number(
        issuance?.renew_target_subscription_id ||
          renewableTargets?.[0]?.subscription?.id ||
          0,
      ),
    );
  }, [issuance?.purchase_mode, issuance?.renew_target_subscription_id, renewableTargets]);

  const renewTargetOptions = useMemo(() => {
    return renewableTargets.map((item) => {
      const sub = item?.subscription || {};
      return {
        value: Number(sub?.id || 0),
        label: `${t('订阅')} #${sub?.id || '-'} · ${t('至')} ${timestamp2string(
          sub?.end_time || 0,
        )}`,
      };
    });
  }, [renewableTargets, t]);

  const handleConfirm = async () => {
    if (!issuanceId) return;
    if (purchaseMode === 'renew' && Number(renewTargetSubscriptionId || 0) <= 0) {
      showError(t('请选择续费目标订阅'));
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/subscription/self/issuances/${issuanceId}/confirm`,
        {
          purchase_mode: purchaseMode,
          renew_target_subscription_id:
            purchaseMode === 'renew' ? Number(renewTargetSubscriptionId || 0) : 0,
        },
      );
      const { success, message, data } = res.data || {};
      if (success) {
        showSuccess(data?.message || t('套餐发放成功'));
        onSuccess?.(data || null);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('发放套餐失败'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('套餐待发放')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={720}
      centered
    >
      <Spin spinning={loading}>
        {!issuance ? (
          <div className='py-8 text-center text-gray-500'>{t('暂无待发放记录')}</div>
        ) : (
          <div className='space-y-5'>
            <div className='rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-4'>
              <div className='flex flex-wrap items-start justify-between gap-3'>
                <div>
                  <div className='text-lg font-semibold'>
                    {issuance?.plan_title || plan?.title || '-'}
                  </div>
                  <div className='mt-2 text-sm text-[var(--semi-color-text-2)]'>
                    {t('来源')}: {issuance?.source_type || '-'} · {t('创建时间')}:{' '}
                    {timestamp2string(issuance?.created_time || 0)}
                  </div>
                </div>
                <Space wrap>
                  <Tag color='blue' shape='circle'>
                    {t('待发放')}
                  </Tag>
                  <Tag color='white' shape='circle'>
                    {t('份数')} {issuance?.purchase_quantity || 1}
                  </Tag>
                  {Number(plan?.total_amount || 0) > 0 ? (
                    <Tag color='white' shape='circle'>
                      {t('额度')} {renderQuota(plan?.total_amount || 0)}
                    </Tag>
                  ) : null}
                </Space>
              </div>
            </div>

            <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
              <div>
                <div className='mb-2 text-sm font-medium'>{t('发放方式')}</div>
                <Select
                  value={purchaseMode}
                  style={{ width: '100%' }}
                  optionList={[
                    { label: t('叠加（新增一条订阅）'), value: 'stack' },
                    {
                      label:
                        renewableTargets.length > 0
                          ? t('续费（延长现有订阅）')
                          : `${t('续费（延长现有订阅）')} · ${t('当前无可续费订阅')}`,
                      value: 'renew',
                      disabled: renewableTargets.length === 0,
                    },
                  ]}
                  onChange={(value) => setPurchaseMode(value || 'stack')}
                />
              </div>
              <div>
                <div className='mb-2 text-sm font-medium'>{t('续费目标')}</div>
                <Select
                  value={
                    purchaseMode === 'renew' && renewTargetSubscriptionId > 0
                      ? renewTargetSubscriptionId
                      : undefined
                  }
                  style={{ width: '100%' }}
                  disabled={purchaseMode !== 'renew'}
                  placeholder={
                    renewableTargets.length > 0
                      ? t('请选择续费目标订阅')
                      : t('当前无可续费订阅')
                  }
                  optionList={renewTargetOptions}
                  onChange={(value) =>
                    setRenewTargetSubscriptionId(Number(value || 0))
                  }
                />
              </div>
            </div>

            {Number(plan?.purchase_quantity_max || 0) > 0 ? (
              <div className='rounded-xl border border-dashed border-[var(--semi-color-border)] p-4 text-sm text-[var(--semi-color-text-2)]'>
                {t('该记录已锁定购买份数')} {issuance?.purchase_quantity || 1}
                {t('，这里只确认叠加还是续费，不再修改数量。')}
              </div>
            ) : null}

            <div className='flex justify-end gap-2'>
              <Button onClick={onCancel}>{t('稍后处理')}</Button>
              <Button
                theme='solid'
                type='primary'
                loading={submitting}
                onClick={handleConfirm}
              >
                {t('立即发放')}
              </Button>
            </div>
          </div>
        )}
      </Spin>
    </Modal>
  );
};

export default SubscriptionIssuanceModal;
