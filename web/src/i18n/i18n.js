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

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

const localeLoaders = {
  'zh-CN': () => import('./locales/zh-CN.json'),
  en: () => import('./locales/en.json'),
  fr: () => import('./locales/fr.json'),
  'zh-TW': () => import('./locales/zh-TW.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
};

const normalizeLanguage = (lng) => {
  if (!lng) return 'zh-CN';
  if (lng.startsWith('zh-TW') || lng.startsWith('zh-HK')) return 'zh-TW';
  if (lng.startsWith('zh')) return 'zh-CN';
  const shortLng = lng.split('-')[0];
  return localeLoaders[lng] ? lng : localeLoaders[shortLng] ? shortLng : 'zh-CN';
};

const loadLocale = async (lng) => {
  const normalizedLng = normalizeLanguage(lng);
  if (i18n.hasResourceBundle(normalizedLng, 'translation')) {
    return normalizedLng;
  }
  const loader = localeLoaders[normalizedLng];
  if (!loader) {
    return 'zh-CN';
  }
  const module = await loader();
  const resources = module.default || module;
  i18n.addResourceBundle(
    normalizedLng,
    'translation',
    resources.translation || resources,
    true,
    true,
  );
  return normalizedLng;
};

const loadLocaleWhenIdle = (lng) => {
  const load = () => {
    i18n.changeLanguage(lng).catch((error) => {
      console.error('Failed to load locale:', error);
    });
  };

  if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
    window.requestIdleCallback(load, { timeout: 4000 });
  } else {
    setTimeout(load, 2000);
  }
};

const initPromise = i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    load: 'currentOnly',
    resources: {},
    fallbackLng: 'zh-CN',
    nsSeparator: false,
    interpolation: {
      escapeValue: false,
    },
  });

const changeLanguage = i18n.changeLanguage.bind(i18n);
i18n.changeLanguage = async (lng, callback) => {
  const normalizedLng = await loadLocale(lng);
  return changeLanguage(normalizedLng, callback);
};

initPromise.then(() => {
  const normalizedLng = normalizeLanguage(i18n.language);
  if (normalizedLng === 'zh-CN') {
    loadLocaleWhenIdle(normalizedLng);
  } else {
    i18n.changeLanguage(normalizedLng);
  }
});

export default i18n;
