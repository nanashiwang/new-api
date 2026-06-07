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
};

export const normalizePublicLanguage = (language) => {
  if (!language) return 'zh-CN';
  if (language.startsWith('zh-TW') || language.startsWith('zh-HK')) {
    return 'zh-TW';
  }
  if (language.startsWith('zh')) return 'zh-CN';
  return 'en';
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
