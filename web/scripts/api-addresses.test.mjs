import test from 'node:test';
import assert from 'node:assert/strict';
import { getApiAddresses } from '../src/helpers/apiAddresses.js';
test('preserves exact configured paths and trailing slashes without appending v1', () => {
  const rows = getApiAddresses({
    api_info: [
      {
        route: ' Asia ',
        url: ' https://asia.example/proxy/v1/ ',
        description: ' Region ',
      },
      { url: 'https://api.example' },
    ],
  });
  assert.deepEqual(rows[0], {
    label: 'Asia',
    url: 'https://asia.example/proxy/v1/',
    description: 'Region',
  });
  assert.equal(rows[1].url, 'https://api.example');
});
test('disabled or invalid configured addresses fall back to server then origin', () => {
  const status = {
    api_info_enabled: false,
    api_info: [{ url: 'https://hidden.example' }],
    server_address: ' https://server.example/proxy/ ',
  };
  assert.equal(getApiAddresses(status)[0].url, 'https://server.example/proxy/');
  assert.equal(
    getApiAddresses(
      {
        api_info: [
          null,
          { url: 'javascript:alert(1)' },
          { url: 'https://user:secret@example.com' },
        ],
      },
      'http://localhost:5177',
    )[0].url,
    'http://localhost:5177',
  );
});
test('deduplicates addresses and handles absent status and malformed data', () => {
  assert.deepEqual(getApiAddresses(undefined, 'https://example.com'), []);
  assert.deepEqual(getApiAddresses({ api_info: {}, server_address: 42 }), []);
  assert.equal(
    getApiAddresses({
      api_info: [{ url: 'https://a.example' }, { url: ' https://a.example ' }],
    }).length,
    1,
  );
});
