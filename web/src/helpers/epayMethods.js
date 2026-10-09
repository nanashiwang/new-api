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

export const EPAY_USDT_TRC20 = 'usdt.trc20';

export function parseEpayMethods(value) {
  const methods = typeof value === 'string' ? JSON.parse(value || '[]') : value;
  if (
    !Array.isArray(methods) ||
    methods.some(
      (method) =>
        !method ||
        Array.isArray(method) ||
        typeof method !== 'object' ||
        typeof method.type !== 'string' ||
        Object.values(method).some((item) => typeof item !== 'string'),
    )
  ) {
    throw new Error('Invalid Epay methods');
  }
  return methods;
}

export function isEpayUsdtEnabled(value) {
  try {
    return parseEpayMethods(value).some(
      (method) => method.type === EPAY_USDT_TRC20,
    );
  } catch {
    return false;
  }
}

export function setEpayUsdtEnabled(value, enabled) {
  const methods = [...parseEpayMethods(value)];
  if (!enabled) {
    return JSON.stringify(
      methods.filter((method) => method.type !== EPAY_USDT_TRC20),
      null,
      2,
    );
  }
  if (!methods.some((method) => method.type === EPAY_USDT_TRC20)) {
    methods.push({
      name: 'USDT / TRC20',
      type: EPAY_USDT_TRC20,
      color: '#26A17B',
    });
  }
  return JSON.stringify(methods, null, 2);
}
