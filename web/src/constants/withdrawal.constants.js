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

export const AFF_WITHDRAWAL_MINIMUM_QUOTA = 250;
export const AFF_WITHDRAWAL_MINIMUM_PAYMENT_CNY = 50;

export const calculateAffWithdrawalPaymentAmount = (
  quota,
  quotaPerUnit,
  price,
) => {
  const numericQuota = Number(quota);
  const numericQuotaPerUnit = Number(quotaPerUnit);
  const numericPrice = Number(price);
  if (
    !Number.isFinite(numericQuota) ||
    !Number.isFinite(numericQuotaPerUnit) ||
    !Number.isFinite(numericPrice) ||
    numericQuota <= 0 ||
    numericQuotaPerUnit <= 0 ||
    numericPrice <= 0
  ) {
    return 0;
  }
  return (numericQuota / numericQuotaPerUnit) * numericPrice;
};

export const meetsAffWithdrawalMinimum = (quota, quotaPerUnit, price) => {
  const numericQuota = Number(quota);
  if (
    !Number.isInteger(numericQuota) ||
    numericQuota < AFF_WITHDRAWAL_MINIMUM_QUOTA
  ) {
    return false;
  }
  const paymentAmount = calculateAffWithdrawalPaymentAmount(
    numericQuota,
    quotaPerUnit,
    price,
  );
  return paymentAmount + Number.EPSILON >= AFF_WITHDRAWAL_MINIMUM_PAYMENT_CNY;
};
