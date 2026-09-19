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

import React from 'react';
import { Banner, Modal, Table } from '@douyinfe/semi-ui';
import { renderNumber, renderQuota } from '../../../../helpers';

export default function LogGroupSummaryModal({
  groupSummaryVisible,
  setGroupSummaryVisible,
  groupSummaryLoading,
  groupSummaryRows,
  groupSummaryError,
  t,
}) {
  const totals = groupSummaryRows.reduce(
    (sum, row) => {
      for (const key of [
        'request_count',
        'quota',
        'prompt_tokens',
        'completion_tokens',
      ]) {
        sum[key] += Number(row[key] || 0);
      }
      return sum;
    },
    { request_count: 0, quota: 0, prompt_tokens: 0, completion_tokens: 0 },
  );
  return (
    <Modal
      title={t('分组用量汇总')}
      visible={groupSummaryVisible}
      onCancel={() => setGroupSummaryVisible(false)}
      footer={null}
      width={880}
      style={{ maxWidth: '96vw' }}
    >
      <Banner
        type='info'
        description={t(
          '按当前筛选和完整时间范围汇总消费记录，不受日志分页影响；失败、退款等记录不计入。',
        )}
        className='mb-3'
      />
      {groupSummaryError ? (
        <Banner type='danger' description={groupSummaryError} />
      ) : (
        <Table
          loading={groupSummaryLoading}
          dataSource={groupSummaryRows}
          rowKey='group'
          pagination={{ pageSize: 10 }}
          scroll={{ x: 760 }}
          columns={[
            {
              title: t('分组'),
              dataIndex: 'group',
              render: (value) => value || t('未分组'),
            },
            {
              title: t('消费记录数'),
              dataIndex: 'request_count',
              render: renderNumber,
            },
            {
              title: t('消耗额度'),
              dataIndex: 'quota',
              render: (value) => renderQuota(value),
            },
            {
              title: t('输入 Token'),
              dataIndex: 'prompt_tokens',
              render: renderNumber,
            },
            {
              title: t('输出 Token'),
              dataIndex: 'completion_tokens',
              render: renderNumber,
            },
          ]}
          footer={() =>
            groupSummaryLoading ? null : (
              <div>
                {t('合计')}：{renderNumber(totals.request_count)}{' '}
                {t('消费记录数')}
                {' · '}
                {renderQuota(totals.quota)}
                {' · '}
                {t('输入 Token')} {renderNumber(totals.prompt_tokens)}
                {' · '}
                {t('输出 Token')} {renderNumber(totals.completion_tokens)}
              </div>
            )
          }
        />
      )}
    </Modal>
  );
}
