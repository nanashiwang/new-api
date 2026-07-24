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
  Typography,
} from '@douyinfe/semi-ui';
import { IconMore } from '@douyinfe/semi-icons';
import {
  renderGroup,
  renderNumber,
  renderQuota,
  timestamp2string,
} from '../../../helpers';

const { Text } = Typography;

const renderTimestamp = (text) => (text ? timestamp2string(text) : '-');

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

const getRegisterSourceMeta = (source, t) => {
  const raw = typeof source === 'string' ? source.trim() : '';
  const normalized = raw || 'unknown';
  if (normalized.startsWith('custom_oauth:')) {
    const provider = normalized.slice('custom_oauth:'.length) || t('未知来源');
    return {
      label: `${t('OAuth 注册')}: ${provider}`,
      color: 'light-blue',
      raw: normalized,
    };
  }
  switch (normalized) {
    case 'password':
      return { label: t('密码注册'), color: 'blue', raw: normalized };
    case 'admin':
      return { label: t('管理员创建'), color: 'green', raw: normalized };
    case 'github':
      return { label: 'GitHub', color: 'light-blue', raw: normalized };
    case 'discord':
      return { label: 'Discord', color: 'light-blue', raw: normalized };
    case 'oidc':
      return { label: 'OIDC', color: 'light-blue', raw: normalized };
    case 'linuxdo':
      return { label: 'LinuxDO', color: 'light-blue', raw: normalized };
    case 'wechat':
      return { label: 'WeChat', color: 'light-blue', raw: normalized };
    default:
      return { label: t('未知来源'), color: 'grey', raw: normalized };
  }
};

const renderRegisterSource = (source, record, t) => {
  const meta = getRegisterSourceMeta(source, t);
  const tooltipContent = (
    <div className='text-xs max-w-xs break-all'>
      <div>
        {t('注册来源')}: {meta.raw}
      </div>
      <div>
        {t('注册 IP')}: {record?.register_ip || '-'}
      </div>
      {record?.register_user_agent ? (
        <div>User-Agent: {record.register_user_agent}</div>
      ) : null}
    </div>
  );
  return (
    <Tooltip content={tooltipContent} position='top'>
      <Tag color={meta.color} shape='circle'>
        {meta.label}
      </Tag>
    </Tooltip>
  );
};

const renderRegisterIP = (ip, record, t) => {
  const value = typeof ip === 'string' && ip.trim() ? ip.trim() : '-';
  if (value === '-') {
    return (
      <Tag color='white' shape='circle'>
        -
      </Tag>
    );
  }
  const blacklistTooltip = (
    <div className='text-xs max-w-xs break-all'>
      <div>{t('注册 IP 已拉黑')}</div>
      <div>
        {t('命中规则')}: {record?.register_ip_blacklist_cidr || '-'}
      </div>
      {record?.register_ip_blacklist_reason ? (
        <div>
          {t('原因')}: {record.register_ip_blacklist_reason}
        </div>
      ) : null}
    </div>
  );
  return (
    <Space spacing={4} wrap>
      <Tooltip content={record?.register_user_agent || value} position='top'>
        <Tag color='white' shape='circle'>
          {value}
        </Tag>
      </Tooltip>
      {record?.register_ip_blacklisted ? (
        <Tooltip content={blacklistTooltip} position='top'>
          <Tag color='red' shape='circle'>
            {t('已拉黑')}
          </Tag>
        </Tooltip>
      ) : null}
    </Space>
  );
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

const renderEmail = (email) => {
  const value = typeof email === 'string' ? email.trim() : '';
  if (!value) {
    return <span>-</span>;
  }
  return (
    <Text copyable ellipsis={{ showTooltip: true }} style={{ maxWidth: 220 }}>
      {value}
    </Text>
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
      {record?.register_ip_blacklisted ? (
        <div>
          {t('注册 IP 已拉黑')}: {record.register_ip_blacklist_cidr || '-'}
        </div>
      ) : null}
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      {content}
    </Tooltip>
  );
};

const renderContentSafety = (record, t) => {
  const level = record?.content_safety_level || 'normal';
  const count = Number(record?.content_safety_count || 0);
  const metaByLevel = {
    normal: { label: t('正常'), color: 'grey' },
    warning_1: { label: t('警告 1/3'), color: 'yellow' },
    warning_2: { label: t('警告 2/3'), color: 'orange' },
    final_warning: { label: t('最终警告 3/3'), color: 'red' },
    disabled: { label: t('风控停用'), color: 'red' },
    review_required: { label: t('待复核'), color: 'orange' },
  };
  const meta = metaByLevel[level] || metaByLevel.review_required;
  const tooltipContent = (
    <div className='text-xs max-w-sm break-all'>
      <div className='font-semibold mb-1'>
        {count > 0
          ? t('已确认的上游内容安全拒绝')
          : t('最近30天无上游内容安全拒绝记录')}
      </div>
      <div>{t('最近30天触发次数')}: {count}</div>
      <div>{t('最近触发时间')}: {renderTimestamp(record?.content_safety_last_at)}</div>
      <div>{t('最近错误码')}: {record?.content_safety_last_code || '-'}</div>
      <div>{t('最近使用模型')}: {record?.content_safety_last_model || '-'}</div>
      <div>{t('最近渠道 ID')}: {record?.content_safety_last_channel_id || '-'}</div>
      <div>{t('最近请求 ID')}: {record?.content_safety_last_request_id || '-'}</div>
      <div>{t('当前处理状态')}: {meta.label}</div>
      <div className='mt-1 opacity-80'>
        {t('该记录表示上游明确拒绝，不代表已判定用户主观恶意。')}
      </div>
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      <Tag
        color={meta.color}
        shape='circle'
        size='small'
        style={
          level === 'disabled'
            ? { backgroundColor: '#7f1d1d', color: '#fff' }
            : undefined
        }
      >
        {meta.label}
      </Tag>
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
    blacklistUserIP,
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
      node: 'divider',
    },
    {
      node: 'item',
      name: t('拉黑注册 IP'),
      disabled: !record?.register_ip,
      type: 'danger',
      onClick: () => {
        if (record?.register_ip) {
          blacklistUserIP?.(record);
        }
      },
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
  showInviteRelationsModal,
  openInviteRelationsUser,
  blacklistUserIP,
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
      title: t('邮箱'),
      dataIndex: 'email',
      width: 240,
      render: (text) => renderEmail(text),
    },
    {
      title: t('注册来源'),
      dataIndex: 'register_source',
      render: (text, record) => renderRegisterSource(text, record, t),
    },
    {
      title: t('注册 IP'),
      dataIndex: 'register_ip',
      render: (text, record) => renderRegisterIP(text, record, t),
    },
    {
      title: t('状态'),
      dataIndex: 'info',
      render: (text, record, index) =>
        renderStatistics(text, record, showEnableDisableModal, t),
    },
    {
      title: t('内容风控'),
      dataIndex: 'content_safety_level',
      key: 'content_safety_level',
      render: (text, record) => renderContentSafety(record, t),
    },
    {
      title: t('套餐情况'),
      dataIndex: 'subscription_status',
      key: 'subscription_status',
      render: (text, record) =>
        renderSubscriptionStatus(record, t, showUserSubscriptionsModal),
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
      title: t('创建时间'),
      dataIndex: 'created_at',
      render: renderTimestamp,
    },
    {
      title: t('最后登录'),
      dataIndex: 'last_login_at',
      render: renderTimestamp,
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
          blacklistUserIP,
          t,
        }),
    },
  ];
};
