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

const TOKEN_GROUP_VENDOR_ORDER = [
  'OpenAI',
  'Claude',
  'Anthropic',
  'Gemini',
  'Google',
  'Grok',
  'xAI',
  'Kimi',
  'Moonshot',
  'DeepSeek',
  'MiMo',
];

const TOKEN_GROUP_TIER_ORDER = ['企业专属', '优质', '标准', '特价', '限时福利'];

const TOKEN_GROUP_OTHER_ORDER = [
  'auto',
  'default',
  '官方渠道',
  '第三方渠道',
  '企业专属',
  '分销商',
];

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
  ['mimo', 'MiMo'],
  ['xiaomi', 'MiMo'],
]);

const compareByKnownOrder = (left, right, orderedValues) => {
  const leftIndex = orderedValues.indexOf(left);
  const rightIndex = orderedValues.indexOf(right);
  if (leftIndex >= 0 && rightIndex >= 0) return leftIndex - rightIndex;
  if (leftIndex >= 0) return -1;
  if (rightIndex >= 0) return 1;
  return String(left).localeCompare(String(right), 'zh-Hans');
};

const resolveTokenGroupTier = (groupName) =>
  TOKEN_GROUP_TIER_ORDER.find((tier) => String(groupName).includes(tier)) || '';

const isThirdPartyGroup = (groupName) =>
  /第三方|third[\s_-]*party/i.test(String(groupName));

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
      return compareByKnownOrder(
        leftVendor,
        rightVendor,
        TOKEN_GROUP_VENDOR_ORDER,
      );
    })
    .map(([vendor, count]) => ({
      value: vendor,
      label: `${vendor === TOKEN_GROUP_OTHER_VENDOR ? otherLabel : vendor} · ${count}`,
    }));
};

export const sortTokenGroupOptions = (groups = []) =>
  [...groups].sort((left, right) => {
    const leftName = String(left?.value || left?.label || '').trim();
    const rightName = String(right?.value || right?.label || '').trim();
    const leftVendor = resolveTokenGroupVendor(leftName);
    const rightVendor = resolveTokenGroupVendor(rightName);

    const vendorOrder = compareByKnownOrder(
      leftVendor,
      rightVendor,
      TOKEN_GROUP_VENDOR_ORDER,
    );
    if (vendorOrder !== 0) return vendorOrder;

    if (leftVendor === TOKEN_GROUP_OTHER_VENDOR) {
      const otherOrder = compareByKnownOrder(
        leftName,
        rightName,
        TOKEN_GROUP_OTHER_ORDER,
      );
      if (otherOrder !== 0) return otherOrder;
    } else {
      const tierOrder = compareByKnownOrder(
        resolveTokenGroupTier(leftName),
        resolveTokenGroupTier(rightName),
        TOKEN_GROUP_TIER_ORDER,
      );
      if (tierOrder !== 0) return tierOrder;

      const sourceOrder =
        Number(isThirdPartyGroup(leftName)) -
        Number(isThirdPartyGroup(rightName));
      if (sourceOrder !== 0) return sourceOrder;
    }

    return leftName.localeCompare(rightName, 'zh-Hans');
  });

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
