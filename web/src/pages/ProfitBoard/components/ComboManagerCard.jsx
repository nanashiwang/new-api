import React from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Input,
  Radio,
  Select,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Layers3, Pencil, Plus, Trash2 } from 'lucide-react';

const { Text } = Typography;

const ComboManagerCard = ({
  draft,
  setDraft,
  channelOptions,
  options,
  isMobile,
  addOrUpdateBatch,
  editingBatchId,
  resetDraft,
  batches,
  batchDigest,
  editBatch,
  removeBatch,
  batchValidationError,
  t,
}) => (
  <Card bordered={false} title={t('渠道组合')}>
    <div className='space-y-4'>
      {/* 创建/编辑区域 */}
      <div className='rounded-xl border border-dashed border-semi-color-border bg-semi-color-fill-0 p-4'>
        <Text type='tertiary' size='small' className='mb-3 block'>
          {editingBatchId ? t('编辑组合') : t('新建组合')}
        </Text>
        <div className='flex flex-col gap-3'>
          <div className='flex flex-wrap items-center gap-3'>
            <Radio.Group
              type='button'
              value={draft.scope_type}
              onChange={(event) =>
                setDraft((prev) => ({
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
            <Input
              value={draft.name}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, name: value }))
              }
              placeholder={t('组合名称，例如：OpenAI 主力')}
              size='small'
              style={{ flex: 1, minWidth: 160 }}
            />
          </div>
          <Select
            multiple
            filter
            maxTagCount={isMobile ? 2 : 4}
            optionList={
              draft.scope_type === 'channel'
                ? channelOptions
                : (options.tags || []).map((item) => ({
                    label: item,
                    value: item,
                  }))
            }
            value={
              draft.scope_type === 'channel'
                ? draft.channel_ids || []
                : draft.tags || []
            }
            onChange={(value) =>
              draft.scope_type === 'channel'
                ? setDraft((prev) => ({ ...prev, channel_ids: value || [] }))
                : setDraft((prev) => ({ ...prev, tags: value || [] }))
            }
            placeholder={
              draft.scope_type === 'channel'
                ? t('选择渠道')
                : t('选择标签')
            }
            size='small'
            style={{ width: '100%' }}
          />
          <div className='flex items-center gap-2'>
            <Button
              icon={<Plus size={14} />}
              size='small'
              onClick={addOrUpdateBatch}
            >
              {editingBatchId ? t('保存') : t('添加')}
            </Button>
            {editingBatchId ? (
              <Button type='tertiary' size='small' onClick={resetDraft}>
                {t('取消')}
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      {/* 已有组合列表 */}
      {batches.length > 0 ? (
        <div className='space-y-2'>
          <div className='flex items-center gap-2 px-1'>
            <Layers3 size={14} className='text-semi-color-text-2' />
            <Text type='tertiary' size='small'>
              {t('已添加 {{count}} 个组合', { count: batches.length })}
            </Text>
          </div>
          {batches.map((batch) => (
            <div
              key={batch.id}
              className='flex items-center justify-between rounded-lg border border-semi-color-border bg-semi-color-bg-2 px-3 py-2.5'
            >
              <div className='min-w-0 flex-1'>
                <div className='flex items-center gap-2'>
                  <Text strong className='truncate text-sm'>
                    {batch.name}
                  </Text>
                  <Tag
                    color={batch.scope_type === 'channel' ? 'blue' : 'cyan'}
                    size='small'
                  >
                    {batch.scope_type === 'channel'
                      ? t('渠道')
                      : t('标签')}
                  </Tag>
                </div>
                <Text type='tertiary' size='small' className='mt-0.5 block truncate'>
                  {batchDigest(batch)}
                </Text>
              </div>
              <Space className='shrink-0'>
                <Button
                  icon={<Pencil size={13} />}
                  size='small'
                  type='tertiary'
                  onClick={() => editBatch(batch)}
                />
                <Button
                  icon={<Trash2 size={13} />}
                  size='small'
                  type='danger'
                  theme='borderless'
                  onClick={() => removeBatch(batch.id)}
                />
              </Space>
            </div>
          ))}
        </div>
      ) : (
        <Empty
          image={null}
          description={t('还没有组合，在上方添加')}
        />
      )}

      {batchValidationError ? (
        <Banner
          type='danger'
          description={batchValidationError}
          closeIcon={null}
        />
      ) : null}
    </div>
  </Card>
);

export default ComboManagerCard;
