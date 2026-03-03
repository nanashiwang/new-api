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
import { Button, Modal } from '@douyinfe/semi-ui';

const UsersActions = ({
  setShowAddUser,
  selectedKeys = [],
  batchManageUsers,
  loading = false,
  t,
}) => {
  // Add new user
  const handleAddUser = () => {
    setShowAddUser(true);
  };

  // 批量删除需要二次确认，避免误删。
  const handleBatchDelete = () => {
    if (!selectedKeys || selectedKeys.length === 0) {
      return;
    }
    Modal.confirm({
      title: t('批量删除用户'),
      content: t('确定要删除所选的 {{count}} 个用户吗？', {
        count: selectedKeys.length,
      }),
      onOk: async () => {
        await batchManageUsers?.('delete');
      },
    });
  };

  return (
    <div className='flex flex-wrap gap-2 w-full md:w-auto order-2 md:order-1'>
      <Button className='w-full md:w-auto' onClick={handleAddUser} size='small'>
        {t('添加用户')}
      </Button>
      <Button
        type='tertiary'
        className='flex-1 md:flex-initial'
        onClick={() => batchManageUsers?.('enable')}
        disabled={selectedKeys.length === 0 || loading}
        loading={loading}
        size='small'
      >
        {t('批量启用')}
      </Button>
      <Button
        type='tertiary'
        className='flex-1 md:flex-initial'
        onClick={() => batchManageUsers?.('disable')}
        disabled={selectedKeys.length === 0 || loading}
        loading={loading}
        size='small'
      >
        {t('批量禁用')}
      </Button>
      <Button
        type='danger'
        className='w-full md:w-auto'
        onClick={handleBatchDelete}
        disabled={selectedKeys.length === 0 || loading}
        loading={loading}
        size='small'
      >
        {t('批量删除')}
      </Button>
    </div>
  );
};

export default UsersActions;
