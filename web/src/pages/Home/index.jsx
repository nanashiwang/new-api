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

import React, {
  lazy,
  Suspense,
  useContext,
  useEffect,
  useState,
} from 'react';
import { API } from '../../helpers/apiCore';
import { copy } from '../../helpers/clipboard';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { usePublicTranslation } from '../../helpers/publicLocale';
import { Github, Play, FileText, Copy } from 'lucide-react';
import { Link } from 'react-router-dom';

const PublicNoticeModal = lazy(
  () => import('../../components/layout/PublicNoticeModal'),
);

const isRemoteHomePage = (content) => content.startsWith('https://');
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

const Home = () => {
  const { t, language, isChinese } = usePublicTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContent, setHomePageContent] = useState('');
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
    const cachedContent = localStorage.getItem('home_page_content') || '';
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
        if (content && !isRemoteHomePage(content)) {
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
    }
  };

  const handleCopyBaseURL = async () => {
    await copy(serverAddress);
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

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

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
                  <h1
                    className={`text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-bold home-text-primary leading-tight ${isChinese ? 'tracking-wide md:tracking-wider' : ''}`}
                  >
                    <>
                      {t('统一的')}
                      <br />
                      <span className='shine-text'>{t('大模型接口网关')}</span>
                    </>
                  </h1>
                  <p className='text-base md:text-lg lg:text-xl home-text-secondary mt-4 md:mt-6 max-w-xl'>
                    {t('更好的价格，更好的稳定性，只需要将模型基址替换为：')}
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
                        onClick={() => window.open(docsLink, '_blank')}
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
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {isRemoteHomePage(homePageContent) ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
