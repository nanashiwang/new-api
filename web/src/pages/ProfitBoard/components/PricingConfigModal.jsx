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
import React, { useCallback, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Dropdown,
  Input,
  InputNumber,
  Radio,
  Select,
  SideSheet,
  Steps,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { CheckCircle2, ChevronDown, ChevronLeft, ChevronRight, Pencil } from 'lucide-react';
import PricingRuleList from './PricingRuleList';
import ModelSelectorSection from './ModelSelectorSection';
import { getUpstreamCostSourceLabel } from '../utils';

const { Text, Title } = Typography;

const FieldLabel = ({ children }) => (
  <Text type='tertiary' size='small' className='mb-1.5 block'>
    {children}
  </Text>
);

const SectionCard = ({ title, description, aside, children }) => (
  <section className='space-y-4 rounded-2xl border border-semi-color-border bg-semi-color-bg-2 p-4'>
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

const MoneyField = ({ label, value, onChange, helper, t }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3'>
    <FieldLabel>{label}</FieldLabel>
    <InputNumber
      min={0}
      value={value || 0}
      onChange={onChange}
      suffix='USD'
      size='small'
      style={{ width: '100%' }}
    />
    <Text type='tertiary' size='small' className='mt-1.5 block'>
      {helper || t('按请求量分摊')}
    </Text>
  </div>
);

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
  getModelsByChannelIds,
  getModelsByTags,
  isMobile,
  clampNumber,
  localModelMap,
  validationError,
  onOk,
  onCancel,
  t,
}) => {
  const sharedSite = comboConfig?.shared_site || {};
  const [editingCell, setEditingCell] = useState(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [stepTouched, setStepTouched] = useState(false);

  // 当 visible 变化时重置步骤
  const prevVisibleRef = React.useRef(visible);
  if (visible !== prevVisibleRef.current) {
    prevVisibleRef.current = visible;
    if (visible) { setCurrentStep(0); setStepTouched(false); }
  }
  const availableAccounts = (options?.upstream_accounts || []).filter(
    (item) => item.enabled !== false,
  );
  const accountOptions = availableAccounts.map((item) => ({
    label: `${item.name} · ${item.base_url || t('未填写地址')}`,
    value: item.id,
  }));
  const costSource = comboConfig?.cost_source || 'manual_only';
  const costSourceLabel = getUpstreamCostSourceLabel(costSource, t);
  const scopeOptions =
    comboConfig?.scope_type === 'channel' ? channelOptions : tagOptions;
  const selectedScopeCount =
    comboConfig?.scope_type === 'channel'
      ? (comboConfig?.channel_ids || []).length
      : (comboConfig?.tags || []).length;
  const siteModeLabel =
    comboConfig?.site_mode === 'log_quota'
      ? t('智能')
      : comboConfig?.site_mode === 'shared_site_model'
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

  const handleModelNamesChange = useCallback(
    (value) =>
      setComboConfig((prev) => ({
        ...prev,
        shared_site: {
          ...prev.shared_site,
          model_names: value || [],
        },
      })),
    [setComboConfig],
  );

  const handleImportFromScope = useCallback(() => {
    const models =
      comboConfig?.scope_type === 'tag'
        ? getModelsByTags?.(comboConfig?.tags || []) || []
        : getModelsByChannelIds?.(comboConfig?.channel_ids || []) || [];
    if (!models.length) return;
    setComboConfig((prev) => {
      const existing = new Set(prev?.shared_site?.model_names || []);
      models.forEach((m) => existing.add(m));
      return {
        ...prev,
        shared_site: {
          ...prev.shared_site,
          model_names: Array.from(existing).sort(),
        },
      };
    });
  }, [comboConfig?.scope_type, comboConfig?.channel_ids, comboConfig?.tags, getModelsByChannelIds, getModelsByTags, setComboConfig]);

  const scopeHasSelection =
    comboConfig?.scope_type === 'channel'
      ? (comboConfig?.channel_ids || []).length > 0
      : (comboConfig?.tags || []).length > 0;

  const hasSmartAction =
    smartSuggestions &&
    (smartSuggestions.canApplySuggestedName ||
      smartSuggestions.shouldRecommendModes ||
      smartSuggestions.shouldRecommendAccount ||
      (smartSuggestions.copyTemplateOptions || []).length > 0);

  const smartDropdownMenu = useMemo(() => {
    if (!smartSuggestions) return null;
    const items = [];
    items.push({ node: 'item', name: 'rename', onClick: onRegenerateName, children: t('智能命名') });
    if (smartSuggestions.shouldRecommendModes) {
      items.push({ node: 'item', name: 'modes', onClick: onApplyRecommendedModes, children: t('沿用常用模式') });
    }
    if (smartSuggestions.shouldRecommendAccount) {
      items.push({ node: 'item', name: 'account', onClick: onApplyRecommendedAccount, children: t('套用推荐账户') });
    }
    const templates = smartSuggestions.copyTemplateOptions || [];
    if (templates.length > 0) {
      items.push({ node: 'divider' });
      templates.forEach((tpl) => {
        items.push({
          node: 'item',
          name: `tpl-${tpl.value}`,
          onClick: () => onApplyTemplate(tpl.value),
          children: t('复制：') + tpl.label,
        });
      });
    }
    return <Dropdown.Menu>{items.map((item, i) =>
      item.node === 'divider'
        ? <Dropdown.Divider key={`d-${i}`} />
        : <Dropdown.Item key={item.name} onClick={item.onClick}>{item.children}</Dropdown.Item>
    )}</Dropdown.Menu>;
  }, [smartSuggestions, onRegenerateName, onApplyRecommendedModes, onApplyRecommendedAccount, onApplyTemplate, t]);

  const modelPreviewRows = useMemo(() => {
    if (!sharedSite.model_names?.length) return [];
    return (sharedSite.model_names || []).map((modelName) => {
      const preview = resolveSharedSitePreview(sharedSite, modelName);
      return { modelName, preview };
    });
  }, [sharedSite, resolveSharedSitePreview]);

  const siteRuleMap = useMemo(() => {
    const map = new Map();
    (comboConfig?.site_rules || []).forEach((rule) => {
      if (!rule.is_default && rule.model_name) {
        map.set(rule.model_name, rule);
      }
    });
    return map;
  }, [comboConfig?.site_rules]);

  const handlePreviewCellEdit = useCallback(
    (modelName, field, value) => {
      setComboConfig((prev) => {
        const rules = [...(prev?.site_rules || [])];
        const existingIndex = rules.findIndex(
          (r) => !r.is_default && r.model_name === modelName,
        );
        if (existingIndex >= 0) {
          rules[existingIndex] = {
            ...rules[existingIndex],
            [field]: clampNumber(value),
          };
        } else {
          const preview = resolveSharedSitePreview(
            prev?.shared_site || {},
            modelName,
          );
          rules.push({
            model_name: modelName,
            input_price: preview?.input_price || 0,
            output_price: preview?.output_price || 0,
            cache_read_price: preview?.cache_read_price || 0,
            cache_creation_price: preview?.cache_creation_price || 0,
            is_default: false,
            is_custom: false,
            [field]: clampNumber(value),
          });
        }
        return { ...prev, site_rules: rules };
      });
    },
    [setComboConfig, resolveSharedSitePreview, clampNumber],
  );

  const PRICE_FIELDS = useMemo(
    () => [
      { key: 'input_price', label: t('输入') },
      { key: 'output_price', label: t('输出') },
      { key: 'cache_read_price', label: t('缓存读') },
      { key: 'cache_creation_price', label: t('缓存写') },
    ],
    [t],
  );

  const renderEditableCell = useCallback(
    (modelName, field, previewValue) => {
      const override = siteRuleMap.get(modelName);
      const displayValue = override?.[field] ?? previewValue;
      const isOverridden = override != null && override[field] != null;
      const isEditing =
        editingCell?.modelName === modelName && editingCell?.field === field;

      if (isEditing) {
        return (
          <InputNumber
            min={0}
            value={displayValue ?? 0}
            onChange={(val) => handlePreviewCellEdit(modelName, field, val)}
            onBlur={() => setEditingCell(null)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === 'Escape')
                setEditingCell(null);
            }}
            size='small'
            style={{ width: '100%', minWidth: 70 }}
            autoFocus
          />
        );
      }

      return (
        <span
          className={`inline-flex cursor-pointer items-center gap-0.5 rounded px-1 py-0.5 transition-colors hover:bg-semi-color-fill-1 ${isOverridden ? 'font-medium text-blue-600 dark:text-blue-400' : ''}`}
          onClick={() => setEditingCell({ modelName, field })}
        >
          {displayValue?.toFixed(4) || '-'}
          {isOverridden ? (
            <Pencil size={9} className='shrink-0 opacity-60' />
          ) : null}
        </span>
      );
    },
    [editingCell, siteRuleMap, handlePreviewCellEdit],
  );

  const STEP_LABELS = useMemo(
    () => [t('基础信息'), t('收入配置'), t('成本配置')],
    [t],
  );

  const stepValidationError = useMemo(() => {
    if (currentStep === 0) {
      if (!scopeHasSelection) return t('请先选择渠道或标签');
    }
    return '';
  }, [currentStep, scopeHasSelection, t]);

  const footer = (
    <div className='flex items-center justify-between gap-2'>
      <Button onClick={onCancel}>{t('取消')}</Button>
      <div className='flex items-center gap-2'>
        {currentStep > 0 && (
          <Button
            icon={<ChevronLeft size={14} />}
            onClick={() => { setStepTouched(false); setCurrentStep((s) => s - 1); }}
          >
            {t('上一步')}
          </Button>
        )}
        {currentStep < 2 ? (
          <Button
            theme='solid'
            type='primary'
            icon={<ChevronRight size={14} />}
            iconPosition='right'
            onClick={() => {
              if (stepValidationError) { setStepTouched(true); return; }
              setStepTouched(false);
              setCurrentStep((s) => s + 1);
            }}
          >
            {t('下一步')}
          </Button>
        ) : (
          <Button theme='solid' type='primary' onClick={onOk}>
            {isEditing ? t('保存组合') : t('创建组合')}
          </Button>
        )}
      </div>
    </div>
  );

  return (
    <SideSheet
      title={isEditing ? t('编辑组合') : t('新建组合')}
      visible={visible}
      onCancel={onCancel}
      placement='right'
      width={isMobile ? '100%' : 700}
      footer={footer}
      bodyStyle={{ paddingTop: 8 }}
    >
      {comboConfig ? (
        <div className='space-y-4'>
          <div className='flex items-center gap-1.5 rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-3 py-2 text-xs text-semi-color-text-2'>
            <CheckCircle2 size={13} className='shrink-0 text-green-500' />
            <span>{t('保存组合后将自动同步到服务器，无需额外手动操作')}</span>
          </div>

          {validationError ? (
            <Banner
              type='danger'
              closeIcon={null}
              description={validationError}
            />
          ) : null}

          {stepTouched && stepValidationError && currentStep < 2 ? (
            <Banner
              type='warning'
              closeIcon={null}
              description={stepValidationError}
            />
          ) : null}

          {/* 步骤条 */}
          <Steps current={currentStep} onChange={setCurrentStep} size='small'>
            <Steps.Step title={STEP_LABELS[0]} />
            <Steps.Step title={STEP_LABELS[1]} />
            <Steps.Step title={STEP_LABELS[2]} />
          </Steps>

          {/* 摘要行 */}
          <div className='grid grid-cols-2 gap-2'>
            <div className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-2.5 py-1.5'>
              <Text type='tertiary' size='small'>{t('范围')}</Text>
              <div className='text-xs font-medium'>
                {comboConfig.scope_type === 'channel' ? t('按渠道') : t('按标签')}
                {' · '}
                {t('{{count}} 项', { count: selectedScopeCount })}
              </div>
            </div>
            <div className='rounded-lg border border-semi-color-border bg-semi-color-fill-0 px-2.5 py-1.5'>
              <Text type='tertiary' size='small'>{t('收入 / 成本')}</Text>
              <div className='text-xs font-medium'>
                {siteModeLabel} / {upstreamModeLabel}
              </div>
            </div>
          </div>

          {/* Step 0: 基础信息 */}
          {currentStep === 0 && (
          <SectionCard
            title={t('基础信息')}
            description={t('先定义组合范围，再决定后续收入和成本怎么计算')}
          >
            <div className='space-y-3'>
              {/* 组合名称 + 智能操作 Dropdown */}
              <div>
                <div className='mb-1.5 flex items-center justify-between gap-2'>
                  <Text type='tertiary' size='small'>{t('组合名称')}</Text>
                  {hasSmartAction ? (
                    <Dropdown render={smartDropdownMenu} position='bottomRight' trigger='click'>
                      <Button size='small' theme='borderless' type='primary' icon={<ChevronDown size={14} />} iconPosition='right'>
                        {t('快捷操作')}
                      </Button>
                    </Dropdown>
                  ) : (
                    <Button size='small' theme='borderless' type='primary' onClick={onRegenerateName}>
                      {t('智能命名')}
                    </Button>
                  )}
                </div>
                <Input
                  value={comboConfig.name}
                  onChange={onNameChange}
                  placeholder={t('组合名称，例如：OpenAI 主力')}
                />
              </div>

              {/* 范围类型 + 选择 */}
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

              <div>
                <FieldLabel>
                  {comboConfig.scope_type === 'channel'
                    ? t('选择渠道')
                    : t('选择标签')}
                </FieldLabel>
                <Select
                  multiple
                  filter
                  maxTagCount={3}
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
          )}

          {/* Step 1: 收入配置 */}
          {currentStep === 1 && (
          <SectionCard
            title={t('收入配置')}
            description={
              comboConfig.site_mode === 'log_quota'
                ? t('智能模式直接读取日志中已计算的额度作为本站收入')
                : comboConfig.site_mode === 'shared_site_model'
                  ? t('命中本站模型价格时直接读取本地模型价，手动规则只负责补充和兜底')
                  : t('完全按手动规则和固定金额计算本站收入')
            }
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
                <Radio value='log_quota'>{t('智能')}</Radio>
              </Radio.Group>
            }
          >
            {comboConfig.site_mode === 'log_quota' ? (
              <div className='space-y-3 rounded-2xl border border-green-500/20 bg-green-500/[0.05] p-3'>
                <Banner
                  type='success'
                  closeIcon={null}
                  description={t('智能模式直接读取选中模型每条日志的已计算额度（含分组倍率、模型倍率），换算为USD作为本站收入')}
                />
                <ModelSelectorSection
                  modelNames={sharedSite.model_names || []}
                  onModelNamesChange={handleModelNamesChange}
                  modelNameOptions={modelNameOptions}
                  scopeHasSelection={scopeHasSelection}
                  onImportFromScope={handleImportFromScope}
                  t={t}
                />
              </div>
            ) : (
              <>
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

            {comboConfig.site_mode === 'shared_site_model' ? (
              <div className='space-y-3 rounded-2xl border border-blue-500/20 bg-blue-500/[0.05] p-3'>
                <ModelSelectorSection
                  modelNames={sharedSite.model_names || []}
                  onModelNamesChange={handleModelNamesChange}
                  modelNameOptions={modelNameOptions}
                  scopeHasSelection={scopeHasSelection}
                  onImportFromScope={handleImportFromScope}
                  t={t}
                />

                {/* 定价基准 + 分组 */}
                <div className='grid grid-cols-2 gap-3'>
                  <div>
                    <FieldLabel>{t('定价基准')}</FieldLabel>
                    <Radio.Group
                      type='button'
                      value={sharedSite.use_recharge_price ? 'recharge' : 'standard'}
                      onChange={(event) =>
                        setComboConfig((prev) => ({
                          ...prev,
                          shared_site: {
                            ...prev.shared_site,
                            use_recharge_price: event.target.value === 'recharge',
                          },
                        }))
                      }
                      size='small'
                    >
                      <Radio value='standard'>{t('按套餐价')}</Radio>
                      <Radio value='recharge'>{t('按充值价')}</Radio>
                    </Radio.Group>
                  </div>
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
                      size='small'
                      style={{ width: '100%' }}
                    />
                  </div>
                </div>

                {/* 模型价格预览 - 紧凑表格 */}
                {modelPreviewRows.length > 0 ? (
                  <div className='space-y-1.5'>
                    <div className='flex items-center justify-between'>
                      <Text strong size='small'>{t('价格预览')}</Text>
                      <Text type='tertiary' size='small'>{t('点击价格可编辑，编辑后自动写入手动规则')}</Text>
                    </div>
                    <div className='rounded-xl border border-semi-color-border overflow-hidden'>
                      <table className='w-full text-xs'>
                        <thead>
                          <tr className='bg-semi-color-fill-0'>
                            <th className='px-2 py-1.5 text-left font-medium text-semi-color-text-2'>{t('模型')}</th>
                            {PRICE_FIELDS.map(({ key, label }) => (
                              <th key={key} className='px-2 py-1.5 text-right font-medium text-semi-color-text-2'>{label}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {modelPreviewRows.map(({ modelName, preview }) => {
                            const hasOverride = siteRuleMap.has(modelName);
                            return (
                              <tr
                                key={`${comboConfig.id}-${modelName}`}
                                className='border-t border-semi-color-border'
                              >
                                <td className='px-2 py-1.5'>
                                  <div className='flex items-center gap-1.5'>
                                    <span className={`truncate max-w-[140px] ${preview ? '' : 'text-semi-color-text-2'}`}>
                                      {modelName}
                                    </span>
                                    {hasOverride ? (
                                      <Tag color='blue' size='small' className='shrink-0'>{t('已覆盖')}</Tag>
                                    ) : !preview ? (
                                      <Tag color='grey' size='small' className='shrink-0'>{t('未匹配')}</Tag>
                                    ) : null}
                                  </div>
                                </td>
                                {PRICE_FIELDS.map(({ key }) => (
                                  <td key={key} className='px-2 py-1.5 text-right tabular-nums'>
                                    {renderEditableCell(modelName, key, preview?.[key])}
                                  </td>
                                ))}
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
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
              </>
            )}
          </SectionCard>
          )}

          {/* Step 2: 成本配置 */}
          {currentStep === 2 && (
          <SectionCard
            title={t('成本配置')}
            description={
              comboConfig.upstream_mode === 'wallet_observer'
                ? `${t('钱包余额变化')} · ${upstreamAccountLabel}`
                : costSourceLabel
            }
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

            {comboConfig.upstream_mode === 'wallet_observer' ? (
              <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3'>
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
                  emptyContent={t('先去"上游账户"添加')}
                  style={{ width: '100%' }}
                />
              </div>
            ) : (
              <>
                <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3'>
                  <FieldLabel>{t('成本来源')}</FieldLabel>
                  <Select
                    value={costSource}
                    onChange={(value) =>
                      setComboConfig((prev) => ({
                        ...prev,
                        cost_source: value || 'manual_only',
                      }))
                    }
                    optionList={[
                      { label: t('只用手动成本规则'), value: 'manual_only' },
                      {
                        label: t('优先用上游返回费用'),
                        value: 'returned_cost_first',
                      },
                      {
                        label: t('只用上游返回费用'),
                        value: 'returned_cost_only',
                      },
                    ]}
                    style={{ width: '100%' }}
                  />
                  <Text type='tertiary' size='small' className='mt-1.5 block'>
                    {costSource === 'returned_cost_first'
                      ? t('先读取上游返回费用；缺失时再按下方规则回退')
                      : costSource === 'returned_cost_only'
                        ? t('只认上游返回费用；下方规则当前不会参与计算')
                        : t('完全按下方手动规则计算上游费用')}
                  </Text>
                </div>
                {costSource === 'returned_cost_only' ? (
                  <Banner
                    type='warning'
                    closeIcon={null}
                    description={t('当前选择“只用上游返回费用”，下面的手动规则会保留，但本次统计不会参与计算')}
                  />
                ) : null}
                <PricingRuleList
                  comboId={comboConfig.id || 'modal'}
                  field='upstream_rules'
                  title={t('成本定价规则')}
                  description={
                    costSource === 'returned_cost_first'
                      ? t('当上游没有返回费用时，按模型定义手动成本单价')
                      : t('按模型定义上游成本单价')
                  }
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
              </>
            )}
          </SectionCard>
          )}
        </div>
      ) : (
        <div className='py-8 text-center'>
          <Text type='tertiary'>{t('未选择组合')}</Text>
        </div>
      )}
    </SideSheet>
  );
};

export default PricingConfigModal;
