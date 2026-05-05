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
import {
  Banner,
  Button,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconEdit,
  IconPlus,
  IconRefresh,
  IconSearch,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../helpers';

const { Text } = Typography;

const NAME_RULE_OPTIONS = [
  { label: '精确匹配', value: 0 },
  { label: '前缀匹配', value: 1 },
  { label: '包含匹配', value: 2 },
  { label: '后缀匹配', value: 3 },
];

const VISIBILITY_SCOPE_OPTIONS = [
  { label: '全部可见', value: 0, color: 'green' },
  { label: '仅管理员可见', value: 1, color: 'blue' },
  { label: '仅普通用户和访客可见', value: 2, color: 'cyan' },
  { label: '都不可见', value: 3, color: 'red' },
];

const CALL_SCOPE_OPTIONS = [
  { label: '全部可调用', value: 0, color: 'green' },
  { label: '仅管理员可调用', value: 1, color: 'blue' },
  { label: '仅普通用户可调用', value: 2, color: 'cyan' },
  { label: '都不可调用', value: 3, color: 'red' },
];

const EMPTY_FORM = {
  model_name: '',
  name_rule: 0,
  visibility_scope: 0,
  call_scope: 0,
};

function optionLabel(options, value, t) {
  const option = options.find((item) => item.value === Number(value));
  return option ? t(option.label) : t(options[0].label);
}

function scopeTag(options, value, t) {
  const option = options.find((item) => item.value === Number(value)) || options[0];
  return (
    <Tag color={option.color} shape='circle'>
      {t(option.label)}
    </Tag>
  );
}

export default function ModelPermissionSettings() {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [candidates, setCandidates] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [form, setForm] = useState(EMPTY_FORM);

  const candidateOptions = useMemo(
    () => candidates.map((name) => ({ label: name, value: name })),
    [candidates],
  );

  const filteredItems = useMemo(() => {
    const q = keyword.trim().toLowerCase();
    if (!q) return items;
    return items.filter((item) =>
      String(item.model_name || '').toLowerCase().includes(q),
    );
  }, [items, keyword]);

  const loadPermissions = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/model_permissions');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('获取模型权限配置失败'));
        return;
      }
      setItems(data?.items || []);
      setCandidates(data?.candidates || []);
    } catch (error) {
      showError(error.message || t('获取模型权限配置失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPermissions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openCreateModal = () => {
    setEditingItem(null);
    setForm(EMPTY_FORM);
    setModalOpen(true);
  };

  const openEditModal = (record) => {
    setEditingItem(record);
    setForm({
      model_name: record.model_name || '',
      name_rule: Number(record.name_rule || 0),
      visibility_scope: Number(record.visibility_scope || 0),
      call_scope: Number(record.call_scope || 0),
    });
    setModalOpen(true);
  };

  const savePermission = async () => {
    const payload = {
      ...form,
      model_name: form.model_name.trim(),
      name_rule: Number(form.name_rule || 0),
      visibility_scope: Number(form.visibility_scope || 0),
      call_scope: Number(form.call_scope || 0),
    };
    if (!payload.model_name) {
      showError(t('模型名称不能为空'));
      return;
    }

    setSaving(true);
    try {
      const res = editingItem
        ? await API.put(`/api/option/model_permissions/${editingItem.id}`, payload)
        : await API.post('/api/option/model_permissions', payload);
      if (!res.data.success) {
        showError(res.data.message || t('保存失败'));
        return;
      }
      showSuccess(t('保存成功'));
      setModalOpen(false);
      await loadPermissions();
    } catch (error) {
      showError(error.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const deletePermission = async (record) => {
    try {
      const res = await API.delete(`/api/option/model_permissions/${record.id}`);
      if (!res.data.success) {
        showError(res.data.message || t('删除失败'));
        return;
      }
      showSuccess(t('删除成功'));
      await loadPermissions();
    } catch (error) {
      showError(error.message || t('删除失败'));
    }
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'model_name',
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: t('命名规则'),
      dataIndex: 'name_rule',
      width: 120,
      render: (value) => (
        <Tag shape='circle'>{optionLabel(NAME_RULE_OPTIONS, value, t)}</Tag>
      ),
    },
    {
      title: t('展示权限'),
      dataIndex: 'visibility_scope',
      width: 180,
      render: (value) => scopeTag(VISIBILITY_SCOPE_OPTIONS, value, t),
    },
    {
      title: t('调用权限'),
      dataIndex: 'call_scope',
      width: 170,
      render: (value) => scopeTag(CALL_SCOPE_OPTIONS, value, t),
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      width: 170,
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('操作'),
      width: 120,
      render: (_, record) => (
        <Space>
          <Button
            icon={<IconEdit />}
            size='small'
            theme='borderless'
            onClick={() => openEditModal(record)}
          />
          <Popconfirm
            title={t('确定要删除此模型权限配置吗？')}
            onConfirm={() => deletePermission(record)}
          >
            <Button
              icon={<IconDelete />}
              size='small'
              theme='borderless'
              type='danger'
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Banner
        type='info'
        description={t(
          '展示权限只影响用户侧模型列表、模型广场和令牌模型选择；调用权限会在实际 API 调用前拦截。两者互相独立，未配置的模型默认全部可见、全部可调用。',
        )}
        style={{ marginBottom: 16 }}
      />

      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          gap: 12,
          flexWrap: 'wrap',
          marginBottom: 12,
        }}
      >
        <Input
          prefix={<IconSearch />}
          value={keyword}
          onChange={setKeyword}
          placeholder={t('搜索模型名称')}
          style={{ width: 280 }}
        />
        <Space>
          <Button icon={<IconRefresh />} onClick={loadPermissions}>
            {t('刷新')}
          </Button>
          <Button icon={<IconPlus />} theme='solid' onClick={openCreateModal}>
            {t('添加权限')}
          </Button>
        </Space>
      </div>

      <Table
        rowKey='id'
        loading={loading}
        columns={columns}
        dataSource={filteredItems}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title={editingItem ? t('编辑模型权限') : t('添加模型权限')}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={savePermission}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={saving}
      >
        <Space vertical align='start' spacing={12} style={{ width: '100%' }}>
          <div style={{ width: '100%' }}>
            <Text strong>{t('模型名称')}</Text>
            <Select
              value={form.model_name}
              filter
              showClear
              allowCreate
              placeholder={t('可选择已有模型，也可以直接输入自定义模型名')}
              optionList={candidateOptions}
              style={{ width: '100%', marginTop: 6 }}
              onChange={(value) =>
                setForm({ ...form, model_name: String(value || '') })
              }
            />
          </div>

          <div style={{ width: '100%' }}>
            <Text strong>{t('命名规则')}</Text>
            <Select
              value={form.name_rule}
              optionList={NAME_RULE_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              style={{ width: '100%', marginTop: 6 }}
              onChange={(value) => setForm({ ...form, name_rule: Number(value) })}
            />
          </div>

          <div style={{ width: '100%' }}>
            <Text strong>{t('展示权限')}</Text>
            <Select
              value={form.visibility_scope}
              optionList={VISIBILITY_SCOPE_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              style={{ width: '100%', marginTop: 6 }}
              onChange={(value) =>
                setForm({ ...form, visibility_scope: Number(value) })
              }
            />
          </div>

          <div style={{ width: '100%' }}>
            <Text strong>{t('调用权限')}</Text>
            <Select
              value={form.call_scope}
              optionList={CALL_SCOPE_OPTIONS.map((item) => ({
                ...item,
                label: t(item.label),
              }))}
              style={{ width: '100%', marginTop: 6 }}
              onChange={(value) =>
                setForm({ ...form, call_scope: Number(value) })
              }
            />
          </div>
        </Space>
      </Modal>
    </div>
  );
}
