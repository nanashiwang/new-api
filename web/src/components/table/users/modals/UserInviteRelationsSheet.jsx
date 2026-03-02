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
  Card,
  Empty,
  SideSheet,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, renderGroup, showError } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import CardTable from '../../../common/ui/CardTable';

const { Text } = Typography;

const UserInviteRelationsSheet = ({ visible, onCancel, user, t }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [relations, setRelations] = useState({
    user: null,
    inviter: null,
    invitees: {
      items: [],
      total: 0,
      page: 1,
      page_size: 10,
    },
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 按用户维度查询邀请关系：
  // 1. 当前用户基本信息
  // 2. 当前用户的邀请人信息（若存在）
  // 3. 当前用户邀请的下游用户列表（分页）
  const loadRelations = async (targetPage = page, targetPageSize = pageSize) => {
    if (!user?.id) {
      return;
    }
    setLoading(true);
    try {
      const res = await API.get(`/api/user/${user.id}/invite-relations`, {
        params: {
          p: targetPage,
          page_size: targetPageSize,
        },
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载失败'));
        return;
      }
      setRelations({
        user: data?.user || null,
        inviter: data?.inviter || null,
        invitees: data?.invitees || {
          items: [],
          total: 0,
          page: targetPage,
          page_size: targetPageSize,
        },
      });
    } catch (error) {
      showError(t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) {
      return;
    }
    // 每次打开抽屉重置分页，确保不同用户之间不会共享上一次翻页状态。
    setPage(1);
    setPageSize(10);
    loadRelations(1, 10);
  }, [visible, user?.id]);

  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 90,
      },
      {
        title: t('用户名'),
        dataIndex: 'username',
      },
      {
        title: t('显示名称'),
        dataIndex: 'display_name',
        render: (text) => text || '-',
      },
      {
        title: t('分组'),
        dataIndex: 'group',
        render: (group) => renderGroup(group),
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 110,
        render: (status) => {
          if (status === 1) {
            return (
              <Tag color='green' shape='circle' size='small'>
                {t('已启用')}
              </Tag>
            );
          }
          if (status === 2) {
            return (
              <Tag color='red' shape='circle' size='small'>
                {t('已禁用')}
              </Tag>
            );
          }
          return (
            <Tag color='grey' shape='circle' size='small'>
              {t('未知状态')}
            </Tag>
          );
        },
      },
    ],
    [t],
  );

  const invitees = (relations?.invitees?.items || []).map((item) => ({
    ...item,
    key: item?.id,
  }));
  const inviteesTotal = Number(relations?.invitees?.total || 0);
  const relationUser = relations?.user || user || null;
  const relationInviter = relations?.inviter || null;

  return (
    <SideSheet
      visible={visible}
      onCancel={onCancel}
      title={t('邀请关系')}
      placement='right'
      width={isMobile ? '100%' : 900}
      bodyStyle={{ padding: 0 }}
    >
      <Spin spinning={loading}>
        <div className='p-3 space-y-3'>
          <Card className='!rounded-2xl shadow-sm border-0'>
            <div className='flex flex-col gap-2'>
              <Text strong>
                {t('当前用户')}：{relationUser?.id || '-'} /{' '}
                {relationUser?.username || '-'}
              </Text>
              {relationInviter ? (
                <Text>
                  {t('邀请人')}：{relationInviter?.id} /{' '}
                  {relationInviter?.username || '-'}
                </Text>
              ) : (
                <Text type='tertiary'>{t('无邀请人')}</Text>
              )}
              <Space>
                <Tag color='white' shape='circle'>
                  {t('直接邀请人数')}：{relationUser?.aff_count || 0}
                </Tag>
                <Tag color='white' shape='circle'>
                  {t('本次列表总数')}：{inviteesTotal}
                </Tag>
              </Space>
            </div>
          </Card>

          <Card className='!rounded-2xl shadow-sm border-0'>
            <CardTable
              columns={columns}
              dataSource={invitees}
              pagination={{
                currentPage: page,
                pageSize: pageSize,
                total: inviteesTotal,
                pageSizeOpts: [10, 20, 50, 100],
                showSizeChanger: true,
                onPageChange: (nextPage) => {
                  setPage(nextPage);
                  loadRelations(nextPage, pageSize);
                },
                onPageSizeChange: (nextPageSize) => {
                  setPage(1);
                  setPageSize(nextPageSize);
                  loadRelations(1, nextPageSize);
                },
              }}
              size='middle'
              hidePagination={false}
              empty={
                <Empty
                  image={
                    <IllustrationNoResult style={{ width: 150, height: 150 }} />
                  }
                  darkModeImage={
                    <IllustrationNoResultDark
                      style={{ width: 150, height: 150 }}
                    />
                  }
                  description={t('暂无被邀请用户')}
                  style={{ padding: 30 }}
                />
              }
            />
          </Card>
        </div>
      </Spin>
    </SideSheet>
  );
};

export default UserInviteRelationsSheet;
