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
    setActiveTypeKey(key);
    setActivePage(1);
    loadChannels(1, pageSize, idSort, enableTagMode, key);
  };

  const handleVendorChange = (key) => {
    setActiveVendorKey(key);
    setActivePage(1);
    loadChannels(
      1,
      pageSize,
      idSort,
      enableTagMode,
      activeTypeKey,
      undefined,
      key,
    );
  };

  const mimoCount = Number(vendorCounts?.mimo || 0);

  return (
    <div className='mb-2 flex flex-col gap-1'>
      <Typography.Text type='tertiary' size='small'>
        {t('类型')}
      </Typography.Text>
      <Tabs
        activeKey={activeTypeKey}
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

        {CHANNEL_OPTIONS.filter((opt) =>
          availableTypeKeys.includes(String(opt.value)),
        ).map((option) => {
          const key = String(option.value);
          const count = channelTypeCounts[option.value] || 0;
          return (
            <TabPane
              key={key}
              itemKey={key}
              tab={
                <span className='flex items-center gap-2'>
                  {getChannelIcon(option.value)}
                  {option.label}
                  <Tag
                    color={activeTypeKey === key ? 'red' : 'grey'}
                    shape='circle'
                  >
                    {count}
                  </Tag>
                </span>
              }
            />
          );
        })}
      </Tabs>

      {(mimoCount > 0 || activeVendorKey === 'mimo') && (
        <>
          <Typography.Text type='tertiary' size='small'>
            {t('供应商')}
          </Typography.Text>
          <Tabs
            activeKey={activeVendorKey}
            type='card'
            collapsible
            onChange={handleVendorChange}
          >
            <TabPane
              itemKey='all'
              tab={
                <span className='flex items-center gap-2'>
                  {t('全部供应商')}
                  <Tag
                    color={activeVendorKey === 'all' ? 'red' : 'grey'}
                    shape='circle'
                  >
                    {vendorCounts?.all || 0}
                  </Tag>
                </span>
              }
            />
            <TabPane
              itemKey='mimo'
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
          </Tabs>
        </>
      )}
    </div>
  );
};

export default ChannelsTabs;
