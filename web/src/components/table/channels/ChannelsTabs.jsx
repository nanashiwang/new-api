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
import { Tabs, TabPane, Tag, Typography } from '@douyinfe/semi-ui';
import { CHANNEL_OPTIONS } from '../../../constants';
import { getChannelIcon, getLobeHubIcon } from '../../../helpers';
import {
  CHANNEL_CATEGORY_ALL,
  CHANNEL_CATEGORY_MIMO,
  channelTypeCategoryKey,
} from '../../../helpers/channelCategory';

const ChannelsTabs = ({
  enableTagMode,
  activeCategoryKey,
  setActiveCategoryKey,
  categoryCounts,
  availableCategoryKeys,
  loadChannels,
  activePage,
  pageSize,
  idSort,
  setActivePage,
  t,
}) => {
  if (enableTagMode) return null;

  const handleTabChange = (key) => {
    setActiveCategoryKey(key);
    setActivePage(1);
    loadChannels(1, pageSize, idSort, enableTagMode, key);
  };

  const mimoCategoryKey = CHANNEL_CATEGORY_MIMO;
  const mimoCount = Number(categoryCounts?.[mimoCategoryKey] || 0);
  const showMiMo = mimoCount > 0 || activeCategoryKey === mimoCategoryKey;
  const visibleTypeOptions = CHANNEL_OPTIONS.filter(
    (opt) =>
      availableCategoryKeys.includes(channelTypeCategoryKey(opt.value)) ||
      activeCategoryKey === channelTypeCategoryKey(opt.value),
  );
  const hasOpenAIType = visibleTypeOptions.some((option) => option.value === 1);

  const renderMiMoTab = () => (
    <TabPane
      key='vendor:mimo'
      itemKey={mimoCategoryKey}
      tab={
        <span className='flex items-center gap-2'>
          {getLobeHubIcon("Xiaomi.color='#FF6900'", 16)}
          {t('小米 MiMo')}
          <Tag
            color={activeCategoryKey === mimoCategoryKey ? 'red' : 'grey'}
            shape='circle'
          >
            {mimoCount}
          </Tag>
        </span>
      }
    />
  );
  const typeTabs = visibleTypeOptions.flatMap((option) => {
    const key = channelTypeCategoryKey(option.value);
    const count = categoryCounts[key] || 0;
    const tabs = [
      <TabPane
        key={key}
        itemKey={key}
        tab={
          <span className='flex items-center gap-2'>
            {getChannelIcon(option.value)}
            {option.label}
            <Tag
              color={activeCategoryKey === key ? 'red' : 'grey'}
              shape='circle'
            >
              {count}
            </Tag>
          </span>
        }
      />,
    ];
    if (showMiMo && option.value === 1) {
      tabs.push(renderMiMoTab());
    }
    return tabs;
  });

  return (
    <div className='mb-2 flex flex-col gap-1'>
      <Typography.Text type='tertiary' size='small'>
        {t('渠道分类')}
      </Typography.Text>
      <Tabs
        activeKey={activeCategoryKey}
        type='card'
        collapsible
        onChange={handleTabChange}
      >
        <TabPane
          itemKey={CHANNEL_CATEGORY_ALL}
          tab={
            <span className='flex items-center gap-2'>
              {t('全部')}
              <Tag
                color={
                  activeCategoryKey === CHANNEL_CATEGORY_ALL ? 'red' : 'grey'
                }
                shape='circle'
              >
                {categoryCounts[CHANNEL_CATEGORY_ALL] || 0}
              </Tag>
            </span>
          }
        />

        {typeTabs}
        {showMiMo && !hasOpenAIType ? renderMiMoTab() : null}
      </Tabs>
    </div>
  );
};

export default ChannelsTabs;
