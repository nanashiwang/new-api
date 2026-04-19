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
import {
  Button,
  Space,
  Tag,
  Tooltip,
  Dropdown,
} from '@douyinfe/semi-ui';
import { IconMore } from '@douyinfe/semi-icons';
import { renderGroup, renderNumber, renderQuota } from '../../../helpers';

/**
 * 渲染用户角色。
 */
const renderRole = (role, t) => {
  switch (role) {
    case 1:
      return (
        <Tag color='blue' shape='circle'>
          {t('普通用户')}
        </Tag>
      );
    case 10:
      return (
        <Tag color='yellow' shape='circle'>
          {t('管理员')}
        </Tag>
      );
    case 100:
      return (
        <Tag color='orange' shape='circle'>
          {t('超级管理员')}
        </Tag>
      );
    default:
      return (
        <Tag color='red' shape='circle'>
          {t('未知身份')}
        </Tag>
      );
  }
};

/**
 * 渲染用户名，存在备注时一并展示。
 */
const renderUsername = (text, record) => {
  const fallback = record?.id ? `#${record.id}` : '';
  const username = typeof text === 'string' ? text.trim() : text;
  const displayNameField =
    typeof record?.display_name === 'string' ? record.display_name.trim() : '';
  const resolved = username || displayNameField || fallback;
  const remark = record?.remark;
  if (!remark) {
    return <span>{resolved}</span>;
  }
  const maxLen = 10;
  const displayRemark =
    remark.length > maxLen ? remark.slice(0, maxLen) + '…' : remark;
  return (
    <Space spacing={2}>
      <span>{resolved}</span>
      <Tooltip content={remark} position='top' showArrow>
        <Tag color='white' shape='circle' className='!text-xs'>
          <div className='flex items-center gap-1'>
            <div
              className='w-2 h-2 flex-shrink-0 rounded-full'
              style={{ backgroundColor: '#10b981' }}
            />
            {displayRemark}
          </div>
        </Tag>
      </Tooltip>
    </Space>
  );
};

/**
 * 渲染用户状态和调用次数。
 */
const renderStatistics = (text, record, showEnableDisableModal, t) => {
  const isDeleted = record.DeletedAt !== null;

  // 参考原状态列规则确定标签文案与颜色
  let tagColor = 'grey';
  let tagText = t('未知状态');
  if (isDeleted) {
    tagColor = 'red';
    tagText = t('已注销');
  } else if (record.status === 1) {
    tagColor = 'green';
    tagText = t('已启用');
  } else if (record.status === 2) {
    tagColor = 'red';
    tagText = t('已禁用');
  }

  const content = (
    <Tag color={tagColor} shape='circle' size='small'>
      {tagText}
    </Tag>
  );

  const tooltipContent = (
    <div className='text-xs'>
      <div>
        {t('调用次数')}: {renderNumber(record.request_count)}
      </div>
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      {content}
    </Tooltip>
  );
};

const renderWalletQuota = (text, record) => {
  const walletQuota = Number(record?.quota || 0);
  return (
    <Tag color='light-blue' shape='circle'>
      {renderQuota(walletQuota)}
    </Tag>
  );
};

const renderSubscriptionQuota = (text, record, t) => {
  const hasUnlimited = !!record?.subscription_quota_has_unlimited;
  const total = Number(record?.subscription_quota_total || 0);
  const remaining = Number(record?.subscription_quota_remaining || 0);
  const items = record?.subscription_quota_items || [];

  if (hasUnlimited) {
    return (
      <Tag color='white' shape='circle'>
        {t('不限额')}
      </Tag>
    );
  }

  if (total <= 0) {
    return (
      <Tag color='white' shape='circle'>
        -
      </Tag>
    );
  }

  const pct = Math.min(100, Math.max(0, (remaining / total) * 100));
  const barColor =
    pct > 30 ? '#10b981' : pct > 10 ? '#f59e0b' : '#ef4444';

  const formatResetPeriod = (period) => {
    const map = {
      never: t('不重置'),
      daily: t('每天'),
      weekly: t('每周'),
      monthly: t('每月'),
      custom: t('自定义'),
    };
    return map[period] || period || t('不重置');
  };

  const formatDate = (ts) => {
    if (!ts) return '-';
    return new Date(ts * 1000).toLocaleDateString();
  };

  const tooltipContent = (
    <div style={{ minWidth: 180, maxWidth: 260 }}>
      {items.map((item, idx) => {
        const planName = item.plan_title || `#${idx + 1}`;
        return (
          <div
            key={idx}
            style={{
              paddingBottom: idx < items.length - 1 ? 8 : 0,
              marginBottom: idx < items.length - 1 ? 8 : 0,
              borderBottom:
                idx < items.length - 1 ? '1px solid rgba(255,255,255,0.1)' : 'none',
            }}
          >
            <div style={{ fontWeight: 600, marginBottom: 4, fontSize: 12 }}>
              {planName}
            </div>
            <div style={{ fontSize: 11, opacity: 0.85, lineHeight: 1.6 }}>
              {item.has_unlimited ? (
                <div>{t('额度')}: {t('不限额')}</div>
              ) : (
                <div>
                  {t('额度')}: {renderQuota(item.remaining)} / {renderQuota(item.total)}
                </div>
              )}
              <div>{t('刷新周期')}: {formatResetPeriod(item.reset_period)}</div>
              {item.next_reset_time > 0 && (
                <div>{t('下次刷新')}: {formatDate(item.next_reset_time)}</div>
              )}
            </div>
          </div>
        );
      })}
      {items.length === 0 && (
        <div style={{ fontSize: 11 }}>
          {t('套餐额度')}: {renderQuota(remaining)} / {renderQuota(total)}
        </div>
      )}
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      <Tag color='white' shape='circle' style={{ cursor: 'default' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
          <span style={{ fontSize: 12, whiteSpace: 'nowrap' }}>
            {renderQuota(remaining)} / {renderQuota(total)}
          </span>
          <div
            style={{
              width: '100%',
              height: 3,
              borderRadius: 2,
              background: 'rgba(0,0,0,0.12)',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                width: `${pct}%`,
                height: '100%',
                borderRadius: 2,
                background: barColor,
                transition: 'width 0.3s ease',
              }}
            />
          </div>
        </div>
      </Tag>
    </Tooltip>
  );
};

// 已用额度单独展示，查余额时更直观。
const renderUsedQuota = (text, record) => {
  const used = parseInt(record.used_quota) || 0;
  return (
    <Tag color='white' shape='circle'>
      {renderQuota(used)}
    </Tag>
  );
};

/**
 * 渲染订阅状态，没有生效订阅时也允许直接点进去管理。
 */
const renderSubscriptionStatus = (record, t, showUserSubscriptionsModal) => {
  const activeCount = Number(record?.active_subscription_count || 0);
  const pendingCount = Number(record?.pending_subscription_issuance_count || 0);
  const hasActiveSubscription =
    record?.has_active_subscription || activeCount > 0 || pendingCount > 0;
  const dotColor = hasActiveSubscription ? '#10b981' : '#94a3b8';
  const label = hasActiveSubscription
    ? `${t('有套餐')} · ${activeCount}${pendingCount > 0 ? ` + ${pendingCount}${t('待发放')}` : ''}`
    : t('无套餐');
  const isDeleted = record?.DeletedAt !== null;
  const content = (
    <Tag color='white' shape='circle'>
      <div className='flex items-center gap-1'>
        <div
          className='w-2 h-2 rounded-full flex-shrink-0'
          style={{ backgroundColor: dotColor }}
        />
        <span className='text-xs'>{label}</span>
      </div>
    </Tag>
  );

  if (isDeleted) {
    return content;
  }

  return (
    <Button
      type='tertiary'
      theme='borderless'
      size='small'
      className='!px-0 cursor-pointer'
      onClick={() => showUserSubscriptionsModal?.(record)}
    >
      {content}
    </Button>
  );
};

const renderSellableTokenStatus = (record, t, showUserSellableTokensModal) => {
  const activeCount = Number(record?.active_sellable_token_count || 0);
  const pendingCount = Number(record?.pending_sellable_issuance_count || 0);
  const hasTokens = record?.has_sellable_token || activeCount > 0 || pendingCount > 0;
  const label = hasTokens
    ? `${t('有令牌')} · ${activeCount}${pendingCount > 0 ? ` + ${pendingCount}${t('待发放')}` : ''}`
    : t('无令牌');
  const dotColor = hasTokens ? '#06b6d4' : '#94a3b8';
  const content = (
    <Tag color='white' shape='circle'>
      <div className='flex items-center gap-1'>
        <div
          className='w-2 h-2 rounded-full flex-shrink-0'
          style={{ backgroundColor: dotColor }}
        />
        <span className='text-xs'>{label}</span>
      </div>
    </Tag>
  );
  if (record?.DeletedAt !== null) {
    return content;
  }
  return (
    <Button
      type='tertiary'
      theme='borderless'
      size='small'
      className='!px-0 cursor-pointer'
      onClick={() => showUserSellableTokensModal?.(record)}
    >
      {content}
    </Button>
  );
};

/**
 * 渲染邀请相关信息。
 */
const renderInviteInfo = (
  text,
  record,
  t,
  showInviteRelationsModal,
  openInviteRelationsUser,
) => {
  const inviterText =
    record.inviter_id === 0
      ? t('无邀请人')
      : `${t('邀请人')}: ${record.inviter_id}${
          record.inviter_username ? ` (${record.inviter_username})` : ''
        }`;
  return (
    <div>
      <Space spacing={1}>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('邀请')}: {renderNumber(record.aff_count)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('收益')}: {renderQuota(record.aff_history_quota)}
        </Tag>
        {record.inviter_id > 0 ? (
          <Button
            type='tertiary'
            size='small'
            theme='borderless'
            className='!px-0'
            onClick={() =>
              openInviteRelationsUser?.({
                id: record.inviter_id,
                username: record.inviter_username || '',
              })
            }
          >
            <Tag color='white' shape='circle' className='!text-xs cursor-pointer'>
              {inviterText}
            </Tag>
          </Button>
        ) : (
          <Tag color='white' shape='circle' className='!text-xs'>
            {inviterText}
          </Tag>
        )}
        <Button
          type='tertiary'
          size='small'
          theme='borderless'
          onClick={() => showInviteRelationsModal?.(record)}
        >
          {t('查看关系')}
        </Button>
      </Space>
    </div>
  );
};

/**
 * 渲染操作列。
 */
const renderOperations = (
  text,
  record,
  {
    setEditingUser,
    setShowEditUser,
    showPromoteModal,
    showDemoteModal,
    showEnableDisableModal,
    showDeleteModal,
    showResetPasskeyModal,
    showResetTwoFAModal,
    showUserSubscriptionsModal,
    showUserSellableTokensModal,
    t,
  },
) => {
  if (record.DeletedAt !== null) {
    return <></>;
  }

  const moreMenu = [
    {
      node: 'item',
      name: t('订阅管理'),
      onClick: () => showUserSubscriptionsModal(record),
    },
    {
      node: 'item',
      name: t('令牌情况'),
      onClick: () => showUserSellableTokensModal(record),
    },
    {
      node: 'divider',
    },
    {
      node: 'item',
      name: t('重置 Passkey'),
      onClick: () => showResetPasskeyModal(record),
    },
    {
      node: 'item',
      name: t('重置 2FA'),
      onClick: () => showResetTwoFAModal(record),
    },
    {
      node: 'divider',
    },
    {
      node: 'item',
      name: t('注销'),
      type: 'danger',
      onClick: () => showDeleteModal(record),
    },
  ];

  return (
    <Space>
      {record.status === 1 ? (
        <Button
          type='danger'
          size='small'
          onClick={() => showEnableDisableModal(record, 'disable')}
        >
          {t('禁用')}
        </Button>
      ) : (
        <Button
          size='small'
          onClick={() => showEnableDisableModal(record, 'enable')}
        >
          {t('启用')}
        </Button>
      )}
      <Button
        type='tertiary'
        size='small'
        onClick={() => {
          setEditingUser(record);
          setShowEditUser(true);
        }}
      >
        {t('编辑')}
      </Button>
      <Button
        type='warning'
        size='small'
        onClick={() => showPromoteModal(record)}
      >
        {t('提升')}
      </Button>
      <Button
        type='secondary'
        size='small'
        onClick={() => showDemoteModal(record)}
      >
        {t('降级')}
      </Button>
      <Dropdown menu={moreMenu} trigger='click' position='bottomRight'>
        <Button type='tertiary' size='small' icon={<IconMore />} />
      </Dropdown>
    </Space>
  );
};

/**
 * Get users table column definitions
 */
export const getUsersColumns = ({
  t,
  setEditingUser,
  setShowEditUser,
  showPromoteModal,
  showDemoteModal,
  showEnableDisableModal,
  showDeleteModal,
  showResetPasskeyModal,
  showResetTwoFAModal,
  showUserSubscriptionsModal,
  showUserSellableTokensModal,
  showInviteRelationsModal,
  openInviteRelationsUser,
}) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      render: (text, record) => renderUsername(text, record),
    },
    {
      title: t('状态'),
      dataIndex: 'info',
      render: (text, record, index) =>
        renderStatistics(text, record, showEnableDisableModal, t),
    },
    {
      title: t('套餐情况'),
      dataIndex: 'subscription_status',
      key: 'subscription_status',
      render: (text, record) =>
        renderSubscriptionStatus(record, t, showUserSubscriptionsModal),
    },
    {
      title: t('令牌情况'),
      dataIndex: 'sellable_token_status',
      key: 'sellable_token_status',
      render: (text, record) =>
        renderSellableTokenStatus(record, t, showUserSellableTokensModal),
    },
    {
      title: t('钱包额度'),
      dataIndex: 'quota',
      key: 'wallet_quota',
      render: (text, record) => renderWalletQuota(text, record),
    },
    {
      title: t('套餐额度'),
      key: 'subscription_quota',
      render: (text, record) => renderSubscriptionQuota(text, record, t),
    },
    {
      title: t('已使用余额'),
      dataIndex: 'used_quota',
      key: 'used_quota',
      render: (text, record) => renderUsedQuota(text, record),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text, record, index) => {
        return <div>{renderGroup(text)}</div>;
      },
    },
    {
      title: t('角色'),
      dataIndex: 'role',
      render: (text, record, index) => {
        return <div>{renderRole(text, t)}</div>;
      },
    },
    {
      title: t('邀请信息'),
      dataIndex: 'invite',
      render: (text, record, index) =>
        renderInviteInfo(
          text,
          record,
          t,
          showInviteRelationsModal,
          openInviteRelationsUser,
        ),
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      width: 200,
      render: (text, record, index) =>
        renderOperations(text, record, {
          setEditingUser,
          setShowEditUser,
          showPromoteModal,
          showDemoteModal,
          showEnableDisableModal,
          showDeleteModal,
          showResetPasskeyModal,
          showResetTwoFAModal,
          showUserSubscriptionsModal,
          showUserSellableTokensModal,
          t,
        }),
    },
  ];
};
