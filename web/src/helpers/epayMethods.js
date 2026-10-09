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

// Explicit BEpusdt v1.24.2 combinations, shared by configuration and checkout.
export const EPAY_CRYPTO_METHODS = [
  ['USDT', 'trc20', 'TRON (TRC20)', 'USDT / TRC20'],
  ['USDT', 'erc20', 'Ethereum (ERC20)'],
  ['USDT', 'bep20', 'BSC (BEP20)'],
  ['USDT', 'polygon', 'Polygon'],
  ['USDT', 'arbitrum', 'Arbitrum One'],
  ['USDT', 'solana', 'Solana'],
  ['USDC', 'erc20', 'Ethereum (ERC20)'],
  ['USDC', 'bep20', 'BSC (BEP20)'],
  ['USDC', 'polygon', 'Polygon'],
  ['USDC', 'arbitrum', 'Arbitrum One'],
  ['USDC', 'base', 'Base'],
  ['USDC', 'solana', 'Solana'],
].map(([coin, chain, network, name]) => ({
  coin,
  network,
  type: `${coin.toLowerCase()}.${chain}`,
  name: name || `${coin} / ${network}`,
  color: coin === 'USDC' ? '#2775CA' : '#26A17B',
}));

export const getEpayCryptoMethod = (type) =>
  EPAY_CRYPTO_METHODS.find((method) => method.type === type);

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

export function isEpayCryptoEnabled(value, type) {
  try {
    return parseEpayMethods(value).some((method) => method.type === type);
  } catch {
    return false;
  }
}

export function setEpayCryptoEnabled(value, type, enabled) {
  const crypto = getEpayCryptoMethod(type);
  if (!crypto) throw new Error('Unsupported Epay crypto method');
  const methods = [...parseEpayMethods(value)];
  if (!enabled) {
    return JSON.stringify(
      methods.filter((method) => method.type !== type),
      null,
      2,
    );
  }
  if (!methods.some((method) => method.type === type)) {
    methods.push({
      name: crypto.name,
      type,
      color: crypto.color,
    });
  }
  return JSON.stringify(methods, null, 2);
}

export const isEpayUsdtEnabled = (value) =>
  isEpayCryptoEnabled(value, EPAY_USDT_TRC20);
export const setEpayUsdtEnabled = (value, enabled) =>
  setEpayCryptoEnabled(value, EPAY_USDT_TRC20, enabled);
