import React from 'react';
import { Button, InputNumber, Select, Tag, Typography } from '@douyinfe/semi-ui';
import { Plus, Trash2 } from 'lucide-react';

const { Text } = Typography;

const PriceInput = ({ label, value, onChange, clampNumber }) => (
  <div>
    <Text type='tertiary' size='small' className='mb-1 block'>
      {label}
    </Text>
    <InputNumber
      min={0}
      value={value}
      onChange={(nextValue) => onChange(clampNumber(nextValue))}
      suffix='USD/1M'
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
    <div className='space-y-1'>
      <Text strong>{title}</Text>
      {description ? (
        <Text type='tertiary' size='small' className='block'>
          {description}
        </Text>
      ) : null}
    </div>

    {(rules || []).map((rule, index) => (
      <div
        key={`${comboId}-${field}-${index}`}
        className='rounded-2xl border border-semi-color-border bg-semi-color-fill-0 p-3'
      >
        <div className='grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto_auto]'>
          <div>
            <Text type='tertiary' size='small' className='mb-1 block'>
              {field === 'site_rules' ? t('模型') : t('上游模型')}
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
                { label: t('默认规则'), value: '__default__' },
                ...modelNameOptions,
              ]}
              placeholder={
                field === 'site_rules'
                  ? t('选择本站模型或输入自定义')
                  : t('选择或输入上游模型名')
              }
              style={{ width: '100%' }}
            />
          </div>
          <Button
            type={rule.is_default ? 'primary' : 'tertiary'}
            theme={rule.is_default ? 'solid' : 'borderless'}
            onClick={() =>
              onUpdate(comboId, field, index, {
                is_default: !rule.is_default,
                model_name: rule.is_default ? rule.model_name : '',
              })
            }
          >
            {rule.is_default ? t('默认规则') : t('设为默认')}
          </Button>
          <Button
            type='danger'
            theme='light'
            icon={<Trash2 size={14} />}
            onClick={() => onRemove(comboId, field, index)}
          >
            {t('删除')}
          </Button>
        </div>

        <div className='mt-3 flex flex-wrap gap-2'>
          {rule.is_default ? <Tag color='blue'>{t('默认兜底')}</Tag> : null}
          {rule.is_custom ? <Tag color='orange'>{t('自定义模型名')}</Tag> : null}
        </div>

        <div className='mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
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
      type='tertiary'
      icon={<Plus size={14} />}
      onClick={() => onAdd(comboId, field)}
    >
      {field === 'site_rules' ? t('新增本站模型规则') : t('新增上游模型规则')}
    </Button>
  </div>
);

export default PricingRuleList;
