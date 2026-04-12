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

const priceItemStyle = {
  color: 'var(--semi-color-text-1)',
  background: 'var(--semi-color-fill-0)',
  border: '1px solid var(--semi-color-border)',
  borderRadius: 999,
  padding: '2px 8px',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 4,
  fontSize: 12,
  fontWeight: 500,
  lineHeight: '18px',
  minWidth: 156,
  whiteSpace: 'nowrap',
};

const getDisplayCurrencySymbol = (currency) => {
  if (currency === 'CNY') {
    return '¥';
  }
  if (currency === 'CUSTOM') {
    try {
      const statusStr = localStorage.getItem('status');
      if (statusStr) {
        const status = JSON.parse(statusStr);
        return status?.custom_currency_symbol || '¤';
      }
    } catch (e) {
      return '¤';
    }
    return '¤';
  }
  return '$';
};

const formatTokenUnitPrice = ({
  priceUSD,
  tokenUnit,
  displayPrice,
  currency,
  precision,
}) => {
  const unitDivisor = tokenUnit === 'K' ? 1000 : 1;
  const rawAmount =
    typeof displayPrice?.toAmount === 'function'
      ? displayPrice.toAmount(priceUSD)
      : parseFloat(String(displayPrice(priceUSD)).replace(/[^0-9.]/g, ''));
  const numericValue = rawAmount / unitDivisor;
  const symbol = getDisplayCurrencySymbol(currency);
  const formattedValue = Number.isFinite(numericValue)
    ? numericValue.toFixed(precision).replace(/\.?0+$/, '')
    : '0';

  return `${symbol}${formattedValue}`;
};

export const getModelPricingItems = (priceData) => {
  if (!priceData) {
    return [];
  }

  if (priceData.isPerToken) {
    return priceData.pricingItems || [];
  }

  if (!priceData.price || priceData.price === '-') {
    return [];
  }

  return [
    {
      key: 'fixed',
      label: '模型价格',
      value: priceData.price,
      unitLabel: '',
    },
  ];
};

// 模型定价计算工具函数
export const calculateModelPrice = ({
  record,
  selectedGroup,
  groupRatio,
  tokenUnit,
  displayPrice,
  currency,
  precision = 4,
}) => {
  let usedGroup = selectedGroup;
  let usedGroupRatio = groupRatio[selectedGroup];

  if (selectedGroup === 'all' || usedGroupRatio === undefined) {
    let minRatio = Number.POSITIVE_INFINITY;
    if (
      Array.isArray(record.enable_groups) &&
      record.enable_groups.length > 0
    ) {
      record.enable_groups.forEach((g) => {
        const r = groupRatio[g];
        if (r !== undefined && r < minRatio) {
          minRatio = r;
          usedGroup = g;
          usedGroupRatio = r;
        }
      });
    }

    if (usedGroupRatio === undefined) {
      usedGroupRatio = 1;
    }
  }

  if (record.quota_type === 0) {
    const inputRatioPriceUSD = record.model_ratio * 2 * usedGroupRatio;
    const completionRatioPriceUSD =
      record.model_ratio * record.completion_ratio * 2 * usedGroupRatio;
    const unitLabel = tokenUnit === 'K' ? 'K' : 'M';
    const inputPrice = formatTokenUnitPrice({
      priceUSD: inputRatioPriceUSD,
      tokenUnit,
      displayPrice,
      currency,
      precision,
    });
    const completionPrice = formatTokenUnitPrice({
      priceUSD: completionRatioPriceUSD,
      tokenUnit,
      displayPrice,
      currency,
      precision,
    });

    const pricingItems = [
      {
        key: 'input',
        label: '输入',
        value: inputPrice,
        unitLabel,
      },
      {
        key: 'output',
        label: '输出',
        value: completionPrice,
        unitLabel,
      },
    ];

    if (record.supports_cache_read) {
      pricingItems.push({
        key: 'cacheRead',
        label: '缓存读取',
        value: formatTokenUnitPrice({
          priceUSD:
            record.model_ratio * record.cache_ratio * 2 * usedGroupRatio,
          tokenUnit,
          displayPrice,
          currency,
          precision,
        }),
        unitLabel,
      });
    }

    if (record.supports_cache_creation) {
      pricingItems.push({
        key: 'cacheCreation',
        label: '缓存创建',
        value: formatTokenUnitPrice({
          priceUSD:
            record.model_ratio *
            record.cache_creation_ratio *
            2 *
            usedGroupRatio,
          tokenUnit,
          displayPrice,
          currency,
          precision,
        }),
        unitLabel,
      });
    }

    return {
      inputPrice,
      completionPrice,
      cacheReadPrice:
        pricingItems.find((item) => item.key === 'cacheRead')?.value || null,
      cacheCreationPrice:
        pricingItems.find((item) => item.key === 'cacheCreation')?.value ||
        null,
      pricingItems,
      unitLabel,
      isPerToken: true,
      usedGroup,
      usedGroupRatio,
    };
  }

  if (record.quota_type === 1) {
    const priceUSD = parseFloat(record.model_price) * usedGroupRatio;
    const displayVal = displayPrice(priceUSD);

    return {
      price: displayVal,
      pricingItems: [
        {
          key: 'fixed',
          label: '模型价格',
          value: displayVal,
          unitLabel: '',
        },
      ],
      isPerToken: false,
      usedGroup,
      usedGroupRatio,
    };
  }

  return {
    price: '-',
    isPerToken: false,
    usedGroup,
    usedGroupRatio,
  };
};

// 格式化价格信息（用于卡片视图）
export const formatPriceInfo = (priceData, t) => {
  return getModelPricingItems(priceData).map((item) => (
    <span key={item.key} style={priceItemStyle}>
      {t(item.label)} {item.value}
      {item.unitLabel ? `/${item.unitLabel}` : ''}
    </span>
  ));
};
