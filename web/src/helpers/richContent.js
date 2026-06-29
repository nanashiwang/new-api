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

import DOMPurify from 'dompurify';
import { marked } from 'marked';

export const isHttpUrl = (value) => {
  try {
    const url = new URL((value || '').trim());
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
};

export const isHttpsUrl = (value) => {
  try {
    const url = new URL((value || '').trim());
    return url.protocol === 'https:';
  } catch {
    return false;
  }
};

export const isLikelyHtml = (value) => {
  return /<!doctype html|<html[\s>]|<head[\s>]|<body[\s>]|<style[\s>]|<script[\s>]|<\/?[a-z][\s\S]*>/i.test(
    value || '',
  );
};

export const isLikelyStandaloneHtml = (value) => {
  const trimmed = (value || '').trimStart();
  return /^<!doctype html|^<!--|^<html[\s>]|^<head[\s>]|^<body[\s>]|^<style[\s>]|^<script[\s>]|^<\/?[a-z][\s\S]*>/i.test(
    trimmed,
  );
};

const addExternalLinkAttributes = (html) => {
  if (typeof document === 'undefined') {
    return html;
  }

  const template = document.createElement('template');
  template.innerHTML = html;

  template.content.querySelectorAll('a[href]').forEach((link) => {
    const href = link.getAttribute('href') || '';
    const isExternal = /^https?:\/\//i.test(href) || href.startsWith('//');
    if (isExternal) {
      link.setAttribute('target', '_blank');
    }
    if (link.getAttribute('target') === '_blank') {
      link.setAttribute('rel', 'noopener noreferrer');
    }
  });

  return template.innerHTML;
};

export const sanitizeHtmlContent = (html) => {
  const sanitized = DOMPurify.sanitize(html || '');
  return addExternalLinkAttributes(sanitized);
};

export const renderMarkdownContent = (content, options = {}) => {
  const parsed = marked.parse(content || '', {
    breaks: Boolean(options.breaks),
    gfm: true,
  });
  return sanitizeHtmlContent(parsed);
};

export const renderRichContent = (content, options = {}) => {
  if (
    options.mode === 'html' ||
    (!options.mode && isLikelyStandaloneHtml(content))
  ) {
    return sanitizeHtmlContent(content);
  }
  return renderMarkdownContent(content, options);
};
