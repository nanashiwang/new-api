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

import React, { useEffect, useMemo, useState } from 'react';
import ReactDOM from 'react-dom/client';
import axios from 'axios';
import {
  ArrowRight,
  BookOpen,
  Check,
  Copy,
  Gauge,
  KeyRound,
  Network,
  ShieldCheck,
  Sparkles,
  WalletCards,
} from 'lucide-react';
import './styles.css';

const STATUS_KEY = 'status';
const USER_KEY = 'user';
const HOME_CONTENT_KEY = 'home_page_content';

const defaultStatus = {
  system_name: 'New API',
  logo: '/logo.png',
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

function App() {
  const cachedStatus = readJson(STATUS_KEY) || {};
  const cachedUser = readJson(USER_KEY);
  const [status, setStatus] = useState({ ...defaultStatus, ...cachedStatus });
  const [user, setUser] = useState(cachedUser);
  const [homeContent, setHomeContent] = useState(readString(HOME_CONTENT_KEY));
  const [notice, setNotice] = useState('');
  const [noticeOpen, setNoticeOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;

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
        const nextContent = res.data.data || '';
        setHomeContent(nextContent);
        if (nextContent) {
          localStorage.setItem(HOME_CONTENT_KEY, nextContent);
        } else {
          localStorage.removeItem(HOME_CONTENT_KEY);
        }
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, []);

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
          if (res.data?.success && content) {
            setNotice(content);
            setNoticeOpen(true);
          }
        })
        .catch(() => {});
    }, 4500);
    return () => window.clearTimeout(timer);
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
        docsLink && {
          label: '文档',
          href: docsLink,
          external: isRemoteUrl(docsLink),
        },
      isEnabled(modules.about) && { label: '关于', href: '/about' },
    ].filter(Boolean);
  }, [status, user]);

  const serverAddress = status.server_address || window.location.origin;
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

  if (homeContent) {
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
              __html: isRawHtml(homeContent) ? homeContent : homeContent,
            }}
          />
        )}
      </div>
    );
  }

  return (
    <div className='default-home'>
      <header className='topbar'>
        <a className='brand' href='/'>
          <img src={status.logo || '/logo.png'} alt='' />
          <span>{status.system_name || 'New API'}</span>
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
        <section className='hero'>
          <div className='hero-copy'>
            <div className='eyebrow'>
              <Sparkles size={16} />
              AI Application Infrastructure
            </div>
            <h1>
              统一的大模型
              <span>接口网关</span>
            </h1>
            <p>
              聚合 OpenAI、Claude、Gemini、DeepSeek、Qwen、Azure、Midjourney
              等渠道， 提供统一协议、智能调度、计费统计和稳定的 API Key 管理。
            </p>
            <div className='cta-row'>
              <a className='primary' href={user ? '/console' : '/register'}>
                {user ? '进入控制台' : '开始使用'}
                <ArrowRight size={18} />
              </a>
              <a className='secondary' href='/pricing'>
                查看模型广场
              </a>
              {(status.docs_link || localStorage.getItem('docs_link')) && (
                <a
                  className='secondary icon-link'
                  href={status.docs_link || localStorage.getItem('docs_link')}
                >
                  <BookOpen size={17} />
                  文档
                </a>
              )}
            </div>
            <div className='endpoint'>
              <span>Base URL</span>
              <code>{serverAddress}</code>
              <button
                type='button'
                onClick={copyServerAddress}
                aria-label='复制 API 地址'
              >
                {copied ? <Check size={16} /> : <Copy size={16} />}
              </button>
            </div>
          </div>

          <div className='terminal-card' aria-label='API 示例'>
            <div className='terminal-head'>
              <span />
              <span />
              <span />
            </div>
            <pre>{`curl ${serverAddress}/v1/chat/completions \\
  -H "Authorization: Bearer $NEW_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4.1",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'`}</pre>
          </div>
        </section>

        <section className='stats-grid'>
          <Stat icon={<Network />} value='40+' label='上游渠道' />
          <Stat icon={<KeyRound />} value='统一 Key' label='OpenAI 兼容协议' />
          <Stat
            icon={<WalletCards />}
            value='透明计费'
            label='用量与余额统计'
          />
          <Stat
            icon={<ShieldCheck />}
            value='稳定调度'
            label='限流与故障切换'
          />
        </section>

        <section className='feature-grid'>
          <Feature
            icon={<Gauge />}
            title='更轻的首屏'
            text='首页不再挂载完整控制台壳，优先用本地缓存渲染，再后台刷新状态。'
          />
          <Feature
            icon={<Network />}
            title='多渠道聚合'
            text='统一接入主流模型供应商和图像、视频、任务类接口。'
          />
          <Feature
            icon={<WalletCards />}
            title='运营可视化'
            text='保留旧控制台现有发票、收益看板、用量分析等本地定制能力。'
          />
        </section>
      </main>

      {noticeOpen && (
        <div className='notice-backdrop' role='presentation'>
          <div
            className='notice-dialog'
            role='dialog'
            aria-modal='true'
            aria-label='系统公告'
          >
            <h2>系统公告</h2>
            <div>{notice}</div>
            <button
              type='button'
              onClick={() => {
                localStorage.setItem(
                  'notice_close_date',
                  new Date().toDateString(),
                );
                setNoticeOpen(false);
              }}
            >
              我知道了
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function Stat(props) {
  return (
    <article className='stat'>
      <div className='stat-icon'>{props.icon}</div>
      <strong>{props.value}</strong>
      <span>{props.label}</span>
    </article>
  );
}

function Feature(props) {
  return (
    <article className='feature'>
      <div className='feature-icon'>{props.icon}</div>
      <h2>{props.title}</h2>
      <p>{props.text}</p>
    </article>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
