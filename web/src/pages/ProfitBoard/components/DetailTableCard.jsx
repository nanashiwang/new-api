import React from 'react';
import { Card, Space, Table, Tag, Typography } from '@douyinfe/semi-ui';

const { Text } = Typography;

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
  t,
}) => (
  <Card
    bordered={false}
    title={
      <div className='flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between'>
        <span>{t('请求对账明细')}</span>
        <Space wrap>
          {detailFilterText ? <Tag color='light-blue'>{detailFilterText}</Tag> : null}
          <Text type='tertiary'>
            {t('共 {{count}} 条', { count: detailTotal })}
            {report?.detail_truncated ? ` · ${t('结果已截断')}` : ''}
          </Text>
        </Space>
      </div>
    }
  >
    <Text type='tertiary' className='mb-3 block'>
      {t('固定总金额只参与当前时间范围的汇总和图表，不会摊到单条请求明细。')}
    </Text>
    <Table
      columns={detailColumns}
      dataSource={detailRows}
      rowKey='id'
      loading={detailLoading}
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
    />
  </Card>
);

export default DetailTableCard;
