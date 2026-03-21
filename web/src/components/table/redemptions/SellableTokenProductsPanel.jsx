import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Modal,
  Popover,
  Space,
  Switch,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  formatPaymentAmount,
  getCurrencyConfig,
  renderQuota,
  showError,
  showSuccess,
} from '../../../helpers';
import {
  formatConcurrencyLabel,
  formatConcurrencyLong,
  formatWindowLimitLong,
  formatWindowLimitShort,
} from '../../../helpers/render';
import {
  displayAmountToQuota,
  quotaToDisplayAmount,
} from '../../../helpers/quota';
import { useTranslation } from 'react-i18next';
import { Coins } from 'lucide-react';
import CardPro from '../../common/ui/CardPro';
import CardTable from '../../common/ui/CardTable';

const { Text } = Typography;

const PERIOD_LABELS = {
  hourly: '每小时',
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
  custom: '自定义',
};

const renderPeriodLabel = (t, period) =>
  t(PERIOD_LABELS[period] || PERIOD_LABELS.custom);

const defaultValues = {
  name: '',
  subtitle: '',
  sort_order: 0,
  price_amount: 0,
  price_quota: 0,
  total_amount: 0,
  total_quota: 0,
  unlimited_quota: false,
  validity_seconds: 0,
  model_limits_enabled: false,
  model_limits: '',
  allowed_groups: [],
  max_concurrency: 0,
  window_request_limit: 0,
  window_seconds: 0,
  package_enabled: false,
  package_limit_amount: 0,
  package_limit_quota: 0,
  package_period: 'hourly',
  package_custom_seconds: 3600,
  package_period_mode: 'relative',
};

const buildProductPayload = (values) => {
  const payload = {
    product: {
      ...values,
      subtitle: values.subtitle || '',
      sort_order: Number(values.sort_order || 0),
      price_quota: displayAmountToQuota(values.price_amount || 0),
      total_quota: values.unlimited_quota
        ? 0
        : displayAmountToQuota(values.total_amount || 0),
      package_limit_quota: values.package_enabled
        ? displayAmountToQuota(values.package_limit_amount || 0)
        : 0,
      allowed_groups: (values.allowed_groups || []).join(','),
      model_limits: values.model_limits || '',
    },
  };
  delete payload.product.price_amount;
  delete payload.product.total_amount;
  delete payload.product.package_limit_amount;
  return payload;
};

const buildEditFormValues = (record) => {
  const allowedGroups = record.allowed_groups
    ? record.allowed_groups.split(',').filter(Boolean)
    : [];

  return {
    ...record,
    subtitle: record.subtitle || '',
    sort_order: Number(record.sort_order || 0),
    price_amount: Number(record.price_amount || 0),
    total_amount: record.unlimited_quota
      ? 0
      : quotaToDisplayAmount(record.total_quota || 0),
    unlimited_quota: Boolean(record.unlimited_quota),
    package_limit_amount: quotaToDisplayAmount(record.package_limit_quota || 0),
    allowed_groups: allowedGroups,
  };
};

const SellableTokenProductFormFields = ({
  t,
  values,
  formApiRef,
  groups,
  quotaDisplayType,
  displayCurrencyLabel,
}) => {
  const setFieldValue = (field, value) => {
    formApiRef.current?.setValue(field, value);
  };

  return (
    <div className='space-y-4'>
      <Card className='!rounded-xl border border-[var(--semi-color-border)]'>
        <div className='mb-4 flex items-center gap-2 text-blue-500'>
          <Coins size={16} className='flex-shrink-0' />
          <Text strong>{t('基本信息')}</Text>
        </div>

        <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
          <Form.Input
            field='name'
            label={t('名称')}
            rules={[{ required: true, message: t('请输入名称') }]}
            showClear
          />
          <Form.Input
            field='subtitle'
            label={t('副标题')}
            showClear
            placeholder={t('例如：适合团队共享或短期活动')}
          />
          <Form.InputNumber
            field='sort_order'
            label={t('排序')}
            precision={0}
            style={{ width: '100%' }}
            extraText={t('数值越大越靠前，会影响前台推荐顺序')}
          />
          <Form.InputNumber
            field='price_amount'
            label={t('钱包售价')}
            min={0}
            precision={quotaDisplayType === 'TOKENS' ? 0 : 2}
            step={quotaDisplayType === 'TOKENS' ? 1 : 0.01}
            style={{ width: '100%' }}
            extraText={t('按支付货币填写，当前为：') + displayCurrencyLabel}
          />
        </div>

        <div className='mt-4 grid grid-cols-1 gap-4 md:grid-cols-2'>
          <div>
            <div className='mb-1 flex items-center gap-2'>
              <label className='semi-form-field-label'>{t('总额度')}</label>
              <Switch
                size='small'
                checked={Boolean(values.unlimited_quota)}
                onChange={(checked) => {
                  const nextChecked = Boolean(
                    checked?.target?.checked ?? checked,
                  );
                  setFieldValue('unlimited_quota', nextChecked);
                }}
              />
              <span className='text-xs text-[var(--semi-color-text-2)]'>
                {t('无限')}
              </span>
            </div>
            {!values.unlimited_quota ? (
              <Form.InputNumber
                field='total_amount'
                noLabel
                min={quotaDisplayType === 'TOKENS' ? 1 : 0.01}
                precision={quotaDisplayType === 'TOKENS' ? 0 : 2}
                step={quotaDisplayType === 'TOKENS' ? 1 : 0.01}
                style={{ width: '100%' }}
                extraText={
                  t('创建后实际展示为：') +
                  renderQuota(displayAmountToQuota(values.total_amount || 0))
                }
              />
            ) : (
              <div className='mt-1 text-sm text-[var(--semi-color-text-2)]'>
                {t('不限制总额度')}
              </div>
            )}
          </div>

          <Form.InputNumber
            field='validity_seconds'
            label={t('有效期（秒，0=长期）')}
            min={0}
            style={{ width: '100%' }}
            extraText={t('0 表示长期有效')}
          />
        </div>
      </Card>

      <Card className='!rounded-xl border border-[var(--semi-color-border)]'>
        <div className='mb-4 flex items-center gap-2 text-blue-500'>
          <Coins size={16} className='flex-shrink-0' />
          <Text strong>{t('发放与限制')}</Text>
        </div>

        <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
          <Form.Select
            field='allowed_groups'
            label={t('允许分组')}
            multiple
            optionList={groups.map((group) => ({
              label: group,
              value: group,
            }))}
          />
          <div className='grid grid-cols-1 gap-4'>
            <Form.InputNumber
              field='max_concurrency'
              label={t('并发上限')}
              min={0}
              style={{ width: '100%' }}
              extraText={t('0 表示不限制')}
            />
            <Form.InputNumber
              field='window_request_limit'
              label={t('窗口请求上限')}
              min={0}
              style={{ width: '100%' }}
              extraText={t('0 表示不限制')}
            />
            <Form.InputNumber
              field='window_seconds'
              label={t('窗口时长（秒）')}
              min={0}
              style={{ width: '100%' }}
              extraText={t('与窗口请求上限配合使用，0 表示不限制')}
            />
          </div>
        </div>

        <div className='mt-4'>
          <Form.Switch field='model_limits_enabled' label={t('启用模型限制')} />
          {values.model_limits_enabled ? (
            <div className='mt-3'>
              <Form.TextArea
                field='model_limits'
                label={t('模型限制（逗号分隔）')}
                placeholder='gpt-4o, claude-3-7-sonnet'
                rows={3}
              />
            </div>
          ) : null}
        </div>
      </Card>

      <Card className='!rounded-xl border border-[var(--semi-color-border)]'>
        <div className='mb-3 flex items-center gap-2 text-blue-500'>
          <Coins size={16} className='flex-shrink-0' />
          <Text strong>{t('周期额度上限')}</Text>
          <Switch
            checked={Boolean(values.package_enabled)}
            onChange={(checked) => {
              const nextChecked = Boolean(checked?.target?.checked ?? checked);
              setFieldValue('package_enabled', nextChecked);
            }}
            size='small'
          />
        </div>

        {values.package_enabled ? (
          <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
            <Form.InputNumber
              field='package_limit_amount'
              label={t('周期额度')}
              min={quotaDisplayType === 'TOKENS' ? 1 : 0.01}
              precision={quotaDisplayType === 'TOKENS' ? 0 : 2}
              step={quotaDisplayType === 'TOKENS' ? 1 : 0.01}
              style={{ width: '100%' }}
              extraText={
                t('按当前系统显示货币填写，实际展示为：') +
                renderQuota(displayAmountToQuota(values.package_limit_amount || 0))
              }
            />
            <Form.Select
              field='package_period'
              label={t('周期')}
              style={{ width: '100%' }}
              optionList={[
                { label: t('每小时'), value: 'hourly' },
                { label: t('每日'), value: 'daily' },
                { label: t('每周'), value: 'weekly' },
                { label: t('每月'), value: 'monthly' },
                { label: t('自定义'), value: 'custom' },
              ]}
            />
            {values.package_period === 'custom' ? (
              <Form.InputNumber
                field='package_custom_seconds'
                label={t('自定义周期（秒）')}
                min={1}
                style={{ width: '100%' }}
              />
            ) : null}
          </div>
        ) : (
          <Text type='tertiary'>{t('关闭后表示不限制周期额度')}</Text>
        )}
      </Card>
    </div>
  );
};

const SellableTokenProductsPanel = () => {
  const { t } = useTranslation();
  const [products, setProducts] = useState([]);
  const [groups, setGroups] = useState([]);
  const [loading, setLoading] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editingProduct, setEditingProduct] = useState(null);
  const [editVisible, setEditVisible] = useState(false);
  const formApiRef = useRef(null);
  const editFormApiRef = useRef(null);
  const currencyConfig = getCurrencyConfig();
  const quotaDisplayType = currencyConfig.type || 'USD';
  const displayCurrencyLabel =
    quotaDisplayType === 'TOKENS'
      ? t('Token')
      : quotaDisplayType === 'USD'
        ? t('美元')
        : t('当前显示货币');

  const loadProducts = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/redemption/sellable-token-products');
      const { success, message, data } = res.data || {};
      if (success) {
        setProducts(data || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('加载可售令牌商品失败'));
    } finally {
      setLoading(false);
    }
  };

  const loadGroups = async () => {
    try {
      const res = await API.get('/api/group/');
      const { success, data, message } = res.data || {};
      if (success) {
        setGroups(data || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('加载分组失败'));
    }
  };

  useEffect(() => {
    loadProducts();
    loadGroups();
  }, []);

  const handleCreate = async (values) => {
    const payload = buildProductPayload(values);
    try {
      const res = await API.post(
        '/api/redemption/sellable-token-products',
        payload,
      );
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('可售令牌商品创建成功'));
        setVisible(false);
        formApiRef.current?.reset();
        loadProducts();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('创建可售令牌商品失败'));
    }
  };

  const handleEdit = async (values) => {
    const payload = buildProductPayload(values);
    try {
      const res = await API.put(
        `/api/redemption/sellable-token-products/${editingProduct.id}`,
        payload,
      );
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('可售令牌商品更新成功'));
        setEditVisible(false);
        setEditingProduct(null);
        loadProducts();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('更新可售令牌商品失败'));
    }
  };

  const updateStatus = async (product, status) => {
    try {
      const res = await API.put(
        `/api/redemption/sellable-token-products/${product.id}/status`,
        { status },
      );
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('状态已更新'));
        loadProducts();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('更新状态失败'));
    }
  };

  const deleteProduct = (product) => {
    Modal.confirm({
      title: t('确定删除此可售令牌商品？'),
      content: t('删除后不可恢复，已发出的令牌记录仅保留审计信息。'),
      onOk: async () => {
        const res = await API.delete(
          `/api/redemption/sellable-token-products/${product.id}`,
        );
        const { success, message } = res.data || {};
        if (success) {
          showSuccess(t('删除成功'));
          loadProducts();
        } else {
          showError(message);
        }
      },
    });
  };

  const openEditModal = (record) => {
    setEditingProduct(record);
    setEditVisible(true);
    setTimeout(() => {
      editFormApiRef.current?.setValues(buildEditFormValues(record));
    }, 0);
  };

  const columns = useMemo(
    () => [
      {
        title: t('名称'),
        dataIndex: 'name',
        render: (text, record) => {
          const subtitle = record?.subtitle;
          const nameContent = (
            <div style={{ maxWidth: 220 }}>
              <Text strong ellipsis={{ showTooltip: false }}>
                {text}
              </Text>
              {subtitle ? (
                <Text
                  type='tertiary'
                  size='small'
                  style={{ display: 'block' }}
                  ellipsis={{ showTooltip: false }}
                >
                  {subtitle}
                </Text>
              ) : null}
            </div>
          );

          return subtitle ? (
            <Popover
              content={
                <div style={{ width: 260 }}>
                  <Text strong>{text}</Text>
                  <Text
                    type='tertiary'
                    style={{ display: 'block', marginTop: 4 }}
                  >
                    {subtitle}
                  </Text>
                </div>
              }
              position='rightTop'
              showArrow
            >
              <div style={{ cursor: 'pointer' }}>{nameContent}</div>
            </Popover>
          ) : (
            nameContent
          );
        },
      },
      {
        title: t('优先级'),
        dataIndex: 'sort_order',
        width: 96,
        render: (text) => <Text type='tertiary'>{Number(text || 0)}</Text>,
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: (text) => (
          <Tag color={Number(text) === 1 ? 'green' : 'grey'} shape='circle'>
            {Number(text) === 1 ? t('上架') : t('下架')}
          </Tag>
        ),
      },
      {
        title: t('售价'),
        dataIndex: 'price_amount',
        render: (text) => formatPaymentAmount(text || 0),
      },
      {
        title: t('总额度'),
        dataIndex: 'total_quota',
        render: (text, record) =>
          record.unlimited_quota ? t('无限') : renderQuota(text || 0),
      },
      {
        title: t('周期额度上限'),
        key: 'package_limit',
        render: (_, record) =>
          record.package_enabled ? (
            <span>
              {renderQuota(record.package_limit_quota || 0)} /{' '}
              {renderPeriodLabel(t, record.package_period)}
            </span>
          ) : (
            '-'
          ),
      },
      {
        title: t('限制'),
        key: 'limits',
        render: (_, record) => {
          const hasConcurrency = Number(record.max_concurrency || 0) > 0;
          const hasWindow =
            Number(record.window_request_limit || 0) > 0 &&
            Number(record.window_seconds || 0) > 0;

          if (!hasConcurrency && !hasWindow) {
            return (
              <Tag color='white' shape='circle'>
                {t('无限制')}
              </Tag>
            );
          }

          return (
            <Space wrap>
              {hasConcurrency ? (
                <Tooltip
                  content={formatConcurrencyLong(record.max_concurrency, t)}
                  position='top'
                  showArrow
                >
                  <Tag color='white' shape='circle'>
                    {formatConcurrencyLabel(record.max_concurrency, t)}
                  </Tag>
                </Tooltip>
              ) : null}
              {hasWindow ? (
                <Tooltip
                  content={formatWindowLimitLong(
                    record.window_seconds,
                    record.window_request_limit,
                    t,
                  )}
                  position='top'
                  showArrow
                >
                  <Tag color='white' shape='circle'>
                    {formatWindowLimitShort(
                      record.window_seconds,
                      record.window_request_limit,
                      t,
                    )}
                  </Tag>
                </Tooltip>
              ) : null}
            </Space>
          );
        },
      },
      {
        title: '',
        key: 'actions',
        render: (_, record) => (
          <Space>
            {Number(record.status) === 1 ? (
              <Button
                size='small'
                theme='light'
                type='warning'
                onClick={() => updateStatus(record, 2)}
              >
                {t('下架')}
              </Button>
            ) : (
              <Button
                size='small'
                theme='light'
                type='primary'
                onClick={() => updateStatus(record, 1)}
              >
                {t('上架')}
              </Button>
            )}
            <Button
              size='small'
              theme='light'
              type='tertiary'
              onClick={() => openEditModal(record)}
            >
              {t('编辑')}
            </Button>
            <Button
              theme='light'
              type='danger'
              size='small'
              onClick={() => deleteProduct(record)}
            >
              {t('删除')}
            </Button>
          </Space>
        ),
      },
    ],
    [t],
  );

  return (
    <>
      <Modal
        title={t('新建可售令牌商品')}
        visible={visible}
        footer={null}
        onCancel={() => setVisible(false)}
        width={760}
      >
        <Form
          initValues={defaultValues}
          getFormApi={(api) => {
            formApiRef.current = api;
          }}
          onSubmit={handleCreate}
        >
          {({ values }) => (
            <>
              <SellableTokenProductFormFields
                t={t}
                values={values}
                formApiRef={formApiRef}
                groups={groups}
                quotaDisplayType={quotaDisplayType}
                displayCurrencyLabel={displayCurrencyLabel}
              />
              <div className='flex justify-end pb-2 pt-4'>
                <Space>
                  <Button onClick={() => setVisible(false)}>{t('取消')}</Button>
                  <Button theme='solid' type='primary' htmlType='submit'>
                    {t('创建')}
                  </Button>
                </Space>
              </div>
            </>
          )}
        </Form>
      </Modal>

      <Modal
        title={t('编辑可售令牌商品')}
        visible={editVisible}
        footer={null}
        onCancel={() => {
          setEditVisible(false);
          setEditingProduct(null);
        }}
        width={760}
      >
        <Form
          initValues={defaultValues}
          getFormApi={(api) => {
            editFormApiRef.current = api;
          }}
          onSubmit={handleEdit}
        >
          {({ values }) => (
            <>
              <SellableTokenProductFormFields
                t={t}
                values={values}
                formApiRef={editFormApiRef}
                groups={groups}
                quotaDisplayType={quotaDisplayType}
                displayCurrencyLabel={displayCurrencyLabel}
              />
              <div className='flex justify-end pb-2 pt-4'>
                <Space>
                  <Button
                    onClick={() => {
                      setEditVisible(false);
                      setEditingProduct(null);
                    }}
                  >
                    {t('取消')}
                  </Button>
                  <Button theme='solid' type='primary' htmlType='submit'>
                    {t('保存')}
                  </Button>
                </Space>
              </div>
            </>
          )}
        </Form>
      </Modal>

      <CardPro
        type='type1'
        descriptionArea={
          <div className='flex items-center gap-2 text-blue-500'>
            <Coins size={16} className='flex-shrink-0' />
            <Text>{t('可售令牌管理')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex w-full flex-col items-start gap-2 md:flex-row md:items-center md:justify-between'>
            <div className='w-full md:w-auto'>
              <Button
                type='primary'
                onClick={() => setVisible(true)}
                size='small'
              >
                {t('新建可售令牌商品')}
              </Button>
            </div>
          </div>
        }
        t={t}
      >
        <CardTable
          columns={columns}
          dataSource={products}
          pagination={false}
          hidePagination={true}
          loading={loading}
          rowKey='id'
          className='overflow-hidden'
          size='middle'
        />
      </CardPro>
    </>
  );
};

export default SellableTokenProductsPanel;
