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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Input, Modal, Space, Table, Tag } from '@douyinfe/semi-ui';
import { API, renderQuota, showError, showSuccess } from '../../../../helpers';

const RedemptionReviewModal = ({ visible, onCancel, onResolved, t }) => {
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [notes, setNotes] = useState({});
  const [resolvingId, setResolvingId] = useState(0);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 20;

  const loadItems = useCallback(async () => {
    if (!visible) return;
    setLoading(true);
    try {
      const response = await API.get(
        `/api/redemption/review-cases?status=pending&p=${page}&page_size=${pageSize}`,
      );
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('加载失败'));
      }
      setItems(response.data?.data?.items || []);
      setTotal(Number(response.data?.data?.total || 0));
    } catch (error) {
      showError(error?.message || t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [visible, page, t]);

  useEffect(() => {
    if (visible) setPage(1);
  }, [visible]);

  useEffect(() => {
    loadItems();
  }, [loadItems]);

  const resolve = async (record, action) => {
    const note = String(notes[record.id] || '').trim();
    if (action === 'disable' && !note) {
      showError(t('封禁用户必须填写人工复查依据'));
      return;
    }
    setResolvingId(record.id);
    try {
      const response = await API.post(
        `/api/redemption/review-cases/${record.id}/resolve`,
        { action, note },
      );
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('操作失败'));
      }
      setItems((current) => current.filter((item) => item.id !== record.id));
      setTotal((current) => Math.max(0, current - 1));
      onResolved?.();
      if (items.length === 1 && page > 1) {
        setPage((current) => current - 1);
      }
      showSuccess(action === 'disable' ? t('用户已停用') : t('复查记录已忽略'));
    } catch (error) {
      showError(error?.message || t('操作失败'));
    } finally {
      setResolvingId(0);
    }
  };

  const confirmDisable = (record) => {
    const note = String(notes[record.id] || '').trim();
    if (!note) {
      showError(t('封禁用户必须填写人工复查依据'));
      return;
    }
    Modal.confirm({
      title: t('确认封禁该用户？'),
      content: t('该操作会立即停用账户，请确认已核对创建人和兑换码记录。'),
      okType: 'danger',
      onOk: () => resolve(record, 'disable'),
    });
  };

  const columns = useMemo(
    () => [
      { title: t('用户 ID'), dataIndex: 'user_id' },
      { title: t('日期'), dataIndex: 'biz_date' },
      {
        title: t('不同创建人数'),
        dataIndex: 'distinct_creator_count',
      },
      { title: t('小额兑换码数'), dataIndex: 'small_code_count' },
      {
        title: t('合计额度'),
        dataIndex: 'total_quota',
        render: (value) => renderQuota(Number(value || 0)),
      },
      {
        title: t('创建人 ID'),
        dataIndex: 'creator_ids',
        render: (value) => value || '-',
      },
      {
        title: t('兑换码 ID'),
        dataIndex: 'redemption_ids',
        render: (value) => value || '-',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: () => <Tag color='orange'>{t('待人工复查')}</Tag>,
      },
      {
        title: t('复查依据'),
        dataIndex: 'review_note',
        width: 220,
        render: (_, record) => (
          <Input
            value={notes[record.id] || ''}
            placeholder={t('封禁前填写依据')}
            onChange={(value) =>
              setNotes((current) => ({ ...current, [record.id]: value }))
            }
          />
        ),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        fixed: 'right',
        width: 170,
        render: (_, record) => (
          <Space>
            <Button
              size='small'
              loading={resolvingId === record.id}
              onClick={() => resolve(record, 'dismiss')}
            >
              {t('忽略')}
            </Button>
            <Button
              type='danger'
              size='small'
              loading={resolvingId === record.id}
              onClick={() => confirmDisable(record)}
            >
              {t('封禁用户')}
            </Button>
          </Space>
        ),
      },
    ],
    [notes, resolvingId, t, items.length, page],
  );

  return (
    <Modal
      title={t('兑换风险人工复查')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width='94vw'
      bodyStyle={{ maxHeight: '75vh', overflow: 'auto' }}
    >
      <Table
        rowKey='id'
        columns={columns}
        dataSource={items}
        loading={loading}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          onPageChange: setPage,
        }}
        scroll={{ x: 'max-content' }}
        empty={t('暂无待复查记录')}
      />
    </Modal>
  );
};

export default RedemptionReviewModal;
