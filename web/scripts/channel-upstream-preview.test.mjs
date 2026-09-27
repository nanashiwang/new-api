import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import {
  normalizeModelList,
  selectPendingModels,
} from '../src/hooks/channels/upstreamUpdateUtils.js';

// Execute the real hook with a minimal state runner, keeping HTTP and refresh controllable.
const source = readFileSync(
  new URL(
    '../src/hooks/channels/useChannelUpstreamUpdates.jsx',
    import.meta.url,
  ),
  'utf8',
)
  .replace(/^import .*;$/gm, '')
  .replace(
    'export const useChannelUpstreamUpdates',
    'const useChannelUpstreamUpdates',
  );
function harness(post, refresh = async () => {}) {
  const states = [],
    refs = [],
    messages = [];
  let stateCursor, refCursor;
  const hook = new Function(
    'useState',
    'useRef',
    'useEffect',
    'API',
    'showError',
    'showInfo',
    'showSuccess',
    'normalizeModelList',
    'selectPendingModels',
    `${source}; return useChannelUpstreamUpdates;`,
  )(
    (initial) => {
      const i = stateCursor++;
      if (!(i in states)) states[i] = initial;
      return [
        states[i],
        (v) => {
          states[i] = typeof v === 'function' ? v(states[i]) : v;
        },
      ];
    },
    (initial) => (refs[refCursor++] ||= { current: initial }),
    () => {},
    { post },
    ...['error', 'info', 'success'].map(
      (kind) => (text) => messages.push({ kind, text }),
    ),
    normalizeModelList,
    selectPendingModels,
  );
  return {
    messages,
    render() {
      stateCursor = refCursor = 0;
      return hook({ t: (s) => s, refresh });
    },
  };
}
const deferred = () => {
  let resolve;
  const promise = new Promise((r) => {
    resolve = r;
  });
  return { promise, resolve };
};
const response = (data) => ({ data: { success: true, data } });

test('only selected pending models are submitted; unselected additions are not ignored', async () => {
  const calls = [];
  const h = harness(async (url, body) => {
    calls.push(body);
    return response({ added_models: ['a'], removed_models: [] });
  });
  h.render().openUpstreamUpdateModal({ id: 1 }, ['a', 'b'], ['old']);
  await h
    .render()
    .applyUpstreamUpdates({
      addModels: ['a', 'a', 'injected'],
      removeModels: ['unknown'],
    });
  assert.deepEqual(calls, [
    { id: 1, add_models: ['a'], ignore_models: [], remove_models: [] },
  ]);
  assert.equal(h.render().showUpstreamUpdateModal, false);
});
test('empty selection does not submit and refresh failures do not report apply failure', async () => {
  let calls = 0;
  const h = harness(
    async () => {
      calls++;
      return response({});
    },
    async () => {
      throw Error('refresh');
    },
  );
  h.render().openUpstreamUpdateModal({ id: 1 }, ['a']);
  await h.render().applyUpstreamUpdates();
  assert.equal(calls, 0);
  await h.render().applyUpstreamUpdates({ addModels: ['a'] });
  assert.equal(calls, 1);
  assert.deepEqual(
    h.messages.map((m) => m.kind),
    ['success', 'info'],
  );
});
test('closing or switching a preview invalidates older responses', async () => {
  const first = deferred(),
    second = deferred();
  const h = harness((_url, { id }) =>
    id === 1 ? first.promise : second.promise,
  );
  const oldRequest = h.render().detectChannelUpstreamUpdates({ id: 1 });
  assert.equal(h.render().upstreamPreviewLoading, true);
  const latest = h.render().detectChannelUpstreamUpdates({ id: 2 });
  first.resolve(response({ add_models: ['old'] }));
  await oldRequest;
  assert.equal(h.render().upstreamPreviewLoading, true);
  assert.deepEqual(h.render().upstreamUpdateAddModels, []);
  h.render().closeUpstreamUpdateModal();
  second.resolve(response({ add_models: ['new'] }));
  await latest;
  assert.equal(h.render().showUpstreamUpdateModal, false);
  assert.deepEqual(h.render().upstreamUpdateAddModels, []);
});
test('detection errors remain in the preview; failed apply preserves selections for retry', async () => {
  const h = harness(async () => ({
    data: { success: false, message: 'offline' },
  }));
  await h.render().detectChannelUpstreamUpdates({ id: 1 });
  assert.equal(h.render().upstreamPreviewError, 'offline');
  assert.equal(h.render().upstreamPreviewLoading, false);
  h.render().openUpstreamUpdateModal({ id: 1 }, ['a']);
  await h.render().applyUpstreamUpdates({ addModels: ['a'] });
  assert.equal(h.render().showUpstreamUpdateModal, true);
  assert.equal(h.render().upstreamApplyLoading, false);
});
test('double apply is blocked and preview cannot close or switch during submission', async () => {
  const pending = deferred();
  let calls = 0;
  const h = harness(() => {
    calls++;
    return pending.promise;
  });
  h.render().openUpstreamUpdateModal({ id: 1 }, ['a']);
  const apply = h.render().applyUpstreamUpdates({ addModels: ['a'] });
  await h.render().applyUpstreamUpdates({ addModels: ['a'] });
  h.render().closeUpstreamUpdateModal();
  h.render().openUpstreamUpdateModal({ id: 2 }, ['b']);
  assert.equal(h.render().upstreamUpdateChannel.id, 1);
  assert.equal(h.render().showUpstreamUpdateModal, true);
  assert.equal(calls, 1);
  pending.resolve(response({}));
  await apply;
});
