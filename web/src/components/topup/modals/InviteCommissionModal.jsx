import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Card, Modal, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { API, renderQuota, showError } from '../../../helpers';
import CardTable from '../../common/ui/CardTable';

const { Text } = Typography;

const initialData = {
  invitees: {
    items: [],
    total: 0,
    page: 1,
    page_size: 10,
  },
  summary: {
    recharge_total_quota: 0,
  },
};

const InviteCommissionModal = ({ visible, onCancel, t }) => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(initialData);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const latestRequestRef = useRef(0);

  const loadData = async (targetPage = page, targetPageSize = pageSize) => {
    const requestId = latestRequestRef.current + 1;
    latestRequestRef.current = requestId;
    setLoading(true);
    try {
      const res = await API.get('/api/user/invite-recharge-commissions/self', {
        params: {
          p: targetPage,
          page_size: targetPageSize,
        },
      });
      const { success, message, data: payload } = res.data || {};
      if (latestRequestRef.current !== requestId) {
        return;
      }
      if (!success) {
        showError(message || t('加载失败'));
        return;
      }
      setData({
        invitees: payload?.invitees || initialData.invitees,
        summary: payload?.summary || initialData.summary,
      });
    } catch (error) {
      if (latestRequestRef.current === requestId) {
        showError(t('请求失败'));
      }
    } finally {
      if (latestRequestRef.current === requestId) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    if (!visible) {
      latestRequestRef.current += 1;
      setData(initialData);
      setLoading(false);
      return;
    }
    setPage(1);
    setPageSize(10);
    loadData(1, 10);
  }, [visible]);

  const columns = useMemo(
    () => [
      {
        title: t('用户'),
        dataIndex: 'alias',
        render: (alias) => alias || '-',
      },
      {
        title: t('充值返佣'),
        dataIndex: 'recharge_commission_quota',
        render: (quota) => (
          <Tag color='white' shape='circle'>
            {renderQuota(quota || 0)}
          </Tag>
        ),
      },
    ],
    [t],
  );

  const invitees = (data?.invitees?.items || []).map((item, index) => ({
    ...item,
    key: item?.alias || index,
  }));
  const inviteesTotal = Number(data?.invitees?.total || 0);

  return (
    <Modal
      title={t('邀请返佣明细')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={720}
      maskClosable={false}
    >
      <div className='space-y-3'>
        <Card className='!rounded-2xl shadow-sm border-0'>
          <Space wrap>
            <Tag color='white' shape='circle'>
              {t('邀请人数')}：{inviteesTotal}
            </Tag>
            <Tag color='white' shape='circle'>
              {t('充值返佣汇总')}：
              {renderQuota(data?.summary?.recharge_total_quota || 0)}
            </Tag>
          </Space>
          <div className='mt-2'>
            <Text type='tertiary' size='small'>
              {t(
                '仅按匿名用户汇总已结算充值返佣，不展示被邀请人的真实账号信息。',
              )}
            </Text>
          </div>
        </Card>

        <CardTable
          columns={columns}
          dataSource={invitees}
          loading={loading}
          rowKey='key'
          pagination={{
            currentPage: page,
            pageSize: pageSize,
            total: inviteesTotal,
            pageSizeOpts: [10, 20, 50, 100],
            showSizeChanger: true,
            onPageChange: (nextPage) => {
              setPage(nextPage);
              loadData(nextPage, pageSize);
            },
            onPageSizeChange: (nextPageSize) => {
              setPage(1);
              setPageSize(nextPageSize);
              loadData(1, nextPageSize);
            },
          }}
          hidePagination={false}
        />
      </div>
    </Modal>
  );
};

export default InviteCommissionModal;
