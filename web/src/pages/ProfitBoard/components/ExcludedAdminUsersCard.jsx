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
import { Card, Select, Typography } from '@douyinfe/semi-ui';

const { Text } = Typography;

const ExcludedAdminUsersCard = ({
  adminUsers,
  excludedUserIDs,
  onChange,
  t,
}) => {
  const optionList = useMemo(() => {
    const baseOptions = (adminUsers || []).map((user) => ({
      label: `${user.display_name || user.username || `#${user.id}`} · ${user.username}`,
      value: String(user.id),
    }));
    const existingValues = new Set(baseOptions.map((item) => item.value));
    (excludedUserIDs || []).forEach((userID) => {
      const value = String(userID);
      if (!existingValues.has(value)) {
        baseOptions.push({
          label: t('用户 #{{id}}', { id: userID }),
          value,
        });
      }
    });
    return baseOptions;
  }, [adminUsers, excludedUserIDs, t]);

  return (
    <Card
      bordered={false}
      title={t('收入排除')}
      className='rounded-xl'
    >
      <div className='space-y-2'>
        <Text type='tertiary' size='small'>
          {t('选中的管理员请求不计入本站配置收入，但上游费用和利润仍继续统计')}
        </Text>
        <Select
          multiple
          filter
          maxTagCount={3}
          value={(excludedUserIDs || []).map((item) => String(item))}
          optionList={optionList}
          placeholder={t('选择要排除收入的管理员')}
          style={{ width: '100%' }}
          onChange={(value) =>
            onChange(
              (value || [])
                .map((item) => Number(item))
                .filter((item) => Number.isInteger(item) && item > 0),
            )
          }
        />
      </div>
    </Card>
  );
};

export default ExcludedAdminUsersCard;
