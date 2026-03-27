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
  <Card bordered={false} title={t('关注组合')}>
    <div className='space-y-4'>
      <div className='grid gap-3 md:grid-cols-[150px_1fr]'>
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
        >
          <Radio value='channel'>{t('渠道')}</Radio>
          <Radio value='tag'>{t('标签聚合渠道')}</Radio>
        </Radio.Group>
        <Input
          value={draft.name}
          onChange={(value) => setDraft((prev) => ({ ...prev, name: value }))}
          placeholder={t('组合名称，例如：OpenAI 主力组合')}
        />
      </div>
      <Select
        multiple
        filter
        maxTagCount={isMobile ? 2 : 4}
        optionList={
          draft.scope_type === 'channel'
            ? channelOptions
            : (options.tags || []).map((item) => ({ label: item, value: item }))
        }
        value={draft.scope_type === 'channel' ? draft.channel_ids || [] : draft.tags || []}
        onChange={(value) =>
          draft.scope_type === 'channel'
            ? setDraft((prev) => ({ ...prev, channel_ids: value || [] }))
            : setDraft((prev) => ({ ...prev, tags: value || [] }))
        }
        placeholder={draft.scope_type === 'channel' ? t('选择一个或多个渠道') : t('选择一个或多个标签')}
        style={{ width: '100%' }}
      />
      <div className='flex flex-wrap items-center gap-2'>
        <Button icon={<Plus size={16} />} onClick={addOrUpdateBatch}>
          {editingBatchId ? t('保存修改') : t('添加组合')}
        </Button>
        {editingBatchId ? (
          <Button type='tertiary' onClick={resetDraft}>
            {t('取消编辑')}
          </Button>
        ) : null}
      </div>
      <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0/60 p-3'>
        <div className='mb-2 flex items-center gap-2'>
          <Layers3 size={15} />
          <Text strong>{t('当前长期关注的组合')}</Text>
        </div>
        {batches.length > 0 ? (
          <div className='space-y-2'>
            {batches.map((batch) => (
              <div
                key={batch.id}
                className='flex flex-col gap-2 rounded-lg border border-semi-color-border bg-semi-color-bg-2 p-3 lg:flex-row lg:items-center lg:justify-between'
              >
                <div>
                  <Space wrap>
                    <Text strong>{batch.name}</Text>
                    <Tag color={batch.scope_type === 'channel' ? 'blue' : 'cyan'}>
                      {batch.scope_type === 'channel' ? t('渠道') : t('标签聚合渠道')}
                    </Tag>
                  </Space>
                  <Text type='tertiary' className='mt-1 block'>
                    {batchDigest(batch)}
                  </Text>
                </div>
                <Space>
                  <Button
                    icon={<Pencil size={14} />}
                    size='small'
                    type='tertiary'
                    onClick={() => editBatch(batch)}
                  >
                    {t('编辑')}
                  </Button>
                  <Button
                    icon={<Trash2 size={14} />}
                    size='small'
                    type='danger'
                    onClick={() => removeBatch(batch.id)}
                  >
                    {t('删除')}
                  </Button>
                </Space>
              </div>
            ))}
          </div>
        ) : (
          <Empty image={null} description={t('还没有组合，先添加一组渠道或标签')} />
        )}
      </div>
      {batchValidationError ? (
        <Banner type='danger' description={batchValidationError} closeIcon={null} />
      ) : (
        <Text type='tertiary'>
          {t('这里决定你长期盯哪些渠道；顶部累计总览会一直按这些组合统计。')}
        </Text>
      )}
    </div>
  </Card>
);

export default ComboManagerCard;
