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
