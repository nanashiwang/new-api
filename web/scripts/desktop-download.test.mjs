import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import {
  detectDesktopOS,
  formatDesktopDownloadSize,
  normalizeDesktopDownloadCatalog,
  orderDesktopDownloadPackages,
} from '../src/pages/DesktopDownload/desktopDownload.js';

const catalog = normalizeDesktopDownloadCatalog(
  {
    version: '1.2.3',
    notes: 'release',
    pub_date: '2026-08-01T00:00:00Z',
    packages: [
      {
        id: 'windows-x64',
        filename: 'YuanHeng_1.2.3_x64-setup.exe',
        size: 20 * 1024 * 1024,
        url: '/desktop/update/releases/1.2.3/YuanHeng_1.2.3_x64-setup.exe',
      },
      {
        id: 'future-linux',
        filename: 'YuanHeng.AppImage',
        size: 1,
        url: 'https://updates.example.com/YuanHeng.AppImage',
      },
      {
        id: 'macos-arm64',
        filename: 'YuanHeng_1.2.3_aarch64.dmg',
        size: 22 * 1024 * 1024,
        url: 'https://updates.example.com/YuanHeng_1.2.3_aarch64.dmg',
      },
    ],
  },
  'https://yuanheng.example/download',
);
assert.equal(catalog.version, '1.2.3');
assert.deepEqual(
  catalog.packages.map((item) => item.id),
  ['macos-arm64', 'windows-x64'],
);
assert.equal(
  catalog.packages[1].url,
  'https://yuanheng.example/desktop/update/releases/1.2.3/YuanHeng_1.2.3_x64-setup.exe',
);

assert.equal(
  detectDesktopOS({
    userAgent: 'Mozilla/5.0 (Windows NT 10.0)',
    platform: 'Win32',
  }),
  'windows',
);
assert.equal(
  detectDesktopOS({
    userAgent: 'Mozilla/5.0 (Macintosh)',
    platform: 'MacIntel',
  }),
  'macos',
);
assert.equal(
  detectDesktopOS({
    userAgent: 'Mozilla/5.0 (iPad)',
    platform: 'MacIntel',
    maxTouchPoints: 5,
  }),
  'other',
);
assert.equal(
  detectDesktopOS({ userAgent: 'Mozilla/5.0 (X11; Linux x86_64)' }),
  'other',
);

assert.deepEqual(
  orderDesktopDownloadPackages(catalog.packages, 'windows').map(
    (item) => item.id,
  ),
  ['windows-x64', 'macos-arm64'],
);
assert.equal(formatDesktopDownloadSize(22 * 1024 * 1024), '22.0 MB');
assert.equal(formatDesktopDownloadSize(512), '1 KB');

for (const invalid of [
  {
    version: '1.2.3',
    packages: [
      {
        id: 'windows-x64',
        filename: '../bad.exe',
        size: 1,
        url: 'https://example.com/bad.exe',
      },
    ],
  },
  {
    version: '1.2.3',
    packages: [
      {
        id: 'windows-x64',
        filename: 'safe_x64-setup.exe',
        size: 1,
        url: 'javascript:alert(1)',
      },
    ],
  },
  {
    version: '1.2.3',
    packages: [
      {
        id: 'windows-x64',
        filename: 'one_x64-setup.exe',
        size: 1,
        url: 'https://example.com/one_x64-setup.exe',
      },
      {
        id: 'windows-x64',
        filename: 'two_x64-setup.exe',
        size: 1,
        url: 'https://example.com/two_x64-setup.exe',
      },
    ],
  },
  {
    version: '1.2.3',
    packages: [
      {
        id: 'windows-x64',
        filename: 'wrong.dmg',
        size: 1,
        url: 'https://example.com/wrong.dmg',
      },
    ],
  },
  {
    version: '1.2.3',
    packages: [
      {
        id: 'windows-x64',
        filename: 'safe_x64-setup.exe',
        size: 1,
      },
    ],
  },
  {
    version: '1.2.3',
    packages: [
      {
        id: 'windows-x64',
        filename: 'safe_x64-setup.exe',
        size: 1,
        url: 'https://user:secret@example.com/safe_x64-setup.exe',
      },
    ],
  },
]) {
  assert.throws(() => normalizeDesktopDownloadCatalog(invalid));
}

Promise.all([
  readFile(
    new URL('../src/pages/Home/EnterpriseHome.jsx', import.meta.url),
    'utf8',
  ),
  readFile(
    new URL('../src/components/layout/PublicHomeShell.jsx', import.meta.url),
    'utf8',
  ),
  readFile(new URL('../src/index.jsx', import.meta.url), 'utf8'),
  readFile(
    new URL('../src/hooks/common/useNavigation.js', import.meta.url),
    'utf8',
  ),
])
  .then(([homeSource, shellSource, entrySource, navigationSource]) => {
    assert.match(homeSource, /to='\/download'/);
    assert.match(homeSource, /t\('下载客户端'\)/);
    assert.match(shellSource, /DesktopDownload/);
    assert.match(shellSource, /prefetchApp: false/);
    assert.match(entrySource, /download/);
    assert.match(navigationSource, /itemKey: 'download'/);
    assert.match(navigationSource, /to: '\/download'/);
    assert.match(
      navigationSource,
      /if \(link\.itemKey === 'download'\) \{\s*return true;/,
    );

    console.log('desktop download helpers: ok');
  })
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
