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
import { Button, Card, Empty, Tag, Typography } from '@douyinfe/semi-ui';
import { Pencil } from 'lucide-react';
import PricingConfigModal from './PricingConfigModal';
import { getUpstreamCostSourceLabel } from '../utils';

const { Text, Title } = Typography;

const getSiteSummary = (comboConfig, t) => {
  if (comboConfig.site_mode !== 'shared_site_model') {
    return t('手动定价');
  }
  const modelCount = comboConfig.shared_site?.model_names?.length || 0;
  if (modelCount === 0) return t('本站模型价格');
  return t('本站模型价格 · {{count}} 个模型', { count: modelCount });
};

const getUpstreamSummary = (comboConfig, options, t) => {
  if (comboConfig.upstream_mode !== 'wallet_observer') {
    return getUpstreamCostSourceLabel('manual_only', t);
  }
  const account = (options?.upstream_accounts || []).find(
    (item) => item.id === Number(comboConfig.upstream_account_id || 0),
  );
  return account
    ? t('按钱包余额变化 · {{name}}', { name: account.name })
    : t('按钱包余额变化');
};

const normalizeComboConfig = (
  batchId,
  comboConfigs,
  siteConfig,
  upstreamConfig,
  createDefaultComboPricingConfig,
) =>
  comboConfigs.find((item) => item.combo_id === batchId) ||
  createDefaultComboPricingConfig(batchId, undefined, siteConfig, upstreamConfig);

const PricingRulesCard = ({
  batches,
  comboConfigs,
  siteConfig,
  modelNameOptions,
  options,
  resolveSharedSitePreview,
  upstreamConfig,
  isMobile,
  createDefaultComboPricingConfig,
  updateComboConfig,
  localModelMap,
  clampNumber,
  t,
}) => {
  const [editingComboId, setEditingComboId] = useState('');
  const [draftConfig, setDraftConfig] = useState(null);

  const editingBatch = useMemo(
    () => batches.find((item) => item.id === editingComboId) || null,
    [batches, editingComboId],
  );

  useEffect(() => {
    if (!editingComboId) return;
    const nextConfig = normalizeComboConfig(
      editingComboId,
      comboConfigs,
      siteConfig,
      upstreamConfig,
      createDefaultComboPricingConfig,
    );
    setDraftConfig({
      ...nextConfig,
      shared_site: { ...(nextConfig.shared_site || {}) },
      site_rules: [...(nextConfig.site_rules || [])],
      upstream_rules: [...(nextConfig.upstream_rules || [])],
      remote_observer: { ...(nextConfig.remote_observer || {}) },
    });
  }, [
    comboConfigs,
    createDefaultComboPricingConfig,
    editingComboId,
    siteConfig,
    upstreamConfig,
  ]);

  const openEditor = (batchId) => {
    const nextConfig = normalizeComboConfig(
      batchId,
      comboConfigs,
      siteConfig,
      upstreamConfig,
      createDefaultComboPricingConfig,
    );
    setDraftConfig({
      ...nextConfig,
      shared_site: { ...(nextConfig.shared_site || {}) },
      site_rules: [...(nextConfig.site_rules || [])],
      upstream_rules: [...(nextConfig.upstream_rules || [])],
      remote_observer: { ...(nextConfig.remote_observer || {}) },
    });
    setEditingComboId(batchId);
  };

  const closeEditor = () => {
    setEditingComboId('');
    setDraftConfig(null);
  };

  const saveDraft = () => {
    if (!editingComboId || !draftConfig) return;
    updateComboConfig(editingComboId, draftConfig);
    closeEditor();
  };

  return (
    <>
      <Card
        bordered={false}
        className='rounded-xl'
        title={
          <div>
            <Title heading={6} style={{ margin: 0 }}>
              {t('定价设置')}
            </Title>
            <Text type='tertiary' size='small'>
              {t('按组合分别配置收入和成本')}
            </Text>
          </div>
        }
      >
        {batches.length === 0 ? (
          <Empty image={null} description={t('先在上方创建组合')} />
        ) : (
          <div className='space-y-3'>
            {batches.map((batch) => {
              const comboConfig = normalizeComboConfig(
                batch.id,
                comboConfigs,
                siteConfig,
                upstreamConfig,
                createDefaultComboPricingConfig,
              );

              return (
                <div
                  key={batch.id}
                  className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'
                >
                  <div className='flex flex-wrap items-start justify-between gap-3'>
                    <div className='min-w-0 flex-1'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <Title heading={6} style={{ margin: 0 }}>
                          {batch.name}
                        </Title>
                        <Tag
                          color={
                            batch.scope_type === 'channel' ? 'blue' : 'cyan'
                          }
                          size='small'
                        >
                          {batch.scope_type === 'channel' ? t('渠道') : t('标签')}
                        </Tag>
                      </div>
                      <div className='mt-3 grid gap-3 md:grid-cols-2'>
                        <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-3'>
                          <Text type='tertiary' size='small'>
                            {t('收入')}
                          </Text>
                          <div className='mt-1 text-sm font-medium'>
                            {getSiteSummary(comboConfig, t)}
                          </div>
                        </div>
                        <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-3'>
                          <Text type='tertiary' size='small'>
                            {t('成本')}
                          </Text>
                          <div className='mt-1 text-sm font-medium'>
                            {getUpstreamSummary(comboConfig, options, t)}
                          </div>
                        </div>
                      </div>
                    </div>

                    <Button
                      theme='solid'
                      type='tertiary'
                      icon={<Pencil size={14} />}
                      onClick={() => openEditor(batch.id)}
                    >
                      {t('编辑')}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Card>

      <PricingConfigModal
        visible={!!editingComboId}
        batchName={editingBatch?.name || t('组合配置')}
        comboConfig={draftConfig}
        setComboConfig={setDraftConfig}
        modelNameOptions={modelNameOptions}
        options={options}
        resolveSharedSitePreview={resolveSharedSitePreview}
        isMobile={isMobile}
        clampNumber={clampNumber}
        localModelMap={localModelMap}
        onOk={saveDraft}
        onCancel={closeEditor}
        t={t}
      />
    </>
  );
};

export default PricingRulesCard;
