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
  Avatar,
  Badge,
  Button,
  Collapsible,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { IconSearch, IconFilter, IconClose } from '@douyinfe/semi-icons';
import { Coins } from 'lucide-react';
import { API, stringToColor, timestamp2string } from '../../../helpers';
import { isAdmin } from '../../../helpers/utils';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const STATUS_CONFIG = {
  success: { type: 'success', key: '\u6210\u529f' },
  pending: { type: 'warning', key: '\u5f85\u652f\u4ed8' },
  expired: { type: 'danger', key: '\u5df2\u8fc7\u671f' },
};

const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  alipay: '\u652f\u4ed8\u5b9d',
  wxpay: '\u5fae\u4fe1',
};

const EMPTY_FILTERS = {
  keyword: '',
  username: '',
  status: '',
  paymentMethod: '',
};

const STATUS_OPTIONS = [
  { label: '\u5168\u90e8\u72b6\u6001', value: '' },
  { label: '\u5f85\u652f\u4ed8', value: 'pending' },
  { label: '\u6210\u529f', value: 'success' },
  { label: '\u5df2\u8fc7\u671f', value: 'expired' },
];

const PAYMENT_OPTIONS = [
  { label: '\u5168\u90e8\u652f\u4ed8\u65b9\u5f0f', value: '' },
  { label: '\u5fae\u4fe1', value: 'wxpay' },
  { label: '\u652f\u4ed8\u5b9d', value: 'alipay' },
  { label: 'Stripe', value: 'stripe' },
  { label: 'Creem', value: 'creem' },
];

const decodeUnicodeText = (value) =>
  String(value).replace(/\\u([0-9a-fA-F]{4})/g, (_, hex) =>
    String.fromCharCode(parseInt(hex, 16)),
  );

const TopupHistoryModal = ({ visible, onCancel, t }) => {
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState(EMPTY_FILTERS);
  const [showFilters, setShowFilters] = useState(false);

  const isMobile = useIsMobile();
  const userIsAdmin = useMemo(() => isAdmin(), []);
  const translate = (value) => t(decodeUnicodeText(value));

  const loadTopups = async (currentPage, currentPageSize, currentFilters) => {
    setLoading(true);
    try {
      const base = userIsAdmin ? '/api/user/topup' : '/api/user/topup/self';
      const searchParams = new URLSearchParams({
        p: String(currentPage),
        page_size: String(currentPageSize),
      });

      if (currentFilters.keyword) {
        searchParams.set('keyword', currentFilters.keyword.trim());
      }
      if (currentFilters.status) {
        searchParams.set('status', currentFilters.status);
      }
      if (currentFilters.paymentMethod) {
        searchParams.set('payment_method', currentFilters.paymentMethod);
      }
      if (userIsAdmin && currentFilters.username) {
        searchParams.set('username', currentFilters.username.trim());
      }

      const res = await API.get(`${base}?${searchParams.toString()}`);
      const { success, message, data } = res.data;
      if (success) {
        setTopups(data?.items || []);
        setTotal(data?.total || 0);
        return;
      }

      Toast.error({ content: message || translate('\u64cd\u4f5c\u5931\u8d25\uff0c\u8bf7\u91cd\u8bd5') });
    } catch (error) {
      console.error('Load topups error:', error);
      Toast.error({ content: translate('\u52a0\u8f7d\u5145\u503c\u8bb0\u5f55\u5931\u8d25') });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      loadTopups(page, pageSize, appliedFilters);
    }
  }, [visible, page, pageSize, appliedFilters, userIsAdmin]);

  const handleFilterChange = (key, value) => {
    setFilters((prev) => ({
      ...prev,
      [key]: value || '',
    }));
  };

  const applyFilters = (nextFilters = filters) => {
    setPage(1);
    setAppliedFilters({
      ...nextFilters,
      keyword: nextFilters.keyword.trim(),
      username: nextFilters.username.trim(),
    });
  };

  const resetFilters = () => {
    setPage(1);
    setFilters(EMPTY_FILTERS);
    setAppliedFilters(EMPTY_FILTERS);
    setShowFilters(false);
  };

  // 构建已激活的筛选标签列表（收起时展示）
  const activeFilterTags = useMemo(() => {
    const tags = [];
    if (appliedFilters.username) {
      tags.push({ key: 'username', label: `ID/${translate('\u7528\u6237\u540d')}: ${appliedFilters.username}` });
    }
    if (appliedFilters.status) {
      const found = STATUS_OPTIONS.find((o) => o.value === appliedFilters.status);
      tags.push({ key: 'status', label: `${translate('\u72b6\u6001')}: ${found ? translate(found.label) : appliedFilters.status}` });
    }
    if (appliedFilters.paymentMethod) {
      const found = PAYMENT_OPTIONS.find((o) => o.value === appliedFilters.paymentMethod);
      tags.push({ key: 'paymentMethod', label: `${translate('\u652f\u4ed8\u65b9\u5f0f')}: ${found ? translate(found.label) : appliedFilters.paymentMethod}` });
    }
    return tags;
  }, [appliedFilters, translate]);

  const removeFilterTag = (key) => {
    const nextFilters = { ...filters, [key]: '' };
    setFilters(nextFilters);
    applyFilters(nextFilters);
  };

  const filterByUsername = (username) => {
    if (!username) {
      return;
    }
    const nextFilters = {
      ...filters,
      username,
    };
    setFilters(nextFilters);
    applyFilters(nextFilters);
  };

  const handlePageChange = (currentPage) => {
    setPage(currentPage);
  };

  const handlePageSizeChange = (currentPageSize) => {
    setPageSize(currentPageSize);
    setPage(1);
  };

  const handleAdminComplete = async (tradeNo) => {
    try {
      const res = await API.post('/api/user/topup/complete', {
        trade_no: tradeNo,
      });
      const { success, message } = res.data;
      if (success) {
        Toast.success({ content: translate('\u8865\u5355\u6210\u529f') });
        await loadTopups(page, pageSize, appliedFilters);
        return;
      }

      Toast.error({ content: message || translate('\u8865\u5355\u5931\u8d25') });
    } catch (error) {
      Toast.error({ content: translate('\u8865\u5355\u5931\u8d25') });
    }
  };

  const confirmAdminComplete = (tradeNo) => {
    Modal.confirm({
      title: translate('\u786e\u8ba4\u8865\u5355'),
      content: translate('\u662f\u5426\u5c06\u8be5\u8ba2\u5355\u6807\u8bb0\u4e3a\u6210\u529f\u5e76\u4e3a\u7528\u6237\u5165\u8d26\uff1f'),
      onOk: () => handleAdminComplete(tradeNo),
    });
  };

  const renderStatusBadge = (status) => {
    const config = STATUS_CONFIG[status] || { type: 'primary', key: status || '-' };
    return (
      <span className='flex items-center gap-2'>
        <Badge dot type={config.type} />
        <span>{translate(config.key)}</span>
      </span>
    );
  };

  const renderPaymentMethod = (paymentMethod) => {
    const displayName = PAYMENT_METHOD_MAP[paymentMethod];
    return (
      <Tag shape='circle' color='grey'>
        {displayName ? translate(displayName) : paymentMethod || '-'}
      </Tag>
    );
  };

  const isSubscriptionTopup = (record) => {
    const tradeNo = (record?.trade_no || '').toLowerCase();
    return Number(record?.amount || 0) === 0 && tradeNo.startsWith('sub');
  };

  const columns = useMemo(() => {
    const baseColumns = [
      {
        title: translate('\u8ba2\u5355\u53f7'),
        dataIndex: 'trade_no',
        key: 'trade_no',
        width: 200,
        render: (text) => (
          <Text
            copyable
            ellipsis={{ showTooltip: { opts: { style: { wordBreak: 'break-all' } } } }}
            style={{ width: 170, display: 'inline-block' }}
          >
            {text || '-'}
          </Text>
        ),
      },
    ];

    if (userIsAdmin) {
      baseColumns.push({
        title: translate('\u7528\u6237\u540d'),
        dataIndex: 'username',
        key: 'username',
        render: (_, record) => {
          const username = record?.username || '';
          const displayName = record?.display_name || '';
          if (!username) {
            return <Text type='tertiary'>-</Text>;
          }

          return (
            <Space spacing={8} align='center'>
              <Avatar size='extra-small' color={stringToColor(username)}>
                {username.slice(0, 1).toUpperCase()}
              </Avatar>
              <div className='flex flex-col leading-5'>
                {record?.user_id > 0 && (
                  <Text type='tertiary' size='small'>ID: {record.user_id}</Text>
                )}
                <Text
                  link
                  size='small'
                  style={{ cursor: 'pointer' }}
                  onClick={() => filterByUsername(username)}
                >
                  {username}
                </Text>
                {displayName && <Text type='tertiary'>{displayName}</Text>}
              </div>
            </Space>
          );
        },
      });
    }

    baseColumns.push(
      {
        title: translate('\u652f\u4ed8\u65b9\u5f0f'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        render: renderPaymentMethod,
      },
      {
        title: translate('\u5145\u503c\u989d\u5ea6'),
        dataIndex: 'amount',
        key: 'amount',
        render: (amount, record) => {
          if (isSubscriptionTopup(record)) {
            return (
              <Tag color='purple' shape='circle' size='small'>
                SUB
              </Tag>
            );
          }

          return (
            <span className='flex items-center gap-1'>
              <Coins size={16} />
              <Text>{amount ?? 0}</Text>
            </span>
          );
        },
      },
      {
        title: translate('\u652f\u4ed8\u91d1\u989d'),
        dataIndex: 'money',
        key: 'money',
        render: (money) => (
          <Text type='danger'>
            {String.fromCharCode(0x00A5)}
            {Number(money || 0).toFixed(2)}
          </Text>
        ),
      },
      {
        title: translate('\u72b6\u6001'),
        dataIndex: 'status',
        key: 'status',
        render: renderStatusBadge,
      },
      {
        title: translate('\u521b\u5efa\u65f6\u95f4'),
        dataIndex: 'create_time',
        key: 'create_time',
        render: (time) => (time ? timestamp2string(time) : '-'),
      },
    );

    if (userIsAdmin) {
      baseColumns.push({
        title: translate('\u64cd\u4f5c'),
        key: 'action',
        render: (_, record) => {
          if (record.status !== 'pending') {
            return null;
          }

          return (
            <Button
              size='small'
              type='primary'
              theme='outline'
              onClick={() => confirmAdminComplete(record.trade_no)}
            >
              {translate('\u8865\u5355')}
            </Button>
          );
        },
      });
    }

    return baseColumns;
  }, [userIsAdmin, translate]);

  return (
    <Modal
      title={translate('\u652f\u4ed8\u8bb0\u5f55')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
      style={isMobile ? undefined : { width: '1100px', maxWidth: '95vw' }}
    >
      <div className='mb-3'>
        {/* 主搜索行：订单号 + 筛选 + 应用 + 重置 */}
        <div className='flex items-center gap-2'>
          <Input
            prefix={<IconSearch />}
            placeholder={translate('\u8ba2\u5355\u53f7')}
            value={filters.keyword}
            onChange={(value) => handleFilterChange('keyword', value)}
            onEnterPress={() => applyFilters()}
            showClear
            style={{ flex: 1 }}
          />
          <Button
            icon={<IconFilter />}
            theme={showFilters ? 'solid' : 'light'}
            type={activeFilterTags.length > 0 ? 'primary' : 'tertiary'}
            onClick={() => setShowFilters((v) => !v)}
          >
            {translate('\u7b5b\u9009')}
            {activeFilterTags.length > 0 && ` (${activeFilterTags.length})`}
          </Button>
          <Button type='primary' onClick={() => applyFilters()}>
            {translate('\u641c\u7d22')}
          </Button>
          {activeFilterTags.length > 0 && (
            <Button theme='borderless' type='tertiary' onClick={resetFilters}>
              {translate('\u91cd\u7f6e')}
            </Button>
          )}
        </div>

        {/* 收起时：展示已激活筛选标签 */}
        {!showFilters && activeFilterTags.length > 0 && (
          <div className='flex flex-wrap items-center gap-1 mt-2'>
            {activeFilterTags.map((tag) => (
              <Tag
                key={tag.key}
                closable
                size='small'
                color='blue'
                shape='circle'
                onClose={() => removeFilterTag(tag.key)}
              >
                {tag.label}
              </Tag>
            ))}
          </div>
        )}

        {/* 展开时：高级筛选面板 */}
        <Collapsible isOpen={showFilters} keepDOM>
          <div
            className='mt-2 rounded-lg p-3 flex flex-wrap items-end gap-3'
            style={{
              background: 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
            }}
          >
            {userIsAdmin && (
              <div style={{ minWidth: 160, flex: 1 }}>
                <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
                  ID/{translate('\u7528\u6237\u540d')}
                </div>
                <Input
                  placeholder={'ID/' + translate('\u7528\u6237\u540d')}
                  value={filters.username}
                  onChange={(value) => handleFilterChange('username', value)}
                  onEnterPress={() => applyFilters()}
                  showClear
                  size='small'
                />
              </div>
            )}
            <div style={{ minWidth: 120, flex: 1 }}>
              <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
                {translate('\u72b6\u6001')}
              </div>
              <Select
                value={filters.status}
                optionList={STATUS_OPTIONS.map((item) => ({
                  ...item,
                  label: translate(item.label),
                }))}
                onChange={(value) => handleFilterChange('status', value)}
                size='small'
                style={{ width: '100%' }}
              />
            </div>
            <div style={{ minWidth: 130, flex: 1 }}>
              <div className='text-xs mb-1' style={{ color: 'var(--semi-color-text-2)' }}>
                {translate('\u652f\u4ed8\u65b9\u5f0f')}
              </div>
              <Select
                value={filters.paymentMethod}
                optionList={PAYMENT_OPTIONS.map((item) => ({
                  ...item,
                  label: translate(item.label),
                }))}
                onChange={(value) => handleFilterChange('paymentMethod', value)}
                size='small'
                style={{ width: '100%' }}
              />
            </div>
          </div>
        </Collapsible>
      </div>
      <Table
        columns={columns}
        dataSource={topups}
        loading={loading}
        rowKey='id'
        size='small'
        pagination={{
          currentPage: page,
          pageSize,
          total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: handlePageChange,
          onPageSizeChange: handlePageSizeChange,
        }}
        scroll={{ x: '100%' }}
        empty={
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={<IllustrationNoResultDark style={{ width: 150, height: 150 }} />}
            description={translate('\u6682\u65e0\u5145\u503c\u8bb0\u5f55')}
            style={{ padding: 30 }}
          />
        }
      />
    </Modal>
  );
};

export default TopupHistoryModal;