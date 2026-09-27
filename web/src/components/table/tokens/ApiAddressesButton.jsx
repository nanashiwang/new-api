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
