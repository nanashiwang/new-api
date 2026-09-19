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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Form } from '@douyinfe/semi-ui';
import { API, showError } from '../../../../helpers';

// Name search discovers IDs only. Log queries always use explicitly selected IDs.
export default function LogChannelPicker({ formApi, t }) {
  const [options, setOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [truncated, setTruncated] = useState(false);
  const request = useRef(0);
  const timer = useRef(null);
  useEffect(
    () => () => {
      clearTimeout(timer.current);
      request.current++;
    },
    [],
  );

  const search = (value) => {
    clearTimeout(timer.current);
    const seq = ++request.current;
    const keyword = value.trim();
    formApi?.setValue('channel_keyword', keyword);
    setOptions([]);
    setTruncated(false);
    setLoading(false);
    if (!keyword) return;
    // Keep a manual ID entry for deleted channels and historical records.
    if (/^[1-9]\d*$/.test(keyword) && Number.isSafeInteger(Number(keyword))) {
      setOptions([{ value: keyword, label: `${t('渠道 ID')}: ${keyword}` }]);
      return;
    }
    if ([...keyword].length < 2) return;
    setLoading(true);
    timer.current = setTimeout(async () => {
      try {
        const { data: result } = await API.get('/api/log/channel-options', {
          params: { keyword },
        });
        if (seq !== request.current) return;
        if (!result.success) throw new Error(result.message);
        setOptions(
          result.data.items.map((item) => ({
            value: String(item.id),
            label: `${item.id} · ${item.name}${item.status !== 1 ? ` · ${t('已禁用')}` : ''}`,
          })),
        );
        setTruncated(result.data.truncated);
      } catch (error) {
        if (seq === request.current) showError(error.message);
      } finally {
        if (seq === request.current) setLoading(false);
      }
    }, 300);
  };

  return (
    <div className='min-w-0'>
      <div style={{ display: 'none' }}>
        <Form.Input field='channel_keyword' noLabel />
      </div>
      <Form.Select
        field='channel_ids'
        placeholder={t('渠道名称 / ID（多选）')}
        multiple
        maxTagCount={1}
        max={200}
        filter
        remote
        onSearch={search}
        optionList={options}
        loading={loading}
        showClear
        onClear={() => {
          formApi?.setValue('channel_keyword', '');
          clearTimeout(timer.current);
          request.current++;
          setOptions([]);
          setLoading(false);
        }}
        pure
        size='small'
        style={{ width: '100%' }}
        innerBottomSlot={
          <div className='p-2 text-xs text-gray-500'>
            <div>
              {t(
                '按当前名称查找，选中后参与查询；含禁用渠道，旧渠道可输入 ID。',
              )}
            </div>
            {truncated && <div>{t('匹配超过 200 个，请缩小关键词。')}</div>}
            <Button
              size='small'
              disabled={loading || truncated || options.length === 0}
              onClick={() =>
                formApi?.setValue(
                  'channel_ids',
                  options.map((item) => item.value),
                )
              }
            >
              {t('选择全部匹配渠道')} ({options.length})
            </Button>
          </div>
        }
      />
    </div>
  );
}
