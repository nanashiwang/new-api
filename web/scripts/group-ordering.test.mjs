import test from 'node:test';
import assert from 'node:assert/strict';
import { moveGroup } from '../src/helpers/groupOrdering.js';
test('dragging across multiple positions preserves all names and identities', () => {
  const items = Object.freeze([
    { name: 'default' },
    { name: 'premium' },
    { name: 'custom' },
  ]);
  const moved = moveGroup(items, 0, 2);
  assert.deepEqual(
    moved.map((i) => i.name),
    ['premium', 'custom', 'default'],
  );
  assert.equal(moved[2], items[0]);
  assert.deepEqual(moveGroup(moved, 2, 0), items);
});
test('up/down boundaries and invalid stale indices never remove a group', () => {
  const items = ['a', 'b'];
  for (const [from, to] of [
    [-1, 0],
    [0, -1],
    [2, 0],
    [0, 2],
    [0, 0],
    [0, NaN],
    [0, 0.5],
  ])
    assert.equal(moveGroup(items, from, to), items);
  assert.deepEqual(moveGroup(items, 1, 0), ['b', 'a']);
  assert.deepEqual(moveGroup([], 0, 1), []);
});
