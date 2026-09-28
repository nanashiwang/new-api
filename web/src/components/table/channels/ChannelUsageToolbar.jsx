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

import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Input,
  Select,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import {
  formatUsageTime,
  parseUsageTime,
  usagePresetRange,
  validUsageRange,
} from '../../../helpers/channelUsage';
import { renderQuota } from '../../../helpers/dashboardFormat';

export default function ChannelUsageToolbar({
  usageOptions: options,
  changeUsageOptions,
  usageMeta,
  usageError,
  loading,
  refresh,
  t,
}) {
  const [draft, setDraft] = useState(options.range.map(formatUsageTime));
  const [rangeError, setRangeError] = useState(false);
  useEffect(() => {
    setDraft(options.range.map(formatUsageTime));
  }, [options.range]);
  const apply = () => {
    const range = draft.map(parseUsageTime);
    if (!validUsageRange(range)) {
      setRangeError(true);
      return;
    }
    setRangeError(false);
    changeUsageOptions({ ...options, preset: 'custom', range });
  };
  return (
    <div className='flex flex-col gap-3 mb-4'>
      <Space wrap>
        <Button
          theme={!options.enabled ? 'solid' : 'light'}
          onClick={() => changeUsageOptions({ ...options, enabled: false })}
        >
          {t('配置视图')}
        </Button>
        <Button
          theme={options.enabled ? 'solid' : 'light'}
          onClick={() => changeUsageOptions({ ...options, enabled: true })}
        >
          {t('用量视图')}
        </Button>
        {options.enabled && (
          <>
            <Select
              aria-label={t('统计时段')}
              value={options.preset}
              style={{ width: 140 }}
              optionList={[
                { value: 'today', label: t('今天') },
                { value: 'yesterday', label: t('昨天') },
                { value: '24h', label: t('最近24小时') },
                { value: '7d', label: t('最近7天') },
                { value: 'custom', label: t('自定义') },
              ]}
              onChange={(preset) => {
                setRangeError(false);
                changeUsageOptions({
                  ...options,
                  preset,
                  range:
                    preset === 'custom'
                      ? options.range
                      : usagePresetRange(preset),
                });
              }}
            />
            <Select
              aria-label={t('用量排序')}
              value={options.order}
              style={{ width: 170 }}
              optionList={[
                { value: 'desc', label: t('时段消耗从高到低') },
                { value: 'asc', label: t('时段消耗从低到高') },
              ]}
              onChange={(order) => changeUsageOptions({ ...options, order })}
            />
            <Button
              loading={loading}
              onClick={() =>
                options.preset === 'custom'
                  ? refresh()
                  : changeUsageOptions({
                      ...options,
                      range: usagePresetRange(options.preset),
                    })
              }
            >
              {t('更新统计')}
            </Button>
          </>
        )}
      </Space>
      {options.enabled && (
        <>
          {options.preset === 'custom' && (
            <Space wrap>
              <Input
                aria-label={t('开始时间（北京时间）')}
                style={{ width: 210 }}
                value={draft[0]}
                onChange={(v) => setDraft([v, draft[1]])}
              />
              <span>—</span>
              <Input
                aria-label={t('结束时间（北京时间）')}
                style={{ width: 210 }}
                value={draft[1]}
                onChange={(v) => setDraft([draft[0], v])}
              />
              <Button onClick={apply}>{t('应用时段')}</Button>
            </Space>
          )}
          {rangeError && (
            <Banner
              type='warning'
              closeIcon={null}
              description={t('请选择有效时间范围，最长30天且不能超过当前时间')}
            />
          )}
          <Typography.Text type='tertiary' size='small'>
            {t('北京时间，左闭右开，最长30天')} ·{' '}
            {formatUsageTime(options.range[0])} —{' '}
            {formatUsageTime(options.range[1])}
          </Typography.Text>
          <Typography.Text type='tertiary' size='small'>
            {t(
              '时段消耗按保留的消费和错误日志汇总，为站内结算额度，不代表上游成本。日志清理、未记录的失败和后续调账可能影响统计。权重影响请求分配，不保证消耗比例。',
            )}
          </Typography.Text>
          {usageMeta && (
            <Space wrap>
              <Typography.Text strong>
                {t('筛选范围消耗')}：
                {renderQuota(usageMeta.usage_summary.total_quota, 4)}
              </Typography.Text>
              <Typography.Text>
                {t('日志请求数')}：
                {usageMeta.usage_summary.total_requests.toLocaleString()}
              </Typography.Text>
              <Typography.Text type='tertiary' size='small'>
                {t('统计生成时间')}：{formatUsageTime(usageMeta.as_of)}{' '}
                {usageMeta.cached
                  ? t('（缓存，最多{{seconds}}秒）', {
                      seconds: usageMeta.cache_ttl_seconds,
                    })
                  : ''}
              </Typography.Text>
            </Space>
          )}
          {usageMeta?.deleted_usage?.total_requests > 0 && (
            <Typography.Text type='tertiary' size='small'>
              {t('已删除渠道消耗（同一时段及日志筛选，不计入占比）')}：
              {renderQuota(usageMeta.deleted_usage.total_quota, 4)}
            </Typography.Text>
          )}
          {usageError && (
            <Banner type='danger' closeIcon={null} description={usageError} />
          )}
        </>
      )}
    </div>
  );
}
