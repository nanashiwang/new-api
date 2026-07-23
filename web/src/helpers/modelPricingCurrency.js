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

const positiveRate = (value, fallback = 1) => {
  const rate = Number(value);
  return Number.isFinite(rate) && rate > 0 ? rate : fallback;
};

export const resolveModelPricingCurrency = ({
  showWithRecharge,
  quotaDisplayType,
}) => {
  if (showWithRecharge) return 'CNY';
  return quotaDisplayType === 'CUSTOM' ? 'CUSTOM' : 'USD';
};

export const resolveModelPricingRate = ({
  showWithRecharge,
  currency,
  priceConvertMode,
  priceRate,
  packageEffectiveRate,
  customExchangeRate,
}) => {
  if (!showWithRecharge) {
    return currency === 'CUSTOM' ? positiveRate(customExchangeRate) : 1;
  }
  const cnyRate =
    priceConvertMode === 'package' && packageEffectiveRate != null
      ? positiveRate(packageEffectiveRate)
      : positiveRate(priceRate);
  return cnyRate;
};

export const resolveModelPricingSymbol = (currency, customCurrencySymbol) => {
  if (currency === 'CNY') return '¥';
  if (currency === 'CUSTOM') return customCurrencySymbol || '¤';
  return '$';
};
