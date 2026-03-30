import React from 'react';
import {
  Banner,
  Card,
  Collapse,
  Empty,
  Input,
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
  <Card bordered={false} title={t('组合与价格规则')}>
    <div className='space-y-5'>
      <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
        <div className='mb-3 flex items-center justify-between gap-3'>
          <div>
            <Text strong>{t('共享本站模型价格')}</Text>
            <Text type='tertiary' className='mt-1 block'>
              {t('选择这里的模型后，所有使用“读取本站模型价格”的组合都会共用这套读取规则。')}
            </Text>
          </div>
        </div>
        <div className='grid gap-3 lg:grid-cols-[1.4fr_220px_220px]'>
          <Select
            multiple
            allowCreate
            filter
            value={siteConfig.model_names || []}
            onChange={(value) => setSiteConfig((prev) => ({ ...prev, model_names: value || [] }))}
            optionList={modelNameOptions}
            placeholder={t('选择或输入本站模型')}
          />
          <Select
            value={siteConfig.group}
            onChange={(value) => setSiteConfig((prev) => ({ ...prev, group: value }))}
            optionList={[
              { label: t('自动取最低分组倍率'), value: '' },
              ...(options.groups || []).map((item) => ({ label: item, value: item })),
            ]}
          />
          <div className='flex items-center justify-between rounded-xl border border-semi-color-border bg-white px-3 py-2'>
            <Text>{t('按充值价')}</Text>
            <Switch
              checked={siteConfig.use_recharge_price}
              onChange={(checked) =>
                setSiteConfig((prev) => ({ ...prev, use_recharge_price: checked }))
              }
            />
          </div>
        </div>
        <div className='mt-3 grid gap-3 xl:grid-cols-2'>
          {(siteConfig.model_names || []).length > 0 ? (
            (siteConfig.model_names || []).map((modelName) => {
              const preview = resolveSharedSitePreview(modelName);
              return (
                <div key={modelName} className='rounded-xl border border-semi-color-border bg-white p-3'>
                  <div className='mb-2 flex items-center justify-between gap-2'>
                    <Text strong>{modelName}</Text>
                    <Tag color={preview ? 'blue' : 'grey'}>
                      {preview ? t('已回显') : t('未命中本站模型')}
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
            <Empty image={null} description={t('启用共享本站模型价格后，会在这里回显各模型的价格')} />
          )}
        </div>
      </div>

      <div className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'>
        <div className='mb-3 flex items-center justify-between gap-3'>
          <div>
            <Text strong>{t('上游费用来源策略')}</Text>
            <Text type='tertiary' className='mt-1 block'>
              {t('这个策略全页共用，决定优先用上游返回费用还是组合里手动维护的上游价格。')}
            </Text>
          </div>
          <Select
            value={upstreamConfig.cost_source}
            onChange={(value) => setUpstreamConfig((prev) => ({ ...prev, cost_source: value }))}
            optionList={[
              { label: t('仅手动价格'), value: 'manual_only' },
              { label: t('优先上游返回，缺失时手动回退'), value: 'returned_cost_first' },
              { label: t('仅上游返回费用'), value: 'returned_cost_only' },
            ]}
            style={{ width: isMobile ? '100%' : 260 }}
          />
        </div>

        <Collapse accordion={false}>
          {batches.map((batch) => {
            const comboConfig =
              comboConfigs.find((item) => item.combo_id === batch.id) ||
              createDefaultComboPricingConfig(batch.id, siteConfig, upstreamConfig);
            return (
              <Collapse.Panel
                key={batch.id}
                itemKey={batch.id}
                header={
                  <div className='flex flex-wrap items-center gap-2'>
                    <Text strong>{batch.name}</Text>
                    <Tag color={comboConfig.site_mode === 'shared_site_model' ? 'blue' : 'orange'}>
                      {comboConfig.site_mode === 'shared_site_model'
                        ? t('读取本站模型价格')
                        : t('手动本站价格')}
                    </Tag>
                    <Tag color='cyan'>
                      {batch.scope_type === 'channel' ? t('渠道') : t('标签聚合渠道')}
                    </Tag>
                  </div>
                }
              >
                <div className='space-y-4'>
                  <div className='rounded-xl border border-semi-color-border bg-white p-4'>
                    <div className='mb-3 flex flex-wrap items-center justify-between gap-3'>
                      <div>
                        <Text strong>{t('本站价格')}</Text>
                        <Text type='tertiary' className='mt-1 block'>
                          {t('读取本站模型价格时用上面的共享规则；手动模式时可以给多个模型分别配置。')}
                        </Text>
                      </div>
                      <Space wrap>
                        <Select
                          value={comboConfig.site_mode || 'manual'}
                          onChange={(value) => updateComboConfig(batch.id, { site_mode: value })}
                          optionList={[
                            { label: t('手动输入本站价格'), value: 'manual' },
                            { label: t('读取本站模型价格'), value: 'shared_site_model' },
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
                    {comboConfig.site_mode === 'shared_site_model' ? (
                      <Text type='tertiary'>
                        {t('这个组合会直接读取上方共享本站模型规则；如果共享模型未命中，会回退到下面的手动规则。')}
                      </Text>
                    ) : null}
                    <div className='mt-3'>
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
                  </div>

                  <div className='rounded-xl border border-semi-color-border bg-white p-4'>
                    <div className='mb-3 flex flex-wrap items-center justify-between gap-3'>
                      <div>
                        <Text strong>{t('上游价格')}</Text>
                        <Text type='tertiary' className='mt-1 block'>
                          {t('上游手动规则按组合独立维护，可给多个模型分别配置。')}
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
                        suffix='USD 固定总费用'
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

                  <div className='rounded-xl border border-semi-color-border bg-white p-4'>
                    <div className='mb-3 flex flex-wrap items-center justify-between gap-3'>
                      <div>
                        <Text strong>{t('远端额度观测')}</Text>
                        <Text type='tertiary' className='mt-1 block'>
                          {t('仅支持 new-api 远端实例。系统会读取远端钱包已用额度和订阅已用额度的增量，换算成当前组合的上游观测消耗。')}
                        </Text>
                      </div>
                      <div className='flex items-center justify-between rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-2'>
                        <Text>{t('启用')}</Text>
                        <Switch
                          checked={!!comboConfig.remote_observer?.enabled}
                          onChange={(checked) =>
                            updateComboConfig(batch.id, {
                              remote_observer: {
                                ...(comboConfig.remote_observer || {}),
                                enabled: checked,
                              },
                            })
                          }
                        />
                      </div>
                    </div>
                    <div className='grid gap-3 xl:grid-cols-[1.4fr_220px]'>
                      <Input
                        value={comboConfig.remote_observer?.base_url || ''}
                        onChange={(value) =>
                          updateComboConfig(batch.id, {
                            remote_observer: {
                              ...(comboConfig.remote_observer || {}),
                              base_url: value,
                            },
                          })
                        }
                        placeholder={t('https://your-new-api.example.com')}
                        disabled={!comboConfig.remote_observer?.enabled}
                      />
                      <InputNumber
                        min={0}
                        value={comboConfig.remote_observer?.user_id || 0}
                        onChange={(value) =>
                          updateComboConfig(batch.id, {
                            remote_observer: {
                              ...(comboConfig.remote_observer || {}),
                              user_id: Math.max(Number(value || 0), 0),
                            },
                          })
                        }
                        suffix='User ID'
                        disabled={!comboConfig.remote_observer?.enabled}
                      />
                    </div>
                    <div className='mt-3 grid gap-3 xl:grid-cols-[1fr_auto] xl:items-center'>
                      <Input
                        value={comboConfig.remote_observer?.access_token || ''}
                        onChange={(value) =>
                          updateComboConfig(batch.id, {
                            remote_observer: {
                              ...(comboConfig.remote_observer || {}),
                              access_token: value,
                            },
                          })
                        }
                        type='password'
                        mode='password'
                        placeholder={
                          comboConfig.remote_observer?.access_token_masked
                            ? t('留空表示继续使用已保存 token')
                            : t('输入远端用户 access token')
                        }
                        disabled={!comboConfig.remote_observer?.enabled}
                      />
                      {comboConfig.remote_observer?.access_token_masked ? (
                        <Tag color='blue'>
                          {t('已保存 token')} {comboConfig.remote_observer.access_token_masked}
                        </Tag>
                      ) : (
                        <Tag color='grey'>{t('未保存 token')}</Tag>
                      )}
                    </div>
                    <Banner
                      className='mt-3'
                      type='info'
                      closeIcon={null}
                      description={t('认证方式固定为远端用户自己的 access token + New-Api-User。观测成本按远端 used_quota 增量计算，但金额始终按本站额度口径换算。')}
                    />
                  </div>
                </div>
              </Collapse.Panel>
            );
          })}
        </Collapse>
      </div>
    </div>
  </Card>
);

export default PricingRulesCard;
