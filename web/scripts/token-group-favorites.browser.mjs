// Run with a locally installed Playwright, or set PLAYWRIGHT_MODULE to its index.mjs.
// All API calls use synthetic fixtures; no live account, token or upstream call is used.
import fs from 'node:fs';
import { spawn } from 'node:child_process';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';
import path from 'node:path';
const { chromium } = await import(
  process.env.PLAYWRIGHT_MODULE || 'playwright'
);
const root = fileURLToPath(new URL('..', import.meta.url));
const prefix = `.favorites-qa-${process.pid}`;
const port = process.env.FAVORITES_QA_PORT || '5193';
const baseURL = `http://127.0.0.1:${port}`;
const files = [root + '/' + prefix + '.html', root + '/' + prefix + '.jsx'];
const delay = (ms) => new Promise((r) => setTimeout(r, ms));
let browser, server, debugPage;
(async () => {
  fs.writeFileSync(
    files[0],
    '<html><div id="root"></div><script type="module" src="/' +
      prefix +
      '.jsx"></script></html>',
  );
  fs.writeFileSync(
    files[1],
    `import React,{useState} from 'react';
import {createRoot} from 'react-dom/client';
import {BrowserRouter} from 'react-router-dom';
import i18n from 'i18next'; import {initReactI18next} from 'react-i18next';
import {UserContext} from './src/context/User'; import {StatusContext} from './src/context/Status';
import EditTokenModal from './src/components/table/tokens/modals/EditTokenModal';
import './src/index.css';
i18n.use(initReactI18next).init({lng:'zh',resources:{zh:{translation:{}}},interpolation:{escapeValue:false}});
function App(){const [visible,setVisible]=useState(true),[id,setId]=useState(901),[edit,setEdit]=useState({});window.qa={switchUser:setId,open:()=>setVisible(true),close:()=>setVisible(false),edit:()=>{setEdit({id:99});setVisible(true)}};
return <BrowserRouter><UserContext.Provider value={[{user:{id,role:1}},()=>{}]}><StatusContext.Provider value={[{status:{quota_per_unit:500000}},()=>{}]}><EditTokenModal visiable={visible} editingToken={edit} handleClose={()=>setVisible(false)} refresh={async()=>{}}/></StatusContext.Provider></UserContext.Provider></BrowserRouter>};
createRoot(document.getElementById('root')).render(<App/>);`,
  );
  server = spawn(
    process.execPath,
    [
      'node_modules/vite/bin/vite.js',
      '--host',
      '127.0.0.1',
      '--port',
      port,
      '--strictPort',
    ],
    { cwd: root, detached: true, stdio: ['ignore', 'pipe', 'pipe'] },
  );
  let logs = '';
  server.stdout.on('data', (b) => (logs += b));
  server.stderr.on('data', (b) => (logs += b));
  for (let i = 0; i < 80; i++) {
    try {
      if ((await fetch(baseURL + '/' + prefix + '.html')).ok) break;
    } catch {}
    if (i === 79) throw Error(logs);
    await delay(250);
  }
  browser = await chromium.launch({
    headless: true,
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
  });
  const context = await browser.newContext({
    viewport: { width: 1050, height: 1000 },
  });
  const page = await context.newPage();
  debugPage = page;
  page.setDefaultTimeout(10000);
  const errors = [];
  page.on('pageerror', (e) => {
    errors.push(e.message);
    console.error('PAGE ERROR', e.message);
  });
  page.on('console', (m) => {
    if (m.type() === 'error' && !m.text().startsWith('Warning:'))
      console.error('CONSOLE', m.text());
  });
  let groups = {
    'OpenAI · A': { desc: '日常使用', ratio: 1 },
    'Claude · B': { desc: '编程使用', ratio: 2 },
  };
  let submitBody;
  let failGroups = false;
  let holdNextGroups = false;
  let releaseGroups;
  let groupsRequests = 0;
  const modelRequests = [];

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname;
    let data = [];
    if (path === '/api/user/self/groups') {
      groupsRequests += 1;
      data = groups;
      if (failGroups) {
        await route.fulfill({
          json: {
            success: false,
            message: 'Synthetic permission refresh failure',
          },
        });
        return;
      }
      if (holdNextGroups) {
        holdNextGroups = false;
        await new Promise((resolve) => {
          releaseGroups = resolve;
        });
      }
    } else if (path === '/api/health/groups')
      data = { enabled: false, groups: [] };
    else if (path === '/api/token/models') {
      modelRequests.push(new URL(req.url()).searchParams.get('group'));
      data = ['test-model'];
    } else if (path === '/api/token/99')
      data = {
        id: 99,
        name: 'existing',
        group: 'removed',
        expired_time: -1,
        model_limits: '',
        channel_limits: '',
        remain_quota: 500000,
        unlimited_quota: true,
      };
    if (req.method() === 'POST' || req.method() === 'PUT') {
      submitBody = req.postDataJSON();
    }
    await route.fulfill({ json: { success: true, data } });
  });
  await page.addInitScript(() => {
    localStorage.setItem('user', JSON.stringify({ id: 901, role: 1 }));
    localStorage.setItem(
      'token-group-favorites:v1:901',
      JSON.stringify(['OpenAI · A', 'Claude · B', 'removed']),
    );
  });
  await page.goto(baseURL + '/' + prefix + '.html');
  await page.getByText('创建新的令牌', { exact: true }).waitFor();
  await page.getByLabel('我的收藏分组').click();
  await page.getByRole('option', { name: /Claude · B/ }).click();
  await page.getByRole('button', { name: '★ 取消收藏', exact: true }).waitFor();
  await delay(150);
  const requestsBeforeStar = modelRequests.length;
  await page.getByRole('button', { name: '★ 取消收藏', exact: true }).click();
  assert(
    !(
      await page.evaluate(() =>
        JSON.parse(localStorage.getItem('token-group-favorites:v1:901')),
      )
    ).includes('Claude · B'),
  );
  await page
    .getByRole('button', { name: '☆ 收藏当前分组', exact: true })
    .click();
  assert(
    (
      await page.evaluate(() =>
        JSON.parse(localStorage.getItem('token-group-favorites:v1:901')),
      )
    ).includes('Claude · B'),
  );
  await delay(150);
  assert.equal(
    modelRequests.length,
    requestsBeforeStar,
    'starring must not switch groups or reload model permissions',
  );
  assert.equal(
    submitBody,
    undefined,
    'starring must not create or update a token',
  );
  await page.getByLabel('我的收藏分组').click();
  assert.equal(await page.getByRole('option', { name: /removed/ }).count(), 0);
  await page.keyboard.press('Escape');
  await page
    .getByPlaceholder('请输入名称', { exact: true })
    .fill('favorites test');
  await page.getByRole('button', { name: /提交/ }).click();
  await delay(400);
  assert.equal(submitBody?.group, 'Claude · B');
  assert(!('group_vendor' in submitBody));
  assert(!('favorites' in submitBody));
  console.log(
    'PASS create token: favorite cross-vendor choice, add/remove, invalid filtered, original submitted group',
  );
  await page.evaluate(() => window.qa.open());
  await page.getByLabel('我的收藏分组').waitFor();
  await page.evaluate(() => window.qa.switchUser(902));
  await page.getByText('选择分组后可收藏', { exact: true }).waitFor();
  console.log('PASS account isolation');
  await page
    .locator('[role=combobox][aria-labelledby=group_vendor-label]')
    .click();
  await page.getByRole('option', { name: /OpenAI/ }).click();
  await page.locator('[role=combobox][aria-labelledby=group-label]').click();
  await page.getByRole('option', { name: /OpenAI · A/ }).click();
  await page
    .getByRole('button', { name: '☆ 收藏当前分组', exact: true })
    .click();
  assert.deepEqual(
    await page.evaluate(() =>
      JSON.parse(localStorage.getItem('token-group-favorites:v1:902')),
    ),
    ['OpenAI · A'],
  );
  console.log('PASS favorite via original vendor/group selector');
  await page.evaluate(() => window.qa.switchUser(901));
  await page.getByLabel('我的收藏分组').click();
  await page.getByRole('option', { name: /OpenAI · A/ }).click();
  await page
    .locator('[role=combobox][aria-labelledby=group-label]')
    .filter({ hasText: 'OpenAI · A' })
    .waitFor();
  await page.evaluate(() => window.qa.close());
  await delay(250);
  await page.evaluate(() => window.qa.open());
  await page.getByLabel('我的收藏分组').click();
  await page.getByRole('option', { name: /Claude · B/ }).click();
  await page
    .locator('[role=combobox][aria-labelledby=group-label]')
    .filter({ hasText: 'Claude · B' })
    .waitFor();
  console.log('PASS reopen persistence and cross-vendor selection');
  await page.evaluate(() => {
    window.qa.originalSetItem = Storage.prototype.setItem;
    Storage.prototype.setItem = function (key, value) {
      if (key.startsWith('token-group-favorites:')) throw Error('quota');
      return window.qa.originalSetItem.call(this, key, value);
    };
  });
  await page.getByRole('button', { name: '★ 取消收藏', exact: true }).click();
  await page.getByRole('status').filter({ hasText: '无法保存收藏' }).waitFor();
  console.log('PASS storage failure keeps form usable');
  await page.evaluate(() => {
    Storage.prototype.setItem = window.qa.originalSetItem;
  });
  await page.screenshot({
    path: path.join(tmpdir(), 'newapi-favorites-desktop.png'),
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await delay(250);
  assert(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  );
  await page.screenshot({
    path: path.join(tmpdir(), 'newapi-favorites-mobile.png'),
  });
  console.log('PASS mobile no horizontal overflow');
  await page.evaluate(() => window.qa.close());
  await delay(250);
  await page.evaluate(() => window.qa.edit());
  await page.getByText('更新令牌信息', { exact: true }).waitFor();
  await delay(300);
  await page.getByLabel('我的收藏分组').click();
  assert.equal(await page.getByRole('option', { name: /removed/ }).count(), 0);
  await page.getByRole('option', { name: /OpenAI · A/ }).click();
  await page
    .locator('[role=combobox][aria-labelledby=group-label]')
    .filter({ hasText: 'OpenAI · A' })
    .waitFor();
  await page.getByRole('button', { name: /提交/ }).click();
  await delay(400);
  assert.equal(submitBody?.group, 'OpenAI · A');
  await page.evaluate(() => window.qa.open());
  await page.getByLabel('我的收藏分组').click();
  await page.getByLabel('我的收藏分组').locator('input').fill('OpenAI');
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  await page
    .locator('[role=combobox][aria-labelledby=group-label]')
    .filter({ hasText: 'OpenAI · A' })
    .waitFor();
  console.log('PASS keyboard search and selection');
  // Fail closed when refreshing group permissions; a stored favorite is never authority.
  await page.evaluate(() => window.qa.close());
  await delay(250);
  failGroups = true;
  await page.evaluate(() => window.qa.open());
  await page.getByText('选择分组后可收藏', { exact: true }).waitFor();
  await page.getByLabel('我的收藏分组').click();
  await page
    .getByText('暂无可用收藏，请先选择并收藏分组', { exact: true })
    .waitFor();
  assert.equal(await page.getByRole('option').count(), 0);
  await page.keyboard.press('Escape');
  console.log('PASS permission-refresh failure exposes no stored favorites');

  // A late response from account 901 must not replace account 902's latest catalog.
  await page.evaluate(() => window.qa.close());
  await delay(250);
  failGroups = false;
  holdNextGroups = true;
  await page.evaluate(() => window.qa.open());
  for (let i = 0; !releaseGroups && i < 50; i++) await delay(20);
  assert(releaseGroups, 'old account request is in flight');
  groups = { 'Claude · B': { desc: '编程使用', ratio: 2 } };
  const beforeSwitch = groupsRequests;
  await page.evaluate(() => window.qa.switchUser(902));
  for (let i = 0; groupsRequests === beforeSwitch && i < 50; i++)
    await delay(20);
  assert(
    groupsRequests > beforeSwitch,
    'new account requested a fresh catalog',
  );
  await delay(100);
  releaseGroups();
  await delay(200);
  await page.getByText('选择分组后可收藏', { exact: true }).waitFor();
  await page.getByLabel('我的收藏分组').click();
  await page
    .getByText('暂无可用收藏，请先选择并收藏分组', { exact: true })
    .waitFor();
  assert.equal(await page.getByRole('option').count(), 0);
  console.log(
    'PASS delayed old-account response cannot resurrect a revoked favorite',
  );
  assert.deepEqual(errors, []);
  console.log(
    'PASS edit token, preserved unavailable group excluded from favorites, no browser errors',
  );
})()
  .catch(async (e) => {
    console.error(e);
    if (debugPage) {
      console.log('FAIL_BODY', await debugPage.locator('body').innerText());
      await debugPage.screenshot({
        path: path.join(tmpdir(), 'newapi-favorites-failure.png'),
      });
    }
    process.exitCode = 1;
  })
  .finally(async () => {
    if (browser) await browser.close();
    if (server) {
      try {
        process.kill(-server.pid, 'SIGTERM');
      } catch {}
      await delay(400);
      try {
        process.kill(-server.pid, 0);
        process.kill(-server.pid, 'SIGKILL');
      } catch {}
    }
    for (const f of files) fs.rmSync(f, { force: true });
    console.log('QA process cleanup complete');
  });
