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

import React, { lazy, Suspense, useContext, useEffect, useState } from 'react';
import { API } from '../../helpers/apiCore';
import { copy } from '../../helpers/clipboard';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { usePublicTranslation } from '../../helpers/publicLocale';
import { Github, Play, FileText, Copy } from 'lucide-react';
import { Link, useNavigate } from 'react-router-dom';

const PublicNoticeModal = lazy(
  () => import('../../components/layout/PublicNoticeModal'),
);

const isRemoteHomePage = (content) => content.startsWith('https://');
// 内容以 HTML 标签 / 注释 / DOCTYPE 起头时视为原生 HTML，跳过 markdown 解析；
// 否则按 markdown 处理。理由：marked v4 会把缩进 4+ 空格的 HTML 段误识别为 indented
// code block 输出 <pre><code>，让用户的格式化 HTML 渲染成代码。识别为 HTML 的内容
// 直接透传更稳，避免误处理；纯 markdown 文案（无标签）依然走 marked。
const isRawHtmlContent = (content) => {
  const trimmed = content.trimStart();
  return (
    trimmed.startsWith('<!') ||
    trimmed.startsWith('<style') ||
    trimmed.startsWith('<html') ||
    trimmed.startsWith('<div') ||
    trimmed.startsWith('<section') ||
    trimmed.startsWith('<main') ||
    trimmed.startsWith('<nav') ||
    trimmed.startsWith('<header')
  );
};
const getCachedHomePageContent = () => {
  try {
    return localStorage.getItem('home_page_content') || '';
  } catch (error) {
    return '';
  }
};
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

const Home = ({ onLandingChange } = {}) => {
  const { t, language, isChinese } = usePublicTranslation();
  const navigate = useNavigate();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContent, setHomePageContent] = useState(
    getCachedHomePageContent,
  );
  const [homePageResolved, setHomePageResolved] = useState(
    () => getCachedHomePageContent() !== '',
  );
  const [noticeContent, setNoticeContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const useDefaultHome = homePageContent === '';
  const currentEndpoint = endpointItems[endpointIndex]?.value || '';

  const displayHomePageContent = async () => {
    const cachedContent = getCachedHomePageContent();
    if (cachedContent) {
      setHomePageContent(cachedContent);
    }

    try {
      const res = await API.get('/api/home_page_content', {
        skipGlobalLoading: true,
        skipErrorHandler: true,
      });
      const { success, message, data } = res.data;
      if (success) {
        let content = data || '';
        if (content && !isRemoteHomePage(content) && !isRawHtmlContent(content)) {
          const { marked } = await import('marked');
          content = marked.parse(content);
        }
        setHomePageContent(content);
        if (content) {
          localStorage.setItem('home_page_content', content);
        } else {
          localStorage.removeItem('home_page_content');
        }

        if (isRemoteHomePage(data || '')) {
          const iframe = document.querySelector('iframe');
          if (iframe) {
            iframe.onload = () => {
              iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
              iframe.contentWindow.postMessage({ lang: language }, '*');
            };
          }
        }
      } else {
        console.error(message);
      }
    } catch (error) {
      if (cachedContent) return;
      console.error('加载首页内容失败:', error);
    } finally {
      setHomePageResolved(true);
    }
  };

  const handleCopyBaseURL = async () => {
    await copy(serverAddress);
  };

  const handleOpenDocs = () => {
    if (!docsLink) return;
    if (docsLink.startsWith('/')) {
      navigate(docsLink);
      return;
    }
    window.open(docsLink, '_blank', 'noopener,noreferrer');
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice', {
            skipGlobalLoading: true,
            skipErrorHandler: true,
          });
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            const { marked } = await import('marked');
            setNoticeContent(marked.parse(data.trim()));
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    const taskId = window.setTimeout(checkNoticeAndShow, 4500);
    return () => window.clearTimeout(taskId);
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  // 上报当前是否为自定义首页（自带顶部导航）：homePageContent 非空即为远程 iframe
  // 或自定义 HTML，二者都自带 nav。外层 PublicHomeShell 据此隐藏项目 public-header，
  // 避免与自定义 nav 叠成双顶栏。onLandingChange 可选，App.jsx 内联使用 Home 时不传。
  useEffect(() => {
    if (typeof onLandingChange === 'function') {
      onLandingChange(homePageContent !== '');
    }
  }, [homePageContent, onLandingChange]);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  if (!homePageResolved) {
    return (
      <div
        className='w-full min-h-[calc(100vh-64px)] bg-[var(--semi-color-bg-0,#fff)]'
        aria-busy='true'
      />
    );
  }

  return (
    <div className='w-full overflow-x-hidden'>
      {noticeVisible && (
        <Suspense fallback={null}>
          <PublicNoticeModal
            visible={noticeVisible}
            content={noticeContent}
            onClose={() => setNoticeVisible(false)}
          />
        </Suspense>
      )}
      {useDefaultHome ? (
        <div className='home-shell w-full overflow-x-hidden'>
          <div className='home-hero w-full min-h-[500px] md:min-h-[600px] lg:min-h-[700px] relative overflow-x-hidden'>
            <div className='blur-ball blur-ball-indigo' />
            <div className='blur-ball blur-ball-teal' />
            <div className='flex items-center justify-center h-full px-4 py-20 md:py-24 lg:py-32 mt-10'>
              <div className='flex flex-col items-center justify-center text-center max-w-4xl mx-auto'>
                <div className='flex flex-col items-center justify-center mb-6 md:mb-8'>
                  <div className='home-hero-brand'>
                    <img src='/logo.svg' alt='logo' />
                    <span>
                      {statusState?.status?.system_name || '山海衡 AI'}
                    </span>
                  </div>
                  <h1
                    className={`text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-bold home-text-primary leading-tight ${isChinese ? 'tracking-wide md:tracking-wider' : ''}`}
                  >
                    <>
                      {t('统一的')}
                      <br />
                      <span className='shine-text'>{t('大模型接口网关')}</span>
                    </>
                  </h1>
                  <p className='text-xl md:text-2xl lg:text-3xl font-semibold home-text-primary mt-4 md:mt-6'>
                    {t('一衡调山海，万模皆可达')}
                  </p>
                  <p className='text-base md:text-lg lg:text-xl home-text-secondary mt-3 md:mt-4 max-w-xl'>
                    {t('聚合万模，平衡调度，一口稳定接入，只需将模型基址替换为：')}
                  </p>
                  <div className='flex flex-col md:flex-row items-center justify-center gap-4 w-full mt-4 md:mt-6 max-w-md'>
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
                          const nextIndex = endpointItems.findIndex(
                            (item) => item.value === event.target.value,
                          );
                          if (nextIndex >= 0) setEndpointIndex(nextIndex);
                        }}
                        aria-label='API endpoint'
                      >
                        {endpointItems.map((item) => (
                          <option key={item.value} value={item.value}>
                            {item.value}
                          </option>
                        ))}
                      </select>
                      <button
                        type='button'
                        onClick={handleCopyBaseURL}
                        className='home-copy-button'
                        aria-label={t('已复制到剪切板')}
                      >
                        <Copy size={18} />
                      </button>
                    </div>
                  </div>
                </div>

                <div className='flex flex-row gap-4 justify-center items-center'>
                  <Link to='/console'>
                    <span className='home-action-button home-action-primary'>
                      <Play size={isMobile ? 16 : 18} />
                      {t('获取密钥')}
                    </span>
                  </Link>
                  {isDemoSiteMode && statusState?.status?.version ? (
                    <button
                      type='button'
                      className='home-action-button home-action-secondary'
                      onClick={() =>
                        window.open(
                          'https://github.com/QuantumNous/new-api',
                          '_blank',
                        )
                      }
                    >
                      <Github size={isMobile ? 16 : 18} />
                      {statusState.status.version}
                    </button>
                  ) : (
                    docsLink && (
                      <button
                        type='button'
                        className='home-action-button home-action-secondary'
                        onClick={handleOpenDocs}
                      >
                        <FileText size={isMobile ? 16 : 18} />
                        {t('文档')}
                      </button>
                    )
                  )}
                </div>

                <div className='mt-12 md:mt-16 lg:mt-20 w-full'>
                  <div className='flex items-center mb-6 md:mb-8 justify-center'>
                    <span className='home-text-tertiary text-lg md:text-xl lg:text-2xl font-light'>
                      {t('支持众多的大模型供应商')}
                    </span>
                  </div>
                  <div className='home-provider-badges'>
                    {providerBadges.map((provider) => (
                      <span className='home-provider-badge' key={provider}>
                        {provider}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <section className='home-section'>
            <div className='home-section-inner'>
              <h2 className='home-section-title'>
                {t('为什么选择山海衡 AI')}
              </h2>
              <div className='home-feature-grid'>
                {[
                  [
                    '聚合 40+ 大模型',
                    'OpenAI、Claude、Gemini、DeepSeek、Qwen… 一个统一 API 全部调用，无需多处对接。',
                    <Layers size={24} />,
                  ],
                  [
                    '更优的价格',
                    '批量议价、按量计费、缓存命中优惠，成本更低、账单更透明。',
                    <Coins size={24} />,
                  ],
                  [
                    '稳定高可用',
                    '多节点容灾、智能路由、自动重试，保障调用稳定不中断。',
                    <Zap size={24} />,
                  ],
                  [
                    '安全可控',
                    '密钥隔离、用量管控、调用日志与账单全程可查，合规放心。',
                    <ShieldCheck size={24} />,
                  ],
                ].map(([title, desc, icon]) => (
                  <div className='home-feature-card' key={title}>
                    <div className='home-feature-icon'>{icon}</div>
                    <h3>{t(title)}</h3>
                    <p>{t(desc)}</p>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <section className='home-section home-section--alt'>
            <div className='home-section-inner'>
              <h2 className='home-section-title'>{t('三步快速接入')}</h2>
              <div className='home-steps-grid'>
                {[
                  ['1', '注册并获取密钥', '登录控制台，创建专属 API Key。'],
                  ['2', '替换模型基址', '把原有 Base URL 换成本平台地址。'],
                  ['3', '立即开始调用', '沿用原有 SDK 与参数，直接调用 40+ 模型。'],
                ].map(([n, title, desc]) => (
                  <div className='home-step' key={n}>
                    <div className='home-step-no'>{n}</div>
                    <h3>{t(title)}</h3>
                    <p>{t(desc)}</p>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <section className='home-cta'>
            <h2>{t('一衡调山海，万模皆可达')}</h2>
            <p>{t('现在就接入，开启统一的大模型调用体验')}</p>
            <Link to='/console' className='home-cta-btn'>
              {t('立即开始')}
              <ArrowRight size={18} />
            </Link>
          </section>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {isRemoteHomePage(homePageContent) ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            // 自定义首页 HTML 自带顶部 nav 时，PageLayout 已经隐藏了全局 Header，
            // 这里也不再加 mt-[60px] 顶部留白，避免与自定义 nav 之间出现空白带。
            <div dangerouslySetInnerHTML={{ __html: homePageContent }} />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
