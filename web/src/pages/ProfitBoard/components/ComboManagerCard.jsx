import React from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Layers3, Pencil, Plus, Trash2 } from 'lucide-react';

const { Text } = Typography;

const SummaryBlock = ({ label, value }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-3'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <div className='mt-1 text-sm font-medium'>{value}</div>
  </div>
);

const ComboManagerCard = ({
  batches,
  batchDigest,
  resolveComboConfig,
  getSiteSummary,
  getUpstreamSummary,
  batchValidationError,
  isMobile,
  onCreateBatch,
  onEditBatch,
  onRemoveBatch,
  t,
}) => (
  <Card
    bordered={false}
    title={t('渠道组合')}
    headerExtraContent={
      <Button
        icon={<Plus size={14} />}
        size='small'
        theme='solid'
        type='primary'
        onClick={onCreateBatch}
      >
        {t('新建组合')}
      </Button>
    }
  >
    <div className='space-y-4'>
      {batches.length > 0 ? (
        <>
          <div className='flex items-center gap-2 px-1'>
            <Layers3 size={14} className='text-semi-color-text-2' />
            <Text type='tertiary' size='small'>
              {t('已添加 {{count}} 个组合', { count: batches.length })}
            </Text>
          </div>
          <div className='space-y-3'>
            {batches.map((batch) => {
              const comboConfig = resolveComboConfig(batch.id);
              return (
                <div
                  key={batch.id}
                  className='rounded-2xl border border-semi-color-border bg-semi-color-bg-2 p-4'
                >
                  <div className='flex flex-wrap items-start justify-between gap-3'>
                    <div className='min-w-0 flex-1'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <Text strong className='truncate text-base'>
                          {batch.name}
                        </Text>
                        <Tag
                          color={
                            batch.scope_type === 'channel' ? 'blue' : 'cyan'
                          }
                          size='small'
                        >
                          {batch.scope_type === 'channel'
                            ? t('渠道')
                            : t('标签')}
                        </Tag>
                      </div>
                      <Text
                        type='tertiary'
                        size='small'
                        className='mt-1.5 block'
                      >
                        {batchDigest(batch)}
                      </Text>
                      <div className='mt-3 grid gap-3 md:grid-cols-2'>
                        <SummaryBlock
                          label={t('收入')}
                          value={getSiteSummary(comboConfig)}
                        />
                        <SummaryBlock
                          label={t('成本')}
                          value={getUpstreamSummary(comboConfig)}
                        />
                      </div>
                    </div>

                    <Space className='shrink-0'>
                      <Button
                        icon={<Pencil size={14} />}
                        size='small'
                        type='tertiary'
                        onClick={() => onEditBatch(batch)}
                      >
                        {isMobile ? null : t('编辑')}
                      </Button>
                      <Button
                        icon={<Trash2 size={14} />}
                        size='small'
                        type='danger'
                        theme='borderless'
                        onClick={() => onRemoveBatch(batch)}
                      >
                        {isMobile ? null : t('删除')}
                      </Button>
                    </Space>
                  </div>
                </div>
              );
            })}
          </div>
        </>
      ) : (
        <Empty image={null} description={t('还没有组合')}>
          <Button
            icon={<Plus size={14} />}
            theme='solid'
            type='primary'
            onClick={onCreateBatch}
          >
            {t('新建组合')}
          </Button>
        </Empty>
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
