/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import ReactDOM from 'react-dom/client';
import axios from 'axios';
import { Check, Copy, Sparkles, FileText, Github, Play, X } from 'lucide-react';
import './styles.css';

const STATUS_KEY = 'status';
const USER_KEY = 'user';
const HOME_CONTENT_KEY = 'home_page_content';
const PublicPages = lazy(() => import('./publicPages.jsx'));
const API_ENDPOINTS = [
  '/v1/chat/completions',
  '/v1/responses',
  '/v1/responses/compact',
  '/v1/messages',
  '/v1beta/models',
  '/v1/embeddings',
  '/v1/rerank',
  '/v1/images/generations',
  '/v1/images/edits',
  '/v1/images/variations',
  '/v1/audio/speech',
  '/v1/audio/transcriptions',
  '/v1/audio/translations',
];

const providerBadges = [
  'OpenAI',
  'Claude',
  'Gemini',
  'Grok',
  'DeepSeek',
  'Qwen',
  'Azure',
  'Midjourney',
  '30+',
];

const defaultStatus = {
  system_name: 'New API',
  logo: '/logo.svg',
  server_address: window.location.origin,
  docs_link: '',
  self_use_mode_enabled: false,
};

function readJson(key) {
  try {
    const value = localStorage.getItem(key);
    return value ? JSON.parse(value) : null;
  } catch {
    return null;
  }
}

function readString(key) {
  try {
    return localStorage.getItem(key) || '';
  } catch {
    return '';
  }
}

function writeJson(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Storage may be unavailable.
  }
}

function parseHeaderModules(raw) {
  const fallback = {
    home: true,
    console: true,
    pricing: true,
    docs: true,
    about: true,
  };
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw);
    return { ...fallback, ...parsed };
  } catch {
    return fallback;
  }
}

function isEnabled(moduleValue) {
  if (moduleValue && typeof moduleValue === 'object') {
    return moduleValue.enabled !== false;
  }
  return moduleValue !== false;
}

function requiresAuth(moduleValue) {
  return Boolean(
    moduleValue && typeof moduleValue === 'object' && moduleValue.requireAuth,
  );
}

function normalizePathname(pathname) {
  return pathname.replace(/\/+$/, '') || '/';
}

function getPublicPageKind(pathname) {
  const path = normalizePathname(pathname);
  if (path === '/pricing') return 'pricing';
  if (path === '/about') return 'about';
  if (path === '/login') return 'login';
  if (path === '/register') return 'register';
  if (path === '/docs' || path.startsWith('/docs/')) return 'docs';
  return '';
}

function isRemoteUrl(value) {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

function isRawHtml(value) {
  const trimmed = value.trimStart().toLowerCase();
  return [
    '<!doctype',
    '<html',
    '<style',
    '<main',
    '<section',
    '<header',
    '<nav',
    '<div',
  ].some((prefix) => trimmed.startsWith(prefix));
}

async function renderMarkupContent(value) {
  if (!value || isRemoteUrl(value) || isRawHtml(value)) return value || '';
  const { marked } = await import('marked');
  return marked.parse(value);
}

function App() {
  const publicPageKind = getPublicPageKind(window.location.pathname);
  const isHomeRoute = normalizePathname(window.location.pathname) === '/';
  const cachedStatus = readJson(STATUS_KEY) || {};
  const cachedUser = readJson(USER_KEY);
  const [status, setStatus] = useState({ ...defaultStatus, ...cachedStatus });
  const [user, setUser] = useState(cachedUser);
  const [homeContent, setHomeContent] = useState('');
  const [notice, setNotice] = useState('');
  const [noticeOpen, setNoticeOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [endpointIndex, setEndpointIndex] = useState(0);

  useEffect(() => {
    if (!isHomeRoute) return undefined;
    let cancelled = false;
    const cachedHomeContent = readString(HOME_CONTENT_KEY);

    if (cachedHomeContent) {
      renderMarkupContent(cachedHomeContent).then((content) => {
        if (!cancelled) setHomeContent(content);
      });
    }

    axios
      .get('/api/status')
      .then((res) => {
        if (cancelled || !res.data?.success) return;
        const nextStatus = { ...defaultStatus, ...res.data.data };
        setStatus(nextStatus);
        writeJson(STATUS_KEY, nextStatus);
        if (nextStatus.system_name) document.title = nextStatus.system_name;
        if (nextStatus.logo) {
          const icon = document.querySelector("link[rel~='icon']");
          if (icon) icon.href = nextStatus.logo;
        }
      })
      .catch(() => {});

    axios
      .get('/api/home_page_content')
      .then((res) => {
        if (cancelled || !res.data?.success) return;
        const rawContent = res.data.data || '';
        return renderMarkupContent(rawContent).then((nextContent) => {
          if (cancelled) return;
          setHomeContent(nextContent);
          if (rawContent) {
            localStorage.setItem(HOME_CONTENT_KEY, nextContent);
          } else {
            localStorage.removeItem(HOME_CONTENT_KEY);
          }
        });
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, [isHomeRoute]);

  useEffect(() => {
    if (!cachedUser) return;
    let cancelled = false;
    axios
      .get('/api/user/self')
      .then((res) => {
        if (cancelled || !res.data?.success) return;
        setUser(res.data.data);
        writeJson(USER_KEY, res.data.data);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const lastCloseDate = localStorage.getItem('notice_close_date');
    if (lastCloseDate === new Date().toDateString()) return undefined;
    const timer = window.setTimeout(() => {
      axios
        .get('/api/notice')
        .then((res) => {
          const content = res.data?.data?.trim();
          if (!res.data?.success || !content) return undefined;
          return renderMarkupContent(content).then((html) => {
            setNotice(html);
            setNoticeOpen(true);
          });
        })
        .catch(() => {});
    }, 4500);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % API_ENDPOINTS.length);
    }, 3000);
    return () => window.clearInterval(timer);
  }, []);

  const navLinks = useMemo(() => {
    const modules = parseHeaderModules(status.HeaderNavModules);
    const docsLink =
      status.docs_link || localStorage.getItem('docs_link') || '';
    return [
      isEnabled(modules.home) && { label: '首页', href: '/' },
      isEnabled(modules.console) && {
        label: '控制台',
        href: user ? '/console' : '/login',
      },
      isEnabled(modules.pricing) && {
        label: '模型广场',
        href: requiresAuth(modules.pricing) && !user ? '/login' : '/pricing',
      },
      isEnabled(modules.docs) &&
        {
          label: '文档',
          href: docsLink || '/docs',
          external: Boolean(docsLink) && isRemoteUrl(docsLink),
        },
      isEnabled(modules.about) && { label: '关于', href: '/about' },
    ].filter(Boolean);
  }, [status, user]);

  const serverAddress = status.server_address || window.location.origin;
  const docsLink = status.docs_link || localStorage.getItem('docs_link') || '';
  const currentEndpoint = API_ENDPOINTS[endpointIndex] || API_ENDPOINTS[0];
  const showRegister = !status.self_use_mode_enabled;

  const copyServerAddress = async () => {
    try {
      await navigator.clipboard.writeText(serverAddress);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  };

  const closeNoticeToday = () => {
    localStorage.setItem('notice_close_date', new Date().toDateString());
    setNoticeOpen(false);
  };

  if (publicPageKind) {
    return (
      <Suspense
        fallback={
          <div className='default-home public-loading'>正在加载...</div>
        }
      >
        <PublicPages
          page={publicPageKind}
          status={status}
          user={user}
          setUser={setUser}
        />
      </Suspense>
    );
  }

  if (isHomeRoute && homeContent) {
    return (
      <div className='default-home custom-page'>
        {isRemoteUrl(homeContent) ? (
          <iframe
            title='Custom Home Page'
            src={homeContent}
            className='custom-frame'
          />
        ) : (
          <div
            className='custom-content'
            dangerouslySetInnerHTML={{
              __html: homeContent,
            }}
          />
        )}
      </div>
    );
  }

  return (
    <div className='default-home home-shell'>
      <header className='topbar public-header--hero'>
        <a className='brand' href='/'>
          <img src={status.logo || '/logo.svg'} alt='' />
          <span>{status.system_name || '山海衡 AI'}</span>
        </a>
        <nav>
          {navLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              target={link.external ? '_blank' : undefined}
              rel={link.external ? 'noreferrer' : undefined}
            >
              {link.label}
            </a>
          ))}
        </nav>
        <div className='actions'>
          {user ? (
            <a className='user-chip' href='/console/personal'>
              {(user.username || 'U').slice(0, 1).toUpperCase()}
              <span>{user.username}</span>
            </a>
          ) : (
            <>
              <a className='login' href='/login'>
                登录
              </a>
              {showRegister && (
                <a className='primary-small' href='/register'>
                  注册
                </a>
              )}
            </>
          )}
        </div>
      </header>

      <main>
        <section className='home-hero'>
          <div className='blur-ball blur-ball-indigo' />
          <div className='blur-ball blur-ball-teal' />
          <div className='home-hero-inner'>
            <div className='home-hero-brand'>
              <img src={status.logo || '/logo.svg'} alt='' />
              <span>{status.system_name || '山海衡 AI'}</span>
            </div>
            <div className='eyebrow'>
              <Sparkles size={16} />
              AI Application Infrastructure
            </div>
            <h1>
              统一的
              <span className='shine-text'>大模型接口网关</span>
            </h1>
            <p className='home-slogan'>一衡调山海，万模皆可达</p>
            <p className='home-desc'>
              聚合万模，平衡调度，一口稳定接入，只需将模型基址替换为：
            </p>
            <div className='home-endpoint-control'>
              <input
                readOnly
                value={serverAddress}
                className='home-endpoint-input'
                aria-label='Base URL'
              />
              <select
                className='home-endpoint-select'
                value={currentEndpoint}
                onChange={(event) => {
                  const nextIndex = API_ENDPOINTS.findIndex(
                    (item) => item === event.target.value,
                  );
                  if (nextIndex >= 0) setEndpointIndex(nextIndex);
                }}
                aria-label='API endpoint'
              >
                {API_ENDPOINTS.map((endpoint) => (
                  <option key={endpoint} value={endpoint}>
                    {endpoint}
                  </option>
                ))}
              </select>
              <button
                type='button'
                className='home-copy-button'
                onClick={copyServerAddress}
                aria-label='复制 API 地址'
              >
                {copied ? <Check size={18} /> : <Copy size={18} />}
              </button>
            </div>
            <div className='home-action-row'>
              <a
                className='home-action-button home-action-primary'
                href='/console'
              >
                <Play size={18} />
                获取密钥
              </a>
              {status.demo_site_enabled && status.version ? (
                <a
                  className='home-action-button home-action-secondary'
                  href='https://github.com/QuantumNous/new-api'
                  target='_blank'
                  rel='noreferrer'
                >
                  <Github size={18} />
                  {status.version}
                </a>
              ) : (
                docsLink && (
                  <a
                    className='home-action-button home-action-secondary'
                    href={docsLink}
                  >
                    <FileText size={18} />
                    文档
                  </a>
                )
              )}
            </div>
            <div className='home-provider-block'>
              <span>支持众多的大模型供应商</span>
              <div className='home-provider-badges'>
                {providerBadges.map((provider) => (
                  <span className='home-provider-badge' key={provider}>
                    {provider}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className='home-section'>
          <div className='home-section-inner'>
            <h2 className='home-section-title'>为什么选择山海衡 AI</h2>
            <div className='home-feature-grid'>
              {[
                [
                  '聚合 40+ 大模型',
                  'OpenAI、Claude、Gemini、DeepSeek、Qwen... 一个统一 API 全部调用，无需多处对接。',
                  '🔗',
                ],
                [
                  '更优的价格',
                  '批量议价、按量计费、缓存命中优惠，成本更低、账单更透明。',
                  '💰',
                ],
                [
                  '稳定高可用',
                  '多节点容灾、智能路由、自动重试，保障调用稳定不中断。',
                  '⚡',
                ],
                [
                  '安全可控',
                  '密钥隔离、用量管控、调用日志与账单全程可查，合规放心。',
                  '🛡️',
                ],
              ].map(([title, desc, icon]) => (
                <article className='home-feature-card' key={title}>
                  <div className='home-feature-icon'>{icon}</div>
                  <h3>{title}</h3>
                  <p>{desc}</p>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className='home-section home-section--alt'>
          <div className='home-section-inner'>
            <h2 className='home-section-title'>三步快速接入</h2>
            <div className='home-steps-grid'>
              {[
                ['1', '注册并获取密钥', '登录控制台，创建专属 API Key。'],
                ['2', '替换模型基址', '把原有 Base URL 换成本平台地址。'],
                [
                  '3',
                  '立即开始调用',
                  '沿用原有 SDK 与参数，直接调用 40+ 模型。',
                ],
              ].map(([n, title, desc]) => (
                <article className='home-step' key={n}>
                  <div className='home-step-no'>{n}</div>
                  <h3>{title}</h3>
                  <p>{desc}</p>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className='home-cta'>
          <h2>一衡调山海，万模皆可达</h2>
          <p>现在就接入，开启统一的大模型调用体验</p>
          <a href='/console' className='home-cta-btn'>
            立即开始
            <span>→</span>
          </a>
        </section>
      </main>

      {noticeOpen && (
        <div className='public-notice-backdrop' role='presentation'>
          <section
            className='public-notice-modal'
            role='dialog'
            aria-modal='true'
            aria-labelledby='public-notice-title'
          >
            <header className='public-notice-header'>
              <h2 id='public-notice-title'>系统公告</h2>
              <button
                type='button'
                className='public-notice-close'
                aria-label='关闭公告'
                onClick={() => setNoticeOpen(false)}
              >
                <X size={18} />
              </button>
            </header>
            <div
              className='public-notice-content'
              dangerouslySetInnerHTML={{ __html: notice }}
            />
            <footer className='public-notice-footer'>
              <button
                type='button'
                className='public-notice-ghost'
                onClick={closeNoticeToday}
              >
                今日关闭
              </button>
              <button
                type='button'
                className='public-notice-primary'
                onClick={() => setNoticeOpen(false)}
              >
                关闭公告
              </button>
            </footer>
          </section>
        </div>
      )}
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
