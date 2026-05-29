#!/usr/bin/env node
/**
 * prerender.mjs — SEO 阶段 2：build 后预渲染脚本。
 *
 * 流程：
 *   1. 启动 vite preview 服务（基于已构建的 web/dist/）
 *   2. 用 puppeteer-core 连接系统 chromium，访问每个公开路由
 *   3. 等 React 渲染完成，page.content() 拿到完整 HTML
 *   4. 写入 web/dist/<route>/index.html，供后端 LoadPrerendered 在启动时
 *      从 embed.FS 读到并查 map 返回（service/seo_meta.go）
 *
 * 为什么不用 vite-prerender-plugin（与 plan 不同）：
 *   - 项目用 BrowserRouter + 大量 createRoot 副作用代码（Semi UI、i18n、
 *     localStorage 访问），改 SSR 入口工作量大且踩坑面广
 *   - puppeteer 是真浏览器，与生产环境渲染 100% 一致，无需为 SSR 兼容性
 *     做任何源码改动
 *
 * 环境要求：
 *   - 系统 chromium（Docker builder 阶段已 apt install）；优先读
 *     PUPPETEER_EXECUTABLE_PATH 环境变量，默认 /usr/bin/chromium
 *   - 跳过预渲染：设 PRERENDER_SKIP=1（本地开发或 chromium 缺失时使用）
 */

import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const WEB_ROOT = resolve(__dirname, '..')
const DIST_DIR = resolve(WEB_ROOT, 'dist')

// 预渲染的公开路由清单。与 service/seo_meta.go routeMetaMap 中的可索引路径保持一致。
// 注意：根路径 `/` 不需要预渲染（直接覆盖 dist/index.html 会破坏 SPA 入口，
// 反而让所有未识别路由的 fallback 也带上首页内容）。
// 后端按 path 精确查 map：只有命中预渲染产物的路径返回静态 HTML，
// 其余路径继续走 RenderIndexWithMeta 模板替换链路。
const ROUTES = ['/login', '/register', '/pricing', '/about']

const PREVIEW_PORT = Number(process.env.PRERENDER_PREVIEW_PORT || 4173)
const PREVIEW_HOST = process.env.PRERENDER_PREVIEW_HOST || '127.0.0.1'
const CHROMIUM_PATH = process.env.PUPPETEER_EXECUTABLE_PATH || '/usr/bin/chromium'
const PAGE_READY_TIMEOUT_MS = Number(process.env.PRERENDER_TIMEOUT_MS || 20000)
const SKIP = process.env.PRERENDER_SKIP === '1'
const JSON_HEADERS = { 'Content-Type': 'application/json; charset=utf-8' }

const mockStatus = {
  setup: true,
  system_name: 'New API',
  logo: '/logo.png',
  footer_html: '',
  quota_per_unit: 500000,
  display_in_currency: false,
  quota_display_type: 'USD',
  enable_drawing: false,
  enable_task: false,
  enable_data_export: false,
  data_export_default_time: 'hour',
  default_collapse_sidebar: false,
  mj_notify_enabled: false,
  docs_link: 'https://docs.newapi.pro',
  demo_site_enabled: false,
  self_use_mode_enabled: false,
  email_verification: true,
  turnstile_check: false,
  turnstile_site_key: '',
  user_agreement_enabled: true,
  privacy_policy_enabled: true,
  github_oauth: false,
  discord_oauth: false,
  oidc_enabled: false,
  wechat_login: false,
  linuxdo_oauth: false,
  telegram_oauth: false,
  custom_oauth_providers: [],
  announcements: [],
  announcements_enabled: false,
  HeaderNavModules: JSON.stringify({
    home: true,
    console: true,
    pricing: { enabled: true, requireAuth: false },
    docs: false,
    about: false,
    usage: false,
  }),
  SidebarModulesAdmin: JSON.stringify({
    chat: { enabled: true, playground: true, imagePlayground: true, chat: true },
    console: { enabled: true, detail: true, token: true, log: true },
    personal: { enabled: true, topup: true, personal: true },
    admin: { enabled: true, channel: true, models: true, setting: true },
  }),
}

const apiMocks = new Map([
  ['/api/status', { success: true, message: '', data: mockStatus }],
  ['/api/user/self', { success: false, message: '', data: null }],
  ['/api/about', { success: true, message: '', data: '' }],
  [
    '/api/pricing',
    {
      success: true,
      message: '',
      data: [],
      vendors: [],
      group_ratio: {},
      usable_group: [],
      supported_endpoint: {},
      auto_groups: [],
    },
  ],
  ['/api/subscription/plans', { success: true, message: '', data: [] }],
])

if (SKIP) {
  console.log('[prerender] PRERENDER_SKIP=1, skipping. (后端 RenderIndexWithMeta 会兜底处理)')
  process.exit(0)
}

// 不强制依赖 chromium：本地开发、CI apt 装 chromium 失败、镜像里没浏览器等场景，
// 都允许跳过预渲染，主构建继续。后端 GetPrerenderedHTML 找不到产物会自动旁路到模板替换。
if (!existsSync(CHROMIUM_PATH)) {
  console.warn(`[prerender] chromium not found at ${CHROMIUM_PATH}, skipping. (后端 RenderIndexWithMeta 会兜底处理)`)
  process.exit(0)
}

async function startPreviewServer() {
  return new Promise((resolveStart, rejectStart) => {
    const child = spawn(
      'bunx',
      ['vite', 'preview', '--host', PREVIEW_HOST, '--port', String(PREVIEW_PORT), '--strictPort'],
      { cwd: WEB_ROOT, stdio: ['ignore', 'pipe', 'pipe'] },
    )
    let resolved = false
    const onData = (chunk) => {
      const text = chunk.toString()
      process.stdout.write(`[preview] ${text}`)
      if (!resolved && text.includes(`${PREVIEW_HOST}:${PREVIEW_PORT}`)) {
        resolved = true
        resolveStart(child)
      }
    }
    child.stdout.on('data', onData)
    child.stderr.on('data', onData)
    child.on('error', (err) => {
      if (!resolved) {
        resolved = true
        rejectStart(err)
      }
    })
    child.on('exit', (code) => {
      if (!resolved) {
        resolved = true
        rejectStart(new Error(`vite preview exited early with code ${code}`))
      }
    })
    // 兜底超时：若 15s 内 stdout 没看到端口监听字样，仍尝试连接（vite 不同版本输出格式有差异）。
    setTimeout(() => {
      if (!resolved) {
        resolved = true
        resolveStart(child)
      }
    }, 15000)
  })
}

async function prerenderRoute(browser, route) {
  const page = await browser.newPage()
  try {
    await page.setRequestInterception(true)
    page.on('request', (request) => {
      const requestUrl = new URL(request.url())
      const mock = apiMocks.get(requestUrl.pathname)
      if (requestUrl.origin === `http://${PREVIEW_HOST}:${PREVIEW_PORT}` && mock) {
        request.respond({
          status: 200,
          headers: JSON_HEADERS,
          body: JSON.stringify(mock),
        })
        return
      }
      if (requestUrl.hostname === 'challenges.cloudflare.com') {
        request.abort()
        return
      }
      request.continue()
    })

    const url = `http://${PREVIEW_HOST}:${PREVIEW_PORT}${route}`
    console.log(`[prerender] ${url}`)
    await page.goto(url, { waitUntil: 'networkidle0', timeout: PAGE_READY_TIMEOUT_MS })
    // 多保险一层：等 #root 真正有子节点（React 已 mount）
    await page.waitForFunction(
      () => document.querySelector('#root')?.children?.length > 0,
      { timeout: PAGE_READY_TIMEOUT_MS },
    ).catch(() => {
      console.warn(`[prerender] #root 未渲染节点，仍写出当前 HTML：${route}`)
    })
    await page.evaluate(() => {
      document.querySelectorAll('.semi-toast-wrapper').forEach((node) => node.remove())
    })
    const html = await page.content()
    const outDir = resolve(DIST_DIR, route.replace(/^\//, ''))
    await mkdir(outDir, { recursive: true })
    await writeFile(resolve(outDir, 'index.html'), html, 'utf8')
    console.log(`[prerender] wrote ${outDir}/index.html (${html.length} bytes)`)
  } finally {
    await page.close()
  }
}

async function main() {
  let puppeteer
  try {
    puppeteer = await import('puppeteer-core')
  } catch (err) {
    console.warn('[prerender] puppeteer-core not installed, skipping. ' + err.message)
    return
  }

  let previewProcess
  let browser
  try {
    console.log(`[prerender] starting vite preview on ${PREVIEW_HOST}:${PREVIEW_PORT}`)
    previewProcess = await startPreviewServer()
    await new Promise((r) => setTimeout(r, 500))
    console.log(`[prerender] launching chromium at ${CHROMIUM_PATH}`)
    browser = await puppeteer.default.launch({
      executablePath: CHROMIUM_PATH,
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
    })
    for (const route of ROUTES) {
      await prerenderRoute(browser, route)
    }
  } finally {
    if (browser) await browser.close().catch(() => {})
    if (previewProcess && !previewProcess.killed) previewProcess.kill('SIGTERM')
  }
}

main().catch((err) => {
  console.error('[prerender] failed:', err)
  // 不阻断主构建：阶段 1 的 head 注入仍生效，预渲染是锦上添花
  process.exit(0)
})
