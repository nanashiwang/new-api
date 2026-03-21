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
import { Button, Card, Divider, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { ChevronDown, ChevronUp, RefreshCw } from 'lucide-react';
import { renderQuota, timestamp2string } from '../../helpers';

const { Text } = Typography;

const MyTokensCard = ({ t, activeSellableTokens = [], onRefresh }) => {
  const [listCollapsed, setListCollapsed] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  if (activeSellableTokens.length === 0) {
    return null;
  }

  const handleRefresh = async () => {
    if (typeof onRefresh !== 'function' || refreshing) return;
    setRefreshing(true);
    try {
      await onRefresh();
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <Card className='!rounded-xl w-full' bodyStyle={{ padding: '12px' }}>
      <div className='flex items-center justify-between mb-2 gap-3'>
        <div className='flex items-center gap-2 flex-1 min-w-0'>
          <Text strong>{t('我的令牌')}</Text>
        </div>
        <div className='flex items-center gap-2'>
          <Button
            size='small'
            theme='light'
            type='tertiary'
            icon={
              <RefreshCw
                size={12}
                className={refreshing ? 'animate-spin' : ''}
              />
            }
            onClick={handleRefresh}
            loading={refreshing}
          >
            {t('刷新')}
          </Button>
        </div>
      </div>
      <Divider margin={8} />
      <div className='flex items-center justify-between mb-2'>
        <Text type='tertiary' size='small'>
          {t('共')} {activeSellableTokens.length} {t('个生效中')}
        </Text>
        <Button
          size='small'
          theme='borderless'
          type='tertiary'
          icon={listCollapsed ? <ChevronDown size={12} /> : <ChevronUp size={12} />}
          onClick={() => setListCollapsed((collapsed) => !collapsed)}
        >
          {listCollapsed ? t('展开') : t('收起')}
        </Button>
      </div>
      {!listCollapsed && (
        <div className='max-h-64 overflow-y-auto pr-1 semi-table-body'>
          {activeSellableTokens.map((token, idx) => {
            const used = Number(token?.used_quota || 0);
            const remain = Number(token?.remain_quota || 0);
            const total = used + remain;
            const isUnlimited = !!token?.unlimited_quota;
            const expiredTime = Number(token?.expired_time || 0);
            const isNeverExpire = expiredTime === -1 || expiredTime === 0;
            const now = Date.now() / 1000;
            const remainDays =
              !isNeverExpire && expiredTime > now
                ? Math.max(0, Math.ceil((expiredTime - now) / 86400))
                : null;
            const isLast = idx === activeSellableTokens.length - 1;

            return (
              <div key={token?.id || idx}>
                <div className='mb-2 flex items-center justify-between text-xs gap-3'>
                  <div className='flex items-center gap-2 min-w-0'>
                    <span className='font-medium truncate'>
                      {token?.name || `${t('令牌')} #${token?.id}`}
                    </span>
                    <Tag color='white' size='small' shape='circle'>
                      #{token?.id}
                    </Tag>
                  </div>
                  {remainDays !== null ? (
                    <span className='text-gray-500 whitespace-nowrap'>
                      {t('剩余')} {remainDays} {t('天')}
                    </span>
                  ) : (
                    <span className='text-gray-500 whitespace-nowrap'>{t('长期有效')}</span>
                  )}
                </div>
                <div className='mb-2 text-xs text-gray-500'>
                  {t('结束时间')}: {isNeverExpire ? t('长期有效') : timestamp2string(expiredTime)}
                </div>
                <div className='mb-2 text-xs text-gray-500'>
                  {t('额度')}:{' '}
                  {isUnlimited ? (
                    t('无限额度')
                  ) : (
                    <Tooltip content={`${t('原生额度')}：${renderQuota(used)}/${renderQuota(total)}`}>
                      <span>
                        {renderQuota(remain)} / {renderQuota(total)}
                      </span>
                    </Tooltip>
                  )}
                </div>
                {!isLast && <Divider margin={12} />}
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
};

export default MyTokensCard;
