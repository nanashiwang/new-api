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

import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Spin, Typography } from '@douyinfe/semi-ui';
import { Image as ImageIcon } from 'lucide-react';
import { API, showError } from '../../helpers';

const ImagePlayground = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const createSession = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await API.post(
        '/api/image-playground/session',
        {},
        { skipGlobalLoading: true },
      );
      if (res.data?.success && res.data?.data?.url) {
        window.location.assign(res.data.data.url);
        return;
      }
      const message = res.data?.message || t('创建生图会话失败');
      setError(message);
      showError(message);
    } catch (err) {
      const message = err?.message || t('创建生图会话失败');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    createSession();
  }, [createSession]);

  return (
    <div className='flex justify-center px-4 py-10'>
      <Card className='w-full max-w-xl !rounded-2xl border-0 shadow-sm'>
        <div className='flex items-start gap-4'>
          <div className='rounded-2xl bg-blue-50 p-3 text-blue-600'>
            <ImageIcon size={24} />
          </div>
          <div className='flex-1'>
            <Typography.Title heading={4} className='!mb-2'>
              {t('AI 生图')}
            </Typography.Title>
            <Typography.Text type='secondary'>
              {loading
                ? t('正在创建临时访问会话，即将进入生图工作台')
                : t('临时访问会话创建失败，请重试')}
            </Typography.Text>

            <div className='mt-6'>
              {loading ? (
                <Spin tip={t('正在跳转')} />
              ) : (
                <>
                  {error && (
                    <div className='mb-4 text-sm text-red-500'>{error}</div>
                  )}
                  <Button type='primary' onClick={createSession}>
                    {t('重新进入')}
                  </Button>
                </>
              )}
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default ImagePlayground;
