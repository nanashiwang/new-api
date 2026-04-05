import React from 'react';
import { Button, InputNumber, Select, Tag, Typography } from '@douyinfe/semi-ui';
import { Plus, Trash2 } from 'lucide-react';

const { Text } = Typography;

const PriceInput = ({ label, value, onChange, clampNumber }) => (
  <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-3'>
    <Text type='tertiary' size='small' className='mb-1 block'>
      {label}
    </Text>
    <InputNumber
      min={0}
      value={value}
      onChange={(nextValue) => onChange(clampNumber(nextValue))}
      suffix='USD/1M'
      size='small'
      style={{ width: '100%' }}
    />
  </div>
);

const PricingRuleList = ({
  comboId,
  field,
  title,
  description,
  rules,
  modelNameOptions,
  localModelMap,
  clampNumber,
  onUpdate,
  onRemove,
  onAdd,
  t,
}) => (
  <div className='space-y-3'>
    <div className='space-y-0.5'>
      <Text strong size='small'>
        {title}
      </Text>
      {description ? (
        <Text type='tertiary' size='small' className='block'>
          {description}
        </Text>
      ) : null}
    </div>

    {(rules || []).map((rule, index) => (
      <div
        key={`${comboId}-${field}-${index}`}
        className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-4'
      >
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='min-w-0 flex-1'>
            <div className='mb-2 flex flex-wrap items-center gap-2'>
              <Text strong size='small'>
                {t('规则 {{index}}', { index: index + 1 })}
              </Text>
              {rule.is_default ? (
                <Tag color='blue' size='small'>
                  {t('默认')}
                </Tag>
              ) : null}
              {rule.is_custom ? (
                <Tag color='orange' size='small'>
                  {t('自定义')}
                </Tag>
              ) : null}
            </div>
            <Text type='tertiary' size='small' className='mb-1.5 block'>
              {t('模型')}
            </Text>
            <Select
              allowCreate
              filter
              showClear
              value={rule.is_default ? '__default__' : rule.model_name}
              onChange={(value) =>
                onUpdate(comboId, field, index, {
                  model_name: value === '__default__' ? '' : value,
                  is_default: value === '__default__',
                  is_custom:
                    value !== '__default__' && !localModelMap.has(value),
                })
              }
              optionList={[
                { label: t('默认（兜底）'), value: '__default__' },
                ...modelNameOptions,
              ]}
              placeholder={t('选择或输入模型名')}
              size='small'
              style={{ width: '100%' }}
            />
          </div>

          <Button
            type='danger'
            theme='borderless'
            icon={<Trash2 size={13} />}
            size='small'
            onClick={() => onRemove(comboId, field, index)}
          >
            {t('删除')}
          </Button>
        </div>

        <div className='mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
          <PriceInput
            label={t('输入')}
            value={rule.input_price}
            onChange={(value) =>
              onUpdate(comboId, field, index, { input_price: value })
            }
            clampNumber={clampNumber}
          />
          <PriceInput
            label={t('输出')}
            value={rule.output_price}
            onChange={(value) =>
              onUpdate(comboId, field, index, { output_price: value })
            }
            clampNumber={clampNumber}
          />
          <PriceInput
            label={t('缓存读')}
            value={rule.cache_read_price}
            onChange={(value) =>
              onUpdate(comboId, field, index, { cache_read_price: value })
            }
            clampNumber={clampNumber}
          />
          <PriceInput
            label={t('缓存写')}
            value={rule.cache_creation_price}
            onChange={(value) =>
              onUpdate(comboId, field, index, {
                cache_creation_price: value,
              })
            }
            clampNumber={clampNumber}
          />
        </div>
      </div>
    ))}

    <Button
      type='primary'
      theme='light'
      size='small'
      icon={<Plus size={14} />}
      onClick={() => onAdd(comboId, field)}
    >
      {t('添加规则')}
    </Button>
  </div>
);

export default PricingRuleList;
