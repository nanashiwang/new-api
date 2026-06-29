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

import React, { useEffect } from 'react';
import { X } from 'lucide-react';
import { usePublicTranslation } from '../../helpers/publicLocale';
import RichContent from '../common/RichContent';

const PublicNoticeModal = ({ visible, content, onClose }) => {
  const { t } = usePublicTranslation();

  useEffect(() => {
    if (!visible) return;

    const handleKeyDown = (event) => {
      if (event.key === 'Escape') onClose();
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose, visible]);

  if (!visible) return null;

  const closeToday = () => {
    localStorage.setItem('notice_close_date', new Date().toDateString());
    onClose();
  };

  return (
    <div className='public-notice-backdrop' role='presentation'>
      <section
        className='public-notice-modal'
        role='dialog'
        aria-modal='true'
        aria-labelledby='public-notice-title'
      >
        <header className='public-notice-header'>
          <h2 id='public-notice-title'>{t('系统公告')}</h2>
          <button
            type='button'
            className='public-notice-close'
            aria-label={t('关闭公告')}
            onClick={onClose}
          >
            <X size={18} />
          </button>
        </header>
        <RichContent
          className='public-notice-content'
          content={content}
          mode='markdown'
          breaks
        />
        <footer className='public-notice-footer'>
          <button
            type='button'
            className='public-notice-ghost'
            onClick={closeToday}
          >
            {t('今日关闭')}
          </button>
          <button
            type='button'
            className='public-notice-primary'
            onClick={onClose}
          >
            {t('关闭公告')}
          </button>
        </footer>
      </section>
    </div>
  );
};

export default PublicNoticeModal;
