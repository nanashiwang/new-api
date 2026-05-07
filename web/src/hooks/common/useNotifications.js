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

import { useState, useEffect } from 'react';
import { API, isAdmin } from '../../helpers';

export const useNotifications = (statusState) => {
  const [noticeVisible, setNoticeVisible] = useState(false);
  const [announcementUnread, setAnnouncementUnread] = useState(0);
  const [pendingWithdrawalCount, setPendingWithdrawalCount] = useState(0);
  const admin = isAdmin();

  const announcements = statusState?.status?.announcements || [];

  const fetchPendingWithdrawalCount = async () => {
    if (!admin) return;
    try {
      const res = await API.get(
        '/api/user/aff-withdrawals?status=pending&p=1&page_size=1',
      );
      if (res?.data?.success) {
        setPendingWithdrawalCount(Number(res.data.data?.total) || 0);
      }
    } catch (_) {
      // silent: keep previous value to avoid badge flicker
    }
  };

  // Helper functions
  const getAnnouncementKey = (a) =>
    `${a?.publishDate || ''}-${(a?.content || '').slice(0, 30)}`;

  const calculateUnreadCount = () => {
    if (!announcements.length) return 0;
    let readKeys = [];
    try {
      readKeys = JSON.parse(localStorage.getItem('notice_read_keys')) || [];
    } catch (_) {
      readKeys = [];
    }
    const readSet = new Set(readKeys);
    return announcements.filter((a) => !readSet.has(getAnnouncementKey(a)))
      .length;
  };

  const getUnreadKeys = () => {
    if (!announcements.length) return [];
    let readKeys = [];
    try {
      readKeys = JSON.parse(localStorage.getItem('notice_read_keys')) || [];
    } catch (_) {
      readKeys = [];
    }
    const readSet = new Set(readKeys);
    return announcements
      .filter((a) => !readSet.has(getAnnouncementKey(a)))
      .map(getAnnouncementKey);
  };

  // Effects
  useEffect(() => {
    setAnnouncementUnread(calculateUnreadCount());
  }, [announcements]);

  useEffect(() => {
    fetchPendingWithdrawalCount();
  }, [admin]);

  // 操作函数
  const handleNoticeOpen = () => {
    fetchPendingWithdrawalCount();
    setNoticeVisible(true);
  };

  const handleNoticeClose = () => {
    setNoticeVisible(false);
    if (announcements.length) {
      let readKeys = [];
      try {
        readKeys = JSON.parse(localStorage.getItem('notice_read_keys')) || [];
      } catch (_) {
        readKeys = [];
      }
      const mergedKeys = Array.from(
        new Set([...readKeys, ...announcements.map(getAnnouncementKey)]),
      );
      localStorage.setItem('notice_read_keys', JSON.stringify(mergedKeys));
    }
    setAnnouncementUnread(0);
  };

  return {
    noticeVisible,
    unreadCount: announcementUnread + (admin ? pendingWithdrawalCount : 0),
    announcementUnread,
    pendingWithdrawalCount,
    isAdminUser: admin,
    announcements,
    handleNoticeOpen,
    handleNoticeClose,
    getUnreadKeys,
  };
};
