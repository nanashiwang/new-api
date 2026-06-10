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

import { useEffect, useState } from 'react';
import { normalizeLocale } from './localeNormalize';

const LANGUAGE_STORAGE_KEY = 'i18nextLng';
const LANGUAGE_CHANGE_EVENT = 'public-language-change';

const en = {
  首页: 'Home',
  控制台: 'Console',
  模型广场: 'Model Marketplace',
  文档: 'Docs',
  关于: 'About',
  登录: 'Sign in',
  注册: 'Sign up',
  切换主题: 'Switch Theme',
  'common.changeLanguage': 'Change Language',
  版权所有: 'All rights reserved',
  设计与开发由: 'Designed & Developed by',
  统一的: 'The Unified',
  大模型接口网关: 'LLM API Gateway',
  '更好的价格，更好的稳定性，只需要将模型基址替换为：':
    'Better prices and better stability. Just replace the model base URL with:',
  '一衡调山海，万模皆可达': 'Balance the vast — every model within reach.',
  '聚合万模，平衡调度，一口稳定接入，只需将模型基址替换为：':
    'Aggregate every model, balance every request, one stable integration. Just replace the model base URL with:',
  已复制到剪切板: 'Copied',
  获取密钥: 'Get Key',
  支持众多的大模型供应商: 'Supporting various LLM providers',
  系统公告: 'System Notice',
  今日关闭: 'Close Today',
  关闭公告: 'Close Notice',
  // 首页内容板块
  '为什么选择山海衡 AI': 'Why Choose Us',
  '聚合 40+ 大模型': 'Aggregate 40+ Models',
  'OpenAI、Claude、Gemini、DeepSeek、Qwen… 一个统一 API 全部调用，无需多处对接。':
    'OpenAI, Claude, Gemini, DeepSeek, Qwen… call them all through one unified API, no multiple integrations needed.',
  更优的价格: 'Better Pricing',
  '批量议价、按量计费、缓存命中优惠，成本更低、账单更透明。':
    'Bulk pricing, pay-as-you-go, cache-hit discounts — lower costs and clearer bills.',
  稳定高可用: 'Highly Available',
  '多节点容灾、智能路由、自动重试，保障调用稳定不中断。':
    'Multi-node failover, smart routing and auto-retry keep your calls stable and uninterrupted.',
  安全可控: 'Secure & Controllable',
  '密钥隔离、用量管控、调用日志与账单全程可查，合规放心。':
    'Key isolation, usage control, and fully auditable logs and bills — compliant and worry-free.',
  三步快速接入: 'Get Started in Three Steps',
  注册并获取密钥: 'Register & Get a Key',
  '登录控制台，创建专属 API Key。':
    'Sign in to the console and create your own API key.',
  替换模型基址: 'Replace the Base URL',
  '把原有 Base URL 换成本平台地址。':
    'Swap your existing Base URL for this platform’s address.',
  立即开始调用: 'Start Calling',
  '沿用原有 SDK 与参数，直接调用 40+ 模型。':
    'Keep your existing SDK and parameters, and call 40+ models right away.',
  '现在就接入，开启统一的大模型调用体验':
    'Integrate now and unlock a unified LLM experience.',
  立即开始: 'Get Started',
};

// 公开首页仅内置 zh / en 两套词条：先走与控制台一致的归一化（localeNormalize），
// 再把非中文语言收敛到 en，保证与 i18next 检测结果方向一致、跨页不跳变。
export const normalizePublicLanguage = (language) => {
  const normalized = normalizeLocale(language);
  return normalized.startsWith('zh') ? normalized : 'en';
};

export const getPublicLanguage = () => {
  if (typeof window === 'undefined') return 'zh-CN';
  const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
  return normalizePublicLanguage(stored || navigator.language || 'zh-CN');
};

export const setPublicLanguage = (language) => {
  const normalizedLanguage = normalizePublicLanguage(language);
  localStorage.setItem(LANGUAGE_STORAGE_KEY, normalizedLanguage);
  window.dispatchEvent(
    new CustomEvent(LANGUAGE_CHANGE_EVENT, { detail: normalizedLanguage }),
  );
  return normalizedLanguage;
};

export const usePublicTranslation = () => {
  const [language, setLanguage] = useState(getPublicLanguage);

  useEffect(() => {
    const handleLanguageChange = (event) => {
      setLanguage(normalizePublicLanguage(event.detail || getPublicLanguage()));
    };

    window.addEventListener(LANGUAGE_CHANGE_EVENT, handleLanguageChange);
    window.addEventListener('storage', handleLanguageChange);
    return () => {
      window.removeEventListener(LANGUAGE_CHANGE_EVENT, handleLanguageChange);
      window.removeEventListener('storage', handleLanguageChange);
    };
  }, []);

  const isChinese = language.startsWith('zh');
  const t = (key) => (isChinese ? key : en[key] || key);

  return { t, language, isChinese, setLanguage: setPublicLanguage };
};
