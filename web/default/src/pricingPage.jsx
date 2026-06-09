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
import axios from 'axios';
import {
  Activity,
  AlertCircle,
  ArrowRight,
  Brain,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Code2,
  Copy,
  FileText,
  Gauge,
  Grid2X2,
  Image as ImageIcon,
  Info,
  Layers,
  Loader2,
  Music2,
  RotateCcw,
  Search,
  SlidersHorizontal,
  Table2,
  Tags,
  Video,
  X,
  Zap,
} from 'lucide-react';

const FILTER_ALL = 'all';
const PAGE_SIZE = 20;
const DEFAULT_ENDPOINT_MAP = {
  openai: { path: '/v1/chat/completions', method: 'POST', label: 'Chat' },
  'openai-response': { path: '/v1/responses', method: 'POST', label: 'Response' },
  anthropic: { path: '/v1/messages', method: 'POST', label: 'Anthropic' },
  gemini: { path: '/v1beta/models/{model}:generateContent', method: 'POST', label: 'Gemini' },
  embeddings: { path: '/v1/embeddings', method: 'POST', label: 'Embeddings' },
  'jina-rerank': { path: '/v1/rerank', method: 'POST', label: 'Rerank' },
  'image-generation': { path: '/v1/images/generations', method: 'POST', label: 'Image' },
  'openai-video': { path: '/v1/videos', method: 'POST', label: 'Video' },
};

const QUOTA_TYPES = [
  { value: FILTER_ALL, label: '全部模型' },
  { value: 'token', label: 'Token 计费' },
  { value: 'request', label: '按次计费' },
];

const SORT_OPTIONS = [
  { value: 'name', label: '按名称' },
  { value: 'price-low', label: '价格从低到高' },
  { value: 'price-high', label: '价格从高到低' },
];

const MODALITY_ORDER = ['text', 'image', 'audio', 'video', 'file'];
const CAPABILITY_LABELS = {
  streaming: '流式输出',
  function_calling: '函数调用',
  tools: '工具调用',
  json_mode: 'JSON 模式',
  structured_output: '结构化输出',
  reasoning: '推理',
  vision: '视觉',
  caching: '缓存',
  embeddings: '向量',
  web_search: '联网搜索',
  code_interpreter: '代码',
};

const BILLING_PRICING_VARS = [
  { key: 'p', field: 'inputPrice', label: '输入', shortLabel: '输入' },
  { key: 'c', field: 'outputPrice', label: '输出', shortLabel: '输出' },
  { key: 'cr', field: 'cacheReadPrice', label: '缓存读', shortLabel: '缓存读' },
  { key: 'cc', field: 'cacheCreatePrice', label: '缓存写', shortLabel: '缓存写' },
  {
    key: 'cc1h',
    field: 'cacheCreate1hPrice',
    label: '缓存写 1h',
    shortLabel: '缓存写 1h',
  },
  { key: 'img', field: 'imagePrice', label: '图像输入', shortLabel: '图像' },
  {
    key: 'img_o',
    field: 'imageOutputPrice',
    label: '图像输出',
    shortLabel: '图像输出',
  },
  {
    key: 'ai',
    field: 'audioInputPrice',
    label: '音频输入',
    shortLabel: '音频输入',
  },
  {
    key: 'ao',
    field: 'audioOutputPrice',
    label: '音频输出',
    shortLabel: '音频输出',
  },
];

const BILLING_VAR_KEY_TO_FIELD = Object.fromEntries(
  BILLING_PRICING_VARS.map((item) => [item.key, item.field]),
);

const BILLING_VAR_REGEX = new RegExp(
  `\\b(${BILLING_PRICING_VARS.map((item) => item.key).join('|')})\\s*\\*\\s*([\\d.eE+-]+)`,
  'g',
);

function PublicFrame({ status, user, children }) {
  const modules = parseHeaderModules(status.HeaderNavModules);
  const links = [
    isEnabled(modules.home) && { label: '首页', href: '/' },
    isEnabled(modules.console) && { label: '控制台', href: user ? '/console' : '/login' },
    isEnabled(modules.pricing) && {
      label: '模型广场',
      href: requiresAuth(modules.pricing) && !user ? '/login' : '/pricing',
    },
    isEnabled(modules.docs) && { label: '文档', href: status.docs_link || '/docs' },
    isEnabled(modules.about) && { label: '关于', href: '/about' },
  ].filter(Boolean);
  return (
    <div className='default-home public-page pricing-v2-page'>
      <header className='topbar public-topbar'>
        <a className='brand' href='/'>
          <img src={status.logo || '/logo.svg'} alt='' />
          <span>{status.system_name || 'New API'}</span>
        </a>
        <nav>
          {links.map((link) => (
            <a key={link.href} href={link.href}>
              {link.label}
            </a>
          ))}
        </nav>
        <div className='actions'>
          {user ? (
            <a className='user-chip public-user-chip' href='/console/personal'>
              {(user.username || 'U').slice(0, 1).toUpperCase()}
              <span>{user.username || '用户'}</span>
            </a>
          ) : (
            <>
              <a className='login public-login-link' href='/login'>
                登录
              </a>
              {!status.self_use_mode_enabled && (
                <a className='primary-small' href='/register'>
                  注册
                </a>
              )}
            </>
          )}
        </div>
      </header>
      {children}
    </div>
  );
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
    return { ...fallback, ...JSON.parse(raw) };
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

function formatCompactNumber(value, digits = 6) {
  const number = Number(value);
  if (!Number.isFinite(number)) return '-';
  return number
    .toFixed(Math.abs(number) >= 1 ? Math.min(digits, 4) : digits)
    .replace(/\.?0+$/, '') || '0';
}

function displayMoney(value, showRechargePrice, priceRate, digits = 6) {
  const number = Number(value);
  if (!Number.isFinite(number)) return showRechargePrice ? '¥-' : '$-';
  if (!showRechargePrice) return `$${formatCompactNumber(number, digits)}`;
  return `¥${formatCompactNumber(number * Number(priceRate || 1), digits)}`;
}

function parseTags(value) {
  if (!value) return [];
  return String(value)
    .split(/[,;|\s]+/)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function countBy(models, predicate) {
  return models.reduce((count, model) => count + (predicate(model) ? 1 : 0), 0);
}

function formatRatio(ratio) {
  const number = Number(ratio);
  if (!Number.isFinite(number)) return '';
  return `x${formatCompactNumber(number, 3)}`;
}

function getEndpointInfo(endpoint, endpointMap = {}) {
  const raw = endpointMap?.[endpoint];
  if (raw && typeof raw === 'object') {
    return {
      path: raw.path || DEFAULT_ENDPOINT_MAP[endpoint]?.path || endpoint,
      method: raw.method || DEFAULT_ENDPOINT_MAP[endpoint]?.method || 'POST',
      label: raw.label || DEFAULT_ENDPOINT_MAP[endpoint]?.label || endpoint,
    };
  }
  if (typeof raw === 'string') {
    return {
      path: raw,
      method: DEFAULT_ENDPOINT_MAP[endpoint]?.method || 'POST',
      label: DEFAULT_ENDPOINT_MAP[endpoint]?.label || endpoint,
    };
  }
  return DEFAULT_ENDPOINT_MAP[endpoint] || { path: endpoint, method: 'POST', label: endpoint };
}

function splitTopLevelMultiply(expr) {
  const parts = [];
  let start = 0;
  let depth = 0;
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index];
    if (char === '(') depth += 1;
    if (char === ')') depth -= 1;
    if (depth === 0 && expr.slice(index, index + 3) === ' * ') {
      parts.push(expr.slice(start, index).trim());
      start = index + 3;
      index += 2;
    }
  }
  parts.push(expr.slice(start).trim());
  return parts.filter(Boolean);
}

function hasFullOuterParens(expr) {
  if (!expr.startsWith('(') || !expr.endsWith(')')) return false;
  let depth = 0;
  for (let index = 0; index < expr.length; index += 1) {
    if (expr[index] === '(') depth += 1;
    if (expr[index] === ')') depth -= 1;
    if (depth === 0 && index < expr.length - 1) return false;
  }
  return depth === 0;
}

function unwrapOuterParens(expr) {
  let current = expr.trim();
  while (hasFullOuterParens(current)) {
    current = current.slice(1, -1).trim();
  }
  return current;
}

function splitBillingExprAndRequestRules(expr) {
  const trimmed = String(expr || '').trim();
  if (!trimmed) return { billingExpr: '', requestRuleExpr: '' };
  const parts = splitTopLevelMultiply(trimmed);
  if (parts.length <= 1) return { billingExpr: trimmed, requestRuleExpr: '' };

  const ruleParts = [];
  const baseParts = [];
  parts.forEach((part) => {
    if (/^\(.+ \? [\d.eE+-]+ : 1\)$/s.test(part)) {
      ruleParts.push(part);
    } else {
      baseParts.push(part);
    }
  });

  if (ruleParts.length === 0 || baseParts.length !== 1) {
    return { billingExpr: trimmed, requestRuleExpr: '' };
  }
  return {
    billingExpr: unwrapOuterParens(baseParts[0]),
    requestRuleExpr: ruleParts.join(' * '),
  };
}

function stripExprVersion(expr) {
  const match = String(expr || '').match(/^v\d+:([\s\S]*)$/);
  return match ? match[1] : String(expr || '');
}

function parseTierBody(body) {
  const coeffs = {};
  const regex = new RegExp(BILLING_VAR_REGEX.source, 'g');
  let match;
  while ((match = regex.exec(body)) !== null) {
    if (!(match[1] in coeffs)) coeffs[match[1]] = Number(match[2]);
  }
  return Object.fromEntries(
    Object.entries(BILLING_VAR_KEY_TO_FIELD).map(([key, field]) => [
      field,
      coeffs[key] || 0,
    ]),
  );
}

function parseTiersFromExpr(expr) {
  const body = stripExprVersion(expr);
  if (!body) return [];
  try {
    const condGroup =
      '((?:(?:p|c|len)\\s*(?:<=|>=|<|>)\\s*[\\d.eE+]+)' +
      '(?:\\s*&&\\s*(?:p|c|len)\\s*(?:<=|>=|<|>)\\s*[\\d.eE+]+)*)';
    const tierRegex = new RegExp(
      `(?:${condGroup}\\s*\\?\\s*)?tier\\("([^"]*)",\\s*([^)]+)\\)`,
      'g',
    );
    const tiers = [];
    let match;
    while ((match = tierRegex.exec(body)) !== null) {
      tiers.push({
        label: match[2],
        conditions: match[1] || '',
        ...parseTierBody(match[3]),
      });
    }
    return tiers;
  } catch {
    return [];
  }
}

function formatDynamicUnitPrice(valuePerMillionTokens, ratio, tokenUnit, showRechargePrice, priceRate) {
  const divisor = tokenUnit === 'K' ? 1000 : 1;
  const price = (Number(valuePerMillionTokens) * ratio) / divisor;
  return displayMoney(price, showRechargePrice, priceRate);
}

function getDynamicPriceRows(model, groupRatio, selectedGroup, tokenUnit, showRechargePrice, priceRate) {
  if (model.billing_mode !== 'tiered_expr' || !model.billing_expr) return null;
  const { billingExpr, requestRuleExpr } = splitBillingExprAndRequestRules(model.billing_expr);
  const tiers = parseTiersFromExpr(billingExpr);
  if (!tiers.length) {
    return {
      mode: '动态',
      comparable: Number.POSITIVE_INFINITY,
      primary: '特殊计费表达式',
      secondary: requestRuleExpr ? '含请求倍率规则' : '查看详情',
      rows: [],
      dynamic: { tiers: [], rawExpression: model.billing_expr, requestRuleExpr },
    };
  }
  const ratio = getBestGroupRatio(model, groupRatio, selectedGroup);
  const tier = tiers[0];
  const entries = BILLING_PRICING_VARS
    .map((variable) => ({
      key: variable.key,
      label: variable.shortLabel,
      rawValue: Number(tier[variable.field]),
      value: formatDynamicUnitPrice(tier[variable.field], ratio, tokenUnit, showRechargePrice, priceRate),
      suffix: `/ 1${tokenUnit}`,
    }))
    .filter((entry) => Number.isFinite(entry.rawValue) && entry.rawValue > 0);
  const input = entries.find((entry) => entry.key === 'p');
  const output = entries.find((entry) => entry.key === 'c');
  return {
    mode: '动态',
    comparable: Number(tier.inputPrice || 0) * ratio,
    primary: input ? `${input.value} / 1${tokenUnit} 输入` : '动态计费',
    secondary: output ? `${output.value} / 1${tokenUnit} 输出` : `${tiers.length} 个阶梯`,
    rows: entries,
    dynamic: { tiers, rawExpression: model.billing_expr, requestRuleExpr },
  };
}

function getBestGroupRatio(model, groupRatio, selectedGroup = FILTER_ALL) {
  if (selectedGroup !== FILTER_ALL) {
    const selectedRatio = Number(groupRatio?.[selectedGroup]);
    return Number.isFinite(selectedRatio) && selectedRatio > 0 ? selectedRatio : 1;
  }
  const groups = Array.isArray(model.enable_groups) ? model.enable_groups : [];
  const candidates = groups
    .map((group) => Number(groupRatio?.[group]))
    .filter((ratio) => Number.isFinite(ratio) && ratio > 0);
  if (candidates.length > 0) return Math.min(...candidates);
  return 1;
}

function getModelPriceRows(model, groupRatio, selectedGroup, tokenUnit, showRechargePrice, priceRate) {
  const dynamicPrice = getDynamicPriceRows(
    model,
    groupRatio,
    selectedGroup,
    tokenUnit,
    showRechargePrice,
    priceRate,
  );
  if (dynamicPrice) return dynamicPrice;

  const ratio = getBestGroupRatio(model, groupRatio, selectedGroup);
  if (model.quota_type === 1) {
    const price = Number(model.model_price || 0) * ratio;
    return {
      mode: '按次',
      comparable: price,
      primary: `${displayMoney(price, showRechargePrice, priceRate)} / 次`,
      secondary: `分组倍率 ${formatCompactNumber(ratio, 3)}`,
      rows: [{ key: 'request', label: '请求', value: displayMoney(price, showRechargePrice, priceRate), suffix: '/ 次' }],
    };
  }

  const divisor = tokenUnit === 'K' ? 1000 : 1;
  const input = (Number(model.model_ratio || 0) * 2 * ratio) / divisor;
  const rows = [
    { key: 'input', label: '输入', value: displayMoney(input, showRechargePrice, priceRate), suffix: `/ 1${tokenUnit}` },
    {
      key: 'output',
      label: '输出',
      value: displayMoney(input * Number(model.completion_ratio || 0), showRechargePrice, priceRate),
      suffix: `/ 1${tokenUnit}`,
    },
  ];
  const extras = [
    ['cache', '缓存读', model.cache_ratio],
    ['create_cache', '缓存写', model.create_cache_ratio],
    ['image', '图像', model.image_ratio],
    ['audio_input', '音频输入', model.audio_ratio],
    ['audio_output', '音频输出', model.audio_ratio && model.audio_completion_ratio],
  ];
  extras.forEach(([key, label, extraRatio]) => {
    const extra = Number(extraRatio);
    if (Number.isFinite(extra) && extra > 0) {
      rows.push({
        key,
        label,
        value: displayMoney(input * extra, showRechargePrice, priceRate),
        suffix: `/ 1${tokenUnit}`,
      });
    }
  });
  return {
    mode: model.billing_mode === 'tiered_expr' ? '动态' : 'Token',
    comparable: input,
    primary: `${rows[0].value} / 1${tokenUnit} 输入`,
    secondary: `${rows[1].value} / 1${tokenUnit} 输出`,
    rows,
  };
}

function inferMetadata(model) {
  const name = String(model.model_name || '').toLowerCase();
  const endpoints = model.supported_endpoint_types || [];
  const tags = parseTags(model.tags).map((item) => item.toLowerCase());
  const inputs = new Set(['text']);
  const outputs = new Set(['text']);
  const capabilities = new Set(['streaming']);

  if (model.image_ratio != null || /vision|vl|image|omni/.test(name) || endpoints.includes('image-generation')) {
    inputs.add('image');
    capabilities.add('vision');
  }
  if (model.audio_ratio != null || /audio|voice|tts|whisper/.test(name)) inputs.add('audio');
  if (/video|sora|veo|kling|pika/.test(name) || endpoints.includes('openai-video')) outputs.add('video');
  if (endpoints.includes('image-generation')) outputs.add('image');
  if (endpoints.includes('embeddings') || endpoints.includes('jina-rerank')) capabilities.add('embeddings');
  if (/reasoning|thinking|deepseek-r|qwq|^o[1-4]/.test(name)) capabilities.add('reasoning');
  if (/code|coder/.test(name)) capabilities.add('code_interpreter');
  if (/search|online|perplexity/.test(name)) capabilities.add('web_search');
  if (model.cache_ratio != null) capabilities.add('caching');
  if (!endpoints.includes('embeddings') && !endpoints.includes('image-generation')) {
    capabilities.add('function_calling');
    capabilities.add('tools');
    capabilities.add('json_mode');
    capabilities.add('structured_output');
  }
  tags.forEach((tag) => {
    if (tag.includes('vision')) capabilities.add('vision');
    if (tag.includes('reason')) capabilities.add('reasoning');
    if (tag.includes('tool')) capabilities.add('tools');
  });

  const context = model.context_length || (name.includes('gemini') ? 1000000 : name.includes('claude') ? 200000 : 128000);
  const maxOutput = model.max_output_tokens || (endpoints.includes('embeddings') ? 0 : 8192);
  return {
    context,
    maxOutput,
    inputModalities: MODALITY_ORDER.filter((item) => inputs.has(item)),
    outputModalities: MODALITY_ORDER.filter((item) => outputs.has(item)),
    capabilities: Array.from(capabilities),
  };
}

function formatThroughput(value) {
  const number = Number(value);
  if (!Number.isFinite(number) || number <= 0) return '-';
  return number >= 1000 ? `${(number / 1000).toFixed(1)}K t/s` : `${number.toFixed(number < 10 ? 2 : 1)} t/s`;
}

function formatLatency(value) {
  const number = Number(value);
  if (!Number.isFinite(number) || number <= 0) return '-';
  return number >= 1000 ? `${(number / 1000).toFixed(2)}s` : `${Math.round(number)}ms`;
}

function formatUptime(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) return '-';
  return `${number.toFixed(2)}%`;
}

function ModelIcon({ model, size = 'md' }) {
  const source = typeof model.icon === 'string' && /^(https?:)?\/\//.test(model.icon)
    ? model.icon
    : typeof model.vendor_icon === 'string' && /^(https?:)?\/\//.test(model.vendor_icon)
      ? model.vendor_icon
      : '';
  return (
    <span className={`pricing-v2-icon pricing-v2-icon--${size}`}>
      {source ? <img src={source} alt='' /> : <span>{(model.model_name || '?').slice(0, 1).toUpperCase()}</span>}
    </span>
  );
}

function FilterSection({ title, options, value, onChange }) {
  if (!options.length) return null;
  return (
    <section className='pricing-v2-filter-section'>
      <button className='pricing-v2-filter-title' type='button'>
        <span>{title}</span>
        <ChevronDown size={15} />
      </button>
      <div className='pricing-v2-chip-list'>
        {options.map((option) => (
          <button
            key={option.value}
            type='button'
            className={`pricing-v2-filter-chip ${value === option.value ? 'active' : ''}`}
            title={option.label}
            onClick={() => onChange(option.value)}
          >
            <span>{option.label}</span>
            {option.suffix && <em>{option.suffix}</em>}
            {option.count != null && <strong>{option.count}</strong>}
          </button>
        ))}
      </div>
    </section>
  );
}

function PriceRows({ model, groupRatio, group, tokenUnit, showRechargePrice, priceRate }) {
  const price = getModelPriceRows(model, groupRatio, group, tokenUnit, showRechargePrice, priceRate);
  if (price.dynamic && price.rows.length === 0) {
    return (
      <div className='pricing-v2-price-list'>
        <div className='pricing-v2-price-row'>
          <span>动态计费</span>
          <strong>特殊表达式</strong>
        </div>
        <code className='pricing-v2-billing-expr'>{price.dynamic.rawExpression}</code>
      </div>
    );
  }
  return (
    <div className='pricing-v2-price-list'>
      {price.rows.slice(0, 4).map((row) => (
        <div className='pricing-v2-price-row' key={row.key}>
          <span>{row.label}</span>
          <strong>
            {row.value}
            <em>{row.suffix}</em>
          </strong>
        </div>
      ))}
      {price.dynamic?.tiers?.length > 1 && (
        <div className='pricing-v2-price-row muted'>
          <span>阶梯</span>
          <strong>{price.dynamic.tiers.length} 档</strong>
        </div>
      )}
    </div>
  );
}

function DynamicPricingDetails({ model, groupRatio, tokenUnit, showRechargePrice, priceRate }) {
  const price = getModelPriceRows(
    model,
    groupRatio,
    FILTER_ALL,
    tokenUnit,
    showRechargePrice,
    priceRate,
  );
  if (!price.dynamic) return null;
  const ratio = getBestGroupRatio(model, groupRatio, FILTER_ALL);
  const visibleVars = BILLING_PRICING_VARS.filter((variable) =>
    price.dynamic.tiers.some((tier) => Number(tier[variable.field]) > 0),
  );

  if (!price.dynamic.tiers.length) {
    return (
      <section className='pricing-v2-detail-card'>
        <h3><Tags size={15} /> 动态计费表达式</h3>
        <code className='pricing-v2-billing-expr block'>{price.dynamic.rawExpression}</code>
      </section>
    );
  }

  return (
    <section className='pricing-v2-detail-card'>
      <h3><Tags size={15} /> 动态计费阶梯</h3>
      <div className='pricing-v2-table-wrap compact'>
        <table className='pricing-v2-table'>
          <thead>
            <tr>
              <th>阶梯</th>
              <th>条件</th>
              {visibleVars.map((variable) => (
                <th key={variable.field}>{variable.shortLabel}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {price.dynamic.tiers.map((tier, index) => (
              <tr key={`${tier.label}-${index}`}>
                <td>{tier.label || `Tier ${index + 1}`}</td>
                <td>{tier.conditions || '默认'}</td>
                {visibleVars.map((variable) => (
                  <td key={variable.field}>
                    {Number(tier[variable.field]) > 0
                      ? `${formatDynamicUnitPrice(
                          tier[variable.field],
                          ratio,
                          tokenUnit,
                          showRechargePrice,
                          priceRate,
                        )} / 1${tokenUnit}`
                      : '-'}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {price.dynamic.requestRuleExpr && (
        <code className='pricing-v2-billing-expr block'>{price.dynamic.requestRuleExpr}</code>
      )}
    </section>
  );
}

function ModelCard({ model, groupRatio, group, tokenUnit, showRechargePrice, priceRate, endpointMap, perf, onOpen, onCopy, copied }) {
  const price = getModelPriceRows(model, groupRatio, group, tokenUnit, showRechargePrice, priceRate);
  const tags = parseTags(model.tags);
  const endpoints = model.supported_endpoint_types || [];
  const groups = model.enable_groups || [];
  const hidden = Math.max(tags.length - 2, 0) + Math.max(endpoints.length - 2, 0) + Math.max(groups.length - 1, 0);
  return (
    <article className='pricing-v2-model-card'>
      <header>
        <button type='button' className='pricing-v2-model-main' onClick={() => onOpen(model)}>
          <ModelIcon model={model} />
          <span>
            <strong>{model.model_name}</strong>
            <em>{model.vendor_name || '未知供应商'}</em>
          </span>
        </button>
        <div className='pricing-v2-card-actions'>
          <button type='button' onClick={() => onOpen(model)} className='pricing-v2-text-button'>
            详情
            <ChevronRight size={14} />
          </button>
          <button type='button' onClick={() => onCopy(model.model_name)} className='pricing-v2-icon-button' title='复制模型名称'>
            {copied === model.model_name ? <Check size={15} /> : <Copy size={15} />}
          </button>
        </div>
      </header>
      <PriceRows model={model} groupRatio={groupRatio} group={group} tokenUnit={tokenUnit} showRechargePrice={showRechargePrice} priceRate={priceRate} />
      <p>{model.description || '暂无模型描述'}</p>
      <footer>
        <div className='pricing-v2-meta-row'>
          {groups[0] && <span>{groups[0]} 分组</span>}
          <span>{price.mode}</span>
          {perf && <span>{formatUptime(perf.success_rate)} 可用率</span>}
        </div>
        <div className='pricing-v2-chip-list pricing-v2-card-tags'>
          {endpoints.slice(0, 2).map((item) => (
            <span className='pricing-v2-chip' key={item}>
              {getEndpointInfo(item, endpointMap).label}
            </span>
          ))}
          {tags.slice(0, 2).map((item) => (
            <span className='pricing-v2-chip muted' key={item}>
              {item}
            </span>
          ))}
          <span className='pricing-v2-chip muted'>1{tokenUnit}</span>
          {hidden > 0 && <span className='pricing-v2-chip muted'>+{hidden}</span>}
        </div>
      </footer>
    </article>
  );
}

function buildSamples(model, endpointMap, status) {
  const baseUrl = (status.server_address || window.location.origin).replace(/\/+$/, '');
  const endpoint = (model.supported_endpoint_types || [])[0] || 'openai';
  const info = getEndpointInfo(endpoint, endpointMap);
  const path = info.path.replace('{model}', model.model_name);
  const url = `${baseUrl}${path}`;
  const chatBody = endpoint === 'openai-response'
    ? { model: model.model_name, input: 'Explain quantum entanglement in one paragraph.' }
    : { model: model.model_name, messages: [{ role: 'user', content: 'Explain quantum entanglement in one paragraph.' }] };
  const body = endpoint === 'image-generation'
    ? { model: model.model_name, prompt: 'A clean product photo of a futuristic API gateway dashboard' }
    : chatBody;
  const json = JSON.stringify(body, null, 2);
  return [
    {
      lang: 'cURL',
      code: [`curl ${url} \\`, '  -H "Authorization: Bearer $NEW_API_KEY" \\', '  -H "Content-Type: application/json" \\', `  -d '${json.replace(/\n/g, '\n     ')}'`].join('\n'),
    },
    {
      lang: 'Python',
      code: [
        'from openai import OpenAI',
        '',
        'client = OpenAI(',
        `    base_url="${baseUrl}/v1",`,
        '    api_key="<YOUR_API_KEY>",',
        ')',
        '',
        `response = client.chat.completions.create(`,
        `    model="${model.model_name}",`,
        '    messages=[{"role": "user", "content": "Hello"}],',
        ')',
        'print(response.choices[0].message.content)',
      ].join('\n'),
    },
  ];
}

function DetailDrawer({ model, status, groupRatio, usableGroup, endpointMap, autoGroups, tokenUnit, showRechargePrice, priceRate, onClose }) {
  const [tab, setTab] = useState('overview');
  const [perf, setPerf] = useState(null);
  const [perfLoading, setPerfLoading] = useState(false);
  const metadata = useMemo(() => inferMetadata(model), [model]);
  const samples = useMemo(() => buildSamples(model, endpointMap, status), [endpointMap, model, status]);
  const groups = model.enable_groups || [];

  useEffect(() => {
    if (!model || tab !== 'performance') return undefined;
    let cancelled = false;
    setPerfLoading(true);
    axios
      .get('/api/perf-metrics', { params: { model: model.model_name, hours: 24 } })
      .then((res) => {
        if (!cancelled) setPerf(res.data?.data || null);
      })
      .catch(() => {
        if (!cancelled) setPerf(null);
      })
      .finally(() => {
        if (!cancelled) setPerfLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [model, tab]);

  return (
    <div className='pricing-v2-detail-backdrop' role='presentation' onClick={onClose}>
      <aside className='pricing-v2-detail-panel' role='dialog' aria-modal='true' onClick={(event) => event.stopPropagation()}>
        <header className='pricing-v2-detail-header'>
          <div className='pricing-v2-detail-title'>
            <ModelIcon model={model} />
            <div>
              <h2>{model.model_name}</h2>
              <p>{model.vendor_name || '未知供应商'} · {model.quota_type === 0 ? 'Token 计费' : '按次计费'}</p>
            </div>
          </div>
          <button type='button' className='pricing-v2-icon-button' onClick={onClose} aria-label='关闭'>
            <X size={16} />
          </button>
        </header>

        <p className='pricing-v2-detail-desc'>{model.description || model.vendor_description || '暂无模型描述'}</p>

        <div className='pricing-v2-detail-tabs'>
          {[
            ['overview', Info, '概览'],
            ['performance', Activity, '性能'],
            ['api', Code2, 'API'],
          ].map(([key, Icon, label]) => (
            <button key={key} type='button' className={tab === key ? 'active' : ''} onClick={() => setTab(key)}>
              <Icon size={15} />
              {label}
            </button>
          ))}
        </div>

        {tab === 'overview' && (
          <div className='pricing-v2-detail-body'>
            <section className='pricing-v2-detail-card'>
              <h3><Zap size={15} /> 基础价格</h3>
              <PriceRows model={model} groupRatio={groupRatio} group={FILTER_ALL} tokenUnit={tokenUnit} showRechargePrice={showRechargePrice} priceRate={priceRate} />
            </section>
            <DynamicPricingDetails
              model={model}
              groupRatio={groupRatio}
              tokenUnit={tokenUnit}
              showRechargePrice={showRechargePrice}
              priceRate={priceRate}
            />
            <section className='pricing-v2-detail-card'>
              <h3><Gauge size={15} /> 模型能力</h3>
              <div className='pricing-v2-stat-grid'>
                <div><strong>{metadata.context.toLocaleString()}</strong><span>上下文</span></div>
                <div><strong>{metadata.maxOutput.toLocaleString()}</strong><span>最大输出</span></div>
                <div><strong>{metadata.inputModalities.length}</strong><span>输入模态</span></div>
                <div><strong>{metadata.capabilities.length}</strong><span>能力标签</span></div>
              </div>
              <div className='pricing-v2-chip-list'>
                {metadata.inputModalities.map((item) => <ModalityChip key={`in-${item}`} value={item} prefix='输入' />)}
                {metadata.outputModalities.map((item) => <ModalityChip key={`out-${item}`} value={item} prefix='输出' />)}
              </div>
              <div className='pricing-v2-chip-list'>
                {metadata.capabilities.map((item) => (
                  <span className='pricing-v2-chip muted' key={item}>{CAPABILITY_LABELS[item] || item}</span>
                ))}
              </div>
            </section>
            <section className='pricing-v2-detail-card'>
              <h3><Layers size={15} /> 可用分组</h3>
              <div className='pricing-v2-chip-list'>
                {groups.map((item) => (
                  <span className='pricing-v2-chip' key={item}>
                    {item}
                    {groupRatio?.[item] ? ` ${formatRatio(groupRatio[item])}` : ''}
                    {usableGroup?.[item]?.desc ? ` · ${usableGroup[item].desc}` : ''}
                  </span>
                ))}
                {autoGroups?.length > 0 && <span className='pricing-v2-chip muted'>自动链 {autoGroups.filter((item) => groups.includes(item)).join(' -> ') || '-'}</span>}
              </div>
            </section>
            <section className='pricing-v2-detail-card'>
              <h3><SlidersHorizontal size={15} /> 支持端点</h3>
              <div className='pricing-v2-chip-list'>
                {(model.supported_endpoint_types || []).map((item) => {
                  const info = getEndpointInfo(item, endpointMap);
                  return <span className='pricing-v2-chip' key={item}>{info.method} {info.path}</span>;
                })}
              </div>
            </section>
            {parseTags(model.tags).length > 0 && (
              <section className='pricing-v2-detail-card'>
                <h3><Tags size={15} /> 标签</h3>
                <div className='pricing-v2-chip-list'>
                  {parseTags(model.tags).map((item) => <span className='pricing-v2-chip muted' key={item}>{item}</span>)}
                </div>
              </section>
            )}
          </div>
        )}

        {tab === 'performance' && (
          <div className='pricing-v2-detail-body'>
            {perfLoading ? (
              <div className='pricing-v2-empty'><Loader2 className='spin' size={18} /> 正在加载性能数据</div>
            ) : perf?.groups?.length ? (
              <>
                <div className='pricing-v2-stat-grid'>
                  <div><strong>{formatThroughput(avg(perf.groups, 'avg_tps'))}</strong><span>平均吞吐</span></div>
                  <div><strong>{formatLatency(avg(perf.groups, 'avg_latency_ms'))}</strong><span>平均延迟</span></div>
                  <div><strong>{formatLatency(avg(perf.groups, 'avg_ttft_ms'))}</strong><span>首字延迟</span></div>
                  <div><strong>{formatUptime(avg(perf.groups, 'success_rate'))}</strong><span>成功率</span></div>
                </div>
                <section className='pricing-v2-table-wrap compact'>
                  <table className='pricing-v2-table'>
                    <thead><tr><th>分组</th><th>TPS</th><th>TTFT</th><th>延迟</th><th>成功率</th></tr></thead>
                    <tbody>
                      {perf.groups.map((item) => (
                        <tr key={item.group}>
                          <td>{item.group}</td>
                          <td>{formatThroughput(item.avg_tps)}</td>
                          <td>{formatLatency(item.avg_ttft_ms)}</td>
                          <td>{formatLatency(item.avg_latency_ms)}</td>
                          <td>{formatUptime(item.success_rate)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </section>
              </>
            ) : (
              <div className='pricing-v2-empty'><AlertCircle size={18} /> 该模型暂时没有性能数据</div>
            )}
          </div>
        )}

        {tab === 'api' && (
          <div className='pricing-v2-detail-body'>
            <section className='pricing-v2-detail-card'>
              <h3><FileText size={15} /> 调用示例</h3>
              {samples.map((sample) => (
                <div className='pricing-v2-code-block' key={sample.lang}>
                  <div><strong>{sample.lang}</strong></div>
                  <pre>{sample.code}</pre>
                </div>
              ))}
            </section>
          </div>
        )}
      </aside>
    </div>
  );
}

function ModalityChip({ value, prefix }) {
  const icons = {
    text: FileText,
    image: ImageIcon,
    audio: Music2,
    video: Video,
    file: FileText,
  };
  const Icon = icons[value] || Brain;
  return (
    <span className='pricing-v2-chip'>
      <Icon size={13} />
      {prefix} {value}
    </span>
  );
}

function avg(groups, field) {
  const values = (groups || []).map((item) => Number(item[field])).filter((value) => Number.isFinite(value) && value > 0);
  if (!values.length) return 0;
  return values.reduce((sum, value) => sum + value, 0) / values.length;
}

export default function PricingPage({ status = {}, user }) {
  const [models, setModels] = useState([]);
  const [vendors, setVendors] = useState([]);
  const [groupRatio, setGroupRatio] = useState({});
  const [usableGroup, setUsableGroup] = useState({});
  const [endpointMap, setEndpointMap] = useState({});
  const [autoGroups, setAutoGroups] = useState([]);
  const [perfSummary, setPerfSummary] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [query, setQuery] = useState('');
  const [vendor, setVendor] = useState(FILTER_ALL);
  const [group, setGroup] = useState(FILTER_ALL);
  const [quotaType, setQuotaType] = useState(FILTER_ALL);
  const [endpointType, setEndpointType] = useState(FILTER_ALL);
  const [tag, setTag] = useState(FILTER_ALL);
  const [sortBy, setSortBy] = useState('name');
  const [viewMode, setViewMode] = useState('card');
  const [tokenUnit, setTokenUnit] = useState('M');
  const [showRechargePrice, setShowRechargePrice] = useState(false);
  const [page, setPage] = useState(1);
  const [selectedModel, setSelectedModel] = useState(null);
  const [copiedModel, setCopiedModel] = useState('');
  const priceRate = Math.max(Number(status.price || 1), 0.001);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    axios
      .get('/api/pricing')
      .then((res) => {
        if (cancelled) return;
        if (!res.data?.success) {
          setError(res.data?.message || '价格加载失败');
          return;
        }
        const vendorMap = {};
        (res.data.vendors || []).forEach((item) => {
          vendorMap[item.id] = item;
        });
        setVendors(res.data.vendors || []);
        setGroupRatio(res.data.group_ratio || {});
        setUsableGroup(res.data.usable_group || {});
        setEndpointMap(res.data.supported_endpoint || {});
        setAutoGroups(res.data.auto_groups || []);
        setModels(
          (res.data.data || []).map((model) => ({
            ...model,
            vendor_name: vendorMap[model.vendor_id]?.name || model.vendor_name || '未知供应商',
            vendor_icon: vendorMap[model.vendor_id]?.icon || model.vendor_icon || '',
            vendor_description: vendorMap[model.vendor_id]?.description || model.vendor_description || '',
          })),
        );
      })
      .catch(() => setError('价格加载失败，请稍后重试'))
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    axios
      .get('/api/perf-metrics/summary', { params: { hours: 24 } })
      .then((res) => {
        if (cancelled) return;
        const list = res.data?.data?.models || res.data?.data || [];
        const map = {};
        if (Array.isArray(list)) {
          list.forEach((item) => {
            if (item.model || item.model_name) map[item.model || item.model_name] = item;
          });
        }
        setPerfSummary(map);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  const availableGroups = useMemo(() => {
    const modelGroups = new Set();
    models.forEach((model) => (model.enable_groups || []).forEach((item) => modelGroups.add(item)));
    return [...new Set([...Object.keys(usableGroup || {}), ...Object.keys(groupRatio || {}), ...modelGroups])]
      .filter((item) => item && item !== 'auto' && modelGroups.has(item))
      .sort((a, b) => a.localeCompare(b));
  }, [groupRatio, models, usableGroup]);

  const availableEndpoints = useMemo(() => {
    const endpointSet = new Set(Object.keys(endpointMap || {}));
    models.forEach((model) => (model.supported_endpoint_types || []).forEach((item) => endpointSet.add(item)));
    return [...endpointSet].filter(Boolean).sort((a, b) => a.localeCompare(b));
  }, [endpointMap, models]);

  const availableTags = useMemo(() => {
    const tagSet = new Set();
    models.forEach((model) => parseTags(model.tags).forEach((item) => tagSet.add(item)));
    return [...tagSet].sort((a, b) => a.localeCompare(b));
  }, [models]);

  const vendorOptions = useMemo(() => {
    const present = new Set(models.map((model) => model.vendor_name).filter(Boolean));
    const fromApi = vendors.filter((item) => present.has(item.name)).map((item) => item.name);
    const unknown = [...present].filter((item) => !fromApi.includes(item));
    return [...fromApi, ...unknown].sort((a, b) => a.localeCompare(b));
  }, [models, vendors]);

  const filterOptions = useMemo(
    () => ({
      vendor: [
        { value: FILTER_ALL, label: '全部供应商', count: models.length },
        ...vendorOptions.map((item) => ({ value: item, label: item, count: countBy(models, (model) => model.vendor_name === item) })),
      ],
      group: [
        { value: FILTER_ALL, label: '全部分组' },
        ...availableGroups.map((item) => ({ value: item, label: item, suffix: formatRatio(groupRatio?.[item]) })),
      ],
      quota: QUOTA_TYPES.map((item) => ({
        ...item,
        count: item.value === FILTER_ALL ? models.length : countBy(models, (model) => (item.value === 'token' ? model.quota_type === 0 : model.quota_type === 1)),
      })),
      endpoint: [
        { value: FILTER_ALL, label: '全部端点', count: models.length },
        ...availableEndpoints.map((item) => ({ value: item, label: getEndpointInfo(item, endpointMap).label, count: countBy(models, (model) => (model.supported_endpoint_types || []).includes(item)) })),
      ],
      tag: [
        { value: FILTER_ALL, label: '全部标签', count: models.length },
        ...availableTags.map((item) => ({ value: item, label: item, count: countBy(models, (model) => parseTags(model.tags).map((tagItem) => tagItem.toLowerCase()).includes(item.toLowerCase())) })),
      ],
    }),
    [availableEndpoints, availableGroups, availableTags, endpointMap, groupRatio, models, vendorOptions],
  );

  const filteredModels = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    const result = models.filter((model) => {
      if (vendor !== FILTER_ALL && model.vendor_name !== vendor) return false;
      if (group !== FILTER_ALL && !(model.enable_groups || []).includes(group)) return false;
      if (quotaType === 'token' && model.quota_type !== 0) return false;
      if (quotaType === 'request' && model.quota_type !== 1) return false;
      if (endpointType !== FILTER_ALL && !(model.supported_endpoint_types || []).includes(endpointType)) return false;
      if (tag !== FILTER_ALL && !parseTags(model.tags).map((item) => item.toLowerCase()).includes(tag.toLowerCase())) return false;
      if (!keyword) return true;
      return [model.model_name, model.description, model.tags, model.vendor_name, ...(model.supported_endpoint_types || [])]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword));
    });
    return result.sort((a, b) => {
      if (sortBy === 'price-low' || sortBy === 'price-high') {
        const diff = getModelPriceRows(a, groupRatio, group, tokenUnit, showRechargePrice, priceRate).comparable - getModelPriceRows(b, groupRatio, group, tokenUnit, showRechargePrice, priceRate).comparable;
        return sortBy === 'price-low' ? diff : -diff;
      }
      return String(a.model_name || '').localeCompare(String(b.model_name || ''));
    });
  }, [endpointType, group, groupRatio, models, priceRate, query, quotaType, showRechargePrice, sortBy, tag, tokenUnit, vendor]);

  useEffect(() => {
    setPage(1);
  }, [endpointType, group, query, quotaType, sortBy, tag, vendor, viewMode]);

  const totalPages = Math.max(1, Math.ceil(filteredModels.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const pagedModels = filteredModels.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);
  const activeFilterCount = [vendor, group, quotaType, endpointType, tag].filter((item) => item !== FILTER_ALL).length;
  const hasActiveFilters = activeFilterCount > 0;

  const clearFilters = () => {
    setVendor(FILTER_ALL);
    setGroup(FILTER_ALL);
    setQuotaType(FILTER_ALL);
    setEndpointType(FILTER_ALL);
    setTag(FILTER_ALL);
  };

  const copyModelName = async (modelName) => {
    try {
      await navigator.clipboard.writeText(modelName);
      setCopiedModel(modelName);
      window.setTimeout(() => setCopiedModel(''), 1200);
    } catch {
      setError('复制失败，请手动复制模型名称');
    }
  };

  return (
    <PublicFrame status={status} user={user}>
      <main className='pricing-v2-main'>
        <section className='pricing-v2-hero'>
          <p className='public-eyebrow'>Model Square</p>
          <h1>模型广场</h1>
          <p>当前站点已启用 {loading ? '...' : models.length} 个模型</p>
          <span>浏览供应商、分组、端点、标签、计费方式和性能数据，快速比较价格并选择合适模型。</span>
          <label className='pricing-v2-search'>
            <Search size={18} />
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder='搜索模型名称、供应商、端点或标签' />
            {query && <button type='button' onClick={() => setQuery('')} aria-label='清空搜索'><X size={16} /></button>}
          </label>
        </section>

        {error && <div className='pricing-v2-message'><AlertCircle size={16} /> {error}</div>}

        <div className='pricing-v2-layout'>
          <aside className='pricing-v2-sidebar'>
            <div className='pricing-v2-filter-head'>
              <div>
                <strong>筛选</strong>
                <span>按供应商、分组、计费、端点和标签收窄模型</span>
              </div>
              <button type='button' onClick={clearFilters} disabled={!hasActiveFilters}>
                <RotateCcw size={14} />
                重置
              </button>
            </div>
            <FilterSection title='供应商' options={filterOptions.vendor} value={vendor} onChange={setVendor} />
            <FilterSection title='分组' options={filterOptions.group} value={group} onChange={setGroup} />
            <FilterSection title='计费' options={filterOptions.quota} value={quotaType} onChange={setQuotaType} />
            <FilterSection title='端点' options={filterOptions.endpoint} value={endpointType} onChange={setEndpointType} />
            <FilterSection title='标签' options={filterOptions.tag} value={tag} onChange={setTag} />
          </aside>

          <section className='pricing-v2-content'>
            <div className='pricing-v2-toolbar'>
              <div className='pricing-v2-count'>
                <strong>{loading ? '...' : filteredModels.length.toLocaleString()}</strong>
                <span>{activeFilterCount > 0 ? ` / ${models.length.toLocaleString()} 个模型` : ' 个模型'}</span>
              </div>
              <div className='pricing-v2-actions'>
                <select value={sortBy} onChange={(event) => setSortBy(event.target.value)}>
                  {SORT_OPTIONS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                </select>
                <div className='pricing-v2-segment' aria-label='价格模式'>
                  <button type='button' className={!showRechargePrice ? 'active' : ''} onClick={() => setShowRechargePrice(false)}>标准</button>
                  <button type='button' className={showRechargePrice ? 'active' : ''} onClick={() => setShowRechargePrice(true)}>充值</button>
                </div>
                <div className='pricing-v2-segment' aria-label='Token单位'>
                  {['M', 'K'].map((item) => <button type='button' key={item} className={tokenUnit === item ? 'active' : ''} onClick={() => setTokenUnit(item)}>/1{item}</button>)}
                </div>
                <div className='pricing-v2-segment compact' aria-label='视图'>
                  <button type='button' className={viewMode === 'card' ? 'active' : ''} onClick={() => setViewMode('card')} title='卡片视图'><Grid2X2 size={15} /></button>
                  <button type='button' className={viewMode === 'table' ? 'active' : ''} onClick={() => setViewMode('table')} title='表格视图'><Table2 size={15} /></button>
                </div>
                <a className='pricing-v2-key-link' href='/console/token'>创建密钥 <ArrowRight size={14} /></a>
              </div>
            </div>

            {loading ? (
              <div className='pricing-v2-empty'><Loader2 size={18} className='spin' /> 正在加载模型价格</div>
            ) : filteredModels.length === 0 ? (
              <div className='pricing-v2-empty'>
                没有匹配的模型
                {(query || hasActiveFilters) && <button type='button' onClick={() => { setQuery(''); clearFilters(); }}>清空筛选</button>}
              </div>
            ) : viewMode === 'card' ? (
              <div className='pricing-v2-card-grid'>
                {pagedModels.map((model) => (
                  <ModelCard
                    key={model.model_name}
                    model={model}
                    groupRatio={groupRatio}
                    group={group}
                    tokenUnit={tokenUnit}
                    showRechargePrice={showRechargePrice}
                    priceRate={priceRate}
                    endpointMap={endpointMap}
                    perf={perfSummary[model.model_name]}
                    onOpen={setSelectedModel}
                    onCopy={copyModelName}
                    copied={copiedModel}
                  />
                ))}
              </div>
            ) : (
              <section className='pricing-v2-table-wrap'>
                <table className='pricing-v2-table'>
                  <thead>
                    <tr><th>模型</th><th>供应商</th><th>计费</th><th>端点</th><th>价格参考</th><th>可用分组</th></tr>
                  </thead>
                  <tbody>
                    {pagedModels.map((model) => {
                      const price = getModelPriceRows(model, groupRatio, group, tokenUnit, showRechargePrice, priceRate);
                      return (
                        <tr key={model.model_name} onClick={() => setSelectedModel(model)}>
                          <td><div className='pricing-v2-table-model'><ModelIcon model={model} size='sm' /><div><strong>{model.model_name}</strong>{model.description && <span>{model.description}</span>}</div></div></td>
                          <td>{model.vendor_name}</td>
                          <td><span className='pricing-v2-chip muted'>{price.mode}</span></td>
                          <td><div className='pricing-v2-chip-list'>{(model.supported_endpoint_types || []).slice(0, 3).map((item) => <span className='pricing-v2-chip' key={item}>{getEndpointInfo(item, endpointMap).label}</span>)}</div></td>
                          <td><div className='pricing-v2-table-price'><strong>{price.primary}</strong><span>{price.secondary}</span></div></td>
                          <td><div className='pricing-v2-chip-list'>{(model.enable_groups || []).slice(0, 4).map((item) => <span className='pricing-v2-chip muted' key={item}>{item}</span>)}{(model.enable_groups || []).length > 4 && <span className='pricing-v2-chip muted'>+{model.enable_groups.length - 4}</span>}</div></td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </section>
            )}

            {!loading && filteredModels.length > PAGE_SIZE && (
              <div className='pricing-v2-pagination'>
                <span>第 {currentPage} / {totalPages} 页</span>
                <div>
                  <button type='button' onClick={() => setPage((prev) => Math.max(1, prev - 1))} disabled={currentPage <= 1}><ChevronLeft size={15} />上一页</button>
                  <button type='button' onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))} disabled={currentPage >= totalPages}>下一页<ChevronRight size={15} /></button>
                </div>
              </div>
            )}
          </section>
        </div>
      </main>

      {selectedModel && (
        <DetailDrawer
          model={selectedModel}
          status={status}
          groupRatio={groupRatio}
          usableGroup={usableGroup}
          endpointMap={endpointMap}
          autoGroups={autoGroups}
          tokenUnit={tokenUnit}
          showRechargePrice={showRechargePrice}
          priceRate={priceRate}
          onClose={() => setSelectedModel(null)}
        />
      )}
    </PublicFrame>
  );
}
