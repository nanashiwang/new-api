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
import React, {
  useState,
  useCallback,
  useMemo,
  useEffect,
  useRef,
} from 'react';
import { Button, Select, Typography, Popconfirm, Tag } from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconDelete,
  IconChevronUp,
  IconChevronDown,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { GripVertical } from 'lucide-react';
import { moveGroup } from '../../../../helpers/groupOrdering';

const { Text } = Typography;

let _idCounter = 0;
const uid = () => `ag_${++_idCounter}`;

function parseAutoGroups(str) {
  if (!str || !str.trim()) return [];
  try {
    const parsed = JSON.parse(str);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((item) => typeof item === 'string')
      .map((name) => ({ _id: uid(), name }));
  } catch {
    return [];
  }
}

function serializeAutoGroups(items) {
  const names = items.map((i) => i.name).filter(Boolean);
  return names.length === 0 ? '' : JSON.stringify(names);
}

export default function AutoGroupList({ value, groupNames = [], onChange }) {
  const { t } = useTranslation();

  const [items, setItems] = useState(() => parseAutoGroups(value));
  const lastValueRef = useRef(value);
  const [draggedId, setDraggedId] = useState(null);
  const [dropTargetId, setDropTargetId] = useState(null);

  useEffect(() => {
    if (value !== lastValueRef.current) {
      lastValueRef.current = value;
      setItems(parseAutoGroups(value));
      setDraggedId(null);
      setDropTargetId(null);
    }
  }, [value]);

  const emitChange = useCallback(
    (newItems) => {
      setItems(newItems);
      const serialized = serializeAutoGroups(newItems);
      lastValueRef.current = serialized;
      onChange?.(serialized);
    },
    [onChange],
  );

  const groupOptions = useMemo(
    () => groupNames.map((n) => ({ value: n, label: n })),
    [groupNames],
  );

  const addItem = useCallback(() => {
    emitChange([...items, { _id: uid(), name: '' }]);
  }, [items, emitChange]);

  const removeItem = useCallback(
    (id) => {
      emitChange(items.filter((i) => i._id !== id));
    },
    [items, emitChange],
  );

  const updateItem = useCallback(
    (id, name) => {
      emitChange(items.map((i) => (i._id === id ? { ...i, name } : i)));
    },
    [items, emitChange],
  );

  const moveUp = useCallback(
    (index) => {
      if (index <= 0) return;
      emitChange(moveGroup(items, index, index - 1));
    },
    [items, emitChange],
  );

  const moveDown = useCallback(
    (index) => {
      if (index >= items.length - 1) return;
      emitChange(moveGroup(items, index, index + 1));
    },
    [items, emitChange],
  );

  if (items.length === 0) {
    return (
      <div>
        <Text type='tertiary' className='block text-center py-4'>
          {t('暂无自动分组，点击下方按钮添加')}
        </Text>
        <div className='mt-2 flex justify-center'>
          <Button icon={<IconPlus />} theme='outline' onClick={addItem}>
            {t('添加分组')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <Text type='tertiary' size='small' className='block mb-2'>
        {t('拖动手柄调整顺序，也可使用上下按钮或方向键。')}
      </Text>
      <div className='space-y-2'>
        {items.map((item, index) => (
          <div
            key={item._id}
            className={`flex items-center gap-2 rounded ${dropTargetId === item._id ? 'ring-2 ring-[var(--semi-color-primary)]' : ''}`}
            onDragOver={(event) => {
              if (!draggedId) return;
              event.preventDefault();
              event.dataTransfer.dropEffect = 'move';
              setDropTargetId(item._id);
            }}
            onDrop={(event) => {
              if (!draggedId) return;
              event.preventDefault();
              const from = items.findIndex((entry) => entry._id === draggedId);
              emitChange(moveGroup(items, from, index));
              setDraggedId(null);
              setDropTargetId(null);
            }}
          >
            <button
              type='button'
              draggable
              className='shrink-0 cursor-grab p-1 text-[var(--semi-color-text-2)] focus-visible:ring-2'
              aria-label={`${t('调整分组顺序')} ${item.name || index + 1}`}
              onDragStart={(event) => {
                event.dataTransfer.effectAllowed = 'move';
                event.dataTransfer.setData('text/plain', item._id);
                setDraggedId(item._id);
              }}
              onDragEnd={() => {
                setDraggedId(null);
                setDropTargetId(null);
              }}
              onKeyDown={(event) => {
                if (event.key === 'ArrowUp' || event.key === 'ArrowDown') {
                  event.preventDefault();
                  if (event.key === 'ArrowUp') moveUp(index);
                  else moveDown(index);
                }
              }}
            >
              <GripVertical size={16} />
            </button>
            <Tag size='small' color='blue' className='shrink-0'>
              {index + 1}
            </Tag>
            <Select
              size='small'
              filter
              value={item.name || undefined}
              placeholder={t('选择分组')}
              optionList={groupOptions}
              onChange={(v) => updateItem(item._id, v)}
              style={{ flex: 1, minWidth: 0 }}
              allowCreate
              position='bottomLeft'
            />
            <Button
              icon={<IconChevronUp />}
              aria-label={`${t('上移分组')} ${item.name || index + 1}`}
              theme='borderless'
              size='small'
              disabled={index === 0}
              onClick={() => moveUp(index)}
            />
            <Button
              icon={<IconChevronDown />}
              aria-label={`${t('下移分组')} ${item.name || index + 1}`}
              theme='borderless'
              size='small'
              disabled={index === items.length - 1}
              onClick={() => moveDown(index)}
            />
            <Popconfirm
              title={t('确认移除？')}
              onConfirm={() => removeItem(item._id)}
              position='left'
            >
              <Button
                icon={<IconDelete />}
                aria-label={`${t('移除')} ${item.name || index + 1}`}
                type='danger'
                theme='borderless'
                size='small'
              />
            </Popconfirm>
          </div>
        ))}
      </div>
      <div className='mt-3 flex justify-center'>
        <Button icon={<IconPlus />} theme='outline' onClick={addItem}>
          {t('添加分组')}
        </Button>
      </div>
    </div>
  );
}
