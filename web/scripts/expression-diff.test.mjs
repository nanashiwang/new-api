import test from 'node:test';
import assert from 'node:assert/strict';
import { expressionDiff } from '../src/helpers/expressionDiff.js';

test('highlights changed numeric literals without evaluating expressions', () => {
  const parts = expressionDiff('input*0.3+output*1.5', 'input*0.2+output*1.5');
  assert.deepEqual(
    parts.filter((p) => p.changed).map((p) => p.text),
    ['0.3'],
  );
});
test('unchanged or absent baseline has no highlighting', () => {
  for (const baseline of [undefined, null, 7, 'a + b'])
    assert.ok(expressionDiff('a + b', baseline).every((p) => !p.changed));
});
test('shape changes, deletions, whitespace, Unicode and HTML are preserved verbatim', () => {
  for (const [value, baseline] of [
    ['a + b + c', 'a + b'],
    ['a', 'a+b'],
    [' a\n +\tb ', 'a+b'],
    ['价格*2', '价格*1'],
    ['<script>alert(1)</script>', 'x'],
  ]) {
    const parts = expressionDiff(value, baseline);
    assert.equal(parts.map((p) => p.text).join(''), value);
    assert.ok(parts.some((p) => p.changed));
  }
});
test('large expressions use bounded linear comparison', () => {
  const value = 'x+'.repeat(20000) + '2';
  assert.equal(
    expressionDiff(value, value + '+3')
      .map((p) => p.text)
      .join(''),
    value,
  );
});
