import assert from 'node:assert/strict';
import {
  buildCCSwitchDeepLink,
  buildEndpoint,
  chooseDefaultModel,
  getCompatibleModels,
  getRouteOptions,
  normalizeBaseUrl,
} from '../src/utils/ccSwitch.js';

const models = [
  { name: 'gpt-5.4-mini', supported_endpoint_types: ['openai-response'] },
  { name: 'gpt-5.6-sol', supported_endpoint_types: ['openai-response'] },
  { name: 'gpt-image-1.5', supported_endpoint_types: ['image-generation'] },
  { name: 'claude-sonnet-4-5', supported_endpoint_types: ['anthropic'] },
  { name: 'gemini-2.5-pro', supported_endpoint_types: ['gemini'] },
];

assert.equal(
  normalizeBaseUrl('https://example.com/v1/'),
  'https://example.com',
);
assert.equal(normalizeBaseUrl('javascript:alert(1)'), '');
assert.equal(normalizeBaseUrl('not-a-url'), '');
assert.equal(
  normalizeBaseUrl('https://user:pass@example.com/prefix/v1?q=secret#hash'),
  'https://example.com/prefix',
);
assert.equal(
  buildEndpoint('https://example.com/v1/', 'codex'),
  'https://example.com/v1',
);
assert.equal(
  buildEndpoint('https://example.com/v1', 'claude'),
  'https://example.com',
);
assert.equal(chooseDefaultModel(models, 'codex'), 'gpt-5.6-sol');
assert.equal(
  chooseDefaultModel(
    models.filter((item) => item.name !== 'gpt-5.6-sol'),
    'codex',
  ),
  'gpt-5.4-mini',
);
assert.equal(chooseDefaultModel(models, 'claude'), 'claude-sonnet-4-5');
assert.equal(chooseDefaultModel(models, 'gemini'), 'gemini-2.5-pro');
assert.deepEqual(
  getCompatibleModels(models, 'codex').map((item) => item.name),
  ['gpt-5.4-mini', 'gpt-5.6-sol'],
);
assert.equal(chooseDefaultModel(models, 'unknown'), '');
assert.deepEqual(
  getRouteOptions('https://nan.meta-api.vip/', 'https://nan.meta-api.vip'),
  ['https://cn.meta-api.vip', 'https://nan.meta-api.vip'],
);

const link = buildCCSwitchDeepLink({
  app: 'codex',
  name: '测试 & provider',
  baseUrl: 'https://example.com/v1/',
  apiKey: 'sk-test+secret',
  model: 'gpt-5.6-sol',
});
const parsed = new URL(link);
assert.equal(parsed.protocol, 'ccswitch:');
assert.equal(parsed.searchParams.get('endpoint'), 'https://example.com/v1');
assert.equal(parsed.searchParams.get('model'), 'gpt-5.6-sol');
assert.equal(parsed.searchParams.get('apiKey'), 'sk-test+secret');
assert.equal(parsed.searchParams.get('name'), '测试 & provider');

for (const [app, expectedEndpoint, expectedModel] of [
  ['claude', 'https://example.com', 'claude-sonnet-4-5'],
  ['gemini', 'https://example.com', 'gemini-2.5-pro'],
  ['opencode', 'https://example.com/v1', 'gpt-5.6-sol'],
  ['openclaw', 'https://example.com/v1', 'gpt-5.6-sol'],
]) {
  const appLink = new URL(
    buildCCSwitchDeepLink({
      app,
      name: 'provider',
      baseUrl: 'https://example.com/v1',
      apiKey: 'sk-test',
      model: expectedModel,
    }),
  );
  assert.equal(appLink.searchParams.get('app'), app);
  assert.equal(appLink.searchParams.get('endpoint'), expectedEndpoint);
  assert.equal(appLink.searchParams.get('model'), expectedModel);
}

assert.throws(
  () =>
    buildCCSwitchDeepLink({
      app: 'codex',
      baseUrl: 'https://example.com',
      apiKey: '',
      model: 'gpt-5.6-sol',
    }),
  /Incomplete/,
);

console.log('CC Switch configuration tests passed');
