import React from 'react';
import {
  Banner,
  Card,
  Collapse,
  Empty,
  InputNumber,
  Select,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import PricingRuleList from './PricingRuleList';

const { Text } = Typography;

const MoneyField = ({ label, value, onChange, helper, t }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-3'>
    <Text strong size='small' className='block'>
      {label}
    </Text>
    <InputNumber
      min={0}
      value={value || 0}
      onChange={onChange}
      suffix='USD'
      style={{ width: '100%', marginTop: 10 }}
    />
    {helper ? (
      <Text type='tertiary' size='small' className='mt-2 block'>
        {helper}
      </Text>
    ) : null}
  </div>
);

const PricePreviewBlock = ({ title, value, tone }) => (
  <div className='rounded-lg bg-semi-color-fill-0 px-3 py-2'>
    <Text type='tertiary' size='small'>
      {title}
    </Text>
    <div className={`mt-1 text-sm font-semibold ${tone}`}>{value}</div>
  </div>
);

const sharedSummaryText = (comboConfig, t) => {
  if (comboConfig.site_mode !== 'shared_site_model') return t('手动本站价格');
  const modelCount = comboConfig.shared_site?.model_names?.length || 0;
  return modelCount > 0
    ? t('{{count}} 个模型', { count: modelCount })
    : t('未选择模型');
};

const PricingRulesCard = ({
  batches,
  comboConfigs,
  siteConfig,
  modelNameOptions,
  options,
  resolveSharedSitePreview,
  upstreamConfig,
  setUpstreamConfig,
  isMobile,
  createDefaultComboPricingConfig,
  updateComboConfig,
  updateComboRule,
  removeComboRule,
  addComboRule,
  localModelMap,
  clampNumber,
  t,
}) => (
  <Card bordered={false} title={t('组合定价工作台')}>
    <div className='space-y-4'>
      <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
        <div className='flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between'>
          <div className='space-y-1'>
            <Text strong>{t('全局上游成本来源')}</Text>
            <Text type='tertiary' size='small'>
              {t('上游钱包属于全局成本来源；本站价格、固定总额和手动规则在下方按组合分别设置。')}
            </Text>
          </div>
          <div className='grid w-full gap-3 xl:max-w-[680px] xl:grid-cols-[220px_minmax(0,1fr)]'>
            <div>
              <Text type='tertiary' size='small' className='mb-1.5 block'>
                {t('成本口径')}
              </Text>
              <Select
                value={upstreamConfig.upstream_mode || 'manual_rules'}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    upstream_mode: value,
                    cost_source:
                      value === 'wallet_observer'
                        ? 'returned_cost_only'
                        : 'manual_only',
                    upstream_account_id:
                      value === 'wallet_observer'
                        ? prev.upstream_account_id || 0
                        : 0,
                  }))
                }
                optionList={[
                  { label: t('固定模型成本'), value: 'manual_rules' },
                  { label: t('上游钱包扣减'), value: 'wallet_observer' },
                ]}
                style={{ width: '100%' }}
              />
            </div>
            <div>
              <Text type='tertiary' size='small' className='mb-1.5 block'>
                {t('钱包来源')}
              </Text>
              <Select
                value={upstreamConfig.upstream_account_id || 0}
                disabled={upstreamConfig.upstream_mode !== 'wallet_observer'}
                onChange={(value) =>
                  setUpstreamConfig((prev) => ({
                    ...prev,
                    upstream_account_id: Number(value || 0),
                  }))
                }
                optionList={(options.upstream_accounts || [])
                  .filter((item) => item.enabled !== false)
                  .map((item) => ({
                    label: `${item.name} · ${item.base_url}`,
                    value: item.id,
                  }))}
                placeholder={t('选择一个已维护的上游账户')}
                emptyContent={t('还没有上游账户，请先去“上游账户”页签创建')}
                style={{ width: '100%' }}
              />
            </div>
          </div>
        </div>
      </div>

      {batches.length === 0 ? (
        <Empty
          image={null}
          description={t('先创建组合，下面才会出现每个组合的定价工作台')}
        />
      ) : null}

      {batches.length > 0 ? (
        <Collapse defaultActiveKey={[batches[0].id]} accordion={false}>
          {batches.map((batch) => {
            const comboConfig =
              comboConfigs.find((item) => item.combo_id === batch.id) ||
              createDefaultComboPricingConfig(
                batch.id,
                undefined,
                siteConfig,
                upstreamConfig,
              );
            const sharedSite = comboConfig.shared_site || {};
            const usingSharedSite = comboConfig.site_mode === 'shared_site_model';

            return (
              <Collapse.Panel
                key={batch.id}
                itemKey={batch.id}
                header={
                  <div className='flex w-full flex-wrap items-center gap-2 pr-3'>
                    <Text strong>{batch.name}</Text>
                    <Tag color={usingSharedSite ? 'blue' : 'grey'}>
                      {usingSharedSite
                        ? t('本站模型价格')
                        : t('手动本站价格')}
                    </Tag>
                    <Tag color='cyan'>{sharedSummaryText(comboConfig, t)}</Tag>
                    <Tag color='orange'>
                      {t('本站固定总额')} {comboConfig.site_fixed_total_amount || 0}
                      {' USD'}
                    </Tag>
                    <Tag color='amber'>
                      {t('上游固定总额')}{' '}
                      {comboConfig.upstream_fixed_total_amount || 0}
                      {' USD'}
                    </Tag>
                  </div>
                }
              >
                <div className='space-y-4'>
                  <div className='grid gap-4 2xl:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]'>
                    <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
                      <div className='mb-4 flex flex-wrap items-start justify-between gap-3'>
                        <div className='space-y-1'>
                          <Text strong>{t('本站收入规则')}</Text>
                          <Text type='tertiary' size='small'>
                            {t('每个组合独立决定是手动定价，还是读取本站模型价格。')}
                          </Text>
                        </div>
                        <div className='w-full md:w-[240px]'>
                          <Text type='tertiary' size='small' className='mb-1.5 block'>
                            {t('本站价格来源')}
                          </Text>
                          <Select
                            value={comboConfig.site_mode || 'manual'}
                            onChange={(value) =>
                              updateComboConfig(batch.id, { site_mode: value })
                            }
                            optionList={[
                              { label: t('手动输入本站价格'), value: 'manual' },
                              {
                                label: t('读取本站模型价格'),
                                value: 'shared_site_model',
                              },
                            ]}
                            style={{ width: '100%' }}
                          />
                        </div>
                      </div>

                      <div className='grid gap-3 xl:grid-cols-2'>
                        <MoneyField
                          label={t('固定总收入')}
                          value={comboConfig.site_fixed_total_amount}
                          onChange={(value) =>
                            updateComboConfig(batch.id, {
                              site_fixed_total_amount: clampNumber(value),
                            })
                          }
                          helper={t('按请求量分摊到这个组合的时间段内。')}
                          t={t}
                        />
                        <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-3'>
                          <Text strong size='small' className='block'>
                            {t('当前模式')}
                          </Text>
                          <div className='mt-2 flex flex-wrap gap-2'>
                            <Tag color={usingSharedSite ? 'blue' : 'grey'}>
                              {usingSharedSite
                                ? t('本站模型价格')
                                : t('手动本站价格')}
                            </Tag>
                            {usingSharedSite ? (
                              <Tag color={sharedSite.use_recharge_price ? 'green' : 'cyan'}>
                                {sharedSite.use_recharge_price
                                  ? t('按充值价')
                                  : t('按原价')}
                              </Tag>
                            ) : null}
                            {sharedSite.group ? (
                              <Tag color='purple'>
                                {t('分组')} {sharedSite.group}
                              </Tag>
                            ) : null}
                          </div>
                        </div>
                      </div>

                      {usingSharedSite ? (
                        <div className='mt-4 space-y-4 rounded-2xl border border-blue-500/20 bg-blue-500/5 p-4'>
                          <div className='grid gap-3 xl:grid-cols-[minmax(0,1.4fr)_220px_180px]'>
                            <div>
                              <Text type='tertiary' size='small' className='mb-1.5 block'>
                                {t('本站模型')}
                              </Text>
                              <Select
                                multiple
                                filter
                                maxTagCount={isMobile ? 2 : 4}
                                value={sharedSite.model_names || []}
                                onChange={(value) =>
                                  updateComboConfig(batch.id, {
                                    shared_site: {
                                      ...sharedSite,
                                      model_names: value || [],
                                    },
                                  })
                                }
                                optionList={modelNameOptions}
                                placeholder={t('选择这个组合要读取的本站模型')}
                                emptyContent={t('暂无可用模型')}
                                style={{ width: '100%' }}
                              />
                            </div>
                            <div>
                              <Text type='tertiary' size='small' className='mb-1.5 block'>
                                {t('分组倍率')}
                              </Text>
                              <Select
                                value={sharedSite.group || ''}
                                onChange={(value) =>
                                  updateComboConfig(batch.id, {
                                    shared_site: {
                                      ...sharedSite,
                                      group: value,
                                    },
                                  })
                                }
                                optionList={[
                                  { label: t('自动取最低分组倍率'), value: '' },
                                  ...(options.groups || []).map((item) => ({
                                    label: item,
                                    value: item,
                                  })),
                                ]}
                                style={{ width: '100%' }}
                              />
                            </div>
                            <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 px-3 py-2'>
                              <Text type='tertiary' size='small' className='block'>
                                {t('价格口径')}
                              </Text>
                              <div className='mt-2 flex items-center justify-between gap-3'>
                                <Text>{t('按充值价')}</Text>
                                <Switch
                                  checked={!!sharedSite.use_recharge_price}
                                  onChange={(checked) =>
                                    updateComboConfig(batch.id, {
                                      shared_site: {
                                        ...sharedSite,
                                        use_recharge_price: checked,
                                      },
                                    })
                                  }
                                />
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
                                    key={`${batch.id}-${modelName}`}
                                    className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-3'
                                  >
                                    <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                                      <Text strong>{modelName}</Text>
                                      <Tag color={preview ? 'blue' : 'grey'}>
                                        {preview ? t('已命中本站价格') : t('未命中')}
                                      </Tag>
                                    </div>
                                    <div className='grid gap-2 md:grid-cols-2'>
                                      <PricePreviewBlock
                                        title={t('输入')}
                                        value={`${preview?.input_price?.toFixed(4) || '0'} USD/1M`}
                                        tone='text-semi-color-text-0'
                                      />
                                      <PricePreviewBlock
                                        title={t('输出')}
                                        value={`${preview?.output_price?.toFixed(4) || '0'} USD/1M`}
                                        tone='text-semi-color-text-0'
                                      />
                                      <PricePreviewBlock
                                        title={t('缓存读')}
                                        value={`${preview?.cache_read_price?.toFixed(4) || '0'} USD/1M`}
                                        tone='text-semi-color-text-0'
                                      />
                                      <PricePreviewBlock
                                        title={t('缓存写')}
                                        value={`${preview?.cache_creation_price?.toFixed(4) || '0'} USD/1M`}
                                        tone='text-semi-color-text-0'
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
                              description={t('这个组合已启用本站模型价格，但还没有选择模型。')}
                            />
                          )}
                        </div>
                      ) : null}

                      <div className='mt-4'>
                        <PricingRuleList
                          comboId={batch.id}
                          field='site_rules'
                          title={t('本站手动规则')}
                          description={t('当这个组合使用手动定价，或本站模型价格未命中时，使用这里的规则。')}
                          rules={comboConfig.site_rules}
                          modelNameOptions={modelNameOptions}
                          localModelMap={localModelMap}
                          clampNumber={clampNumber}
                          onUpdate={updateComboRule}
                          onRemove={removeComboRule}
                          onAdd={addComboRule}
                          t={t}
                        />
                      </div>
                    </div>

                    <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
                      <div className='mb-4 space-y-1'>
                        <Text strong>{t('上游成本规则')}</Text>
                        <Text type='tertiary' size='small'>
                          {t('这里控制这个组合的手动上游成本和固定总成本；全局钱包模式会覆盖为钱包观测成本。')}
                        </Text>
                      </div>

                      <div className='grid gap-3 xl:grid-cols-2'>
                        <MoneyField
                          label={t('固定总成本')}
                          value={comboConfig.upstream_fixed_total_amount}
                          onChange={(value) =>
                            updateComboConfig(batch.id, {
                              upstream_fixed_total_amount: clampNumber(value),
                            })
                          }
                          helper={t('按请求量分摊到这个组合的时间段内。')}
                          t={t}
                        />
                        <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-3'>
                          <Text strong size='small' className='block'>
                            {t('成本来源摘要')}
                          </Text>
                          <div className='mt-2 flex flex-wrap gap-2'>
                            <Tag color='amber'>
                              {upstreamConfig.upstream_mode === 'wallet_observer'
                                ? t('全局钱包扣减')
                                : t('手动上游成本')}
                            </Tag>
                            {(comboConfig.upstream_rules || []).length > 0 ? (
                              <Tag color='cyan'>
                                {t('{{count}} 条规则', {
                                  count: comboConfig.upstream_rules.length,
                                })}
                              </Tag>
                            ) : null}
                          </div>
                        </div>
                      </div>

                      <div className='mt-4'>
                        <PricingRuleList
                          comboId={batch.id}
                          field='upstream_rules'
                          title={t('上游手动规则')}
                          description={t('当没有使用全局钱包扣减时，这些规则决定这个组合的上游成本。')}
                          rules={comboConfig.upstream_rules}
                          modelNameOptions={modelNameOptions}
                          localModelMap={localModelMap}
                          clampNumber={clampNumber}
                          onUpdate={updateComboRule}
                          onRemove={removeComboRule}
                          onAdd={addComboRule}
                          t={t}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </Collapse.Panel>
            );
          })}
        </Collapse>
      ) : null}
    </div>
  </Card>
);

export default PricingRulesCard;
