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
import { Empty, Typography, Button } from '@douyinfe/semi-ui';
import { Link } from 'react-router-dom';
import { renderQuota } from '../../../helpers/dashboardFormat';
import { channelUsageLogURL } from '../../../helpers/channelUsage';
import CardTable from '../../common/ui/CardTable';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { getChannelsColumns } from './ChannelsColumnDefs';

const ChannelsTable = (channelsData) => {
  const {
    channels,
    usageOptions,
    usageMeta,
    usageError,
    selectedChannels,
    loading,
    searching,
    activePage,
    pageSize,
    channelCount,
    enableBatchDelete,
    compactMode,
    visibleColumns,
    setSelectedChannels,
    handlePageChange,
    handlePageSizeChange,
    handleRow,
    t,
    COLUMN_KEYS,
    // Column functions and data
    updateChannelBalance,
    manageChannel,
    manageTag,
    submitTagEdit,
    testChannel,
    openTagTestModal,
    setCurrentTestChannel,
    setShowModelTestModal,
    setEditingChannel,
    setShowEdit,
    setShowEditTag,
    setEditingTag,
    copySelectedChannel,
    refresh,
    checkOllamaVersion,
    // Multi-key management
    setShowMultiKeyManageModal,
    setCurrentMultiKeyChannel,
    openUpstreamUpdateModal,
    detectChannelUpstreamUpdates,
    channelConcurrencySnapshot,
  } = channelsData;

  // Get all columns
  const allColumns = useMemo(() => {
    return getChannelsColumns({
      t,
      COLUMN_KEYS,
      updateChannelBalance,
      manageChannel,
      manageTag,
      submitTagEdit,
      testChannel,
      openTagTestModal,
      setCurrentTestChannel,
      setShowModelTestModal,
      setEditingChannel,
      setShowEdit,
      setShowEditTag,
      setEditingTag,
      copySelectedChannel,
      refresh,
      activePage,
      channels,
      checkOllamaVersion,
      setShowMultiKeyManageModal,
      setCurrentMultiKeyChannel,
      openUpstreamUpdateModal,
      detectChannelUpstreamUpdates,
      channelConcurrencySnapshot,
    });
  }, [
    t,
    COLUMN_KEYS,
    updateChannelBalance,
    manageChannel,
    manageTag,
    submitTagEdit,
    testChannel,
    openTagTestModal,
    setCurrentTestChannel,
    setShowModelTestModal,
    setEditingChannel,
    setShowEdit,
    setShowEditTag,
    setEditingTag,
    copySelectedChannel,
    refresh,
    activePage,
    channels,
    checkOllamaVersion,
    setShowMultiKeyManageModal,
    setCurrentMultiKeyChannel,
    openUpstreamUpdateModal,
    detectChannelUpstreamUpdates,
    channelConcurrencySnapshot,
  ]);

  // Filter columns based on visibility settings
  const getVisibleColumns = () => {
    return allColumns.filter((column) => visibleColumns[column.key]);
  };

  const visibleColumnsList = useMemo(() => {
    if (!usageOptions.enabled) return getVisibleColumns();
    const keep = [
      COLUMN_KEYS.NAME,
      COLUMN_KEYS.STATUS,
      COLUMN_KEYS.PRIORITY,
      COLUMN_KEYS.WEIGHT,
      COLUMN_KEYS.OPERATE,
    ];
    const cols = allColumns
      .filter((col) => keep.includes(col.key))
      .map((col) =>
        col.key === COLUMN_KEYS.OPERATE
          ? {
              key: col.key,
              title: '',
              width: 76,
              render: (_, row) => (
                <Button
                  size='small'
                  onClick={() => {
                    if (row.children) {
                      setEditingTag(row.tag);
                      setShowEditTag(true);
                    } else {
                      setEditingChannel(row);
                      setShowEdit(true);
                    }
                  }}
                >
                  {t('编辑')}
                </Button>
              ),
            }
          : col,
      );
    cols.splice(
      2,
      0,
      {
        title: t('时段消耗'),
        key: 'period_usage',
        render: (_, row) =>
          usageMeta && row.usage ? (
            row.children?.length > 200 ? (
              <Typography.Text
                title={t('标签超过200个渠道，请展开后查看单渠道日志')}
              >
                {renderQuota(row.usage.total_quota, 4)}
              </Typography.Text>
            ) : (
              <Link
                to={channelUsageLogURL(row, usageMeta)}
                title={t('查看时段日志')}
              >
                {renderQuota(row.usage.total_quota, 4)}
              </Link>
            )
          ) : (
            '—'
          ),
      },
      {
        title: t('消耗占比'),
        key: 'usage_share',
        render: (_, row) =>
          usageMeta && row.usage
            ? usageMeta.usage_summary.total_quota
              ? (
                  (row.usage.total_quota /
                    usageMeta.usage_summary.total_quota) *
                  100
                ).toFixed(2) + '%'
              : '0.00%'
            : '—',
      },
      {
        title: t('日志请求数'),
        key: 'usage_requests',
        render: (_, row) => row.usage?.total_requests?.toLocaleString() ?? '—',
      },
      {
        title: t('日志成功率'),
        key: 'usage_success',
        render: (_, row) =>
          row.usage?.total_requests ? (
            <Typography.Text
              title={
                t('成功/失败') +
                ': ' +
                row.usage.success_requests +
                '/' +
                row.usage.failed_requests
              }
            >
              {(
                (row.usage.success_requests / row.usage.total_requests) *
                100
              ).toFixed(2)}
              %
            </Typography.Text>
          ) : (
            '—'
          ),
      },
    );
    return cols;
  }, [visibleColumns, allColumns, usageOptions.enabled, usageMeta, t]);

  const tableColumns = useMemo(() => {
    return compactMode
      ? visibleColumnsList.map(({ fixed, ...rest }) => rest)
      : visibleColumnsList;
  }, [compactMode, visibleColumnsList]);

  return (
    <CardTable
      columns={tableColumns}
      dataSource={channels}
      scroll={compactMode ? undefined : { x: 'max-content' }}
      pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: channelCount,
        pageSizeOpts: [10, 20, 50, 100],
        showSizeChanger: true,
        onPageSizeChange: handlePageSizeChange,
        onPageChange: handlePageChange,
      }}
      hidePagination={true}
      expandAllRows={false}
      onRow={handleRow}
      rowSelection={
        enableBatchDelete
          ? {
              selectedRowKeys: selectedChannels.map((row) => row.key),
              onChange: (selectedRowKeys, selectedRows) => {
                setSelectedChannels(selectedRows);
              },
            }
          : null
      }
      empty={
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={usageError || t('搜索无结果')}
          style={{ padding: 30 }}
        />
      }
      className='rounded-xl overflow-hidden'
      size='middle'
      loading={loading || searching}
    />
  );
};

export default ChannelsTable;
