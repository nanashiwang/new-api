import React from 'react';
import {
  Banner,
  Button,
  Card,
  DatePicker,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../helpers';

const { Text } = Typography;

const TimeRangePanel = ({
  datePresets,
  dateRange,
  setDateRange,
  currentRangeText,
  currentRangeDuration,
  validationErrors,
  statusSummary,
  report,
  t,
}) => (
  <div className='grid gap-4 xl:grid-cols-[1.15fr_0.85fr]'>
    <Card bordered={false} title={t('时间分析范围')}>
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
        <div className='grid gap-3 md:grid-cols-2'>
          <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
            <Text type='tertiary'>{t('当前时间范围')}</Text>
            <div className='mt-1 font-medium'>{currentRangeText}</div>
          </div>
          <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
            <Text type='tertiary'>{t('时长')}</Text>
            <div className='mt-1 font-medium'>{currentRangeDuration}</div>
          </div>
        </div>
        {validationErrors.length > 0 ? (
          <Banner
            type='danger'
            description={validationErrors[0]}
            closeIcon={null}
          />
        ) : null}
      </div>
    </Card>

    <Card bordered={false} title={t('时间分析状态')}>
      <div className='space-y-3'>
        <div className='flex flex-wrap gap-2'>
          {statusSummary.length > 0 ? (
            statusSummary.map((item) => (
              <Tag key={item.key} color={item.color}>
                {item.text}
              </Tag>
            ))
          ) : (
            <Tag color='grey'>{t('等待时间分析结果')}</Tag>
          )}
        </div>
        <div className='rounded-lg bg-semi-color-fill-0 px-4 py-3'>
          <Text type='tertiary'>{t('时间分析上次更新时间')}</Text>
          <div className='mt-1 font-medium'>
            {report?.meta?.generated_at
              ? timestamp2string(report.meta.generated_at)
              : '-'}
          </div>
        </div>
      </div>
    </Card>
  </div>
);

export default TimeRangePanel;
