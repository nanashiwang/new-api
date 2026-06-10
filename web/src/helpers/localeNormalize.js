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

// 语言归一化的唯一实现，供 i18n/i18n.js（控制台）与 helpers/publicLocale.js
// （公开首页轻量词条）共用。两处此前各有一套规则且不一致：
// 例如未知语言（如 de）首页归 en、控制台归 zh-CN，导致跨页面语言跳变。
// 注意不要在此文件引入 i18next——publicLocale 位于首页入口 chunk，
// 引入会把整个 i18n 库拖进首页关键路径。

export const SUPPORTED_LOCALES = [
  'zh-CN',
  'zh-TW',
  'en',
  'fr',
  'ru',
  'ja',
  'vi',
];

export const normalizeLocale = (lng) => {
  if (!lng) return 'zh-CN';
  if (lng.startsWith('zh-TW') || lng.startsWith('zh-HK')) return 'zh-TW';
  if (lng.startsWith('zh')) return 'zh-CN';
  if (SUPPORTED_LOCALES.includes(lng)) return lng;
  const shortLng = lng.split('-')[0];
  if (SUPPORTED_LOCALES.includes(shortLng)) return shortLng;
  // 未知的非中文语言统一回退英文（而非 zh-CN），与公开首页行为一致
  return 'en';
};
