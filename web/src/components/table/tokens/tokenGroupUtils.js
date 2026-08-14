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

export const TOKEN_GROUP_OTHER_VENDOR = '__other__';

const CANONICAL_VENDOR_NAMES = new Map([
  ['openai', 'OpenAI'],
  ['claude', 'Claude'],
  ['anthropic', 'Anthropic'],
  ['gemini', 'Gemini'],
  ['google', 'Google'],
  ['grok', 'Grok'],
  ['xai', 'xAI'],
  ['kimi', 'Kimi'],
  ['moonshot', 'Moonshot'],
  ['deepseek', 'DeepSeek'],
]);

const normalizeVendorName = (value) => {
  const normalized = String(value || '').trim();
  if (!normalized) return '';
  return CANONICAL_VENDOR_NAMES.get(normalized.toLowerCase()) || normalized;
};

export const resolveTokenGroupVendor = (groupName) => {
  const normalizedGroupName = String(groupName || '').trim();
  if (!normalizedGroupName) return '';

  const separatorIndex = normalizedGroupName.indexOf('·');
  if (separatorIndex > 0) {
    return normalizeVendorName(normalizedGroupName.slice(0, separatorIndex));
  }

  const canonicalVendor = CANONICAL_VENDOR_NAMES.get(
    normalizedGroupName.toLowerCase(),
  );
  return canonicalVendor || TOKEN_GROUP_OTHER_VENDOR;
};

export const buildTokenGroupVendorOptions = (
  groups = [],
  otherLabel = '其他',
) => {
  const vendorCounts = new Map();

  groups.forEach((group) => {
    const vendor = resolveTokenGroupVendor(group?.value);
    if (!vendor) return;
    vendorCounts.set(vendor, (vendorCounts.get(vendor) || 0) + 1);
  });

  return Array.from(vendorCounts.entries())
    .sort(([leftVendor], [rightVendor]) => {
      if (leftVendor === TOKEN_GROUP_OTHER_VENDOR) return 1;
      if (rightVendor === TOKEN_GROUP_OTHER_VENDOR) return -1;
      return 0;
    })
    .map(([vendor, count]) => ({
      value: vendor,
      label: `${vendor === TOKEN_GROUP_OTHER_VENDOR ? otherLabel : vendor} · ${count}`,
    }));
};

export const filterTokenGroupsByVendor = (groups = [], vendor = '') => {
  if (!vendor) return [];
  return groups.filter(
    (group) => resolveTokenGroupVendor(group?.value) === vendor,
  );
};

export const shouldClearTokenGroupForVendor = (groupName, vendor) => {
  const normalizedGroupName = String(groupName || '').trim();
  if (!normalizedGroupName) return false;
  return resolveTokenGroupVendor(normalizedGroupName) !== vendor;
};

export const formatTokenGroupSelectedLabel = (option) => {
  const groupName = String(option?.value || option?.label || '').trim();
  if (!groupName) return '';

  const ratio = option?.ratio;
  if (ratio === undefined || ratio === null || ratio === '') {
    return groupName;
  }

  const ratioText =
    typeof ratio === 'number' || /^\d+(\.\d+)?$/.test(String(ratio))
      ? `${ratio}x`
      : String(ratio);
  return `${groupName} · ${ratioText}`;
};
