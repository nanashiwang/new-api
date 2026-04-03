import React from 'react';
import {
  Banner,
  Card,
  Empty,
  InputNumber,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import PricingRuleList from './PricingRuleList';

const { Text } = Typography;

const PricingRulesCard = ({
  batches,
  comboConfigs,
  siteConfig,
  setSiteConfig,
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
  <Card bordered={false} title={t('收益口径与价格规则')}>
    <div className='space-y-5'>
      <div className='rounded-[28px] border border-semi-color-border bg-[linear-gradient(135deg,#fefce8_0%,#f8fafc_100%)] p-5'>
        <div className='mb-4 flex flex-wrap items-start justify-between gap-3'>
          <div>
            <Text strong>{t('上游成本口径')}</Text>
            <Text type='tertiary' className='mt-1 block'>
              {t('整张收益看板只选一种上游成本模式，避免一部分看手动价格、一部分看钱包扣减。')}
            </Text>
          </div>
          <Select
            value={upstreamConfig.upstream_mode || 'manual_rules'}
            onChange={(value) =>
              setUpstreamConfig((prev) => ({
                ...prev,
                upstream_mode: value,
                cost_source:
                  value === 'wallet_observer' ? 'returned_cost_only' : 'manual_only',
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
            style={{ width: isMobile ? '100%' : 240 }}
          />
        </div>

        {upstreamConfig.upstream_mode === 'wallet_observer' ? (
          <div className='grid gap-3 lg:grid-cols-[minmax(0,1fr)_260px]'>
            <Select
              value={upstreamConfig.upstream_account_id || 0}
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
            />
            <Banner
              type='info'
              closeIcon={null}
              description={t('钱包模式会按所选上游账户的余额变化计算成本，并统一回灌到收益分析。')}
            />
          </div>
        ) : (
          <Banner
            type='info'
            closeIcon={null}
            description={t('固定模型成本模式下，每个组合维护自己的上游模型单价；如果模型不在本站列表里，再手动补录。')}
          />
        )}
      </div>

      <div className='rounded-[28px] border border-semi-color-border bg-semi-color-fill-0 p-5'>
        <div className='mb-3 flex items-center justify-between gap-3'>
          <div>
            <Text strong>{t('共享本站模型价格')}</Text>
            <Text type='tertiary' className='mt-1 block'>
              {t('这里直接取本站全部可见模型，不再只依赖那套偏窄的 local_models。')}
            </Text>
          </div>
        </div>
        <div className='grid gap-3 lg:grid-cols-[1.4fr_220px_220px]'>
          <Select
            multiple
            filter
            maxTagCount={3}
            value={siteConfig.model_names || []}
            onChange={(value) =>
              setSiteConfig((prev) => ({ ...prev, model_names: value || [] }))
            }
            optionList={modelNameOptions}
            placeholder={t('从本站全部可见模型中搜索并选择')}
            emptyContent={t('暂无可用模型')}
          />
          <Select
            value={siteConfig.group}
            onChange={(value) =>
              setSiteConfig((prev) => ({ ...prev, group: value }))
            }
            optionList={[
              { label: t('自动取最低分组倍率'), value: '' },
              ...(options.groups || []).map((item) => ({
                label: item,
                value: item,
              })),
            ]}
          />
          <div className='flex items-center justify-between rounded-xl border border-semi-color-border bg-white px-3 py-2'>
            <Text>{t('按充值价')}</Text>
            <Switch
              checked={siteConfig.use_recharge_price}
              onChange={(checked) =>
                setSiteConfig((prev) => ({
                  ...prev,
                  use_recharge_price: checked,
                }))
              }
            />
          </div>
        </div>
        <div className='mt-3 grid gap-3 xl:grid-cols-2'>
          {(siteConfig.model_names || []).length > 0 ? (
            (siteConfig.model_names || []).map((modelName) => {
              const preview = resolveSharedSitePreview(modelName);
              return (
                <div
                  key={modelName}
                  className='rounded-xl border border-semi-color-border bg-white p-3'
                >
                  <div className='mb-2 flex items-center justify-between gap-2'>
                    <Text strong>{modelName}</Text>
                    <Tag color={preview ? 'blue' : 'grey'}>
                      {preview ? t('已回显') : t('未命中本站价格')}
                    </Tag>
                  </div>
                  <div className='grid gap-2 md:grid-cols-2'>
                    <InputNumber disabled value={preview?.input_price || 0} suffix='USD / 1M 输入' />
                    <InputNumber disabled value={preview?.output_price || 0} suffix='USD / 1M 输出' />
                    <InputNumber disabled value={preview?.cache_read_price || 0} suffix='USD / 1M 缓存读' />
                    <InputNumber disabled value={preview?.cache_creation_price || 0} suffix='USD / 1M 缓存写' />
                  </div>
                </div>
              );
            })
          ) : (
            <Empty image={null} description={t('选中模型后，这里会直接回显本站价格')} />
          )}
        </div>
      </div>

      {batches.map((batch) => {
        const comboConfig =
          comboConfigs.find((item) => item.combo_id === batch.id) ||
          createDefaultComboPricingConfig(batch.id, siteConfig, upstreamConfig);
        return (
          <div
            key={batch.id}
            className='rounded-[28px] border border-semi-color-border bg-white p-5'
          >
            <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
              <div className='flex flex-wrap items-center gap-2'>
                <Text strong>{batch.name}</Text>
                <Tag
                  color={
                    comboConfig.site_mode === 'shared_site_model'
                      ? 'blue'
                      : 'orange'
                  }
                >
                  {comboConfig.site_mode === 'shared_site_model'
                    ? t('共享本站价格')
                    : t('手动本站价格')}
                </Tag>
              </div>
              <Space wrap>
                <Select
                  value={comboConfig.site_mode || 'manual'}
                  onChange={(value) =>
                    updateComboConfig(batch.id, { site_mode: value })
                  }
                  optionList={[
                    { label: t('手动输入本站价格'), value: 'manual' },
                    { label: t('共享本站模型价格'), value: 'shared_site_model' },
                  ]}
                  style={{ width: 220 }}
                />
                <InputNumber
                  min={0}
                  value={comboConfig.site_fixed_total_amount || 0}
                  onChange={(value) =>
                    updateComboConfig(batch.id, {
                      site_fixed_total_amount: clampNumber(value),
                    })
                  }
                  suffix='USD 固定总收入'
                />
              </Space>
            </div>

            <div className='grid gap-4 xl:grid-cols-2'>
              <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
                <div className='mb-3'>
                  <Text strong>{t('本站规则')}</Text>
                  <Text type='tertiary' className='mt-1 block'>
                    {comboConfig.site_mode === 'shared_site_model'
                      ? t('这个组合优先用上面的共享本站模型价格；没命中时再回落到这里的手动规则。')
                      : t('这个组合的本站收入完全按这里的手动规则计算。')}
                  </Text>
                </div>
                <PricingRuleList
                  comboId={batch.id}
                  field='site_rules'
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

              <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
                <div className='mb-3 flex flex-wrap items-center justify-between gap-3'>
                  <div>
                    <Text strong>{t('上游手动规则')}</Text>
                    <Text type='tertiary' className='mt-1 block'>
                      {upstreamConfig.upstream_mode === 'wallet_observer'
                        ? t('钱包模式启用后，这里的规则仅作为备用配置保留，不参与当前统计。')
                        : t('固定模型成本模式下，这里的规则就是当前组合的上游成本。')}
                    </Text>
                  </div>
                  <InputNumber
                    min={0}
                    value={comboConfig.upstream_fixed_total_amount || 0}
                    onChange={(value) =>
                      updateComboConfig(batch.id, {
                        upstream_fixed_total_amount: clampNumber(value),
                      })
                    }
                    suffix='USD 固定总成本'
                  />
                </div>
                <PricingRuleList
                  comboId={batch.id}
                  field='upstream_rules'
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
        );
      })}
    </div>
  </Card>
);

export default PricingRulesCard;
