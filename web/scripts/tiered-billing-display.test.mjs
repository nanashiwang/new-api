import assert from 'node:assert/strict';
import {
  buildTieredBillingBreakdown,
  resolveTieredLogParams,
} from '../src/helpers/tieredBillingDisplay.js';

const tier = {
  inputPrice: 1,
  outputPrice: 6,
  cacheReadPrice: 0.1,
  cacheCreatePrice: 1.25,
  cacheCreate1hPrice: 2,
};

const exactParams = resolveTieredLogParams(
  {
    tiered_params: {
      p: 1000,
      c: 200,
      len: 3000,
      cr: 1792,
      cc: 8,
      cc1h: 4,
    },
  },
  tier,
);
assert.deepEqual(exactParams, {
  p: 1000,
  c: 200,
  len: 3000,
  cr: 1792,
  cc: 8,
  cc1h: 4,
  img: 0,
  img_o: 0,
  ai: 0,
  ao: 0,
});

const exactBreakdown = buildTieredBillingBreakdown(tier, exactParams);
assert.equal(exactBreakdown.items.length, 5);
assert.equal(Number(exactBreakdown.subtotalUSD.toFixed(7)), 0.0023972);

const legacyParams = resolveTieredLogParams(
  {
    prompt_tokens: 3000,
    completion_tokens: 200,
    cache_tokens: 1792,
    cache_creation_tokens: 12,
    cache_creation_tokens_5m: 8,
    cache_creation_tokens_1h: 4,
  },
  tier,
);
assert.equal(legacyParams.p, 1196);
assert.equal(legacyParams.c, 200);
assert.equal(legacyParams.cr, 1792);
assert.equal(legacyParams.cc, 8);
assert.equal(legacyParams.cc1h, 4);
assert.equal(legacyParams.len, 3000);

const claudeParams = resolveTieredLogParams(
  {
    claude: true,
    prompt_tokens: 1200,
    completion_tokens: 100,
    cache_tokens: 800,
    cache_creation_tokens_5m: 20,
    cache_creation_tokens_1h: 10,
  },
  tier,
);
assert.equal(claudeParams.p, 1200);
assert.equal(claudeParams.len, 2030);

console.log('tiered billing display tests passed');
