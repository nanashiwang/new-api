import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(scriptDir, '..');
const playgroundDir = path.join(webDir, 'public', 'image-playground');
const assetsDir = path.join(playgroundDir, 'assets');

const readText = (file) => readFileSync(file, 'utf8');
const assertNotHtmlFallback = (file, label) => {
  const head = readText(file).trimStart().slice(0, 80).toLowerCase();
  assert(!head.startsWith('<!doctype'), `${label} unexpectedly starts with HTML doctype`);
  assert(!head.startsWith('<html'), `${label} unexpectedly starts with HTML`);
};

const indexHtmlPath = path.join(playgroundDir, 'index.html');
const indexHtml = readText(indexHtmlPath);

const collectRefs = (regex) => [...indexHtml.matchAll(regex)].map((match) => match[1]);
const scriptRefs = collectRefs(/<script\b[^>]*\bsrc=["']([^"']+)["']/g);
const stylesheetRefs = collectRefs(/<link\b[^>]*\brel=["']stylesheet["'][^>]*\bhref=["']([^"']+)["']/g);

assert(scriptRefs.length > 0, 'image-playground index.html should reference at least one script');
assert(stylesheetRefs.length > 0, 'image-playground index.html should reference at least one stylesheet');

for (const ref of [...scriptRefs, ...stylesheetRefs]) {
  assert(
    ref.startsWith('./assets/') || ref.startsWith('assets/'),
    `image-playground asset ref should stay relative under assets/: ${ref}`,
  );
  const file = path.join(playgroundDir, ref.replace(/^\.\//, ''));
  assert(existsSync(file), `referenced asset does not exist: ${ref}`);
  assert(statSync(file).size > 0, `referenced asset is empty: ${ref}`);
  assertNotHtmlFallback(file, ref);
}

for (const fileName of readdirSync(assetsDir).filter((name) => name.endsWith('.js'))) {
  const file = path.join(assetsDir, fileName);
  assert(statSync(file).size > 0, `JS chunk is empty: ${fileName}`);
  assertNotHtmlFallback(file, fileName);
  execFileSync(process.execPath, ['--check', file], { stdio: 'pipe' });
}

const swPath = path.join(playgroundDir, 'sw.js');
const swSourcePath = path.join(webDir, 'apps', 'image-playground', 'public', 'sw.js');
const mainSourcePath = path.join(webDir, 'apps', 'image-playground', 'src', 'main.tsx');
const sw = readText(swPath);
const swSource = readText(swSourcePath);
const mainSource = readText(mainSourcePath);

assert.match(sw, /gpt-image-playground-v0\.1\.6/, 'built sw.js should bump cache name to clear stale caches');
assert.equal(sw, swSource, 'built sw.js should match source sw.js');
assert(!sw.includes("'./index.html'"), 'sw.js must not precache index.html');
assert(!sw.includes('cache.put(\'./index.html\''), 'sw.js must not runtime-cache HTML shell');
assert(!sw.includes('caches.match(\'./index.html\''), 'sw.js must not serve cached HTML shell fallback');
assert(sw.includes("request.destination === 'script'"), 'sw.js should explicitly avoid caching scripts');
assert(sw.includes("request.destination === 'style'"), 'sw.js should explicitly avoid caching styles');
assert(sw.includes("fetch(request, { cache: 'no-store' })"), 'sw.js navigations should bypass browser cache');
assert(mainSource.includes("updateViaCache: 'none'"), 'service worker registration should bypass HTTP cache');

console.log('image-playground static guard checks passed');
