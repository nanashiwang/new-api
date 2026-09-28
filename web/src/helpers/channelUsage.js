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

// Accounting windows use Asia/Shanghai and an exclusive end, independent of browser timezone.
export const MAX_USAGE_RANGE = 30 * 86400;
export function usagePresetRange(preset, now = Math.floor(Date.now() / 1000)) {
  const midnight = Math.floor((now + 28800) / 86400) * 86400 - 28800;
  if (preset === 'today') return [midnight, now];
  if (preset === 'yesterday') return [midnight - 86400, midnight];
  return [now - (preset === '7d' ? 7 * 86400 : 86400), now];
}
export function validUsageRange(range, now = Math.floor(Date.now() / 1000)) {
  return (
    range?.length === 2 &&
    range.every(Number.isInteger) &&
    range[0] > 0 &&
    range[1] > range[0] &&
    range[1] - range[0] <= MAX_USAGE_RANGE &&
    range[1] <= now + 1
  );
}
export function formatUsageTime(timestamp) {
  return new Date(timestamp * 1000 + 28800000)
    .toISOString()
    .slice(0, 19)
    .replace('T', ' ');
}
export function parseUsageTime(value) {
  if (!/^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}$/.test(value)) return NaN;
  const timestamp = Math.floor(
    Date.parse(value.replace(' ', 'T') + '+08:00') / 1000,
  );
  return Number.isFinite(timestamp) &&
    formatUsageTime(timestamp) === value.replace('T', ' ')
    ? timestamp
    : NaN;
}
export function initialChannelUsage(search) {
  const q = new URLSearchParams(search);
  const range = [Number(q.get('start_time')), Number(q.get('end_time'))];
  return {
    enabled: q.get('view') === 'usage',
    range: validUsageRange(range) ? range : usagePresetRange('today'),
    preset: validUsageRange(range) ? 'custom' : 'today',
    order: 'desc',
  };
}
export function sumChannelUsage(children) {
  return children.reduce(
    (sum, ch) => {
      for (const key of [
        'total_quota',
        'total_requests',
        'success_requests',
        'failed_requests',
      ])
        sum[key] += ch.usage?.[key] || 0;
      return sum;
    },
    {
      total_quota: 0,
      total_requests: 0,
      success_requests: 0,
      failed_requests: 0,
    },
  );
}
export function channelUsageLogURL(record, metadata) {
  const q = new URLSearchParams({
    source: 'channel-usage',
    start_time: metadata.start_time,
    end_time: metadata.end_time,
    model: metadata.model
      ? '%' + metadata.model.replace(/^%+|%+$/g, '') + '%'
      : '',
    group: metadata.group || '',
  });
  q.set(
    'channel_ids',
    (record.children || [record]).map((ch) => ch.id).join(','),
  );
  return '/console/log?' + q;
}
