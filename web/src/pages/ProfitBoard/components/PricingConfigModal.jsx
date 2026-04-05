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
import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
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

const FieldLabel = ({ children }) => (
  <Text type='tertiary' size='small' className='mb-1.5 block'>
    {children}
  </Text>
);

const SectionCard = ({ title, description, aside, children }) => (
  <section className='space-y-4 rounded-2xl border border-semi-color-border bg-semi-color-bg-2 p-4 md:p-5'>
    <div className='flex flex-wrap items-start justify-between gap-3'>
      <div className='space-y-1'>
        <Title heading={6} style={{ margin: 0 }}>
          {title}
        </Title>
        {description ? (
          <Text type='tertiary' size='small'>
            {description}
          </Text>
        ) : null}
      </div>
      {aside}
    </div>
    {children}
  </section>
);

const SummaryBlock = ({ label, value }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-3'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <div className='mt-1 text-sm font-medium'>{value}</div>
  </div>
);

const MoneyField = ({ label, value, onChange, helper, t }) => (
  <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
    <FieldLabel>{label}</FieldLabel>
    <InputNumber
      min={0}
      value={value || 0}
      onChange={onChange}
      suffix='USD'
      style={{ width: '100%' }}
    />
    <Text type='tertiary' size='small' className='mt-2 block'>
      {helper || t('按请求量分摊')}
    </Text>
  </div>
);

const PricePreviewBlock = ({ title, value }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 px-3 py-2.5'>
    <Text type='tertiary' size='small'>
      {title}
    </Text>
    <div className='mt-1 text-sm font-semibold tabular-nums'>{value}</div>
  </div>
);

const SmartSuggestionPanel = ({
  smartSuggestions,
  selectedTemplateId,
  setSelectedTemplateId,
  onRegenerateName,
  onApplyRecommendedModes,
  onApplyRecommendedAccount,
  onApplyTemplate,
  t,
}) => {
  if (!smartSuggestions) return null;

  const hasQuickAction =
    smartSuggestions.canApplySuggestedName ||
    smartSuggestions.shouldRecommendModes ||
    smartSuggestions.shouldRecommendAccount ||
    (smartSuggestions.copyTemplateOptions || []).length > 0;

  if (!hasQuickAction) return null;

  return (
    <div className='rounded-2xl border border-dashed border-blue-500/30 bg-blue-500/[0.06] p-4'>
      <div className='flex flex-wrap items-center gap-2'>
        <Tag color='blue'>{t('智能建议')}</Tag>
        {smartSuggestions.canApplySuggestedName ? (
          <Button
            size='small'
            type='primary'
            theme='borderless'
            onClick={onRegenerateName}
          >
            {t('使用建议名称')}
          </Button>
        ) : null}
        {smartSuggestions.shouldRecommendModes ? (
          <Button
            size='small'
            type='primary'
            theme='borderless'
            onClick={onApplyRecommendedModes}
          >
            {t('沿用常用模式')}
          </Button>
        ) : null}
        {smartSuggestions.shouldRecommendAccount ? (
          <Button
            size='small'
            type='primary'
            theme='borderless'
            onClick={onApplyRecommendedAccount}
          >
            {t('套用推荐账户')}
          </Button>
        ) : null}
      </div>

      <div className='mt-3 space-y-2'>
        {smartSuggestions.canApplySuggestedName ? (
          <Text type='tertiary' size='small' className='block'>
            {t('建议名称')}: {smartSuggestions.suggestedName}
          </Text>
        ) : null}
        {smartSuggestions.shouldRecommendModes ? (
          <Text type='tertiary' size='small' className='block'>
            {t('常用模式')}: {smartSuggestions.recommendedModeLabel}
          </Text>
        ) : null}
        {smartSuggestions.shouldRecommendAccount ? (
          <Text type='tertiary' size='small' className='block'>
            {t('推荐账户')}: {smartSuggestions.recommendedAccountName}
          </Text>
        ) : null}
      </div>

      {(smartSuggestions.copyTemplateOptions || []).length > 0 ? (
        <div className='mt-3 flex flex-wrap items-end gap-3'>
          <div className='min-w-[220px] flex-1'>
            <FieldLabel>{t('复制已有组合的定价')}</FieldLabel>
            <Select
              value={selectedTemplateId}
              onChange={(value) => setSelectedTemplateId(value || '')}
              optionList={smartSuggestions.copyTemplateOptions}
              placeholder={t('选择一个已有组合')}
              style={{ width: '100%' }}
              size='small'
            />
          </div>
          <Button
            size='small'
            type='primary'
            theme='light'
            disabled={!selectedTemplateId}
            onClick={() => onApplyTemplate(selectedTemplateId)}
          >
            {t('复制定价')}
          </Button>
        </div>
      ) : null}
    </div>
  );
};

const PricingConfigModal = ({
  visible,
  isEditing,
  comboConfig,
  setComboConfig,
  onNameChange,
  onRegenerateName,
  onApplyRecommendedModes,
  onApplyTemplate,
  onApplyRecommendedAccount,
  smartSuggestions,
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
  const [selectedTemplateId, setSelectedTemplateId] = useState('');

  useEffect(() => {
    if (!visible) {
      setSelectedTemplateId('');
    }
  }, [visible, comboConfig?.id]);

  const sharedSite = comboConfig?.shared_site || {};
  const availableAccounts = (options?.upstream_accounts || []).filter(
    (item) => item.enabled !== false,
  );
  const accountOptions = availableAccounts.map((item) => ({
    label: `${item.name} · ${item.base_url || t('未填写地址')}`,
    value: item.id,
  }));
  const scopeOptions =
    comboConfig?.scope_type === 'channel' ? channelOptions : tagOptions;
  const selectedScopeCount =
    comboConfig?.scope_type === 'channel'
      ? (comboConfig?.channel_ids || []).length
      : (comboConfig?.tags || []).length;
  const siteModeLabel =
    comboConfig?.site_mode === 'shared_site_model'
      ? t('本站模型价格')
      : t('手动定价');
  const upstreamAccountLabel =
    accountOptions.find(
      (item) => Number(item.value) === Number(comboConfig?.upstream_account_id || 0),
    )?.label || t('未绑定');
  const upstreamModeLabel =
    comboConfig?.upstream_mode === 'wallet_observer'
      ? t('钱包余额变化')
      : t('模型单价');

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
      centered
      width={isMobile ? 360 : 980}
      okText={isEditing ? t('保存组合') : t('创建组合')}
      cancelText={t('取消')}
      bodyStyle={{
        paddingTop: 8,
        maxHeight: '72vh',
        overflowY: 'auto',
      }}
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

          <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-5'>
            <SummaryBlock
              label={t('组合名称')}
              value={comboConfig.name?.trim() || t('未命名组合')}
            />
            <SummaryBlock
              label={t('范围类型')}
              value={
                comboConfig.scope_type === 'channel' ? t('按渠道') : t('按标签')
              }
            />
            <SummaryBlock
              label={
                comboConfig.scope_type === 'channel'
                  ? t('已选渠道')
                  : t('已选标签')
              }
              value={t('{{count}} 项', { count: selectedScopeCount })}
            />
            <SummaryBlock label={t('收入')} value={siteModeLabel} />
            <SummaryBlock label={t('成本')} value={upstreamModeLabel} />
          </div>

          <SectionCard
            title={t('基础信息')}
            description={t('先定义组合范围，再决定后续收入和成本怎么计算')}
          >
            <SmartSuggestionPanel
              smartSuggestions={smartSuggestions}
              selectedTemplateId={selectedTemplateId}
              setSelectedTemplateId={setSelectedTemplateId}
              onRegenerateName={onRegenerateName}
              onApplyRecommendedModes={onApplyRecommendedModes}
              onApplyRecommendedAccount={onApplyRecommendedAccount}
              onApplyTemplate={onApplyTemplate}
              t={t}
            />

            <div className='grid gap-4 lg:grid-cols-2'>
              <div className='lg:col-span-2'>
                <div className='mb-1.5 flex items-center justify-between gap-2'>
                  <Text type='tertiary' size='small'>
                    {t('组合名称')}
                  </Text>
                  <Button
                    size='small'
                    theme='borderless'
                    type='primary'
                    onClick={onRegenerateName}
                  >
                    {t('智能命名')}
                  </Button>
                </div>
                <Input
                  value={comboConfig.name}
                  onChange={onNameChange}
                  placeholder={t('组合名称，例如：OpenAI 主力')}
                />
              </div>

              <div>
                <FieldLabel>{t('范围类型')}</FieldLabel>
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

              <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-4 py-3'>
                <Text type='tertiary' size='small'>
                  {comboConfig.scope_type === 'channel'
                    ? t('当前已选渠道')
                    : t('当前已选标签')}
                </Text>
                <div className='mt-1 text-sm font-semibold'>
                  {t('{{count}} 项', { count: selectedScopeCount })}
                </div>
              </div>

              <div className='lg:col-span-2'>
                <FieldLabel>
                  {comboConfig.scope_type === 'channel'
                    ? t('选择渠道')
                    : t('选择标签')}
                </FieldLabel>
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
                      : setComboConfig((prev) => ({
                          ...prev,
                          tags: value || [],
                        }))
                  }
                  placeholder={
                    comboConfig.scope_type === 'channel'
                      ? t('选择渠道')
                      : t('选择标签')
                  }
                  style={{ width: '100%' }}
                />
              </div>
            </div>
          </SectionCard>

          <SectionCard
            title={t('收入配置')}
            description={t('定义本站侧收入来源，固定金额会额外按请求量分摊')}
            aside={
              <Radio.Group
                type='button'
                value={comboConfig.site_mode || 'manual'}
                onChange={(event) =>
                  setComboConfig((prev) => ({
                    ...prev,
                    site_mode: event.target.value,
                  }))
                }
                size='small'
              >
                <Radio value='manual'>{t('手动定价')}</Radio>
                <Radio value='shared_site_model'>{t('本站模型价格')}</Radio>
              </Radio.Group>
            }
          >
            <div className='grid gap-4 xl:grid-cols-2'>
              <MoneyField
                label={t('固定总收入')}
                value={comboConfig.site_fixed_total_amount}
                onChange={(value) =>
                  setComboConfig((prev) => ({
                    ...prev,
                    site_fixed_total_amount: clampNumber(value),
                  }))
                }
                helper={t('会额外计入组合收入，再按请求量均摊')}
                t={t}
              />
              <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
                <FieldLabel>{t('当前收入口径')}</FieldLabel>
                <div className='text-sm font-semibold'>{siteModeLabel}</div>
                <Text type='tertiary' size='small' className='mt-2 block'>
                  {comboConfig.site_mode === 'shared_site_model'
                    ? t('命中本站模型价格时直接读取本地模型价，手动规则只负责补充和兜底')
                    : t('完全按手动规则和固定金额计算本站收入')}
                </Text>
              </div>
            </div>

            {comboConfig.site_mode === 'shared_site_model' ? (
              <div className='space-y-4 rounded-2xl border border-blue-500/20 bg-blue-500/[0.05] p-4'>
                <div className='grid gap-4 xl:grid-cols-2'>
                  <div>
                    <FieldLabel>{t('模型')}</FieldLabel>
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
                      style={{ width: '100%' }}
                    />
                  </div>

                  <div className='grid gap-4 sm:grid-cols-2'>
                    <div>
                      <FieldLabel>{t('分组')}</FieldLabel>
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
                        style={{ width: '100%' }}
                      />
                    </div>

                    <div className='flex items-end'>
                      <div className='flex w-full items-center justify-between rounded-xl border border-semi-color-border bg-semi-color-bg-1 px-3 py-2.5'>
                        <div>
                          <div className='text-sm font-medium'>
                            {t('按充值价')}
                          </div>
                          <Text type='tertiary' size='small'>
                            {t('开启后使用模型充值倍率')}
                          </Text>
                        </div>
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
                  </div>
                </div>

                {(sharedSite.model_names || []).length > 0 ? (
                  <div className='grid gap-3 xl:grid-cols-2'>
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
                          <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-4'>
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
              title={
                comboConfig.site_mode === 'shared_site_model'
                  ? t('手动规则补充')
                  : t('手动定价规则')
              }
              description={
                comboConfig.site_mode === 'shared_site_model'
                  ? t('用于补充未覆盖模型，或设置默认兜底价格')
                  : t('按模型定义本站收入单价')
              }
              rules={comboConfig.site_rules}
              modelNameOptions={modelNameOptions}
              localModelMap={localModelMap}
              clampNumber={clampNumber}
              onUpdate={(_, field, index, patch) => updateRule(field, index, patch)}
              onRemove={(_, field, index) => removeRule(field, index)}
              onAdd={(_, field) => addRule(field)}
              t={t}
            />
          </SectionCard>

          <SectionCard
            title={t('成本配置')}
            description={t('定义上游侧成本来源，固定金额会额外按请求量分摊')}
            aside={
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
                <Radio value='manual_rules'>{t('按模型单价')}</Radio>
                <Radio value='wallet_observer'>{t('钱包余额变化')}</Radio>
              </Radio.Group>
            }
          >
            <div className='grid gap-4 xl:grid-cols-2'>
              <MoneyField
                label={t('固定总成本')}
                value={comboConfig.upstream_fixed_total_amount}
                onChange={(value) =>
                  setComboConfig((prev) => ({
                    ...prev,
                    upstream_fixed_total_amount: clampNumber(value),
                  }))
                }
                helper={t('会额外计入组合成本，再按请求量均摊')}
                t={t}
              />
              <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
                <FieldLabel>{t('当前成本口径')}</FieldLabel>
                <div className='text-sm font-semibold'>{upstreamModeLabel}</div>
                <Text type='tertiary' size='small' className='mt-2 block'>
                  {comboConfig.upstream_mode === 'wallet_observer'
                    ? upstreamAccountLabel
                    : t('完全按手动成本规则计算上游费用')}
                </Text>
              </div>
            </div>

            {comboConfig.upstream_mode === 'wallet_observer' ? (
              <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
                <FieldLabel>{t('上游账户')}</FieldLabel>
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
                  style={{ width: '100%' }}
                />
              </div>
            ) : (
              <PricingRuleList
                comboId={comboConfig.id || 'modal'}
                field='upstream_rules'
                title={t('成本定价规则')}
                description={t('按模型定义上游成本单价')}
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
            )}
          </SectionCard>
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
