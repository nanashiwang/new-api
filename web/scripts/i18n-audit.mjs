import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webRoot = path.resolve(__dirname, '..');
const srcDir = path.join(webRoot, 'src');
const localesDir = path.join(srcDir, 'i18n', 'locales');
const reportDir = path.join(webRoot, '.i18n-report');
const baseLocale = 'zh-CN';
const variantSuffixes = new Set([
  'zero',
  'one',
  'two',
  'few',
  'many',
  'other',
  'male',
  'female',
]);

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function listFilesRecursive(dirPath) {
  const entries = fs.readdirSync(dirPath, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const fullPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) {
      files.push(...listFilesRecursive(fullPath));
      continue;
    }
    files.push(fullPath);
  }
  return files;
}

function ensureDir(dirPath) {
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
  }
}

function extractI18nKeysFromContent(content) {
  const keys = new Set();
  const translationCalls = /(?:\bi18next\.)?\bt\s*\(\s*(['"`])((?:\\.|(?!\1)[\s\S])*?)\1/gm;
  let match;
  while ((match = translationCalls.exec(content)) !== null) {
    const key = match[2];
    if (!key || key.includes('${')) {
      continue;
    }
    keys.add(key);
  }
  return keys;
}

function extractSuspiciousHardcodedStrings(content) {
  const suspicious = new Set();
  const stringLiteral = /(['"`])((?:\\.|(?!\1)[\s\S])*?)\1/gm;
  let match;
  while ((match = stringLiteral.exec(content)) !== null) {
    const value = match[2];
    if (!value || value.includes('${')) {
      continue;
    }
    const hasCjk = /[\u3400-\u9fff]/.test(value);
    if (!hasCjk) {
      continue;
    }
    suspicious.add(value);
  }
  return suspicious;
}

function sortObjectByKey(input) {
  const output = {};
  const keys = Object.keys(input).sort((a, b) => a.localeCompare(b));
  for (const key of keys) {
    output[key] = input[key];
  }
  return output;
}

function isVariantKeyUsed(key, usedKeys) {
  if (!key.includes('_')) {
    return false;
  }

  const parts = key.split('_');
  let changed = false;

  while (parts.length > 1) {
    const last = parts[parts.length - 1];
    if (!variantSuffixes.has(last)) {
      break;
    }
    parts.pop();
    changed = true;
    const candidate = parts.join('_');
    if (usedKeys.has(candidate)) {
      return true;
    }
  }

  return changed && usedKeys.has(parts.join('_'));
}

const localeFiles = fs
  .readdirSync(localesDir)
  .filter((name) => name.endsWith('.json'))
  .sort((a, b) => a.localeCompare(b));

const localeMaps = {};
for (const fileName of localeFiles) {
  const locale = fileName.replace('.json', '');
  const content = readJson(path.join(localesDir, fileName));
  localeMaps[locale] = content.translation || {};
}

if (!localeMaps[baseLocale]) {
  console.error(`Base locale "${baseLocale}" is missing.`);
  process.exit(1);
}

const sourceFiles = listFilesRecursive(srcDir).filter((filePath) =>
  /\.(js|jsx|ts|tsx)$/.test(filePath) &&
  !filePath.includes(`${path.sep}src${path.sep}i18n${path.sep}`),
);

const usedKeys = new Set();
const suspiciousByFile = {};

for (const filePath of sourceFiles) {
  const content = fs.readFileSync(filePath, 'utf8');
  const keysInFile = extractI18nKeysFromContent(content);
  for (const key of keysInFile) {
    usedKeys.add(key);
  }

  const suspicious = extractSuspiciousHardcodedStrings(content);
  const filtered = [...suspicious].filter((item) => !usedKeys.has(item));
  if (filtered.length > 0) {
    suspiciousByFile[path.relative(webRoot, filePath)] = filtered.sort((a, b) =>
      a.localeCompare(b),
    );
  }
}

const baseKeys = new Set(Object.keys(localeMaps[baseLocale]));

const missingByLocale = {};
const unusedByLocale = {};
const stats = {
  baseLocale,
  usedKeyCount: usedKeys.size,
  localeStats: {},
};

for (const [locale, translations] of Object.entries(localeMaps)) {
  const localeKeys = new Set(Object.keys(translations));
  const missing = [...baseKeys].filter((key) => !localeKeys.has(key)).sort((a, b) =>
    a.localeCompare(b),
  );
  const unused = [...localeKeys]
    .filter((key) => !usedKeys.has(key) && !isVariantKeyUsed(key, usedKeys))
    .sort((a, b) => a.localeCompare(b));
  missingByLocale[locale] = missing;
  unusedByLocale[locale] = unused;
  stats.localeStats[locale] = {
    keyCount: localeKeys.size,
    missingCount: missing.length,
    unusedCount: unused.length,
  };
}

ensureDir(reportDir);
fs.writeFileSync(
  path.join(reportDir, 'missing-by-locale.json'),
  JSON.stringify(sortObjectByKey(missingByLocale), null, 2),
  'utf8',
);
fs.writeFileSync(
  path.join(reportDir, 'unused-by-locale.json'),
  JSON.stringify(sortObjectByKey(unusedByLocale), null, 2),
  'utf8',
);
fs.writeFileSync(
  path.join(reportDir, 'suspicious-hardcoded.json'),
  JSON.stringify(sortObjectByKey(suspiciousByFile), null, 2),
  'utf8',
);
fs.writeFileSync(
  path.join(reportDir, 'stats.json'),
  JSON.stringify(stats, null, 2),
  'utf8',
);

console.log('i18n audit generated in .i18n-report/');
for (const [locale, localeStat] of Object.entries(stats.localeStats)) {
  console.log(
    `${locale}: keys=${localeStat.keyCount}, missing=${localeStat.missingCount}, unused=${localeStat.unusedCount}`,
  );
}
