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
  Button,
  Input,
  Modal,
  SideSheet,
  Space,
  Table,
  Tag,
  TextArea,
} from '@douyinfe/semi-ui';
import { IconDelete, IconSearch } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, timestamp2string } from '../../../../helpers';

const IPBlacklistModal = ({ visible, onCancel, t }) => {
  const [items, setItems] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [ip, setIp] = useState('');
  const [reason, setReason] = useState('');
  const [adding, setAdding] = useState(false);

  const loadItems = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextKeyword = keyword,
  ) => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/ip-blacklist', {
        params: {
          keyword: nextKeyword.trim(),
          p: nextPage,
          page_size: nextPageSize,
        },
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('请求失败'));
        return;
      }
      setItems(Array.isArray(data?.items) ? data.items : []);
      setTotal(Number(data?.total || 0));
      setPage(Number(data?.page || nextPage));
      setPageSize(Number(data?.page_size || nextPageSize));
    } catch (error) {
      showError(error?.message || t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  const showCurrentIPConfirm = (data, retry) => {
    Modal.confirm({
      title: t('确认拉黑当前 IP'),
      content: t(
        '规则 {{rule}} 会拉黑你当前访问 IP {{ip}}。确认后你可能无法继续访问后台。',
        {
          rule: data?.rule || '',
          ip: data?.client_ip || '',
        },
      ),
      type: 'warning',
      onOk: retry,
    });
  };

  const addItem = async (confirmCurrentIP = false) => {
    const nextIP = ip.trim();
    if (!nextIP) {
      showError(t('请输入 IP 或 CIDR'));
      return;
    }
    setAdding(true);
    try {
      const res = await API.post('/api/user/ip-blacklist', {
        ip: nextIP,
        reason,
        confirm_current_ip: confirmCurrentIP,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        if (data?.code === 'current_ip_requires_confirmation' && !confirmCurrentIP) {
          showCurrentIPConfirm(data, () => addItem(true));
          return;
        }
        showError(message || t('操作失败，请重试'));
        return;
      }
      if (data?.created === false) {
        showSuccess(t('该 IP 已在黑名单中'));
      } else {
        showSuccess(t('已添加 IP 黑名单'));
      }
      setIp('');
      setReason('');
      await loadItems(1, pageSize);
    } catch (error) {
      showError(error?.message || t('操作失败，请重试'));
    } finally {
      setAdding(false);
    }
  };

  const deleteItem = (item) => {
    Modal.confirm({
      title: t('解除 IP 拉黑'),
      content: `${t('确定要解除拉黑吗？')} ${item.cidr}`,
      type: 'warning',
      onOk: async () => {
        try {
          const res = await API.delete(`/api/user/ip-blacklist/${item.id}`);
          const { success, message } = res.data || {};
          if (!success) {
            showError(message || t('操作失败，请重试'));
            return;
          }
          showSuccess(t('已解除 IP 拉黑'));
          await loadItems(page, pageSize);
        } catch (error) {
          showError(error?.message || t('操作失败，请重试'));
        }
      },
    });
  };

  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 80,
      },
      {
        title: t('IP / CIDR'),
        dataIndex: 'cidr',
        render: (text, record) => (
          <Space spacing={4}>
            <Tag color={record.ip_version === 6 ? 'light-blue' : 'blue'} shape='circle'>
              IPv{record.ip_version}
            </Tag>
            <span className='font-mono'>{text}</span>
          </Space>
        ),
      },
      {
        title: t('原因'),
        dataIndex: 'reason',
        render: (text) => text || '-',
      },
      {
        title: t('来源用户'),
        dataIndex: 'source_user_id',
        width: 100,
        render: (text) => (text ? `#${text}` : '-'),
      },
      {
        title: t('创建管理员'),
        dataIndex: 'created_by',
        width: 110,
        render: (text) => (text ? `#${text}` : '-'),
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_at',
        width: 170,
        render: (text) => (text ? timestamp2string(text) : '-'),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        width: 90,
        render: (_, record) => (
          <Button
            type='danger'
            theme='borderless'
            size='small'
            icon={<IconDelete />}
            onClick={() => deleteItem(record)}
          />
        ),
      },
    ],
    [t, page, pageSize],
  );

  useEffect(() => {
    if (visible) {
      loadItems(1, pageSize);
    }
  }, [visible]);

  return (
    <SideSheet
      visible={visible}
      onCancel={onCancel}
      title={t('IP 黑名单')}
      placement='right'
      width={860}
      bodyStyle={{ padding: 16 }}
    >
      <div className='flex flex-col gap-3'>
        <div className='grid grid-cols-1 md:grid-cols-[minmax(220px,1fr)_minmax(260px,1fr)_auto] gap-2'>
          <Input
            value={ip}
            onChange={setIp}
            placeholder={t('IP 或 CIDR，例如 1.2.3.4 或 2001:db8::/64')}
            showClear
          />
          <TextArea
            value={reason}
            onChange={setReason}
            placeholder={t('原因')}
            autosize={{ minRows: 1, maxRows: 2 }}
          />
          <Button
            type='primary'
            theme='solid'
            loading={adding}
            onClick={() => addItem(false)}
          >
            {t('添加')}
          </Button>
        </div>

        <div className='flex flex-col md:flex-row gap-2 md:items-center md:justify-between'>
          <Input
            prefix={<IconSearch />}
            value={keyword}
            onChange={setKeyword}
            onEnterPress={() => loadItems(1, pageSize)}
            placeholder={t('搜索 IP、CIDR 或原因')}
            showClear
            className='w-full md:w-80'
          />
          <Space>
            <Button type='tertiary' onClick={() => loadItems(1, pageSize)}>
              {t('查询')}
            </Button>
            <Button
              type='tertiary'
              onClick={() => {
                setKeyword('');
                loadItems(1, pageSize, '');
              }}
            >
              {t('重置')}
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={items}
          loading={loading}
          rowKey='id'
          pagination={{
            currentPage: page,
            pageSize,
            total,
            pageSizeOpts: [10, 20, 50, 100],
            showSizeChanger: true,
            onPageChange: (nextPage) => loadItems(nextPage, pageSize),
            onPageSizeChange: (nextPageSize) => loadItems(1, nextPageSize),
          }}
        />
      </div>
    </SideSheet>
  );
};

export default IPBlacklistModal;
