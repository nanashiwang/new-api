import assert from 'node:assert/strict';

import { groupWarningDetailsByScope } from './utils.js';

const grouped = groupWarningDetailsByScope([
  {
    scope_type: 'tag',
    scope_label: 'api.aipaibox.com - 满血反重力',
    scope_value: 'api.aipaibox.com - 满血反重力',
    model_name: 'claude-sonnet-4-6',
    count: 14,
  },
  {
    scope_type: 'tag',
    scope_label: 'api.aipaibox.com - 满血反重力',
    scope_value: 'api.aipaibox.com - 满血反重力',
    model_name: 'claude-opus-4-6',
    count: 10,
  },
  {
    scope_type: 'channel',
    scope_label: '同名渠道',
    scope_value: '101',
    model_name: 'gpt-5.4',
    count: 3,
  },
  {
    scope_type: 'channel',
    scope_label: '同名渠道',
    scope_value: '202',
    model_name: 'gpt-5.3',
    count: 2,
  },
]);

assert.equal(grouped.length, 3);

assert.deepEqual(grouped[0], {
  scopeKey: 'tag:api.aipaibox.com - 满血反重力',
  scopeType: 'tag',
  scopeLabel: 'api.aipaibox.com - 满血反重力',
  scopeValue: 'api.aipaibox.com - 满血反重力',
  totalCount: 24,
  displayHint: '',
  models: [
    { modelName: 'claude-sonnet-4-6', count: 14 },
    { modelName: 'claude-opus-4-6', count: 10 },
  ],
});

assert.equal(grouped[1].scopeKey, 'channel:101');
assert.equal(grouped[1].displayHint, '#101');
assert.deepEqual(grouped[1].models, [{ modelName: 'gpt-5.4', count: 3 }]);

assert.equal(grouped[2].scopeKey, 'channel:202');
assert.equal(grouped[2].displayHint, '#202');
assert.deepEqual(grouped[2].models, [{ modelName: 'gpt-5.3', count: 2 }]);
