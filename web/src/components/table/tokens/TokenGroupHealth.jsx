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

import React, { useState } from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { IconChevronDown, IconChevronUp } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { renderRatio } from '../../../helpers';
import {
  GroupHealthCollectionNote,
  GroupHealthPanel,
  GroupHealthScopeNote,
  GroupHealthStatus,
  GroupHealthSummary,
  resolveGroupHealthMetric,
} from '../../group-health';

const { Text } = Typography;

// Keep health data out of the option value and label: filtering and the saved
// token group must continue using the configured group, even during refreshes.
export const TokenGroupHealthOption = ({
  item,
  stats,
  loading,
  error,
  collectionEnabled = true,
}) => {
  const { t } = useTranslation();
  const { disabled, selected, focused, value, label, ratio } = item;

  return (
    <div
      role='option'
      aria-selected={Boolean(selected)}
      aria-disabled={Boolean(disabled)}
      className={item.className}
      style={{
        ...item.style,
        padding: '10px 16px',
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.5 : 1,
        backgroundColor: selected
          ? 'var(--semi-color-primary-light-default)'
          : focused
            ? 'var(--semi-color-fill-0)'
            : 'transparent',
      }}
      onClick={disabled ? undefined : item.onClick}
      onMouseEnter={disabled ? undefined : item.onMouseEnter}
    >
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0 flex-1'>
          <Text strong style={{ overflowWrap: 'anywhere' }}>
            {value}
          </Text>
          {label && label !== value ? (
            <div className='mt-1'>
              <Text size='small' type='secondary'>
                {label}
              </Text>
            </div>
          ) : null}
        </div>
        {ratio !== undefined && ratio !== null && ratio !== '' ? (
          <span className='shrink-0'>{renderRatio(ratio)}</span>
        ) : null}
      </div>
      <div className='mt-2 text-xs'>
        {value === 'auto' ? (
          <Text type='tertiary' size='small'>
            {t('自动选择分组，表现按实际调用分组统计')}
          </Text>
        ) : error ? (
          <Text type='tertiary' size='small'>
            {t('健康数据暂不可用')}
          </Text>
        ) : loading ? (
          <Text type='tertiary' size='small'>
            {t('正在加载健康数据')}
          </Text>
        ) : !collectionEnabled ? (
          <Text type='tertiary' size='small'>
            {t('分组健康统计未启用')}
          </Text>
        ) : (
          <div className='flex flex-wrap items-center gap-x-2 gap-y-1'>
            <Text type='tertiary' size='small'>
              {t('近24小时')}
            </Text>
            <GroupHealthSummary
              stats={stats}
              metric={resolveGroupHealthMetric(stats, value)}
              compact
            />
          </div>
        )}
      </div>
    </div>
  );
};

export const TokenGroupHealth = ({
  group,
  stats,
  loading,
  error,
  onRetry,
  collectionEnabled = true,
  collectionData,
}) => {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const metric = resolveGroupHealthMetric(stats, group);

  if (!group) return null;

  return (
    <div
      className='mb-3 rounded-xl p-3'
      style={{
        border: '1px solid var(--semi-color-border)',
        background: 'var(--semi-color-fill-0)',
      }}
    >
      <div className='mb-2 flex flex-wrap items-center justify-between gap-2'>
        <Text strong size='small'>
          {t('当前分组表现')}
        </Text>
        <Text type='tertiary' size='small'>
          {t('近24小时')} · {t('全部模型')}
        </Text>
      </div>
      {group !== 'auto' && !loading && !error ? (
        <GroupHealthCollectionNote data={collectionData} />
      ) : null}
      {group === 'auto' ? (
        <Text type='secondary' size='small'>
          {t('自动选择分组，表现按实际调用分组统计')}
        </Text>
      ) : loading || error ? (
        <GroupHealthStatus loading={loading} error={error} onRetry={onRetry} />
      ) : !collectionEnabled ? (
        <Text type='secondary' size='small'>
          {t('分组健康统计未启用')}
        </Text>
      ) : (
        <>
          <div className='flex flex-wrap items-start justify-between gap-2'>
            {!expanded ? (
              <GroupHealthSummary stats={stats} metric={metric} />
            ) : null}
            <Button
              size='small'
              theme='borderless'
              type='tertiary'
              icon={expanded ? <IconChevronUp /> : <IconChevronDown />}
              aria-expanded={expanded}
              onClick={() => setExpanded((current) => !current)}
            >
              {expanded ? t('收起趋势') : t('查看24小时趋势')}
            </Button>
          </div>
          {expanded ? (
            <div className='mt-3'>
              <GroupHealthPanel
                stats={stats || { group, series: [] }}
                metric={metric}
                showTitle={false}
              />
              <GroupHealthScopeNote />
            </div>
          ) : null}
          <div className='mt-2'>
            <Text type='tertiary' size='small'>
              {t('分组内全部模型的调用统计，具体模型表现可在模型广场查看')}
            </Text>
          </div>
        </>
      )}
    </div>
  );
};
