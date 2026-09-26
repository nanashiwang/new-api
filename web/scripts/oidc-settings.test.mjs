import test from 'node:test';
import assert from 'node:assert/strict';
import { prepareOIDCSettingsUpdate } from '../src/helpers/oidcSettings.js';

const baseline = Object.freeze({
  'oidc.enabled': true,
  'oidc.well_known': 'https://old.example/.well-known/openid-configuration',
  'oidc.client_id': 'old-client',
  'oidc.client_secret': '',
  'oidc.authorization_endpoint': 'https://old.example/auth',
  'oidc.token_endpoint': 'https://old.example/token',
  'oidc.user_info_endpoint': 'https://old.example/user',
});
const offline = () => {
  throw new Error('Discovery should not run');
};
const metadata = {
  authorization_endpoint: 'https://new.example/auth',
  token_endpoint: 'https://new.example/token',
  userinfo_endpoint: 'https://new.example/user',
};

test('unchanged discovery does not block client edits or resend blank secrets', async () => {
  const result = await prepareOIDCSettingsUpdate(
    { ...baseline, 'oidc.client_id': 'new-client' },
    baseline,
    offline,
  );
  assert.deepEqual(result.options, [
    { key: 'oidc.client_id', value: 'new-client' },
  ]);
  assert.equal(result.discovered, false);
  assert.deepEqual(
    (await prepareOIDCSettingsUpdate(baseline, baseline, offline)).options,
    [],
  );
});

test('trims changed URLs and preserves input state while discovering endpoints', async () => {
  const input = Object.freeze({
    ...baseline,
    'oidc.well_known': ' https://new.example/discovery ',
  });
  const result = await prepareOIDCSettingsUpdate(
    input,
    baseline,
    async (url) => {
      assert.equal(url, 'https://new.example/discovery');
      return metadata;
    },
  );
  assert.equal(result.discovered, true);
  assert.equal(
    result.values['oidc.authorization_endpoint'],
    metadata.authorization_endpoint,
  );
  assert.equal(
    input['oidc.authorization_endpoint'],
    baseline['oidc.authorization_endpoint'],
  );
  const saved = {
    ...baseline,
    ...Object.fromEntries(result.options.map(({ key, value }) => [key, value])),
  };
  assert.deepEqual(
    (await prepareOIDCSettingsUpdate(saved, saved, offline)).options,
    [],
  );
});

test('manual configuration, clearing discovery, and disabling remain independent', async () => {
  const input = { ...baseline, 'oidc.well_known': '', 'oidc.enabled': false };
  const result = await prepareOIDCSettingsUpdate(input, baseline, offline);
  assert.deepEqual(result.options, [{ key: 'oidc.well_known', value: '' }]);
  assert.equal(result.values['oidc.enabled'], false);
  const trimmed = await prepareOIDCSettingsUpdate(
    { ...baseline, 'oidc.well_known': ` ${baseline['oidc.well_known']} ` },
    baseline,
    offline,
  );
  assert.deepEqual(trimmed.options, []);
});

test('an existing URL can still discover missing initial endpoints', async () => {
  const input = { ...baseline, 'oidc.token_endpoint': '' };
  const result = await prepareOIDCSettingsUpdate(
    input,
    baseline,
    async () => metadata,
  );
  assert.equal(result.discovered, true);
  assert.equal(result.values['oidc.token_endpoint'], metadata.token_endpoint);
});

test('failed or incomplete discovery does not produce partial updates', async () => {
  const input = Object.freeze({
    ...baseline,
    'oidc.well_known': 'https://new.example/discovery',
  });
  for (const discover of [
    async () => {
      throw new Error('offline');
    },
    async () => ({ token_endpoint: 'only-one' }),
  ]) {
    await assert.rejects(
      prepareOIDCSettingsUpdate(input, baseline, discover),
      /获取 OIDC 配置失败/,
    );
    assert.equal(input['oidc.token_endpoint'], baseline['oidc.token_endpoint']);
  }
  await assert.rejects(
    prepareOIDCSettingsUpdate(
      { ...baseline, 'oidc.well_known': 'file:///discovery' },
      baseline,
      offline,
    ),
    /Well-Known URL/,
  );
});
