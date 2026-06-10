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

// 公开首页停留期间预取控制台壳层（AppShell 连带 semi-ui JS/CSS、i18n 及语言包）
// 与登录/注册表单 chunk，使「首页 → 登录/注册/控制台」的路由切换无需再等网络。
// 动态 import 与 App.jsx 中 lazy() 的模块标识一致，Vite 会复用同一 chunk，
// 预取完成后 Suspense 直接命中模块缓存。

let prefetchPromise = null;

export const prefetchAppShell = () => {
  if (prefetchPromise) return prefetchPromise;
  prefetchPromise = Promise.allSettled([
    import('../components/layout/AppShell'),
    import('../components/auth/LoginForm'),
    import('../components/auth/RegisterForm'),
  ]);
  return prefetchPromise;
};

// 首页关键资源就绪后，利用浏览器空闲时间触发预取；
// 不支持 requestIdleCallback 的环境退化为短延时。
export const schedulePrefetchAppShell = () => {
  if (typeof window === 'undefined') return;
  if ('requestIdleCallback' in window) {
    window.requestIdleCallback(() => prefetchAppShell(), { timeout: 3000 });
  } else {
    window.setTimeout(prefetchAppShell, 1500);
  }
};
