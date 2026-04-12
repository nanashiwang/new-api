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

import { useState, useRef } from 'react';
import { ITEMS_PER_PAGE } from '../../constants';

/**
 * 提取列表页共用的分页、加载、搜索和请求竞态控制状态。
 * useUsersData / useTokensData 等 Hook 均可使用此基础状态，
 * 避免重复声明 useState / useRef。
 */
export function usePaginatedList(initialPageSize = ITEMS_PER_PAGE) {
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(initialPageSize);
  const [searching, setSearching] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState([]);

  // 请求序列号，用于防止异步请求竞态（stale response 问题）
  const requestCounter = useRef(0);

  /** 生成下一个请求 ID，每次调用自增 */
  const nextRequestId = () => ++requestCounter.current;

  /** 判断给定 ID 是否仍是最新请求 */
  const isLatestRequest = (id) => id === requestCounter.current;

  return {
    loading,
    setLoading,
    activePage,
    setActivePage,
    pageSize,
    setPageSize,
    searching,
    setSearching,
    selectedKeys,
    setSelectedKeys,
    requestCounter,
    nextRequestId,
    isLatestRequest,
  };
}
