import test from 'node:test';
import assert from 'node:assert/strict';
import { globSync, readFileSync } from 'node:fs';
import {
  createProviderIconRegistry,
  resolveProviderIcon,
} from '../src/helpers/providerIconRegistry.js';

test('POSIX and Windows paths register only supported JS icon components', () => {
  for (const root of [
    '/build/web/node_modules/',
    'D:\\a\\new-api\\web\\node_modules\\',
  ]) {
    const loader = () => {
      throw Error('must not import before rendering');
    };
    const files = [
      'OpenAI/components/Mono.js',
      'Claude/components/Color.js',
      'Gemma/components/Simple.js',
      'LobeHub/components/Morden.js',
      'OpenAI/index.js',
      'OpenAI/components/Mono.d.ts',
      'OpenAI/components/Unknown.js',
    ];
    const registry = createProviderIconRegistry(
      Object.fromEntries(
        files.map((file) => [root + '@lobehub/icons/es/' + file, loader]),
      ),
    );
    assert.deepEqual(
      [...registry.keys()],
      ['OpenAI.Mono', 'Claude.Color', 'Gemma.Simple', 'LobeHub.Morden'],
    );
    assert.equal(
      resolveProviderIcon(registry, 'OpenAI', 'Color'),
      'OpenAI.Mono',
    );
    assert.equal(resolveProviderIcon(registry, 'Unknown', 'Color'), null);
    assert.equal(resolveProviderIcon(registry, '../../OpenAI'), null);
  }
});
test('installed package covers every configured provider and special variant', () => {
  const files = [
    ...globSync('node_modules/@lobehub/icons/es/*/components/*.js'),
  ];
  const registry = createProviderIconRegistry(
    Object.fromEntries(files.map((file) => [file, () => {}])),
  );
  const source = readFileSync(
    new URL('../src/helpers/providerIcons.jsx', import.meta.url),
    'utf8',
  );
  for (const match of source.matchAll(/provider\('([^']+)'\)/g))
    assert.ok(resolveProviderIcon(registry, match[1]), match[1]);
  for (const [name, variant] of [
    ['Claude', 'Color'],
    ['OpenAI', 'Avatar'],
    ['Gemma', 'Simple'],
  ])
    assert.ok(registry.has(`${name}.${variant}`));
  // This installed version predates Morden; preserve the existing Mono fallback.
  assert.equal(
    resolveProviderIcon(registry, 'LobeHub', 'Morden'),
    'LobeHub.Mono',
  );
});
