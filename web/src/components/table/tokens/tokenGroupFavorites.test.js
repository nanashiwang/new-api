import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  availableFavorites,
  favoriteStorageKey,
  readGroupFavorites,
  toggleGroupFavorite,
} from './groupFavoriteStorage.js';

test('favorites use an account-scoped key and never an anonymous shared key', () => {
  assert.equal(favoriteStorageKey(undefined), null);
  assert.notEqual(favoriteStorageKey(1), favoriteStorageKey(2));
});
test('corrupt, blocked, non-array and mixed storage are safe', () => {
  for (const input of ['broken', '{}', 'null', 'true']) {
    assert.deepEqual(readGroupFavorites({ getItem: () => input }, 'key'), []);
  }
  assert.deepEqual(
    readGroupFavorites(
      {
        getItem() {
          throw Error('blocked');
        },
      },
      'key',
    ),
    [],
  );
  assert.deepEqual(
    readGroupFavorites({ getItem: () => '["a",null,"",1,"a","b"]' }, 'key'),
    ['a', 'b'],
  );
  assert.deepEqual(readGroupFavorites(null, null), []);
});
test('only currently permitted groups become shortcuts across vendors', () => {
  const groups = [{ value: 'OpenAI · A' }, { value: 'Claude · B' }];
  assert.deepEqual(
    availableFavorites(groups, ['removed', 'Claude · B', 'OpenAI · A']),
    groups,
  );
  assert.deepEqual(availableFavorites([], ['removed']), []);
});
test('toggling is reversible and does not mutate values or input', () => {
  const before = ['OpenAI · A'];
  const next = toggleGroupFavorite(before, 'Claude · B');
  assert.deepEqual(before, ['OpenAI · A']);
  assert.deepEqual(next, ['OpenAI · A', 'Claude · B']);
  assert.deepEqual(toggleGroupFavorite(next, 'Claude · B'), before);
});
