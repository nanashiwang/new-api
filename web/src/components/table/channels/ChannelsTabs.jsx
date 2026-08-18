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

const ChannelsTabs = ({
  enableTagMode,
  activeTypeKey,
  setActiveTypeKey,
  channelTypeCounts,
  availableTypeKeys,
  activeVendorKey,
  setActiveVendorKey,
  vendorCounts,
  loadChannels,
  activePage,
  pageSize,
  idSort,
  setActivePage,
  t,
}) => {
  if (enableTagMode) return null;

  const handleTabChange = (key) => {
    const isMiMo = key === 'vendor:mimo';
    const nextTypeKey = isMiMo ? 'all' : key;
    const nextVendorKey = isMiMo ? 'mimo' : 'all';
    setActiveTypeKey(nextTypeKey);
    setActiveVendorKey(nextVendorKey);
    setActivePage(1);
    loadChannels(
      1,
      pageSize,
      idSort,
      enableTagMode,
      nextTypeKey,
      undefined,
      nextVendorKey,
    );
  };

  const mimoCount = Number(vendorCounts?.mimo || 0);
  const showMiMo = mimoCount > 0 || activeVendorKey === 'mimo';
  const activeTabKey =
    activeVendorKey === 'mimo' ? 'vendor:mimo' : activeTypeKey;
  const visibleTypeOptions = CHANNEL_OPTIONS.filter((opt) =>
    availableTypeKeys.includes(String(opt.value)),
  );
  const hasOpenAIType = visibleTypeOptions.some((option) => option.value === 1);

  const renderMiMoTab = () => (
    <TabPane
      key='vendor:mimo'
      itemKey='vendor:mimo'
      tab={
        <span className='flex items-center gap-2'>
          {getLobeHubIcon("Xiaomi.color='#FF6900'", 16)}
          小米 MiMo
          <Tag
            color={activeVendorKey === 'mimo' ? 'red' : 'grey'}
            shape='circle'
          >
            {mimoCount}
          </Tag>
        </span>
      }
    />
  );
  const typeTabs = visibleTypeOptions.flatMap((option) => {
    const key = String(option.value);
    const count = channelTypeCounts[option.value] || 0;
    const tabs = [
      <TabPane
        key={key}
        itemKey={key}
        tab={
          <span className='flex items-center gap-2'>
            {getChannelIcon(option.value)}
            {option.label}
            <Tag color={activeTabKey === key ? 'red' : 'grey'} shape='circle'>
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
        {t('类型')}
      </Typography.Text>
      <Tabs
        activeKey={activeTabKey}
        type='card'
        collapsible
        onChange={handleTabChange}
      >
        <TabPane
          itemKey='all'
          tab={
            <span className='flex items-center gap-2'>
              {t('全部')}
              <Tag
                color={activeTypeKey === 'all' ? 'red' : 'grey'}
                shape='circle'
              >
                {channelTypeCounts['all'] || 0}
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
