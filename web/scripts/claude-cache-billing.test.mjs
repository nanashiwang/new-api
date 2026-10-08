import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import { getCacheCreationBreakdown } from '../src/helpers/cacheCreation.js';
import { resolveTieredLogParams } from '../src/helpers/tieredBillingDisplay.js';

const root = fileURLToPath(new URL('..', import.meta.url));
const require = createRequire(import.meta.url);
const parser = require('@babel/parser');
const extract = (file, names) => {
  const source = fs.readFileSync(`${root}/${file}`, 'utf8');
  const ast = parser.parse(source, { sourceType: 'module', plugins: ['jsx'] });
  return ast.program.body
    .flatMap((node) => {
      const d =
        node.type === 'ExportNamedDeclaration' ? node.declaration : node;
      const name = d?.id?.name || d?.declarations?.[0]?.id?.name;
      return names.includes(name) ? [source.slice(d.start, d.end)] : [];
    })
    .join('\n');
};
const render = extract('src/helpers/render.jsx', ['renderClaudeModelPrice']);
const prices = extract('src/helpers/price.jsx', [
  'getDisplayCurrencySymbol',
  'formatTokenUnitPrice',
  'calculateModelPrice',
]);
const { renderClaudeModelPrice, calculateModelPrice } = vm.runInNewContext(
  `${render}\n${prices}\n({renderClaudeModelPrice,calculateModelPrice})`,
  {
    getCacheCreationBreakdown,
    getEffectiveRatio: (ratio) => ({ ratio, label: 'group' }),
    getCurrencyConfig: () => ({ symbol: '$', rate: 1 }),
    shouldUseRatioBillingProcess: () => false,
    renderBillingArticle: (lines) => lines.filter(Boolean),
    buildBillingPriceText: (key, vars) => ({ key, vars }),
    buildBillingText: (key, vars) => ({ key, vars }),
    formatBillingDisplayPrice: (value, rate) => value * rate,
    i18next: { t: (key) => key },
  },
);
const record = {
  prompt_tokens: 400,
  completion_tokens: 0,
  model_ratio: 1,
  completion_ratio: 5,
  group_ratio: 1,
  model_price: -1,
  cache_creation_tokens: 400,
  cache_creation_tokens_5m: 80,
  cache_creation_tokens_1h: 120,
  cache_creation_ratio: 1.25,
  cache_creation_ratio_5m: 1.25,
  cache_creation_ratio_1h: 2,
};
assert.deepEqual(getCacheCreationBreakdown(record), {
  total: 400,
  fiveMinute: 80,
  oneHour: 120,
  unclassified: 200,
  conflict: false,
});
const lines = renderClaudeModelPrice(record);
assert(
  Math.abs(
    lines.find((line) => line.vars?.total !== undefined).vars.total - 0.00198,
  ) < 1e-12,
);
assert(
  lines.some(
    (line) => line.key.includes('未细分缓存创建') && line.vars.tokens === 200,
  ),
);
for (const [r, total, rest] of [
  [{ cache_creation_tokens: 400 }, 400, 400],
  [{ cache_creation_tokens_5m: 80, cache_creation_tokens_1h: 120 }, 200, 0],
  [{ cache_creation_tokens: 0 }, 0, 0],
]) {
  const d = getCacheCreationBreakdown(r);
  assert.equal(d.total, total);
  assert.equal(d.unclassified, rest);
}
assert(
  getCacheCreationBreakdown({
    cache_creation_tokens: 0,
    cache_creation_tokens_1h: 10,
  }).conflict,
);
const tier = {
  inputPrice: 2,
  outputPrice: 10,
  cacheReadPrice: 0.2,
  cacheCreatePrice: 2.5,
  cacheCreate1hPrice: 4,
};
const params = resolveTieredLogParams({ ...record, claude: true }, tier);
assert.equal(params.cc, 280);
assert.equal(params.cc1h, 120);
assert.equal(params.len, 800);

for (const ratio of [0, 0.45, 1]) {
  for (const tokenUnit of ['K', 'M']) {
    for (const [currency, symbol, conversion] of [
      ['USD', '$', 1],
      ['CNY', '¥', 7],
      ['CUSTOM', '⚡️', 5],
    ]) {
      const displayPrice = (v) => `${symbol}${v * conversion}`;
      displayPrice.toAmount = (v) => v * conversion;
      displayPrice.currencySymbol = symbol;
      const p = calculateModelPrice({
        record: {
          quota_type: 0,
          model_ratio: 10,
          completion_ratio: 5,
          supports_cache_creation: true,
          cache_creation_ratio: 1.25,
          supports_cache_creation_ttl: true,
          cache_creation_ratio_1h: 2,
        },
        selectedGroup: 'test',
        groupRatio: { test: ratio },
        tokenUnit,
        displayPrice,
        currency,
        precision: 8,
      });
      const item = p.pricingItems.find(
        (item) => item.key === 'cacheCreation1h',
      );
      assert(item);
      assert.equal(
        Number(item.value.replace(symbol, '')),
        (40 * ratio * conversion) / (tokenUnit === 'K' ? 1000 : 1),
      );
    }
  }
}
const displayPrice = (v) => `$${v}`;
displayPrice.toAmount = (v) => v;
const generic = calculateModelPrice({
  record: {
    quota_type: 0,
    model_ratio: 1,
    supports_cache_creation: true,
    cache_creation_ratio: 1.25,
  },
  selectedGroup: 'test',
  groupRatio: { test: 1 },
  tokenUnit: 'M',
  displayPrice,
  currency: 'USD',
});
assert(!generic.pricingItems.some((item) => item.key === 'cacheCreation1h'));
console.log(
  'PASS cache totals/remainders/conflicts, log formula, legacy tiered normalization, 18 currency/group/unit combinations and generic fallback',
);
