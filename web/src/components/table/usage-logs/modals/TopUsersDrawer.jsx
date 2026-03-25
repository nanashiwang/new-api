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

import React, { useMemo } from 'react';
import {
  Banner,
  Button,
  Empty,
  Radio,
  Select,
  SideSheet,
  Space,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import { renderNumber, renderQuota } from '../../../../helpers';

const { Text } = Typography;
const RadioGroup = Radio.Group;

const limitOptions = [10, 20, 50].map((value) => ({
  label: String(value),
  value,
}));

const orderOptions = (t) => [
  { label: t('降序'), value: 'desc' },
  { label: t('升序'), value: 'asc' },
];

const viewModeOptions = (t) => [
  { label: t('同时看两个'), value: 'both' },
  { label: t('只看额度'), value: 'quota' },
  { label: t('只看请求数'), value: 'requests' },
];

const TopUsersDrawer = ({
  showTopUsersDrawer,
  setShowTopUsersDrawer,
  topUsersData,
  topUsersLoading,
  topUsersViewMode,
  setTopUsersViewMode,
  topUsersQuotaOrder,
  setTopUsersQuotaOrder,
  topUsersRequestOrder,
  setTopUsersRequestOrder,
  topUsersLimit,
  setTopUsersLimit,
  selectTopUser,
  refreshTopUsers,
  currentTopUsersLogType,
  t,
}) => {
  const rankColumns = useMemo(
    () => [
      {
        title: '#',
        dataIndex: 'rank',
        width: 56,
        render: (_, __, index) => index + 1,
      },
      {
        title: t('用户'),
        dataIndex: 'username',
        render: (text, record) => (
          <Button
            type='tertiary'
            theme='borderless'
            style={{ padding: 0, height: 'auto' }}
            onClick={() => selectTopUser(record.username)}
          >
            {text}
          </Button>
        ),
      },
      {
        title: t('分组'),
        dataIndex: 'group',
        render: (text) => text || '-',
      },
      {
        title: t('消耗额度'),
        dataIndex: 'quota',
        render: (value) => renderQuota(value),
      },
      {
        title: t('请求数'),
        dataIndex: 'request_count',
        render: (value) => renderNumber(value),
      },
      {
        title: t('输入 Tokens'),
        dataIndex: 'prompt_tokens',
        render: (value) => renderNumber(value),
      },
      {
        title: t('输出 Tokens'),
        dataIndex: 'completion_tokens',
        render: (value) => renderNumber(value),
      },
    ],
    [selectTopUser, t],
  );

  const showQuotaTable =
    topUsersViewMode === 'both' || topUsersViewMode === 'quota';
  const showRequestTable =
    topUsersViewMode === 'both' || topUsersViewMode === 'requests';
  const consumeOnlyMode =
    currentTopUsersLogType !== 0 && currentTopUsersLogType !== 2;

  const renderTable = (title, dataSource, emptyDescription) => (
    <div style={{ marginTop: 16 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 12,
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <Text strong>{title}</Text>
        <Text type='tertiary' size='small'>
          {t('点击用户名可直接筛选日志')}
        </Text>
      </div>
      <Table
        rowKey={(record) => `${title}-${record.user_id}-${record.username}`}
        columns={rankColumns}
        dataSource={dataSource}
        loading={topUsersLoading}
        size='small'
        pagination={false}
        scroll={{ x: 'max-content' }}
        empty={
          <Empty
            description={emptyDescription}
            image={null}
            style={{ padding: 24 }}
          />
        }
      />
    </div>
  );

  return (
    <SideSheet
      title={t('大用户榜单')}
      visible={showTopUsersDrawer}
      onCancel={() => setShowTopUsersDrawer(false)}
      width={900}
      bodyStyle={{ padding: 20 }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {consumeOnlyMode && (
          <Banner
            type='warning'
            closeIcon={null}
            description={t('当前榜单始终按消费日志统计，已忽略当前非消费日志类型筛选。')}
          />
        )}

        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: 12,
            flexWrap: 'wrap',
          }}
        >
          <RadioGroup
            type='button'
            value={topUsersViewMode}
            onChange={(e) => setTopUsersViewMode(e.target.value)}
          >
            {viewModeOptions(t).map((option) => (
              <Radio value={option.value} key={option.value}>
                {option.label}
              </Radio>
            ))}
          </RadioGroup>

          <Space wrap>
            <Text type='tertiary' size='small'>
              {t('榜单条数')}
            </Text>
            <Select
              value={topUsersLimit}
              optionList={limitOptions}
              onChange={(value) => setTopUsersLimit(Number(value))}
              style={{ width: 112 }}
            />
            <Button type='tertiary' onClick={refreshTopUsers}>
              {t('刷新榜单')}
            </Button>
          </Space>
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
            gap: 12,
          }}
        >
          <div>
            <Text type='tertiary' size='small'>
              {t('额度榜排序')}
            </Text>
            <Select
              value={topUsersQuotaOrder}
              optionList={orderOptions(t)}
              onChange={(value) => setTopUsersQuotaOrder(value)}
              style={{ width: '100%', marginTop: 6 }}
            />
          </div>
          <div>
            <Text type='tertiary' size='small'>
              {t('请求数榜排序')}
            </Text>
            <Select
              value={topUsersRequestOrder}
              optionList={orderOptions(t)}
              onChange={(value) => setTopUsersRequestOrder(value)}
              style={{ width: '100%', marginTop: 6 }}
            />
          </div>
        </div>

        {showQuotaTable
          ? renderTable(
              t('按消耗额度排序'),
              topUsersData.by_quota || [],
              t('暂无额度榜数据'),
            )
          : null}

        {showRequestTable
          ? renderTable(
              t('按请求数排序'),
              topUsersData.by_requests || [],
              t('暂无请求数榜数据'),
            )
          : null}
      </div>
    </SideSheet>
  );
};

export default TopUsersDrawer;
