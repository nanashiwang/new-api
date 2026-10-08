// Synthetic browser QA only. No request may reach a real API.
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer, loadConfigFromFile } from 'vite';
import { stop as stopEsbuild } from 'esbuild';
const { chromium } = await import(
  process.env.PLAYWRIGHT_MODULE || 'playwright'
);
const root = fileURLToPath(new URL('..', import.meta.url));
const fixture = await fs.mkdtemp(path.join(root, '.claude-cache-qa-'));
const output = await fs.mkdtemp(
  path.join(os.tmpdir(), 'claude-cache-browser-'),
);
let server, browser, page;
try {
  await fs.writeFile(
    path.join(fixture, 'index.html'),
    '<html><div id="root"></div><script type="module" src="./entry.jsx"></script></html>',
  );
  await fs.writeFile(
    path.join(fixture, 'entry.jsx'),
    `
import React,{useState} from 'react';import {createRoot} from 'react-dom/client';
import {BrowserRouter} from 'react-router-dom';import i18n from 'i18next';import {initReactI18next} from 'react-i18next';
import Editor from '/src/pages/Setting/Ratio/components/ModelPricingEditor';
import {renderClaudeModelPrice} from '/src/helpers/render';import {calculateModelPrice} from '/src/helpers/price';
import {UserContext} from '/src/context/User';import {StatusContext} from '/src/context/Status';import '/src/index.css';
i18n.use(initReactI18next).init({lng:'zh',resources:{zh:{translation:{}}},interpolation:{escapeValue:false}});
const initial={ModelRatio:'{"claude-cache-audit":10,"generic-model":1}',CompletionRatio:'{"claude-cache-audit":5,"generic-model":1}',CacheRatio:'{"claude-cache-audit":0.05}',CreateCacheRatio:'{"claude-cache-audit":1.25,"generic-model":1.25}',CacheCreationPriceMeta:'{"claude-cache-audit":1.6}'};
function App(){const [options,setOptions]=useState(initial),[version,setVersion]=useState(0);
window.qa={reload:()=>setVersion(v=>v+1),setOptions};
const displayPrice=v=>'$'+v;displayPrice.toAmount=v=>v;displayPrice.currencySymbol='$';
const prices=calculateModelPrice({record:{quota_type:0,model_ratio:10,completion_ratio:5,supports_cache_creation:true,cache_creation_ratio:1.25,supports_cache_creation_ttl:true,cache_creation_ratio_1h:2},selectedGroup:'test',groupRatio:{test:1},tokenUnit:'M',displayPrice,currency:'USD'});
return <BrowserRouter><UserContext.Provider value={[{user:{id:901,role:100}},()=>{}]}><StatusContext.Provider value={[{status:{quota_per_unit:500000}},()=>{}]}>
<main style={{padding:12}}><section data-testid="prices">{prices.pricingItems.map(p=><p key={p.key}>{p.label} {p.value}</p>)}</section>
<section data-testid="formula">{renderClaudeModelPrice({prompt_tokens:400,completion_tokens:0,model_ratio:1,completion_ratio:5,group_ratio:1,model_price:-1,cache_creation_tokens:400,cache_creation_tokens_5m:80,cache_creation_tokens_1h:120,cache_creation_ratio:1.25,cache_creation_ratio_5m:1.25,cache_creation_ratio_1h:2})}</section>
<Editor key={version} options={options} refresh={async()=>setOptions(window.savedOptions || initial)}/></main>
</StatusContext.Provider></UserContext.Provider></BrowserRouter>}
createRoot(document.getElementById('root')).render(<App/>);`,
  );
  const { config } = await loadConfigFromFile(
    { command: 'serve', mode: 'development' },
    path.join(root, 'vite.config.js'),
  );
  server = await createServer({
    ...config,
    configFile: false,
    root,
    server: { host: '127.0.0.1', port: 0, open: false, proxy: {} },
    plugins: [
      ...config.plugins
        .flat(Infinity)
        .filter((p) => p?.name !== '@code-inspector/vite'),
      {
        name: 'deny-unmocked-api',
        configureServer(vite) {
          vite.middlewares.use((req, res, next) => {
            if (/^\/(?:api|v1)\//.test(req.url || '')) {
              res.statusCode = 502;
              res.end('blocked QA request');
              return;
            }
            next();
          });
        },
      },
    ],
  });
  await server.listen();
  const base = `http://127.0.0.1:${server.httpServer.address().port}`;
  browser = await chromium.launch({
    headless: true,
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
  });
  page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  page.setDefaultTimeout(15000);
  const errors = [],
    saved = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.addInitScript(() => {
    localStorage.setItem('user', JSON.stringify({ id: 901, role: 100 }));
    localStorage.setItem('quota_display_type', 'USD');
    localStorage.setItem('quota_per_unit', '500000');
  });
  await page.route('**/*', async (route) => {
    const req = route.request(),
      url = new URL(req.url());
    if (url.origin !== base) {
      errors.push('Unexpected external request');
      await route.abort();
      return;
    }
    if (!url.pathname.startsWith('/api/')) {
      await route.continue();
      return;
    }
    if (req.method() === 'PUT') {
      assert.equal(url.pathname, '/api/option/');
      const data = req.postDataJSON();
      saved.push(data);
      await page.evaluate((data) => {
        window.savedOptions = {
          ...(window.savedOptions || {
            ModelRatio: '{"claude-cache-audit":10,"generic-model":1}',
            CompletionRatio: '{"claude-cache-audit":5,"generic-model":1}',
            CacheRatio: '{"claude-cache-audit":0.05}',
            CreateCacheRatio:
              '{"claude-cache-audit":1.25,"generic-model":1.25}',
            CacheCreationPriceMeta: '{"claude-cache-audit":1.6}',
          }),
        };
        window.savedOptions[data.key] = data.value;
      }, data);
    } else {
      assert.equal(req.method(), 'GET');
    }
    await route.fulfill({ json: { success: true, data: [] } });
  });
  await page.goto(`${base}/${path.basename(fixture)}/index.html`);
  await page
    .getByTestId('prices')
    .getByText('缓存创建价格（1小时） $40', { exact: true })
    .waitFor();
  await page
    .getByTestId('formula')
    .getByText(/未细分缓存创建：200 tokens/)
    .waitFor();
  await page
    .getByTestId('formula')
    .getByText(/0.001980/)
    .waitFor();
  const label = page.getByText('缓存创建价格（5分钟）', { exact: true });
  await label.waitFor();
  await page
    .getByText(
      '1小时缓存创建价格：40 USD / 1M tokens（5分钟价格 × 1.6，自动计算）',
      { exact: true },
    )
    .waitFor();
  const input = label.locator('../..').locator('input[type=text]');
  await input.fill('30');
  await page
    .getByText(
      '1小时缓存创建价格：48 USD / 1M tokens（5分钟价格 × 1.6，自动计算）',
      { exact: true },
    )
    .waitFor();
  await page.getByText('应用更改', { exact: true }).click();
  await page.waitForFunction(
    () =>
      window.savedOptions?.CreateCacheRatio &&
      JSON.parse(window.savedOptions.CreateCacheRatio)['claude-cache-audit'] ===
        1.5,
  );
  assert(!saved.some((s) => s.key === 'CacheCreationPriceMeta'));
  assert.equal(
    JSON.parse(saved.find((s) => s.key === 'CreateCacheRatio').value)[
      'generic-model'
    ],
    1.25,
  );
  await page.evaluate(() => window.qa.reload());
  await page
    .getByText(
      '1小时缓存创建价格：48 USD / 1M tokens（5分钟价格 × 1.6，自动计算）',
      { exact: true },
    )
    .waitFor();
  await page.screenshot({
    path: path.join(output, 'desktop.png'),
    fullPage: true,
  });
  console.log(
    'PASS editor derives 1h price, saves only original base configuration, preserves generic model and survives reopen',
  );
  await page.getByText('generic-model', { exact: true }).first().click();
  await page
    .getByText('未提供缓存时长能力信息，当前显示通用缓存创建价。', {
      exact: true,
    })
    .waitFor();
  assert.equal(await page.getByText(/1小时缓存创建价格：48 USD/).count(), 0);
  await page.setViewportSize({ width: 390, height: 844 });
  await page
    .getByText('未提供缓存时长能力信息，当前显示通用缓存创建价。', {
      exact: true,
    })
    .waitFor();
  assert(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
    'mobile layout must not overflow horizontally',
  );
  await page.screenshot({
    path: path.join(output, 'mobile.png'),
    fullPage: true,
  });
  assert.deepEqual(errors, []);
  console.log(
    'PASS generic model does not acquire TTL, formula includes remainder, public price includes 1h; browser errors=0; ' +
      output,
  );
} catch (error) {
  if (page)
    await page
      .screenshot({ path: path.join(output, 'failure.png'), fullPage: true })
      .catch(() => {});
  console.error('Browser QA artifacts: ' + output);
  throw error;
} finally {
  await Promise.allSettled([browser?.close(), server?.close()]);
  await stopEsbuild();
  await fs.rm(fixture, { recursive: true, force: true });
  console.log(
    'CLEANUP browser, Vite and esbuild stopped; temporary entry removed',
  );
}
