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

const REDIRECT_BASE_ORIGIN = 'https://new-api.invalid';
const OAUTH_REDIRECT_STORAGE_PREFIX = 'new-api:oauth-login-redirect:';
const OAUTH_REDIRECT_MAX_AGE_MS = 15 * 60 * 1000;

function getBrowserStorage() {
  return typeof window !== 'undefined' ? window.localStorage : null;
}

export function getSafeInternalRedirectPath(path, fallback = '') {
  if (
    typeof path !== 'string' ||
    path === '' ||
    !path.startsWith('/') ||
    path.startsWith('//') ||
    /[\\\u0000-\u001f\u007f]/.test(path)
  ) {
    return fallback;
  }

  try {
    let decoded = path;
    for (let index = 0; index < 2; index += 1) {
      const nextDecoded = decodeURIComponent(decoded);
      if (nextDecoded === decoded) break;
      decoded = nextDecoded;
    }
    if (decoded.startsWith('//') || /[\\\u0000-\u001f\u007f]/.test(decoded)) {
      return fallback;
    }

    const parsed = new URL(path, REDIRECT_BASE_ORIGIN);
    if (parsed.origin !== REDIRECT_BASE_ORIGIN) return fallback;
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch (error) {
    return fallback;
  }
}

export function getSafeLoginRedirectPath(search, fallback = '/console') {
  let rawSearch = search;
  if (typeof rawSearch !== 'string') {
    rawSearch = typeof window !== 'undefined' ? window.location.search : '';
  }

  const values = new URLSearchParams(rawSearch).getAll('next');
  if (values.length !== 1) return fallback;
  return getSafeInternalRedirectPath(values[0], fallback);
}

export function buildAuthPathWithNext(authPath, next) {
  const safeNext = getSafeInternalRedirectPath(next);
  if (!safeNext) return authPath;
  const params = new URLSearchParams({ next: safeNext });
  return `${authPath}?${params.toString()}`;
}

export function isBackendLoginRedirectPath(path) {
  return path === '/api/forum/sso/start';
}

export function navigateToLoginRedirect(path, navigate, locationObject) {
  const safePath = getSafeInternalRedirectPath(path, '/console');
  if (isBackendLoginRedirectPath(safePath)) {
    const targetLocation =
      locationObject ||
      (typeof window !== 'undefined' ? window.location : null);
    if (!targetLocation || typeof targetLocation.assign !== 'function') {
      throw new Error('browser location is required for backend redirect');
    }
    targetLocation.assign(safePath);
    return 'document';
  }
  navigate(safePath);
  return 'client';
}

function oauthRedirectStorageKey(state) {
  if (typeof state !== 'string' || state === '' || state.length > 256)
    return '';
  return `${OAUTH_REDIRECT_STORAGE_PREFIX}${state}`;
}

export function rememberOAuthLoginRedirect(
  state,
  path,
  storage = getBrowserStorage(),
  now = Date.now(),
) {
  const key = oauthRedirectStorageKey(state);
  const safePath = getSafeInternalRedirectPath(path);
  if (!key || !safePath || !storage) return false;

  try {
    storage.setItem(key, JSON.stringify({ path: safePath, created_at: now }));
    return true;
  } catch (error) {
    return false;
  }
}

export function consumeOAuthLoginRedirect(
  state,
  fallback = '/console/token',
  storage = getBrowserStorage(),
  now = Date.now(),
) {
  const key = oauthRedirectStorageKey(state);
  if (!key || !storage) return fallback;

  let raw = null;
  try {
    raw = storage.getItem(key);
    storage.removeItem(key);
  } catch (error) {
    return fallback;
  }
  if (!raw) return fallback;

  try {
    const saved = JSON.parse(raw);
    if (
      !Number.isFinite(saved?.created_at) ||
      saved.created_at > now ||
      now - saved.created_at > OAUTH_REDIRECT_MAX_AGE_MS
    ) {
      return fallback;
    }
    return getSafeInternalRedirectPath(saved.path, fallback);
  } catch (error) {
    return fallback;
  }
}
