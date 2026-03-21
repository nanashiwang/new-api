import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Modal,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, timestamp2string, renderQuota } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const PERIOD_LABELS = {
  hourly: '每小时',
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
  custom: '自定义',
};

const renderPeriodLabel = (t, period) => t(PERIOD_LABELS[period] || PERIOD_LABELS.custom);

const SellableTokenIssuanceModal = ({
  visible,
  issuanceId,
  onCancel,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState(null);
  const [formApi, setFormApi] = useState(null);

  const issuance = detail?.issuance || null;
  const product = issuance?.product || null;
  const renewableTargets = detail?.renewable_targets || [];
  const groupOptions = detail?.group_options || [];

  const initialValues = useMemo(() => {
    const firstGroup = groupOptions?.[0]?.value || '';
    const firstTarget = renewableTargets?.[0]?.id || 0;
    return {
      mode: renewableTargets.length > 0 ? 'stack' : 'new',
      target_token_id: firstTarget,
      name: product?.name || '',
      group: issuance?.requested_group || firstGroup,
    };
  }, [groupOptions, issuance?.requested_group, product?.name, renewableTargets]);

  useEffect(() => {
    if (formApi && detail) {
      formApi.setValues(initialValues);
    }
  }, [detail, formApi, initialValues]);

  const loadIssuance = async () => {
    if (!issuanceId) {
      setDetail(null);
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(`/api/user/sellable-token/issuances/${issuanceId}`);
      const { success, message, data } = res.data || {};
      if (success) {
        setDetail(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('加载待发放令牌失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      loadIssuance();
    } else {
      setDetail(null);
    }
  }, [visible, issuanceId]);

  const handleSubmit = async (values) => {
    if (!issuanceId) return;
    setSubmitting(true);
    try {
      const payload = {
        mode: values.mode || 'stack',
        target_token_id: Number(values.target_token_id || 0),
        name: values.name || '',
        group: values.group || '',
      };
      const res = await API.post(
        `/api/user/sellable-token/issuances/${issuanceId}/confirm`,
        payload,
      );
      const { success, message, data } = res.data || {};
      if (success) {
        showSuccess(t('可售令牌发放成功'));
        onSuccess?.(data?.token || null);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('发放可售令牌失败'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('发放可售令牌')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      centered
      width={640}
    >
      <Spin spinning={loading}>
        {!issuance ? (
          <div className='py-6 text-center text-gray-500'>{t('暂无待发放记录')}</div>
        ) : (
          <Form initValues={initialValues} getFormApi={setFormApi} onSubmit={handleSubmit}>
            {({ values }) => (
              <div className='space-y-4'>
                <div className='rounded-xl border border-[var(--semi-color-border)] p-3'>
                  <Space wrap>
                    <Tag color='blue' shape='circle'>
                      {product?.name || t('可售令牌')}
                    </Tag>
                    <Tag color='white' shape='circle'>
                      {t('总额度')} {renderQuota(product?.total_quota || 0)}
                    </Tag>
                    {product?.validity_seconds > 0 ? (
                      <Tag color='white' shape='circle'>
                        {t('有效期')} {product.validity_seconds}s
                      </Tag>
                    ) : (
                      <Tag color='green' shape='circle'>{t('长期有效')}</Tag>
                    )}
                  </Space>
                  <div className='mt-3 text-sm text-gray-600'>
                    <div>{t('来源')}: {issuance?.source_type || '-'}</div>
                    <div>{t('创建时间')}: {timestamp2string(issuance?.created_time || 0)}</div>
                    {product?.package_enabled ? (
                      <div>
                        {t('周期额度上限')}: {renderQuota(product?.package_limit_quota || 0)} /{' '}
                        {renderPeriodLabel(t, product?.package_period)}
                      </div>
                    ) : null}
                  </div>
                </div>

                <Form.Select
                  field='mode'
                  label={t('发放方式')}
                  style={{ width: '100%' }}
                  optionList={[
                    { label: t('叠加新令牌'), value: 'stack' },
                    {
                      label:
                        renewableTargets.length > 0
                          ? t('续费已有令牌')
                          : `${t('续费已有令牌')} (${t('暂无可续费目标')})`,
                      value: 'renew',
                      disabled: renewableTargets.length === 0,
                    },
                  ]}
                />

                {values.mode === 'renew' ? (
                  <Form.Select
                    field='target_token_id'
                    label={t('续费目标')}
                    style={{ width: '100%' }}
                    optionList={renewableTargets.map((token) => ({
                      label: `${token.name} (#${token.id}) · ${t('到期')} ${token.expired_time === -1 ? t('长期') : timestamp2string(token.expired_time)}`,
                      value: token.id,
                    }))}
                    rules={[{ required: true, message: t('请选择续费目标') }]}
                  />
                ) : null}

                <Form.Input
                  field='name'
                  label={t('令牌名称')}
                  placeholder={t('请输入令牌名称')}
                  maxLength={50}
                />

                <Form.Select
                  field='group'
                  label={t('分组')}
                  style={{ width: '100%' }}
                  optionList={groupOptions.map((item) => ({
                    label: item.label || item.value,
                    value: item.value,
                  }))}
                  rules={[{ required: true, message: t('请选择分组') }]}
                />

                <div className='flex justify-end'>
                  <Space>
                    <Button onClick={onCancel}>{t('取消')}</Button>
                    <Button theme='solid' type='primary' htmlType='submit' loading={submitting}>
                      {t('确认发放')}
                    </Button>
                  </Space>
                </div>
              </div>
            )}
          </Form>
        )}
      </Spin>
    </Modal>
  );
};

export default SellableTokenIssuanceModal;
