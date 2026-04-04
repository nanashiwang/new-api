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
import React, { useState, useMemo } from 'react';
import {
  Button,
  Card,
  Dropdown,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  ChevronDown,
  Columns3,
  ChevronRight,
} from 'lucide-react';

const { Text } = Typography;

const allColumnKeys = [
  'created_at',
  'batch_name',
  'channel_name',
  'model_name',
  'configured_site_revenue_usd',
  'configured_profit_usd',
  'upstream_cost_usd',
  'actual_site_revenue_usd',
  'actual_profit_usd',
  'configured_actual_delta_usd',
  'site_pricing_source',
];

const defaultVisibleColumns = [
  'created_at',
  'batch_name',
  'channel_name',
  'model_name',
  'configured_site_revenue_usd',
  'configured_profit_usd',
];

const columnLabels = {
  created_at: '时间',
  batch_name: '组合',
  channel_name: '渠道',
  model_name: '模型',
  configured_site_revenue_usd: '本站配置收入',
  configured_profit_usd: '配置利润',
  upstream_cost_usd: '上游费用',
  actual_site_revenue_usd: '本站实际收入',
  actual_profit_usd: '实际利润',
  configured_actual_delta_usd: '配置与实际差值',
  site_pricing_source: '本站配置来源',
};

const ExpandableRow = ({
  record,
  formatMoney,
  status,
  sitePricingSourceLabelMap,
  t,
}) => (
  <div className='grid gap-4 bg-semi-color-fill-0 p-4 sm:grid-cols-2 lg:grid-cols-4'>
    <div>
      <Text type='tertiary' size='small'>
        {t('本站实际收入')}
      </Text>
      <div className='mt-1 font-medium text-emerald-600 dark:text-emerald-400'>
        {formatMoney(record.actual_site_revenue_usd, status)}
      </div>
    </div>
    <div>
      <Text type='tertiary' size='small'>
        {t('上游费用')}
      </Text>
      <div className='mt-1 font-medium text-amber-600 dark:text-amber-400'>
        {record.upstream_cost_known
          ? formatMoney(record.upstream_cost_usd, status)
          : '-'}
      </div>
    </div>
    <div>
      <Text type='tertiary' size='small'>
        {t('实际利润')}
      </Text>
      <div className='mt-1 font-medium text-violet-600 dark:text-violet-400'>
        {record.upstream_cost_known
          ? formatMoney(record.actual_profit_usd, status)
          : '-'}
      </div>
    </div>
    <div>
      <Text type='tertiary' size='small'>
        {t('配置与实际差值')}
      </Text>
      <div className='mt-1 font-medium'>
        {formatMoney(record.configured_actual_delta_usd, status)}
      </div>
    </div>
    <div className='sm:col-span-2 lg:col-span-4'>
      <Text type='tertiary' size='small'>
        {t('本站配置来源')}
      </Text>
      <div className='mt-1'>
        <Tag color={record.site_pricing_known ? 'blue' : 'grey'}>
          {sitePricingSourceLabelMap[record.site_pricing_source] ||
            record.site_pricing_source ||
            t('未知')}
        </Tag>
      </div>
    </div>
  </div>
);

const DetailTableCard = ({
  detailFilterText,
  detailRows,
  detailTotal,
  detailPage,
  detailPageSize,
  setDetailPage,
  setDetailPageSize,
  detailColumns,
  detailLoading,
  report,
  isMobile,
  formatMoney,
  status,
  sitePricingSourceLabelMap,
  t,
}) => {
  const [visibleColumns, setVisibleColumns] = useState(defaultVisibleColumns);
  const [expandedRowKeys, setExpandedRowKeys] = useState([]);

  const filteredColumns = useMemo(() => {
    return detailColumns.filter((col) =>
      visibleColumns.includes(col.dataIndex),
    );
  }, [detailColumns, visibleColumns]);

  const columnsWithExpand = useMemo(() => {
    return [
      {
        title: '',
        dataIndex: '__expand__',
        width: 40,
        render: (value, record, index) => {
          const isExpanded = expandedRowKeys.includes(record.id);
          return (
            <button
              onClick={() => {
                setExpandedRowKeys((prev) =>
                  isExpanded
                    ? prev.filter((k) => k !== record.id)
                    : [...prev, record.id],
                );
              }}
              className='flex h-6 w-6 items-center justify-center rounded hover:bg-semi-color-fill-1'
            >
              {isExpanded ? (
                <ChevronDown size={14} />
              ) : (
                <ChevronRight size={14} />
              )}
            </button>
          );
        },
      },
      ...filteredColumns,
    ];
  }, [filteredColumns, expandedRowKeys]);

  const toggleColumn = (key) => {
    setVisibleColumns((prev) => {
      if (prev.includes(key)) {
        if (prev.length <= 3) return prev;
        return prev.filter((k) => k !== key);
      }
      return [...prev, key];
    });
  };

  const columnDropdownItems = allColumnKeys.map((key) => ({
    node: 'item',
    name: (
      <div className='flex items-center gap-2'>
        {visibleColumns.includes(key) ? (
          <span className='text-semi-color-primary'>✓</span>
        ) : (
          <span className='text-semi-color-text-3'>○</span>
        )}
        <span>{t(columnLabels[key])}</span>
      </div>
    ),
    onClick: () => toggleColumn(key),
  }));

  return (
    <Card
      bordered={false}
      className='rounded-xl'
      title={
        <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
          <span className='font-medium'>{t('请求对账明细')}</span>
          <Space wrap>
            {detailFilterText ? (
              <Tag color='light-blue'>{detailFilterText}</Tag>
            ) : null}
            <Text type='tertiary' size='small'>
              {t('共 {{count}} 条', { count: detailTotal })}
              {report?.detail_truncated ? ` · ${t('结果已截断')}` : ''}
            </Text>
          </Space>
        </div>
      }
      headerExtraContent={
        <Dropdown
          trigger='click'
          position='bottomRight'
          render={
            <Dropdown.Menu>
              <Dropdown.Title>{t('显示列')}</Dropdown.Title>
              {columnDropdownItems.map((item, idx) => (
                <Dropdown.Item key={idx} onClick={item.onClick}>
                  {item.name}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          }
        >
          <Button icon={<Columns3 size={14} />} type='tertiary' size='small'>
            {t('列设置')}
          </Button>
        </Dropdown>
      }
    >
      <Table
        columns={columnsWithExpand}
        dataSource={detailRows}
        rowKey='id'
        loading={detailLoading}
        expandedRowKeys={expandedRowKeys}
        expandedRowRender={(record) => (
          <ExpandableRow
            record={record}
            formatMoney={formatMoney}
            status={status}
            sitePricingSourceLabelMap={sitePricingSourceLabelMap}
            t={t}
          />
        )}
        onExpandedRowsChange={(keys) => setExpandedRowKeys(keys)}
        pagination={{
          currentPage: detailPage,
          pageSize: detailPageSize,
          total: detailTotal,
          showSizeChanger: true,
          pageSizeOpts: isMobile ? [8, 12, 20] : [12, 20, 50, 100],
          onPageChange: (page) => setDetailPage(page),
          onPageSizeChange: (size) => {
            setDetailPage(1);
            setDetailPageSize(size);
          },
        }}
        empty={t('当前时间范围暂无明细')}
        scroll={{ x: 'max-content' }}
      />
    </Card>
  );
};

export default DetailTableCard;
