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
