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

import React, { useState, useEffect } from 'react';
import { Card, Table, Tabs, TabPane, Spin, Button, Select, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconRefresh, IconFilter } from '@douyinfe/semi-icons';
import { API } from '../../helpers';
import { showError } from '../../helpers/notification';
import { useTranslation } from 'react-i18next';
import { CHANNEL_TYPE_MAP } from '../../constants/channel.constants';

const { Text } = Typography;

const ChannelMonitorPanel = ({ CARD_PROPS, ILLUSTRATION_SIZE }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [groupBy, setGroupBy] = useState('channel');
  const [timeRange, setTimeRange] = useState(86400); // 默认24小时
  const [channelStats, setChannelStats] = useState([]);
  const [groupStats, setGroupStats] = useState([]);

  // 表格列定义 - 按渠道
  const channelColumns = [
    {
      title: t('渠道名称'),
      dataIndex: 'channel_name',
      key: 'channel_name',
      render: (text, record) => (
        <Space>
          <Text strong>{text || t('未知渠道')}</Text>
          {record.channel_group && (
            <Tag size="small" color="blue">
              {record.channel_group}
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: t('渠道类型'),
      dataIndex: 'channel_type',
      key: 'channel_type',
      render: (type) => CHANNEL_TYPE_MAP[type] || t('未知类型'),
    },
    {
      title: t('总请求数'),
      dataIndex: 'total_requests',
      key: 'total_requests',
      sorter: (a, b) => a.total_requests - b.total_requests,
      render: (val) => val?.toLocaleString() || 0,
    },
    {
      title: t('成功率'),
      dataIndex: 'success_rate',
      key: 'success_rate',
      sorter: (a, b) => a.success_rate - b.success_rate,
      render: (rate) => {
        const color = rate >= 95 ? 'green' : rate >= 80 ? 'orange' : 'red';
        return (
          <Tag color={color} size="large">
            {rate?.toFixed(2)}%
          </Tag>
        );
      },
    },
    {
      title: t('成功/失败'),
      key: 'success_failed',
      render: (_, record) => (
        <Space>
          <Text type="success">{record.success_requests?.toLocaleString() || 0}</Text>
          <Text type="tertiary">/</Text>
          <Text type="danger">{record.failed_requests?.toLocaleString() || 0}</Text>
        </Space>
      ),
    },
    {
      title: t('总消耗额度'),
      dataIndex: 'total_quota',
      key: 'total_quota',
      sorter: (a, b) => a.total_quota - b.total_quota,
      render: (quota) => `$${((quota || 0) / 500000).toFixed(4)}`,
    },
    {
      title: t('平均响应时间'),
      dataIndex: 'avg_response_time',
      key: 'avg_response_time',
      sorter: (a, b) => a.avg_response_time - b.avg_response_time,
      render: (time) => `${(time || 0).toFixed(0)}ms`,
    },
  ];

  // 表格列定义 - 按分组
  const groupColumns = [
    {
      title: t('渠道分组'),
      dataIndex: 'channel_group',
      key: 'channel_group',
      render: (text) => (
        <Tag size="large" color="cyan">
          {text || t('默认分组')}
        </Tag>
      ),
    },
    {
      title: t('总请求数'),
      dataIndex: 'total_requests',
      key: 'total_requests',
      sorter: (a, b) => a.total_requests - b.total_requests,
      render: (val) => val?.toLocaleString() || 0,
    },
    {
      title: t('成功率'),
      dataIndex: 'success_rate',
      key: 'success_rate',
      sorter: (a, b) => a.success_rate - b.success_rate,
      render: (rate) => {
        const color = rate >= 95 ? 'green' : rate >= 80 ? 'orange' : 'red';
        return (
          <Tag color={color} size="large">
            {rate?.toFixed(2)}%
          </Tag>
        );
      },
    },
    {
      title: t('成功/失败'),
      key: 'success_failed',
      render: (_, record) => (
        <Space>
          <Text type="success">{record.success_requests?.toLocaleString() || 0}</Text>
          <Text type="tertiary">/</Text>
          <Text type="danger">{record.failed_requests?.toLocaleString() || 0}</Text>
        </Space>
      ),
    },
    {
      title: t('总消耗额度'),
      dataIndex: 'total_quota',
      key: 'total_quota',
      sorter: (a, b) => a.total_quota - b.total_quota,
      render: (quota) => `$${((quota || 0) / 500000).toFixed(4)}`,
    },
    {
      title: t('平均响应时间'),
      dataIndex: 'avg_response_time',
      key: 'avg_response_time',
      sorter: (a, b) => a.avg_response_time - b.avg_response_time,
      render: (time) => `${(time || 0).toFixed(0)}ms`,
    },
  ];

  // 加载监控数据
  const loadMonitorData = async () => {
    setLoading(true);
    try {
      const endTime = Math.floor(Date.now() / 1000);
      const startTime = endTime - timeRange;

      const res = await API.get('/api/monitor/channels', {
        params: {
          start_time: startTime,
          end_time: endTime,
          group_by: groupBy,
        },
      });

      if (res.data.success) {
        if (groupBy === 'channel') {
          setChannelStats(res.data.data || []);
        } else {
          setGroupStats(res.data.data || []);
        }
      } else {
        showError(res.data.message || t('加载监控数据失败'));
      }
    } catch (error) {
      showError(t('加载监控数据失败') + ': ' + (error.message || ''));
    } finally {
      setLoading(false);
    }
  };

  // 初始加载和依赖变化时重新加载
  useEffect(() => {
    loadMonitorData();
  }, [groupBy, timeRange]);

  // 时间范围选项
  const timeRangeOptions = [
    { value: 3600, label: t('最近1小时') },
    { value: 21600, label: t('最近6小时') },
    { value: 86400, label: t('最近24小时') },
    { value: 259200, label: t('最近3天') },
    { value: 604800, label: t('最近7天') },
    { value: 2592000, label: t('最近30天') },
  ];

  return (
    <Card
      {...CARD_PROPS}
      title={
        <Space>
          <IconFilter />
          <span>{t('渠道监控统计')}</span>
        </Space>
      }
      headerExtraContent={
        <Space>
          <Select
            value={timeRange}
            onChange={setTimeRange}
            style={{ width: 140 }}
            size="small"
          >
            {timeRangeOptions.map((opt) => (
              <Select.Option key={opt.value} value={opt.value}>
                {opt.label}
              </Select.Option>
            ))}
          </Select>
          <Button
            icon={<IconRefresh />}
            onClick={loadMonitorData}
            loading={loading}
            size="small"
          >
            {t('刷新')}
          </Button>
        </Space>
      }
    >
      <Spin spinning={loading}>
        <Tabs
          type="line"
          activeKey={groupBy}
          onChange={setGroupBy}
        >
          <TabPane tab={t('按渠道统计')} itemKey="channel">
            <Table
              columns={channelColumns}
              dataSource={channelStats}
              pagination={{
                pageSize: 10,
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50, 100],
              }}
              rowKey={(record) => `channel_${record.channel_id}`}
              size="small"
              empty={t('暂无数据')}
            />
          </TabPane>
          <TabPane tab={t('按分组统计')} itemKey="group">
            <Table
              columns={groupColumns}
              dataSource={groupStats}
              pagination={{
                pageSize: 10,
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50, 100],
              }}
              rowKey={(record) => `group_${record.channel_group}`}
              size="small"
              empty={t('暂无数据')}
            />
          </TabPane>
        </Tabs>
      </Spin>
    </Card>
  );
};

export default ChannelMonitorPanel;
