import React from 'react';
import { Button, InputNumber, Select } from '@douyinfe/semi-ui';
import { Plus } from 'lucide-react';

const PricingRuleList = ({
  comboId,
  field,
  rules,
  modelNameOptions,
  localModelMap,
  clampNumber,
  onUpdate,
  onRemove,
  onAdd,
  t,
}) => (
  <div className='space-y-2'>
    {(rules || []).map((rule, index) => (
      <div
        key={`${comboId}-${field}-${index}`}
        className='grid gap-2 rounded-xl border border-semi-color-border bg-semi-color-fill-0 p-3 lg:grid-cols-[1.3fr_repeat(4,160px)_auto_auto]'
      >
        <Select
          allowCreate
          filter
          value={rule.is_default ? '__default__' : rule.model_name}
          onChange={(value) =>
            onUpdate(comboId, field, index, {
              model_name: value === '__default__' ? '' : value,
              is_default: value === '__default__',
              is_custom: value !== '__default__' && !localModelMap.has(value),
            })
          }
          optionList={[
            { label: t('默认规则'), value: '__default__' },
            ...modelNameOptions,
          ]}
          placeholder={t('选择或输入模型')}
        />
        <InputNumber
          min={0}
          value={rule.input_price}
          onChange={(value) => onUpdate(comboId, field, index, { input_price: clampNumber(value) })}
          suffix='输入'
        />
        <InputNumber
          min={0}
          value={rule.output_price}
          onChange={(value) => onUpdate(comboId, field, index, { output_price: clampNumber(value) })}
          suffix='输出'
        />
        <InputNumber
          min={0}
          value={rule.cache_read_price}
          onChange={(value) =>
            onUpdate(comboId, field, index, { cache_read_price: clampNumber(value) })
          }
          suffix='缓存读'
        />
        <InputNumber
          min={0}
          value={rule.cache_creation_price}
          onChange={(value) =>
            onUpdate(comboId, field, index, { cache_creation_price: clampNumber(value) })
          }
          suffix='缓存写'
        />
        <Button
          type={rule.is_default ? 'primary' : 'tertiary'}
          onClick={() =>
            onUpdate(comboId, field, index, {
              is_default: !rule.is_default,
              model_name: rule.is_default ? rule.model_name : '',
            })
          }
        >
          {rule.is_default ? t('默认中') : t('设默认')}
        </Button>
        <Button type='danger' onClick={() => onRemove(comboId, field, index)}>
          {t('删除')}
        </Button>
      </div>
    ))}
    <Button type='tertiary' icon={<Plus size={14} />} onClick={() => onAdd(comboId, field)}>
      {field === 'site_rules' ? t('新增本站模型规则') : t('新增上游模型规则')}
    </Button>
  </div>
);

export default PricingRuleList;
