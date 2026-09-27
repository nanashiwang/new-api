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
import React, { useContext, useState } from 'react';
import { Button, Modal, Typography, Empty } from '@douyinfe/semi-ui';
import { StatusContext } from '../../../context/Status';
import { getApiAddresses } from '../../../helpers/apiAddresses';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

export default function ApiAddressesButton({ copyText, t }) {
  const [statusState] = useContext(StatusContext);
  const [visible, setVisible] = useState(false);
  const [copying, setCopying] = useState('');
  const isMobile = useIsMobile();
  const status = statusState?.status;
  const addresses = getApiAddresses(status, window.location.origin);
  const handleCopy = async (url) => {
    if (copying) return;
    setCopying(url);
    try {
      await copyText(url);
    } finally {
      setCopying('');
    }
  };
  return (
    <>
      <Button
        size='small'
        type='tertiary'
        className='flex-1 md:flex-initial'
        disabled={!status}
        aria-haspopup='dialog'
        aria-expanded={visible}
        onClick={() => setVisible(true)}
      >
        {t('API 地址')}
      </Button>
      <Modal
        title={t('API 地址')}
        visible={visible}
        onCancel={() => setVisible(false)}
        footer={null}
        size={isMobile ? 'full-width' : 'medium'}
      >
        {addresses.length ? (
          <ul className='m-0 p-0 list-none space-y-4'>
            {addresses.map((address) => (
              <li
                key={address.url}
                className='min-w-0 rounded-lg border border-[var(--semi-color-border)] p-3'
              >
                <Typography.Text strong className='break-all'>
                  {address.labelKey ? t(address.labelKey) : address.label}
                </Typography.Text>
                {address.description && (
                  <p className='text-sm text-[var(--semi-color-text-2)] break-words'>
                    {address.description}
                  </p>
                )}
                <div className='flex items-start gap-2 mt-2'>
                  <code className='min-w-0 flex-1 whitespace-pre-wrap break-all select-text'>
                    {address.url}
                  </code>
                  <Button
                    size='small'
                    className='shrink-0'
                    aria-label={`${t('复制')} ${address.url}`}
                    loading={copying === address.url}
                    disabled={!!copying}
                    onClick={() => handleCopy(address.url)}
                  >
                    {t('复制')}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <Empty description={t('暂无API信息')} />
        )}
      </Modal>
    </>
  );
}
