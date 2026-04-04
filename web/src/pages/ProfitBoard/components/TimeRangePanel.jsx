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
import {
  Banner,
  Button,
  Card,
  DatePicker,
  Typography,
} from '@douyinfe/semi-ui';

const { Text } = Typography;

const TimeRangePanel = ({
  datePresets,
  dateRange,
  setDateRange,
  validationErrors,
  t,
}) => (
  <Card bordered={false} className='rounded-xl' title={t('时间范围')}>
    <div className='space-y-4'>
      <div className='flex flex-wrap gap-2'>
        {datePresets.map((item) => (
          <Button
            key={item.label}
            type='tertiary'
            size='small'
            onClick={() => setDateRange(item.value)}
          >
            {t(item.label)}
          </Button>
        ))}
      </div>
      <DatePicker
        type='dateTimeRange'
        value={dateRange}
        onChange={(value) => setDateRange(value)}
        style={{ width: '100%' }}
      />
      {validationErrors.length > 0 ? (
        <Banner
          type='danger'
          description={validationErrors[0]}
          closeIcon={null}
        />
      ) : null}
    </div>
  </Card>
);

export default TimeRangePanel;
