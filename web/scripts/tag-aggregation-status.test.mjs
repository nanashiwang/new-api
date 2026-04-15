import assert from 'node:assert/strict';
import { getTagAggregationStatus } from '../src/components/table/channels/tagAggregationStatus.js';

const allEnabled = getTagAggregationStatus([
  { status: 1, effective_available: true },
  { status: 1, effective_available: true },
]);

assert.equal(allEnabled.totalCount, 2);
assert.equal(allEnabled.disabledCount, 0);
assert.equal(allEnabled.enabledCount, 2);
assert.equal(allEnabled.isAllEnabled, true);
assert.equal(allEnabled.isAllDisabled, false);
assert.equal(allEnabled.isMixed, false);

const mixedWithPendingRetry = getTagAggregationStatus([
  { status: 1, effective_available: true },
  { status: 1, effective_available: false },
  { status: 2, effective_available: false },
  { status: 3, effective_available: false },
  { status: 1, effective_available: true },
]);

assert.equal(mixedWithPendingRetry.totalCount, 5);
assert.equal(mixedWithPendingRetry.enabledCount, 2);
assert.equal(mixedWithPendingRetry.disabledCount, 3);
assert.equal(mixedWithPendingRetry.progressPercent, 40);
assert.equal(mixedWithPendingRetry.progressStroke, 'var(--semi-color-warning)');
assert.equal(mixedWithPendingRetry.isMixed, true);

const allDisabled = getTagAggregationStatus([
  { status: 2, effective_available: false },
  { status: 3, effective_available: false },
  { status: 1, effective_available: false },
]);

assert.equal(allDisabled.totalCount, 3);
assert.equal(allDisabled.enabledCount, 0);
assert.equal(allDisabled.disabledCount, 3);
assert.equal(allDisabled.isAllEnabled, false);
assert.equal(allDisabled.isAllDisabled, true);
assert.equal(allDisabled.isMixed, false);

const mostlyEnabled = getTagAggregationStatus([
  { status: 1, effective_available: true },
  { status: 1, effective_available: true },
  { status: 1, effective_available: true },
  { status: 1, effective_available: false },
]);

assert.equal(mostlyEnabled.progressPercent, 75);
assert.equal(mostlyEnabled.progressStroke, 'var(--semi-color-success)');

console.log('tag aggregation status checks passed');
