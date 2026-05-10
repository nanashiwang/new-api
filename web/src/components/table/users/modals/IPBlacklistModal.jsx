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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Input,
  Modal,
  SideSheet,
  Space,
  Table,
  Tag,
  TextArea,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconAlertTriangle,
  IconDelete,
  IconRefresh,
  IconSearch,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess, timestamp2string } from '../../../../helpers';

const { Text } = Typography;

const DEFAULT_RULE_USERS_STATE = {
  items: [],
  total: 0,
  page: 1,
  pageSize: 10,
  loading: false,
  loaded: false,
  selectedRowKeys: [],
};

const getUserLabel = (user) => {
  const username = typeof user?.username === 'string' ? user.username.trim() : '';
  const displayName =
    typeof user?.display_name === 'string' ? user.display_name.trim() : '';
  return username || displayName || `#${user?.id || ''}`;
};

const IPBlacklistModal = ({ visible, onCancel, t, onChanged }) => {
  const [items, setItems] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [ip, setIp] = useState('');
  const [reason, setReason] = useState('');
  const [adding, setAdding] = useState(false);
  const [ruleUsers, setRuleUsers] = useState({});
  const [managingKey, setManagingKey] = useState('');

  const getRuleUsersState = (ruleId) =>
    ruleUsers[ruleId] || DEFAULT_RULE_USERS_STATE;

  const patchRuleUsersState = (ruleId, patch) => {
    setRuleUsers((prev) => ({
      ...prev,
      [ruleId]: {
        ...DEFAULT_RULE_USERS_STATE,
        ...(prev[ruleId] || {}),
        ...patch,
      },
    }));
  };

  const loadItems = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextKeyword = keyword,
  ) => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/ip-blacklist', {
        params: {
          keyword: nextKeyword.trim(),
          p: nextPage,
          page_size: nextPageSize,
        },
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('请求失败'));
        return;
      }
      setItems(Array.isArray(data?.items) ? data.items : []);
      setTotal(Number(data?.total || 0));
      setPage(Number(data?.page || nextPage));
      setPageSize(Number(data?.page_size || nextPageSize));
    } catch (error) {
      showError(error?.message || t('请求失败'));
    } finally {
      setLoading(false);
    }
  };

  const loadRuleUsers = async (
    ruleId,
    nextPage = 1,
    nextPageSize = DEFAULT_RULE_USERS_STATE.pageSize,
  ) => {
    if (!ruleId) {
      return;
    }
    patchRuleUsersState(ruleId, {
      loading: true,
      page: nextPage,
      pageSize: nextPageSize,
    });
    try {
      const res = await API.get(`/api/user/ip-blacklist/${ruleId}/users`, {
        params: {
          p: nextPage,
          page_size: nextPageSize,
        },
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('请求失败'));
        patchRuleUsersState(ruleId, { loading: false, loaded: true });
        return;
      }
      patchRuleUsersState(ruleId, {
        items: Array.isArray(data?.items) ? data.items : [],
        total: Number(data?.total || 0),
        page: Number(data?.page || nextPage),
        pageSize: Number(data?.page_size || nextPageSize),
        loading: false,
        loaded: true,
        selectedRowKeys: [],
      });
    } catch (error) {
      showError(error?.message || t('请求失败'));
      patchRuleUsersState(ruleId, { loading: false, loaded: true });
    }
  };

  const showCurrentIPConfirm = (data, retry) => {
    Modal.confirm({
      title: t('确认拉黑当前 IP'),
      content: t(
        '规则 {{rule}} 会拉黑你当前访问 IP {{ip}}。确认后你可能无法继续访问后台。',
        {
          rule: data?.rule || '',
          ip: data?.client_ip || '',
        },
      ),
      type: 'warning',
      onOk: retry,
    });
  };

  const addItem = async (confirmCurrentIP = false) => {
    const nextIP = ip.trim();
    if (!nextIP) {
      showError(t('请输入 IP 或 CIDR'));
      return;
    }
    setAdding(true);
    try {
      const res = await API.post('/api/user/ip-blacklist', {
        ip: nextIP,
        reason,
        confirm_current_ip: confirmCurrentIP,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        if (data?.code === 'current_ip_requires_confirmation' && !confirmCurrentIP) {
          showCurrentIPConfirm(data, () => addItem(true));
          return;
        }
        showError(message || t('操作失败，请重试'));
        return;
      }
      showSuccess(data?.created === false ? t('该 IP 已在黑名单中') : t('已添加 IP 黑名单'));
      setIp('');
      setReason('');
      await loadItems(1, pageSize);
      await onChanged?.();
    } catch (error) {
      showError(error?.message || t('操作失败，请重试'));
    } finally {
      setAdding(false);
    }
  };

  const deleteItem = (item) => {
    Modal.confirm({
      title: t('解除 IP 拉黑'),
      content: `${t('确定要解除拉黑吗？')} ${item.cidr}`,
      type: 'warning',
      onOk: async () => {
        try {
          const res = await API.delete(`/api/user/ip-blacklist/${item.id}`);
          const { success, message } = res.data || {};
          if (!success) {
            showError(message || t('操作失败，请重试'));
            return;
          }
          showSuccess(t('已解除 IP 拉黑'));
          await loadItems(page, pageSize);
          await onChanged?.();
        } catch (error) {
          showError(error?.message || t('操作失败，请重试'));
        }
      },
    });
  };

  const executeManageRuleUsers = async (record, action, scope, selectedIds) => {
    const state = getRuleUsersState(record.id);
    const actionKey = `${record.id}:${scope}:${action}`;
    setManagingKey(actionKey);
    try {
      const res = await API.post(`/api/user/ip-blacklist/${record.id}/users/manage`, {
        action,
        scope,
        user_ids: scope === 'selected' ? selectedIds : [],
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('操作失败，请重试'));
        return;
      }
      showSuccess(
        t('处理完成: {{success}}个成功, {{failed}}个失败', {
          success: Number(data?.success_count || 0),
          failed: Number(data?.failed_count || 0),
        }),
      );
      await loadRuleUsers(record.id, state.page, state.pageSize);
      await loadItems(page, pageSize);
      await onChanged?.();
    } catch (error) {
      showError(error?.message || t('操作失败，请重试'));
    } finally {
      setManagingKey('');
    }
  };

  const manageRuleUsers = (record, action, scope) => {
    const state = getRuleUsersState(record.id);
    const selectedIds = state.selectedRowKeys || [];
    if (scope === 'selected' && selectedIds.length === 0) {
      showError(t('请先选择命中账号'));
      return;
    }

    const actionText = action === 'delete' ? t('注销') : t('禁用');
    const scopeText = scope === 'all' ? t('全部命中账号') : t('选中账号');
    Modal.confirm({
      title: `${actionText}${scopeText}`,
      content:
        scope === 'all'
          ? t('将对该 IP 黑名单规则下的全部命中账号执行操作，root 或无权限账号会被自动跳过。')
          : t('将只处理当前选中的命中账号，root 或无权限账号会被自动跳过。'),
      type: 'warning',
      okText: actionText,
      onOk: () => executeManageRuleUsers(record, action, scope, selectedIds),
    });
  };

  const renderUserStatus = (record) => {
    if (record?.deleted_at) {
      return (
        <Tag color='red' shape='circle'>
          {t('已注销')}
        </Tag>
      );
    }
    if (record?.status === 1) {
      return (
        <Tag color='green' shape='circle'>
          {t('已启用')}
        </Tag>
      );
    }
    if (record?.status === 2) {
      return (
        <Tag color='red' shape='circle'>
          {t('已禁用')}
        </Tag>
      );
    }
    return (
      <Tag color='grey' shape='circle'>
        {t('未知状态')}
      </Tag>
    );
  };

  const renderUserRole = (role) => {
    if (role === 100) {
      return (
        <Tag color='orange' shape='circle'>
          {t('超级管理员')}
        </Tag>
      );
    }
    if (role === 10) {
      return (
        <Tag color='yellow' shape='circle'>
          {t('管理员')}
        </Tag>
      );
    }
    return (
      <Tag color='blue' shape='circle'>
        {t('普通用户')}
      </Tag>
    );
  };

  const renderMatchedPreview = (record) => {
    const count = Number(record?.matched_user_count || 0);
    const previews = Array.isArray(record?.matched_users) ? record.matched_users : [];
    return (
      <div className='flex flex-col gap-1'>
        <Space spacing={4}>
          <Tag color={count > 0 ? 'red' : 'white'} shape='circle'>
            {t('{{count}} 个账号', { count })}
          </Tag>
          {count > previews.length ? (
            <Text type='tertiary' size='small'>
              {t('展开查看全部')}
            </Text>
          ) : null}
        </Space>
        {previews.length > 0 ? (
          <div className='flex flex-wrap gap-1'>
            {previews.slice(0, 5).map((user) => (
              <Tooltip
                key={user.id}
                content={`${getUserLabel(user)} · ${user.register_ip || '-'}`}
                position='top'
              >
                <Tag color='white' shape='circle' className='!text-xs'>
                  #{user.id}
                </Tag>
              </Tooltip>
            ))}
          </div>
        ) : null}
      </div>
    );
  };

  const userColumns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 80,
        render: (text) => `#${text}`,
      },
      {
        title: t('账号'),
        dataIndex: 'username',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Space spacing={4} wrap>
              <span className='font-medium'>{getUserLabel(record)}</span>
              {renderUserRole(record.role)}
            </Space>
            {record.display_name && record.display_name !== record.username ? (
              <Text type='tertiary' size='small'>
                {record.display_name}
              </Text>
            ) : null}
          </div>
        ),
      },
      {
        title: t('注册 IP'),
        dataIndex: 'register_ip',
        width: 190,
        render: (text) => (
          <Tag color='white' shape='circle' className='font-mono'>
            {text || '-'}
          </Tag>
        ),
      },
      {
        title: t('注册来源'),
        dataIndex: 'register_source',
        width: 130,
        render: (text) => text || '-',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 110,
        render: (_, record) => renderUserStatus(record),
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_at',
        width: 170,
        render: (text) => (text ? timestamp2string(text) : '-'),
      },
    ],
    [t],
  );

  const renderRuleUsers = (record) => {
    const state = getRuleUsersState(record.id);
    const selectedCount = (state.selectedRowKeys || []).length;
    const totalCount = state.loaded
      ? Number(state.total || 0)
      : Number(record?.matched_user_count || 0);
    const disableAllActions = state.loading || totalCount <= 0;

    return (
      <div
        className='rounded-2xl border p-3 md:p-4'
        style={{
          borderColor: 'var(--semi-color-border)',
          background: 'var(--semi-color-fill-0)',
        }}
      >
        <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between gap-3 mb-3'>
          <div className='flex flex-col gap-1'>
            <Text strong>{t('该规则命中的注册账号')}</Text>
            <Text type='tertiary' size='small'>
              {t('拉黑 IP 只做标记；这里的禁用和注销需要你手动确认。')}
            </Text>
          </div>
          <Space wrap>
            <Tag color='white' shape='circle'>
              {t('已选 {{count}} 个', { count: selectedCount })}
            </Tag>
            <Button
              size='small'
              type='tertiary'
              icon={<IconRefresh />}
              loading={state.loading}
              onClick={() => loadRuleUsers(record.id, state.page, state.pageSize)}
            >
              {t('刷新')}
            </Button>
            <Button
              size='small'
              type='warning'
              disabled={selectedCount === 0 || state.loading}
              loading={managingKey === `${record.id}:selected:disable`}
              onClick={() => manageRuleUsers(record, 'disable', 'selected')}
            >
              {t('禁用选中')}
            </Button>
            <Button
              size='small'
              type='danger'
              disabled={selectedCount === 0 || state.loading}
              loading={managingKey === `${record.id}:selected:delete`}
              onClick={() => manageRuleUsers(record, 'delete', 'selected')}
            >
              {t('注销选中')}
            </Button>
            <Button
              size='small'
              type='warning'
              theme='outline'
              disabled={disableAllActions}
              loading={managingKey === `${record.id}:all:disable`}
              onClick={() => manageRuleUsers(record, 'disable', 'all')}
            >
              {t('禁用全部')}
            </Button>
            <Button
              size='small'
              type='danger'
              theme='outline'
              disabled={disableAllActions}
              loading={managingKey === `${record.id}:all:delete`}
              onClick={() => manageRuleUsers(record, 'delete', 'all')}
            >
              {t('注销全部')}
            </Button>
          </Space>
        </div>
        <Table
          columns={userColumns}
          dataSource={state.items}
          loading={state.loading}
          rowKey='id'
          size='small'
          scroll={{ x: 'max-content' }}
          rowSelection={{
            selectedRowKeys: state.selectedRowKeys || [],
            getCheckboxProps: (user) => ({
              disabled: !!user?.deleted_at || user?.role === 100,
            }),
            onChange: (selectedRowKeys) => {
              patchRuleUsersState(record.id, {
                selectedRowKeys: selectedRowKeys || [],
              });
            },
          }}
          pagination={{
            currentPage: state.page,
            pageSize: state.pageSize,
            total: state.total,
            pageSizeOpts: [10, 20, 50, 100],
            showSizeChanger: true,
            onPageChange: (nextPage) =>
              loadRuleUsers(record.id, nextPage, state.pageSize),
            onPageSizeChange: (nextPageSize) =>
              loadRuleUsers(record.id, 1, nextPageSize),
          }}
        />
      </div>
    );
  };

  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 70,
      },
      {
        title: t('IP / CIDR'),
        dataIndex: 'cidr',
        width: 260,
        render: (text, record) => (
          <div className='flex flex-col gap-1'>
            <Space spacing={4} wrap>
              <Tag color={record.ip_version === 6 ? 'light-blue' : 'blue'} shape='circle'>
                IPv{record.ip_version}
              </Tag>
              <span className='font-mono font-medium'>{text}</span>
            </Space>
            {record.ip && record.ip !== text ? (
              <Text type='tertiary' size='small'>
                {t('原始输入')}: {record.ip}
              </Text>
            ) : null}
          </div>
        ),
      },
      {
        title: t('命中账号'),
        dataIndex: 'matched_user_count',
        width: 220,
        render: (_, record) => renderMatchedPreview(record),
      },
      {
        title: t('来源 / 原因'),
        dataIndex: 'reason',
        render: (text, record) => (
          <div className='flex flex-col gap-1'>
            <Text>{text || '-'}</Text>
            <Space spacing={4} wrap>
              <Tag color='white' shape='circle'>
                {t('来源用户')}: {record.source_user_id ? `#${record.source_user_id}` : '-'}
              </Tag>
              <Tag color='white' shape='circle'>
                {t('管理员')}: {record.created_by ? `#${record.created_by}` : '-'}
              </Tag>
            </Space>
          </div>
        ),
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_at',
        width: 170,
        render: (text) => (text ? timestamp2string(text) : '-'),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        width: 90,
        render: (_, record) => (
          <Button
            type='danger'
            theme='borderless'
            size='small'
            icon={<IconDelete />}
            onClick={(event) => {
              event?.stopPropagation?.();
              deleteItem(record);
            }}
          />
        ),
      },
    ],
    [t, ruleUsers, managingKey, page, pageSize],
  );

  useEffect(() => {
    if (visible) {
      loadItems(1, pageSize);
      return;
    }
    setRuleUsers({});
  }, [visible]);

  return (
    <SideSheet
      visible={visible}
      onCancel={onCancel}
      title={t('IP 黑名单')}
      placement='right'
      width='min(1120px, calc(100vw - 24px))'
      bodyStyle={{ padding: 20 }}
    >
      <div className='flex flex-col gap-4'>
        <div
          className='rounded-2xl border px-4 py-3 flex gap-3'
          style={{
            borderColor: 'rgba(245, 158, 11, 0.35)',
            background: 'rgba(245, 158, 11, 0.08)',
          }}
        >
          <IconAlertTriangle
            className='mt-0.5 flex-shrink-0'
            style={{ color: 'var(--semi-color-warning)' }}
          />
          <div className='flex flex-col gap-1'>
            <Text strong>{t('IP 规则是父级，命中账号在展开区处理')}</Text>
            <Text type='tertiary' size='small'>
              {t('拉黑 IP 不会自动禁用账号；用户列表只显示“已拉黑”标记，禁用或注销需要在这里手动执行。')}
            </Text>
          </div>
        </div>

        <div
          className='rounded-2xl border p-3 md:p-4'
          style={{ borderColor: 'var(--semi-color-border)' }}
        >
          <div className='grid grid-cols-1 lg:grid-cols-[minmax(260px,1fr)_minmax(280px,1fr)_auto] gap-2'>
            <Input
              value={ip}
              onChange={setIp}
              placeholder={t('IP 或 CIDR，例如 1.2.3.4 或 2001:db8::/64')}
              showClear
            />
            <TextArea
              value={reason}
              onChange={setReason}
              placeholder={t('原因')}
              autosize={{ minRows: 1, maxRows: 2 }}
            />
            <Button
              type='primary'
              theme='solid'
              loading={adding}
              onClick={() => addItem(false)}
            >
              {t('添加规则')}
            </Button>
          </div>
        </div>

        <div className='flex flex-col md:flex-row gap-2 md:items-center md:justify-between'>
          <Input
            prefix={<IconSearch />}
            value={keyword}
            onChange={setKeyword}
            onEnterPress={() => loadItems(1, pageSize)}
            placeholder={t('搜索 IP、CIDR 或原因')}
            showClear
            className='w-full md:w-96'
          />
          <Space>
            <Button type='tertiary' onClick={() => loadItems(1, pageSize)}>
              {t('查询')}
            </Button>
            <Button
              type='tertiary'
              onClick={() => {
                setKeyword('');
                loadItems(1, pageSize, '');
              }}
            >
              {t('重置')}
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={items}
          loading={loading}
          rowKey='id'
          size='middle'
          className='rounded-2xl overflow-hidden'
          scroll={{ x: 'max-content' }}
          expandedRowRender={renderRuleUsers}
          rowExpandable={() => true}
          onExpand={(expanded, record) => {
            if (!expanded || !record?.id) {
              return;
            }
            const state = getRuleUsersState(record.id);
            if (!state.loaded) {
              loadRuleUsers(record.id, 1, state.pageSize);
            }
          }}
          pagination={{
            currentPage: page,
            pageSize,
            total,
            pageSizeOpts: [10, 20, 50, 100],
            showSizeChanger: true,
            onPageChange: (nextPage) => loadItems(nextPage, pageSize),
            onPageSizeChange: (nextPageSize) => loadItems(1, nextPageSize),
          }}
        />
      </div>
    </SideSheet>
  );
};

export default IPBlacklistModal;
