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

// 水墨山峦剪影：多层远近山形，由远到近墨色加深，靠 CSS 变量 --ink-mountain-*
// 取色以适配明暗主题。纯内联 SVG，无外部资源，preserveAspectRatio 让其
// 始终贴底铺满 hero 宽度。装饰性元素，aria-hidden。
const InkMountains = () => (
  <svg
    className='ink-mountains'
    viewBox='0 0 1440 320'
    preserveAspectRatio='none'
    aria-hidden='true'
    focusable='false'
  >
    {/* 最远层：淡墨远山 */}
    <path
      className='ink-mountain ink-mountain--far'
      d='M0 220 L120 180 L240 205 L360 150 L480 195 L600 140 L720 185 L840 130 L960 180 L1080 145 L1200 190 L1320 160 L1440 200 L1440 320 L0 320 Z'
    />
    {/* 中景层 */}
    <path
      className='ink-mountain ink-mountain--mid'
      d='M0 260 L160 215 L300 250 L440 195 L560 245 L700 190 L860 250 L1000 205 L1160 255 L1300 210 L1440 250 L1440 320 L0 320 Z'
    />
    {/* 近景层：浓墨主峰 */}
    <path
      className='ink-mountain ink-mountain--near'
      d='M0 300 L180 250 L360 295 L520 245 L680 290 L820 255 L1000 298 L1180 260 L1340 298 L1440 270 L1440 320 L0 320 Z'
    />
  </svg>
);

export default InkMountains;
