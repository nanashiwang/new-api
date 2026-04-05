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
import React from 'react';
import {
  Banner,
  Input,
  InputNumber,
  Modal,
  Radio,
  Select,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import PricingRuleList from './PricingRuleList';

const { Text, Title } = Typography;

const MoneyField = ({ label, value, onChange, t }) => (
  <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <InputNumber
      min={0}
      value={value || 0}
      onChange={onChange}
      suffix='USD'
      style={{ marginTop: 10, width: '100%' }}
    />
    <Text type='tertiary' size='small' className='mt-2 block'>
      {t('按请求量分摊')}
    </Text>
  </div>
);

const PricePreviewBlock = ({ title, value }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-2.5'>
    <Text type='tertiary' size='small'>
      {title}
    </Text>
    <div className='mt-1 text-sm font-semibold tabular-nums'>{value}</div>
  </div>
);

const PricingConfigModal = ({
  visible,
  isEditing,
  comboConfig,
  setComboConfig,
  channelOptions,
  tagOptions,
  modelNameOptions,
  options,
  resolveSharedSitePreview,
  isMobile,
  clampNumber,
  localModelMap,
  validationError,
  onOk,
  onCancel,
  t,
}) => {
  const sharedSite = comboConfig?.shared_site || {};
  const accountOptions = (options?.upstream_accounts || [])
    .filter((item) => item.enabled !== false)
    .map((item) => ({
      label: `${item.name} · ${item.base_url || t('未填写地址')}`,
      value: item.id,
    }));
  const scopeOptions = comboConfig?.scope_type === 'channel'
    ? channelOptions
    : tagOptions;

  const updateRule = (field, index, patch) =>
    setComboConfig((prev) => ({
      ...prev,
      [field]: (prev?.[field] || []).map((item, itemIndex) =>
        itemIndex === index ? { ...item, ...patch } : item,
      ),
    }));

  const removeRule = (field, index) =>
    setComboConfig((prev) => ({
      ...prev,
      [field]: (prev?.[field] || []).filter((_, itemIndex) => itemIndex !== index),
    }));

  const addRule = (field) =>
    setComboConfig((prev) => ({
      ...prev,
      [field]: [
        ...(prev?.[field] || []),
        {
          model_name: '',
          input_price: 0,
          output_price: 0,
          cache_read_price: 0,
          cache_creation_price: 0,
          is_default: false,
          is_custom: false,
        },
      ],
    }));

  return (
    <Modal
      title={isEditing ? t('编辑组合') : t('新建组合')}
      visible={visible}
      onOk={onOk}
      onCancel={onCancel}
      size='large'
      centered
      okText={isEditing ? t('保存组合') : t('创建组合')}
      cancelText={t('取消')}
      bodyStyle={{ paddingTop: 8 }}
    >
      {comboConfig ? (
        <div className='space-y-5'>
          <Banner
            type='info'
            closeIcon={null}
            description={t('关闭弹窗后，仍需点击页面顶部“保存配置”才会提交到服务器')}
          />

          {validationError ? (
            <Banner
              type='danger'
              closeIcon={null}
              description={validationError}
            />
          ) : null}

          <section className='space-y-3 rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
            <div className='flex flex-wrap items-center justify-between gap-3'>
              <div>
                <Text type='tertiary' size='small'>
                  {t('组合范围')}
                </Text>
                <Title heading={6} style={{ margin: '6px 0 0' }}>
                  {comboConfig.name?.trim() || t('未命名组合')}
                </Title>
              </div>
              <Tag
                color={comboConfig.scope_type === 'channel' ? 'blue' : 'cyan'}
              >
                {comboConfig.scope_type === 'channel' ? t('按渠道') : t('按标签')}
              </Tag>
            </div>

            <div className='grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]'>
              <div>
                <Text type='tertiary' size='small' className='mb-1.5 block'>
                  {t('组合名称')}
                </Text>
                <Input
                  value={comboConfig.name}
                  onChange={(value) =>
                    setComboConfig((prev) => ({ ...prev, name: value }))
                  }
                  placeholder={t('组合名称，例如：OpenAI 主力')}
                />
              </div>
              <div>
                <Text type='tertiary' size='small' className='mb-1.5 block'>
                  {t('范围类型')}
                </Text>
                <Radio.Group
                  type='button'
                  value={comboConfig.scope_type}
                  onChange={(event) =>
                    setComboConfig((prev) => ({
                      ...prev,
                      scope_type: event.target.value,
                      channel_ids: [],
                      tags: [],
                    }))
                  }
                  size='small'
                >
                  <Radio value='channel'>{t('按渠道')}</Radio>
                  <Radio value='tag'>{t('按标签')}</Radio>
                </Radio.Group>
              </div>
            </div>

            <div>
              <Text type='tertiary' size='small' className='mb-1.5 block'>
                {comboConfig.scope_type === 'channel' ? t('选择渠道') : t('选择标签')}
              </Text>
              <Select
                multiple
                filter
                maxTagCount={isMobile ? 2 : 4}
                optionList={scopeOptions}
                value={
                  comboConfig.scope_type === 'channel'
                    ? comboConfig.channel_ids || []
                    : comboConfig.tags || []
                }
                onChange={(value) =>
                  comboConfig.scope_type === 'channel'
                    ? setComboConfig((prev) => ({
                        ...prev,
                        channel_ids: value || [],
                      }))
                    : setComboConfig((prev) => ({ ...prev, tags: value || [] }))
                }
                placeholder={
                  comboConfig.scope_type === 'channel'
                    ? t('选择渠道')
                    : t('选择标签')
                }
                style={{ width: '100%' }}
              />
            </div>
          </section>

          <section className='space-y-3'>
            <div className='flex items-center justify-between gap-3'>
              <Title heading={6} style={{ margin: 0 }}>
                {t('收入')}
              </Title>
              <Select
                value={comboConfig.site_mode || 'manual'}
                onChange={(value) =>
                  setComboConfig((prev) => ({ ...prev, site_mode: value }))
                }
                optionList={[
                  { label: t('手动定价'), value: 'manual' },
                  { label: t('读取本站模型价格'), value: 'shared_site_model' },
                ]}
                size='small'
                style={{ width: 180 }}
              />
            </div>

            <MoneyField
              label={t('固定总收入')}
              value={comboConfig.site_fixed_total_amount}
              onChange={(value) =>
                setComboConfig((prev) => ({
                  ...prev,
                  site_fixed_total_amount: clampNumber(value),
                }))
              }
              t={t}
            />

            {comboConfig.site_mode === 'shared_site_model' ? (
              <div className='space-y-4 rounded-2xl border border-blue-500/20 bg-blue-500/5 p-4'>
                <div className='grid gap-3 xl:grid-cols-[minmax(0,1.5fr)_180px_150px]'>
                  <div>
                    <Text type='tertiary' size='small' className='mb-1.5 block'>
                      {t('模型')}
                    </Text>
                    <Select
                      multiple
                      filter
                      maxTagCount={isMobile ? 2 : 4}
                      value={sharedSite.model_names || []}
                      onChange={(value) =>
                        setComboConfig((prev) => ({
                          ...prev,
                          shared_site: {
                            ...prev.shared_site,
                            model_names: value || [],
                          },
                        }))
                      }
                      optionList={modelNameOptions}
                      placeholder={t('选择模型')}
                      emptyContent={t('暂无可用模型')}
                      size='small'
                      style={{ width: '100%' }}
                    />
                  </div>
                  <div>
                    <Text type='tertiary' size='small' className='mb-1.5 block'>
                      {t('分组')}
                    </Text>
                    <Select
                      value={sharedSite.group || ''}
                      onChange={(value) =>
                        setComboConfig((prev) => ({
                          ...prev,
                          shared_site: {
                            ...prev.shared_site,
                            group: value,
                          },
                        }))
                      }
                      optionList={[
                        { label: t('自动最低'), value: '' },
                        ...((options?.groups || []).map((item) => ({
                          label: item,
                          value: item,
                        })) || []),
                      ]}
                      size='small'
                      style={{ width: '100%' }}
                    />
                  </div>
                  <div className='flex items-center gap-2 rounded-xl border border-semi-color-border bg-semi-color-bg-1 px-3 py-2'>
                    <Text size='small'>{t('按充值价')}</Text>
                    <Switch
                      checked={!!sharedSite.use_recharge_price}
                      onChange={(checked) =>
                        setComboConfig((prev) => ({
                          ...prev,
                          shared_site: {
                            ...prev.shared_site,
                            use_recharge_price: checked,
                          },
                        }))
                      }
                      size='small'
                    />
                  </div>
                </div>

                {(sharedSite.model_names || []).length > 0 ? (
                  <div className='grid gap-3 md:grid-cols-2'>
                    {(sharedSite.model_names || []).map((modelName) => {
                      const preview = resolveSharedSitePreview(
                        sharedSite,
                        modelName,
                      );
                      return (
                        <div
                          key={`${comboConfig.id}-${modelName}`}
                          className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'
                        >
                          <div className='mb-3 flex items-center justify-between gap-2'>
                            <Text strong>{modelName}</Text>
                            <Tag color={preview ? 'blue' : 'grey'}>
                              {preview ? t('已匹配') : t('未匹配')}
                            </Tag>
                          </div>
                          <div className='grid gap-2 sm:grid-cols-2'>
                            <PricePreviewBlock
                              title={t('输入')}
                              value={`${preview?.input_price?.toFixed(4) || '0'} USD/1M`}
                            />
                            <PricePreviewBlock
                              title={t('输出')}
                              value={`${preview?.output_price?.toFixed(4) || '0'} USD/1M`}
                            />
                            <PricePreviewBlock
                              title={t('缓存读')}
                              value={`${preview?.cache_read_price?.toFixed(4) || '0'} USD/1M`}
                            />
                            <PricePreviewBlock
                              title={t('缓存写')}
                              value={`${preview?.cache_creation_price?.toFixed(4) || '0'} USD/1M`}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <Banner
                    type='warning'
                    closeIcon={null}
                    description={t('已启用本站模型价格，但还没有选择模型')}
                  />
                )}
              </div>
            ) : null}

            <PricingRuleList
              comboId={comboConfig.id || 'modal'}
              field='site_rules'
              title={t('手动定价规则')}
              rules={comboConfig.site_rules}
              modelNameOptions={modelNameOptions}
              localModelMap={localModelMap}
              clampNumber={clampNumber}
              onUpdate={(_, field, index, patch) => updateRule(field, index, patch)}
              onRemove={(_, field, index) => removeRule(field, index)}
              onAdd={(_, field) => addRule(field)}
              t={t}
            />
          </section>

          <section className='space-y-3'>
            <Title heading={6} style={{ margin: 0 }}>
              {t('成本')}
            </Title>

            <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
              <Text type='tertiary' size='small' className='mb-2 block'>
                {t('成本计算方式')}
              </Text>
              <Radio.Group
                type='button'
                value={comboConfig.upstream_mode || 'manual_rules'}
                onChange={(event) =>
                  setComboConfig((prev) => ({
                    ...prev,
                    upstream_mode: event.target.value,
                    upstream_account_id:
                      event.target.value === 'wallet_observer'
                        ? Number(prev.upstream_account_id || 0)
                        : 0,
                  }))
                }
                size='small'
              >
                <Radio value='wallet_observer'>{t('按钱包余额变化')}</Radio>
                <Radio value='manual_rules'>{t('按模型单价')}</Radio>
              </Radio.Group>

              {comboConfig.upstream_mode === 'wallet_observer' ? (
                <div className='mt-3'>
                  <Text type='tertiary' size='small' className='mb-1.5 block'>
                    {t('上游账户')}
                  </Text>
                  <Select
                    value={comboConfig.upstream_account_id || 0}
                    onChange={(value) =>
                      setComboConfig((prev) => ({
                        ...prev,
                        upstream_account_id: Number(value || 0),
                      }))
                    }
                    optionList={accountOptions}
                    placeholder={t('选择钱包账户')}
                    emptyContent={t('先去“上游账户”添加')}
                    size='small'
                    style={{ width: '100%' }}
                  />
                </div>
              ) : null}
            </div>

            <MoneyField
              label={t('固定总成本')}
              value={comboConfig.upstream_fixed_total_amount}
              onChange={(value) =>
                setComboConfig((prev) => ({
                  ...prev,
                  upstream_fixed_total_amount: clampNumber(value),
                }))
              }
              t={t}
            />

            {comboConfig.upstream_mode === 'manual_rules' ? (
              <PricingRuleList
                comboId={comboConfig.id || 'modal'}
                field='upstream_rules'
                title={t('成本定价规则')}
                rules={comboConfig.upstream_rules}
                modelNameOptions={modelNameOptions}
                localModelMap={localModelMap}
                clampNumber={clampNumber}
                onUpdate={(_, field, index, patch) =>
                  updateRule(field, index, patch)
                }
                onRemove={(_, field, index) => removeRule(field, index)}
                onAdd={(_, field) => addRule(field)}
                t={t}
              />
            ) : null}
          </section>
        </div>
      ) : (
        <div className='py-8 text-center'>
          <Text type='tertiary'>{t('未选择组合')}</Text>
        </div>
      )}
    </Modal>
  );
};

export default PricingConfigModal;
