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

export function normalizePaymentCurrency(currency) {
  const normalized = String(currency || '')
    .trim()
    .toUpperCase();
  if (normalized === 'RMB') return 'CNY';
  return /^[A-Z]{3}$/.test(normalized) ? normalized : '';
}

export function getLegacyPaymentCurrency(paymentMethod) {
  const normalizedMethod = String(paymentMethod || '')
    .trim()
    .toLowerCase();
  if (normalizedMethod === 'stripe') return 'USD';
  if (normalizedMethod === 'waffo' || normalizedMethod.startsWith('waffo:')) {
    return '';
  }
  return normalizedMethod ? 'CNY' : '';
}

export function createPaymentQuote(
  rawAmount,
  rawCurrency,
  paymentMethod,
  estimated = true,
) {
  const amount = Number(rawAmount);
  if (!Number.isFinite(amount) || amount <= 0) return null;

  const hasCurrency = String(rawCurrency || '').trim() !== '';
  const currency = hasCurrency
    ? normalizePaymentCurrency(rawCurrency)
    : getLegacyPaymentCurrency(paymentMethod);

  return {
    amount,
    currency,
    estimated: estimated !== false,
  };
}

export function formatPaymentMoney(amount, currency, locale = 'zh-CN') {
  const numericAmount = Number(amount || 0);
  const safeAmount = Number.isFinite(numericAmount) ? numericAmount : 0;
  const normalizedCurrency = normalizePaymentCurrency(currency);

  if (!normalizedCurrency) {
    return safeAmount.toLocaleString(locale || 'zh-CN', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  }

  try {
    return new Intl.NumberFormat(locale || 'zh-CN', {
      style: 'currency',
      currency: normalizedCurrency,
    }).format(safeAmount);
  } catch (_) {
    return `${normalizedCurrency} ${safeAmount.toFixed(2)}`;
  }
}
