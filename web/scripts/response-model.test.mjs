import assert from 'node:assert/strict';
import { getResponseModelInfo } from '../src/helpers/responseModel.js';

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
    mismatch: false,
  },
);
console.log('response model diagnostics: passed');
