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
import { Typography } from '@douyinfe/semi-ui';

const { Text } = Typography;

function buildChannelText(record) {
  const channelId = record?.channel ?? '-';
  const channelName = record?.channel_name ? ` (${record.channel_name})` : '';
  return `#${channelId}${channelName}`;
}

function pushIfPresent(lines, label, value) {
  if (value === undefined || value === null || value === '') {
    return;
  }
  lines.push({ label, value });
}

export function renderRouteDecisionContent(record, other, t) {
  const adminInfo = other?.admin_info || {};
  const affinity = adminInfo?.channel_affinity || null;
  const channelChain = Array.isArray(adminInfo?.use_channel)
    ? adminInfo.use_channel.filter(Boolean)
    : [];
  const requestConversion = Array.isArray(other?.request_conversion)
    ? other.request_conversion.filter(Boolean)
    : [];

  const lines = [];

  pushIfPresent(lines, t('最终命中渠道'), buildChannelText(record));
  if (channelChain.length > 0) {
    pushIfPresent(lines, t('渠道链路'), channelChain.join(' -> '));
  }
  pushIfPresent(lines, t('使用分组'), affinity?.using_group || record?.group || '');
  if (affinity?.rule_name) {
    pushIfPresent(lines, t('亲和性规则'), affinity.rule_name);
  }
  if (affinity?.selected_group && affinity.selected_group !== affinity?.using_group) {
    pushIfPresent(lines, t('亲和性选组'), affinity.selected_group);
  }
  if (adminInfo?.is_multi_key) {
    pushIfPresent(
      lines,
      t('多 Key 命中'),
      t('第 {{index}} 个 Key', {
        index: Number(adminInfo.multi_key_index || 0) + 1,
      }),
    );
  }
  if (other?.is_model_mapped && other?.upstream_model_name) {
    pushIfPresent(
      lines,
      t('模型映射'),
      `${record?.model_name || '-'} -> ${other.upstream_model_name}`,
    );
  }
  if (requestConversion.length > 0) {
    pushIfPresent(lines, t('请求格式转换'), requestConversion.join(' -> '));
  }
  pushIfPresent(lines, t('请求路径'), other?.request_path);
  if (record?.type === 5) {
    pushIfPresent(lines, t('错误类型'), other?.error_type);
    pushIfPresent(lines, t('错误代码'), other?.error_code);
    pushIfPresent(lines, t('状态码'), other?.status_code);
    pushIfPresent(lines, t('拦截原因'), other?.reject_reason);
  }

  if (lines.length === 0) {
    return null;
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        maxWidth: 720,
      }}
    >
      {lines.map((item) => (
        <div
          key={`${item.label}-${item.value}`}
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 10,
            lineHeight: 1.6,
          }}
        >
          <Text
            strong
            style={{
              minWidth: 96,
              color: 'var(--semi-color-text-1)',
              flexShrink: 0,
            }}
          >
            {item.label}
          </Text>
          <Text
            style={{
              whiteSpace: 'normal',
              wordBreak: 'break-word',
              color: 'var(--semi-color-text-0)',
            }}
          >
            {String(item.value)}
          </Text>
        </div>
      ))}
    </div>
  );
}
