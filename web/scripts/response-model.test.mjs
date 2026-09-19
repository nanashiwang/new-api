import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import {
  getResponseModelInfo,
  isResponseModelMismatch,
} from '../src/helpers/responseModel.js';

const cases = JSON.parse(
  readFileSync(
    new URL(
      '../../relay/common/testdata/response_model_cases.json',
      import.meta.url,
    ),
    'utf8',
  ),
);
for (const { name, observation, mismatch } of cases) {
  for (const legacy of [true, false, undefined, 'false']) {
    assert.equal(
      isResponseModelMismatch({ ...observation, mismatch: legacy }),
      mismatch,
      `${name}: ignore stale/missing flags (${legacy})`,
    );
    const display = getResponseModelInfo(
      { response_model: { ...observation, mismatch: legacy } },
      observation.requested_model,
    );
    if (observation.returned_model.trim()) {
      assert.equal(display.mismatch, mismatch, `${name}: rendered verdict`);
    } else {
      assert.equal(display, null);
    }
  }
}
assert.equal(isResponseModelMismatch(), false);
assert.equal(isResponseModelMismatch({ returned_model: 12 }), false);
assert.equal(
  isResponseModelMismatch({ requested_model: {}, returned_model: 'model' }),
  true,
);

assert.equal(getResponseModelInfo({}, 'old-model'), null);
assert.equal(
  getResponseModelInfo({ response_model: { returned_model: {} } }, 'old-model'),
  null,
);
assert.deepEqual(
  getResponseModelInfo(
    {
      response_model: {
        requested_model: 'alias',
        upstream_model: 'mapped',
        returned_model: 'declared',
        mismatch: true,
      },
    },
    'alias',
  ),
  {
    requested: 'alias',
    upstream: 'mapped',
    returned: 'declared',
    mismatch: true,
  },
);
assert.deepEqual(
  getResponseModelInfo(
    { response_model: { returned_model: 'declared', mismatch: 'false' } },
    'old-model',
  ),
  {
    requested: 'old-model',
    upstream: 'old-model',
    returned: 'declared',
    mismatch: true,
  },
);
assert.equal(
  getResponseModelInfo(
    {
      upstream_model_name: 'mapped',
      response_model: { returned_model: 'vendor/mapped', mismatch: true },
    },
    'alias',
  ).mismatch,
  false,
  'old rows without names use log routing fields, never the stored flag',
);
console.log('response model diagnostics: passed');
