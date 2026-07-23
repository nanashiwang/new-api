import assert from 'node:assert/strict';
import {
  resolveModelPricingCurrency,
  resolveModelPricingRate,
  resolveModelPricingSymbol,
} from '../src/helpers/modelPricingCurrency.js';

assert.equal(
  resolveModelPricingCurrency({
    showWithRecharge: true,
    quotaDisplayType: 'CUSTOM',
  }),
  'CUSTOM',
);
assert.equal(
  resolveModelPricingCurrency({
    showWithRecharge: false,
    quotaDisplayType: 'CUSTOM',
  }),
  'USD',
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
    currency: 'CUSTOM',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    usdExchangeRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  12.5,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CNY',
    priceConvertMode: 'package',
    priceRate: 7.2,
    usdExchangeRate: 7.2,
    packageEffectiveRate: 6.4,
    customExchangeRate: 12.5,
  }),
  6.4,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CUSTOM',
    priceConvertMode: 'package',
    priceRate: 7.2,
    usdExchangeRate: 7.2,
    packageEffectiveRate: 6.4,
    customExchangeRate: 12.5,
  }),
  (6.4 * 12.5) / 7.2,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CUSTOM',
    priceConvertMode: 'recharge',
    priceRate: 6.5,
    usdExchangeRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  (6.5 * 12.5) / 7.2,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: false,
    currency: 'USD',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    usdExchangeRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  1,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CUSTOM',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    usdExchangeRate: 7.2,
    packageEffectiveRate: null,
    customExchangeRate: 0,
  }),
  1,
);
assert.equal(
  resolveModelPricingRate({
    showWithRecharge: true,
    currency: 'CUSTOM',
    priceConvertMode: 'recharge',
    priceRate: 7.2,
    usdExchangeRate: 0,
    packageEffectiveRate: null,
    customExchangeRate: 12.5,
  }),
  12.5,
);

assert.equal(resolveModelPricingSymbol('CUSTOM', 'G'), 'G');
assert.equal(resolveModelPricingSymbol('CUSTOM', ''), '¤');
assert.equal(resolveModelPricingSymbol('CNY'), '¥');
assert.equal(resolveModelPricingSymbol('USD'), '$');

console.log('model pricing currency tests passed');
