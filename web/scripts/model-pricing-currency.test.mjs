import assert from 'node:assert/strict';
import {
  formatModelPricingUnitPrice,
  resolveModelPricingCurrency,
  resolveModelPricingRate,
  resolveModelPricingSymbol,
} from '../src/helpers/modelPricingCurrency.js';

assert.equal(
  resolveModelPricingCurrency({
    showWithRecharge: true,
    quotaDisplayType: 'CUSTOM',
  }),
  'CNY',
);
assert.equal(
  resolveModelPricingCurrency({
    showWithRecharge: false,
    quotaDisplayType: 'USD',
  }),
  'USD',
);
assert.equal(
  resolveModelPricingCurrency({
    showWithRecharge: false,
    quotaDisplayType: 'CUSTOM',
  }),
  'CUSTOM',
);
assert.equal(
  resolveModelPricingCurrency({
    showWithRecharge: true,
    quotaDisplayType: 'USD',
  }),
  'CNY',
);

assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CNY',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  7.2,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CNY',
    priceConvertMode: 'package',
    priceRate: 7.2,
    packageEffectiveRate: 6.4,
    customExchangeRate: 12.5,
  }),
  6.4,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: false,
    currency: 'CUSTOM',
    priceConvertMode: 'recharge',
    priceRate: 6.5,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  12.5,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: false,
    currency: 'USD',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  1,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: false,
    currency: 'CUSTOM',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 0,
  }),
  1,
);

assert.equal(resolveModelPricingSymbol('CUSTOM', 'G'), 'G');
assert.equal(resolveModelPricingSymbol('CUSTOM', ''), '¤');
assert.equal(resolveModelPricingSymbol('CNY'), '¥');
assert.equal(resolveModelPricingSymbol('USD'), '$');

const packageDisplayPrice = () => '';
packageDisplayPrice.toAmount = (usdAmount) => usdAmount * 6.4;
packageDisplayPrice.currencySymbol = '¥';
assert.equal(formatModelPricingUnitPrice(2.5, packageDisplayPrice), '¥16.0000');

const customDisplayPrice = () => '';
customDisplayPrice.toAmount = (usdAmount) => usdAmount * 12.5;
customDisplayPrice.currencySymbol = '⚡️';
assert.equal(
  formatModelPricingUnitPrice(0.1, customDisplayPrice, 6),
  '⚡️1.250000',
);

console.log('model pricing currency tests passed');
