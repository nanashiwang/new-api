import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import { resolveTokenGroupVendor } from '../src/components/table/tokens/tokenGroupUtils.js';

test('log vendor SQL aliases stay identical to frontend vendor recognition', () => {
  const frontend = readFileSync(
    new URL(
      '../src/components/table/tokens/tokenGroupUtils.js',
      import.meta.url,
    ),
    'utf8',
  );
  const backend = readFileSync(
    new URL('../../model/log_scope.go', import.meta.url),
    'utf8',
  );
  const frontendMap = frontend
    .split('const CANONICAL_VENDOR_NAMES = new Map([')[1]
    .split(']);')[0];
  const backendMap = backend
    .split('var logVendorAliases = map[string]string{')[1]
    .split('\n}')[0];
  const expected = [...frontendMap.matchAll(/\['([^']+)', '([^']+)'\]/g)].map(
    ([, alias, canonical]) => [alias, canonical.toLowerCase()],
  );
  const actual = [...backendMap.matchAll(/"([^"]+)":\s*"([^"]+)"/g)].map(
    ([, alias, canonical]) => [alias, canonical],
  );
  assert.ok(expected.length > 30);
  assert.deepEqual(actual.sort(), expected.sort());
  for (const [alias, canonical] of expected) {
    assert.equal(resolveTokenGroupVendor(alias).toLowerCase(), canonical);
    assert.equal(
      resolveTokenGroupVendor(`${alias} · test`).toLowerCase(),
      canonical,
    );
  }
});
