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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import axios from 'axios';
import {
  AlertCircle,
  ArrowRight,
  BookOpen,
  ChevronLeft,
  ChevronRight,
  Check,
  Copy,
  ExternalLink,
  FileText,
  Github,
  Grid2X2,
  KeyRound,
  Layers,
  Loader2,
  Lock,
  Mail,
  RotateCcw,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  Table2,
  Tags,
  User,
  X,
} from 'lucide-react';
import usageMarkdown from '../../../docs/NAN_USAGE.md?raw';
import clientsMarkdown from '../../../docs/NAN_CLIENTS.md?raw';
import codexMarkdown from '../../../docs/NAN_CLIENT_CODEX.md?raw';
import claudeCodeMarkdown from '../../../docs/NAN_CLIENT_CLAUDE_CODE.md?raw';
import claudeCodeOpenAIMarkdown from '../../../docs/NAN_CLIENT_CLAUDE_CODE_OPENAI.md?raw';
import geminiMarkdown from '../../../docs/NAN_CLIENT_GEMINI.md?raw';
import opencodeMarkdown from '../../../docs/NAN_CLIENT_OPENCODE.md?raw';
import openclawMarkdown from '../../../docs/NAN_CLIENT_OPENCLAW.md?raw';
import ccSwitchMarkdown from '../../../docs/NAN_CLIENT_CCSWITCH.md?raw';
import troubleshootingMarkdown from '../../../docs/NAN_TROUBLESHOOTING.md?raw';

const defaultStatus = {
  system_name: 'New API',
  logo: '/logo.svg',
  server_address: window.location.origin,
};

const DOCS = [
  {
    key: 'usage',
    path: '/docs',
    group: '基础',
    title: '平台教程',
    description: '注册、令牌、调用、充值、发票、扣费和安全提醒',
    content: usageMarkdown,
  },
  {
    key: 'clients',
    path: '/docs/clients',
    group: '客户端',
    title: '客户端接入',
    description: 'Codex、Claude Code、Gemini、OpenCode、OpenClaw、CC Switch',
    content: clientsMarkdown,
  },
  {
    key: 'client-codex',
    path: '/docs/clients/codex',
    group: '客户端',
    title: 'Codex',
    description: 'Codex 一键配置与手动配置',
    content: codexMarkdown,
  },
  {
    key: 'client-claude-code',
    path: '/docs/clients/claude-code',
    group: '客户端',
    title: 'Claude Code',
    description: 'Claude Code 使用 Claude 模型',
    content: claudeCodeMarkdown,
  },
  {
    key: 'client-claude-code-openai',
    path: '/docs/clients/claude-code-openai',
    group: '客户端',
    title: 'Claude Code OpenAI',
    description: 'Claude Code 使用 OpenAI / Codex 模型',
    content: claudeCodeOpenAIMarkdown,
  },
  {
    key: 'client-gemini',
    path: '/docs/clients/gemini',
    group: '客户端',
    title: 'Gemini CLI',
    description: 'Gemini CLI 配置与验证',
    content: geminiMarkdown,
  },
  {
    key: 'client-opencode',
    path: '/docs/clients/opencode',
    group: '客户端',
    title: 'OpenCode',
    description: 'OpenCode 环境变量与配置文件',
    content: opencodeMarkdown,
  },
  {
    key: 'client-openclaw',
    path: '/docs/clients/openclaw',
    group: '客户端',
    title: 'OpenClaw',
    description: 'OpenClaw 本地网关与多模型配置',
    content: openclawMarkdown,
  },
  {
    key: 'client-ccswitch',
    path: '/docs/clients/ccswitch',
    group: '客户端',
    title: 'CC Switch',
    description: '一键导入配置与聊天入口边界',
    content: ccSwitchMarkdown,
  },
  {
    key: 'troubleshooting',
    path: '/docs/troubleshooting',
    group: '排查',
    title: '常见报错排查',
    description: '按错误文本排查注册、认证、模型、客户端、网络、支付问题',
    content: troubleshootingMarkdown,
  },
];

const DOC_LINK_REPLACEMENTS = [
  [/\.\/NAN_USAGE\.md/g, '/docs'],
  [/\.\/NAN_CLIENTS\.md/g, '/docs/clients'],
  [/\.\/NAN_CLIENT_CODEX\.md/g, '/docs/clients/codex'],
  [/\.\/NAN_CLIENT_CLAUDE_CODE\.md/g, '/docs/clients/claude-code'],
  [
    /\.\/NAN_CLIENT_CLAUDE_CODE_OPENAI\.md/g,
    '/docs/clients/claude-code-openai',
  ],
  [/\.\/NAN_CLIENT_GEMINI\.md/g, '/docs/clients/gemini'],
  [/\.\/NAN_CLIENT_OPENCODE\.md/g, '/docs/clients/opencode'],
  [/\.\/NAN_CLIENT_OPENCLAW\.md/g, '/docs/clients/openclaw'],
  [/\.\/NAN_CLIENT_CCSWITCH\.md/g, '/docs/clients/ccswitch'],
  [/\.\/NAN_TROUBLESHOOTING\.md/g, '/docs/troubleshooting'],
  [/\.\/images\//g, '/docs/images/'],
];

const DOC_GROUPS = DOCS.reduce((groups, doc) => {
  if (!groups.includes(doc.group)) groups.push(doc.group);
  return groups;
}, []);

function writeJson(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Storage may be unavailable.
  }
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

async function renderMarkupContent(value) {
  if (!value || isRemoteUrl(value) || isRawHtml(value)) return value || '';
  const { marked } = await import('marked');
  return marked.parse(value);
}

function normalizePathname(pathname) {
  return pathname.replace(/\/+$/, '') || '/';
}

function getSafeLoginRedirectPath() {
  const next = new URLSearchParams(window.location.search).get('next');
  return next && next.startsWith('/') && !next.startsWith('//')
    ? next
    : '/console';
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

function buildNavLinks(status, user) {
  const modules = parseHeaderModules(status.HeaderNavModules);
  const docsLink = status.docs_link || localStorage.getItem('docs_link') || '';
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
    isEnabled(modules.docs) && {
      label: '文档',
      href: docsLink || '/docs',
      external: Boolean(docsLink) && isRemoteUrl(docsLink),
    },
    isEnabled(modules.about) && { label: '关于', href: '/about' },
  ].filter(Boolean);
}

function PublicHeader({ status, user }) {
  const links = buildNavLinks(status, user);
  return (
    <header className='topbar public-topbar'>
      <a className='brand' href='/'>
        <img src={status.logo || '/logo.svg'} alt='' />
        <span>{status.system_name || 'New API'}</span>
      </a>
      <nav>
        {links.map((link) => (
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
          <a className='user-chip public-user-chip' href='/console/personal'>
            {(user.username || 'U').slice(0, 1).toUpperCase()}
            <span>{user.username}</span>
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
  );
}

function PublicFrame({ status, user, children, className = '' }) {
  return (
    <div className={`default-home public-page ${className}`}>
      <PublicHeader status={status} user={user} />
      {children}
    </div>
  );
}

function PageHero({ eyebrow, title, description, actions }) {
  return (
    <section className='public-hero'>
      <div>
        <p className='public-eyebrow'>{eyebrow}</p>
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      {actions && <div className='public-hero-actions'>{actions}</div>}
    </section>
  );
}

function Message({ type = 'info', children }) {
  return (
    <div className={`public-message public-message--${type}`}>
      {type === 'success' ? <Check size={16} /> : <AlertCircle size={16} />}
      <span>{children}</span>
    </div>
  );
}

function formatCompactNumber(value, digits = 6) {
  const number = Number(value);
  if (!Number.isFinite(number)) return '-';
  const fixed = number.toFixed(digits);
  return fixed.replace(/\.?0+$/, '') || '0';
}

function formatUsdPrice(value, digits = 6) {
  return `$${formatCompactNumber(value, digits)}`;
}

const PRICING_PAGE_SIZE = 20;
const FILTER_ALL = 'all';
const QUOTA_TYPES = [
  { value: 'all', label: '全部模型' },
  { value: 'token', label: '按量计费' },
  { value: 'request', label: '按次计费' },
];
const SORT_OPTIONS = [
  { value: 'name', label: '名称' },
  { value: 'price-low', label: '价格从低到高' },
  { value: 'price-high', label: '价格从高到低' },
];
const ENDPOINT_LABELS = {
  openai: 'Chat',
  'openai-response': 'Response',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  'jina-rerank': 'Rerank',
  'image-generation': 'Image',
  embeddings: 'Embeddings',
  'openai-video': 'Video',
};

function parseTags(value) {
  if (!value) return [];
  return String(value)
    .split(/[,;|]+/)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function isImageSource(value) {
  return typeof value === 'string' && /^(https?:)?\/\//.test(value);
}

function getModelInitial(modelName) {
  return (modelName || '?').trim().slice(0, 1).toUpperCase() || '?';
}

function getEndpointLabel(endpoint, endpointMap = {}) {
  return (
    ENDPOINT_LABELS[endpoint] ||
    endpointMap?.[endpoint]?.path ||
    endpoint ||
    'Endpoint'
  );
}

function countBy(models, predicate) {
  return models.reduce((count, model) => count + (predicate(model) ? 1 : 0), 0);
}

function formatGroupRatio(ratio) {
  const number = Number(ratio);
  if (!Number.isFinite(number)) return '';
  return `x${formatCompactNumber(number, 3)}`;
}

function getBestGroupRatio(model, groupRatio, selectedGroup = FILTER_ALL) {
  if (selectedGroup !== FILTER_ALL) {
    const selectedRatio = Number(groupRatio?.[selectedGroup]);
    return Number.isFinite(selectedRatio) && selectedRatio > 0
      ? selectedRatio
      : 1;
  }
  const groups = Array.isArray(model.enable_groups) ? model.enable_groups : [];
  const candidates = groups
    .map((group) => Number(groupRatio?.[group]))
    .filter((ratio) => Number.isFinite(ratio) && ratio > 0);
  if (candidates.length === 0) {
    const allRatios = Object.values(groupRatio || {})
      .map(Number)
      .filter((ratio) => Number.isFinite(ratio) && ratio > 0);
    return allRatios.length > 0 ? Math.min(...allRatios) : 1;
  }
  return Math.min(...candidates);
}

function displayMoney(value, showRechargePrice, priceRate, digits = 6) {
  const number = Number(value);
  if (!Number.isFinite(number)) return showRechargePrice ? '¥-' : '$-';
  if (!showRechargePrice) return formatUsdPrice(number, digits);
  return `¥${formatCompactNumber(number * priceRate, digits)}`;
}

function getModelPrices(
  model,
  groupRatio,
  selectedGroup = FILTER_ALL,
  tokenUnit = 'M',
  showRechargePrice = false,
  priceRate = 1,
) {
  const ratio = getBestGroupRatio(model, groupRatio, selectedGroup);
  if (model.quota_type === 1) {
    return {
      type: '按次',
      primary: `${displayMoney(
        Number(model.model_price || 0) * ratio,
        showRechargePrice,
        priceRate,
      )} / 次`,
      secondary: `分组倍率 ${formatCompactNumber(ratio, 3)}`,
      rows: [
        {
          label: '按次',
          value: displayMoney(
            Number(model.model_price || 0) * ratio,
            showRechargePrice,
            priceRate,
          ),
        },
      ],
    };
  }
  const divisor = tokenUnit === 'K' ? 1000 : 1;
  const input = (Number(model.model_ratio || 0) * 2 * ratio) / divisor;
  const output = input * Number(model.completion_ratio || 0);
  const rows = [
    {
      label: '输入',
      value: displayMoney(input, showRechargePrice, priceRate),
    },
    {
      label: '输出',
      value: displayMoney(output, showRechargePrice, priceRate),
    },
  ];
  if (model.supports_cache_read && Number(model.cache_ratio) > 0) {
    rows.push({
      label: '缓存',
      value: displayMoney(
        input * Number(model.cache_ratio),
        showRechargePrice,
        priceRate,
      ),
    });
  }
  return {
    type: '按量',
    primary: `${displayMoney(
      input,
      showRechargePrice,
      priceRate,
    )} / 1${tokenUnit} 输入`,
    secondary: `${displayMoney(
      output,
      showRechargePrice,
      priceRate,
    )} / 1${tokenUnit} 输出`,
    rows,
  };
}

function getComparablePrice(model, groupRatio, selectedGroup) {
  const ratio = getBestGroupRatio(model, groupRatio, selectedGroup);
  if (model.quota_type === 1) return Number(model.model_price || 0) * ratio;
  return Number(model.model_ratio || 0) * 2 * ratio;
}

function PricingFilterSection({ title, options, value, onChange }) {
  if (!options.length) return null;
  return (
    <section className='pricing-filter-section'>
      <h3>{title}</h3>
      <div className='pricing-filter-chips'>
        {options.map((option) => (
          <button
            type='button'
            key={option.value}
            className={`pricing-filter-chip ${
              value === option.value ? 'active' : ''
            }`}
            onClick={() => onChange(option.value)}
            title={option.label}
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

function PricingPage({ status, user }) {
  const [models, setModels] = useState([]);
  const [vendors, setVendors] = useState([]);
  const [groupRatio, setGroupRatio] = useState({});
  const [usableGroup, setUsableGroup] = useState({});
  const [endpointMap, setEndpointMap] = useState({});
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
  const priceRate = Number(status.price || 1);

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
        setModels(
          (res.data.data || [])
            .map((model) => ({
              ...model,
              vendor_name: vendorMap[model.vendor_id]?.name || '未知供应商',
              vendor_icon: vendorMap[model.vendor_id]?.icon || '',
              vendor_description: vendorMap[model.vendor_id]?.description || '',
            }))
            .sort((a, b) => {
              const aGpt = a.model_name?.startsWith('gpt') ? 0 : 1;
              const bGpt = b.model_name?.startsWith('gpt') ? 0 : 1;
              return aGpt - bGpt || a.model_name.localeCompare(b.model_name);
            }),
        );
      })
      .catch(() => setError('价格加载失败，请稍后重试'))
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const availableGroups = useMemo(() => {
    const modelGroups = new Set();
    models.forEach((model) => {
      (model.enable_groups || []).forEach((item) => modelGroups.add(item));
    });
    const groups = [
      ...new Set([
        ...Object.keys(usableGroup || {}),
        ...Object.keys(groupRatio || {}),
        ...modelGroups,
      ]),
    ];
    return groups
      .filter((item) => item && item !== 'auto' && modelGroups.has(item))
      .sort((a, b) => a.localeCompare(b));
  }, [groupRatio, models, usableGroup]);

  const availableEndpoints = useMemo(() => {
    const endpointSet = new Set(Object.keys(endpointMap || {}));
    models.forEach((model) => {
      (model.supported_endpoint_types || []).forEach((item) =>
        endpointSet.add(item),
      );
    });
    return [...endpointSet].filter(Boolean).sort((a, b) => a.localeCompare(b));
  }, [endpointMap, models]);

  const availableTags = useMemo(() => {
    const tagSet = new Set();
    models.forEach((model) => {
      parseTags(model.tags).forEach((item) => tagSet.add(item));
    });
    return [...tagSet].sort((a, b) => a.localeCompare(b));
  }, [models]);

  const vendorOptions = useMemo(() => {
    const present = new Set(models.map((model) => model.vendor_name));
    const fromApi = vendors
      .filter((item) => present.has(item.name))
      .map((item) => item.name);
    const unknown = [...present].filter((item) => !fromApi.includes(item));
    return [...fromApi, ...unknown].sort((a, b) => a.localeCompare(b));
  }, [models, vendors]);

  const filterOptions = useMemo(
    () => ({
      vendor: [
        { value: FILTER_ALL, label: '全部供应商', count: models.length },
        ...vendorOptions.map((item) => ({
          value: item,
          label: item,
          count: countBy(models, (model) => model.vendor_name === item),
        })),
      ],
      group: [
        { value: FILTER_ALL, label: '全部分组' },
        ...availableGroups.map((item) => ({
          value: item,
          label: item,
          suffix: formatGroupRatio(groupRatio?.[item]),
        })),
      ],
      quota: QUOTA_TYPES.map((item) => ({
        ...item,
        count:
          item.value === FILTER_ALL
            ? models.length
            : countBy(models, (model) =>
                item.value === 'token'
                  ? model.quota_type === 0
                  : model.quota_type === 1,
              ),
      })),
      endpoint: [
        { value: FILTER_ALL, label: '全部端点', count: models.length },
        ...availableEndpoints.map((item) => ({
          value: item,
          label: getEndpointLabel(item, endpointMap),
          count: countBy(models, (model) =>
            (model.supported_endpoint_types || []).includes(item),
          ),
        })),
      ],
      tag: [
        { value: FILTER_ALL, label: '全部标签', count: models.length },
        ...availableTags.map((item) => ({
          value: item,
          label: item,
          count: countBy(models, (model) =>
            parseTags(model.tags)
              .map((tagItem) => tagItem.toLowerCase())
              .includes(item.toLowerCase()),
          ),
        })),
      ],
    }),
    [
      availableEndpoints,
      availableGroups,
      availableTags,
      endpointMap,
      groupRatio,
      models,
      vendorOptions,
    ],
  );

  const filteredModels = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    const result = models.filter((model) => {
      if (vendor !== FILTER_ALL && model.vendor_name !== vendor) return false;
      if (
        group !== FILTER_ALL &&
        !(model.enable_groups || []).includes(group)
      ) {
        return false;
      }
      if (quotaType === 'token' && model.quota_type !== 0) return false;
      if (quotaType === 'request' && model.quota_type !== 1) return false;
      if (
        endpointType !== FILTER_ALL &&
        !(model.supported_endpoint_types || []).includes(endpointType)
      ) {
        return false;
      }
      if (
        tag !== FILTER_ALL &&
        !parseTags(model.tags)
          .map((item) => item.toLowerCase())
          .includes(tag.toLowerCase())
      ) {
        return false;
      }
      if (!keyword) return true;
      return [
        model.model_name,
        model.description,
        model.tags,
        model.vendor_name,
        ...(model.supported_endpoint_types || []),
      ]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword));
    });
    return result.sort((a, b) => {
      if (sortBy === 'price-low' || sortBy === 'price-high') {
        const diff =
          getComparablePrice(a, groupRatio, group) -
          getComparablePrice(b, groupRatio, group);
        return sortBy === 'price-low' ? diff : -diff;
      }
      return a.model_name.localeCompare(b.model_name);
    });
  }, [
    endpointType,
    group,
    groupRatio,
    models,
    query,
    quotaType,
    sortBy,
    tag,
    vendor,
  ]);

  useEffect(() => {
    setPage(1);
  }, [endpointType, group, query, quotaType, sortBy, tag, vendor, viewMode]);

  const totalPages = Math.max(
    1,
    Math.ceil(filteredModels.length / PRICING_PAGE_SIZE),
  );
  const currentPage = Math.min(page, totalPages);
  const pagedModels = filteredModels.slice(
    (currentPage - 1) * PRICING_PAGE_SIZE,
    currentPage * PRICING_PAGE_SIZE,
  );

  const hasActiveFilters =
    vendor !== FILTER_ALL ||
    group !== FILTER_ALL ||
    quotaType !== FILTER_ALL ||
    endpointType !== FILTER_ALL ||
    tag !== FILTER_ALL;
  const activeFilterCount = [
    vendor,
    group,
    quotaType,
    endpointType,
    tag,
  ].filter((item) => item !== FILTER_ALL).length;

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

  const renderModelIcon = (model) => {
    const source = isImageSource(model.icon) ? model.icon : model.vendor_icon;
    if (isImageSource(source)) {
      return <img src={source} alt='' />;
    }
    return <span>{getModelInitial(model.model_name)}</span>;
  };

  const renderPriceRows = (model) => {
    const price = getModelPrices(
      model,
      groupRatio,
      group,
      tokenUnit,
      showRechargePrice,
      priceRate,
    );
    if (model.billing_mode === 'tiered_expr' && model.billing_expr) {
      return (
        <div className='pricing-price-list'>
          <div className='pricing-price-row'>
            <span>动态计费</span>
            <strong>按表达式</strong>
          </div>
          <code>{model.billing_expr}</code>
        </div>
      );
    }
    return (
      <div className='pricing-price-list'>
        {price.rows.map((item) => (
          <div className='pricing-price-row' key={item.label}>
            <span>{item.label}</span>
            <strong>
              {item.value}
              {model.quota_type === 0 ? ` / 1${tokenUnit}` : ''}
            </strong>
          </div>
        ))}
      </div>
    );
  };

  const renderModelCard = (model) => {
    const tags = parseTags(model.tags);
    const groups = model.enable_groups || [];
    const endpoints = model.supported_endpoint_types || [];
    const hiddenCount =
      Math.max(groups.length - 1, 0) +
      Math.max(tags.length - 2, 0) +
      Math.max(endpoints.length - 2, 0);
    return (
      <article className='pricing-model-card' key={model.model_name}>
        <div className='pricing-card-head'>
          <button
            type='button'
            className='pricing-card-main'
            onClick={() => setSelectedModel(model)}
          >
            <span className='pricing-model-icon'>{renderModelIcon(model)}</span>
            <span className='pricing-card-title'>
              <strong>{model.model_name}</strong>
              <em>{model.vendor_name}</em>
            </span>
          </button>
          <div className='pricing-card-actions'>
            <button
              type='button'
              onClick={() => setSelectedModel(model)}
              className='pricing-text-button'
            >
              详情
              <ChevronRight size={14} />
            </button>
            <button
              type='button'
              className='pricing-icon-button'
              onClick={() => copyModelName(model.model_name)}
              title='复制模型名称'
            >
              {copiedModel === model.model_name ? (
                <Check size={15} />
              ) : (
                <Copy size={15} />
              )}
            </button>
          </div>
        </div>

        {renderPriceRows(model)}

        <p className='pricing-card-desc'>
          {model.description || '暂无模型描述'}
        </p>

        <div className='pricing-card-foot'>
          <div className='pricing-meta-row'>
            {groups[0] && <span>{groups[0]} 分组</span>}
            <span>
              {model.quota_type === 0 ? 'Token-based' : 'Per Request'}
            </span>
          </div>
          <div className='pricing-tag-row'>
            {endpoints.slice(0, 2).map((item) => (
              <span className='pricing-chip' key={item}>
                {getEndpointLabel(item, endpointMap)}
              </span>
            ))}
            {tags.slice(0, 2).map((item) => (
              <span className='pricing-chip pricing-chip--muted' key={item}>
                {item}
              </span>
            ))}
            <span className='pricing-chip pricing-chip--muted'>
              1{tokenUnit}
            </span>
            {hiddenCount > 0 && (
              <span className='pricing-chip pricing-chip--muted'>
                +{hiddenCount}
              </span>
            )}
          </div>
        </div>
      </article>
    );
  };

  return (
    <PublicFrame status={status} user={user} className='pricing-page'>
      <main className='public-main pricing-main'>
        <section className='pricing-intro'>
          <p className='public-eyebrow'>Model Square</p>
          <h1>模型广场</h1>
          <p>当前站点已启用 {models.length} 个模型</p>
          <span>
            浏览供应商、分组、端点和标签，快速比较价格并选择合适模型。
          </span>
          <label className='pricing-search-hero'>
            <Search size={18} />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='搜索模型名称、供应商、端点或标签'
            />
            {query && (
              <button
                type='button'
                onClick={() => setQuery('')}
                aria-label='清空搜索'
              >
                <X size={16} />
              </button>
            )}
          </label>
        </section>

        {error && <Message type='error'>{error}</Message>}

        <div className='pricing-layout-grid'>
          <aside className='pricing-sidebar'>
            <div className='pricing-filter-head'>
              <div>
                <strong>筛选</strong>
                <span>按供应商、分组、类型和标签收窄模型</span>
              </div>
              <button
                type='button'
                className='pricing-filter-reset'
                onClick={clearFilters}
                disabled={!hasActiveFilters}
              >
                <RotateCcw size={14} />
                重置
              </button>
            </div>
            <PricingFilterSection
              title='供应商'
              options={filterOptions.vendor}
              value={vendor}
              onChange={setVendor}
            />
            <PricingFilterSection
              title='分组'
              options={filterOptions.group}
              value={group}
              onChange={setGroup}
            />
            <PricingFilterSection
              title='计费'
              options={filterOptions.quota}
              value={quotaType}
              onChange={setQuotaType}
            />
            <PricingFilterSection
              title='端点'
              options={filterOptions.endpoint}
              value={endpointType}
              onChange={setEndpointType}
            />
            <PricingFilterSection
              title='标签'
              options={filterOptions.tag}
              value={tag}
              onChange={setTag}
            />
          </aside>

          <section className='pricing-content'>
            <div className='pricing-toolbar'>
              <div className='pricing-count'>
                <strong>{loading ? '...' : filteredModels.length}</strong>
                <span>
                  {activeFilterCount > 0
                    ? ` / ${models.length} 个模型`
                    : ' 个模型'}
                </span>
              </div>
              <div className='pricing-toolbar-actions'>
                <select
                  value={sortBy}
                  onChange={(event) => setSortBy(event.target.value)}
                >
                  {SORT_OPTIONS.map((item) => (
                    <option key={item.value} value={item.value}>
                      {item.label}
                    </option>
                  ))}
                </select>
                <div className='pricing-segment' aria-label='价格模式'>
                  <button
                    type='button'
                    className={!showRechargePrice ? 'active' : ''}
                    onClick={() => setShowRechargePrice(false)}
                  >
                    标准
                  </button>
                  <button
                    type='button'
                    className={showRechargePrice ? 'active' : ''}
                    onClick={() => setShowRechargePrice(true)}
                  >
                    充值
                  </button>
                </div>
                <div className='pricing-segment' aria-label='Token单位'>
                  {['M', 'K'].map((item) => (
                    <button
                      type='button'
                      key={item}
                      className={tokenUnit === item ? 'active' : ''}
                      onClick={() => setTokenUnit(item)}
                    >
                      /1{item}
                    </button>
                  ))}
                </div>
                <div
                  className='pricing-segment pricing-view-tabs'
                  aria-label='视图'
                >
                  <button
                    type='button'
                    className={viewMode === 'card' ? 'active' : ''}
                    onClick={() => setViewMode('card')}
                    title='卡片视图'
                  >
                    <Grid2X2 size={15} />
                  </button>
                  <button
                    type='button'
                    className={viewMode === 'table' ? 'active' : ''}
                    onClick={() => setViewMode('table')}
                    title='表格视图'
                  >
                    <Table2 size={15} />
                  </button>
                </div>
                <a className='pricing-key-link' href='/console/token'>
                  创建密钥
                  <ArrowRight size={14} />
                </a>
              </div>
            </div>

            {loading ? (
              <div className='pricing-empty-card'>
                <Loader2 size={18} className='spin' />
                正在加载模型价格
              </div>
            ) : filteredModels.length === 0 ? (
              <div className='pricing-empty-card'>
                没有匹配的模型
                {(query || hasActiveFilters) && (
                  <button
                    type='button'
                    onClick={() => {
                      setQuery('');
                      clearFilters();
                    }}
                  >
                    清空筛选
                  </button>
                )}
              </div>
            ) : viewMode === 'card' ? (
              <div className='pricing-card-grid'>
                {pagedModels.map((model) => renderModelCard(model))}
              </div>
            ) : (
              <section className='pricing-table-wrap'>
                <table className='pricing-table'>
                  <thead>
                    <tr>
                      <th>模型</th>
                      <th>供应商</th>
                      <th>计费</th>
                      <th>端点</th>
                      <th>价格参考</th>
                      <th>可用分组</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pagedModels.map((model) => {
                      const price = getModelPrices(
                        model,
                        groupRatio,
                        group,
                        tokenUnit,
                        showRechargePrice,
                        priceRate,
                      );
                      return (
                        <tr
                          key={model.model_name}
                          onClick={() => setSelectedModel(model)}
                        >
                          <td>
                            <div className='pricing-model-name'>
                              <span className='pricing-model-icon'>
                                {renderModelIcon(model)}
                              </span>
                              <div>
                                <strong>{model.model_name}</strong>
                                {model.description && (
                                  <span>{model.description}</span>
                                )}
                              </div>
                            </div>
                          </td>
                          <td>{model.vendor_name}</td>
                          <td>
                            <span className='public-pill'>{price.type}</span>
                          </td>
                          <td>
                            <div className='pricing-tag-row'>
                              {(model.supported_endpoint_types || [])
                                .slice(0, 3)
                                .map((item) => (
                                  <span className='pricing-chip' key={item}>
                                    {getEndpointLabel(item, endpointMap)}
                                  </span>
                                ))}
                            </div>
                          </td>
                          <td>
                            <div className='pricing-price'>
                              <strong>{price.primary}</strong>
                              <span>{price.secondary}</span>
                            </div>
                          </td>
                          <td>
                            <div className='pricing-groups'>
                              {(model.enable_groups || [])
                                .slice(0, 4)
                                .map((item) => (
                                  <span key={item}>{item}</span>
                                ))}
                              {(model.enable_groups || []).length > 4 && (
                                <span>+{model.enable_groups.length - 4}</span>
                              )}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </section>
            )}

            {!loading && filteredModels.length > PRICING_PAGE_SIZE && (
              <div className='pricing-pagination'>
                <span>
                  第 {currentPage} / {totalPages} 页
                </span>
                <div>
                  <button
                    type='button'
                    onClick={() => setPage((prev) => Math.max(1, prev - 1))}
                    disabled={currentPage <= 1}
                  >
                    <ChevronLeft size={15} />
                    上一页
                  </button>
                  <button
                    type='button'
                    onClick={() =>
                      setPage((prev) => Math.min(totalPages, prev + 1))
                    }
                    disabled={currentPage >= totalPages}
                  >
                    下一页
                    <ChevronRight size={15} />
                  </button>
                </div>
              </div>
            )}
          </section>
        </div>

        {selectedModel && (
          <div
            className='pricing-detail-backdrop'
            role='presentation'
            onClick={() => setSelectedModel(null)}
          >
            <aside
              className='pricing-detail-panel'
              role='dialog'
              aria-modal='true'
              aria-label='模型详情'
              onClick={(event) => event.stopPropagation()}
            >
              <header>
                <div className='pricing-detail-title'>
                  <span className='pricing-model-icon'>
                    {renderModelIcon(selectedModel)}
                  </span>
                  <div>
                    <h2>{selectedModel.model_name}</h2>
                    <p>{selectedModel.vendor_name}</p>
                  </div>
                </div>
                <button
                  type='button'
                  className='pricing-icon-button'
                  onClick={() => setSelectedModel(null)}
                  aria-label='关闭'
                >
                  <X size={16} />
                </button>
              </header>
              <p className='pricing-detail-desc'>
                {selectedModel.description || '暂无模型描述'}
              </p>
              {renderPriceRows(selectedModel)}
              <section className='pricing-detail-section'>
                <h3>
                  <Layers size={15} />
                  可用分组
                </h3>
                <div className='pricing-tag-row'>
                  {(selectedModel.enable_groups || []).map((item) => (
                    <span className='pricing-chip' key={item}>
                      {item}
                      {groupRatio?.[item]
                        ? ` ${formatGroupRatio(groupRatio[item])}`
                        : ''}
                    </span>
                  ))}
                </div>
              </section>
              <section className='pricing-detail-section'>
                <h3>
                  <SlidersHorizontal size={15} />
                  支持端点
                </h3>
                <div className='pricing-tag-row'>
                  {(selectedModel.supported_endpoint_types || []).map(
                    (item) => (
                      <span className='pricing-chip' key={item}>
                        {getEndpointLabel(item, endpointMap)}
                      </span>
                    ),
                  )}
                </div>
              </section>
              {parseTags(selectedModel.tags).length > 0 && (
                <section className='pricing-detail-section'>
                  <h3>
                    <Tags size={15} />
                    标签
                  </h3>
                  <div className='pricing-tag-row'>
                    {parseTags(selectedModel.tags).map((item) => (
                      <span
                        className='pricing-chip pricing-chip--muted'
                        key={item}
                      >
                        {item}
                      </span>
                    ))}
                  </div>
                </section>
              )}
            </aside>
          </div>
        )}
      </main>
    </PublicFrame>
  );
}
function AboutFallback() {
  const currentYear = new Date().getFullYear();
  return (
    <div className='about-fallback'>
      <FileText size={34} />
      <h2>管理员暂时未设置任何关于内容</h2>
      <p>可在设置页面设置关于内容，支持 HTML 和 Markdown。</p>
      <p>
        <a
          href='https://github.com/QuantumNous/new-api'
          target='_blank'
          rel='noreferrer'
        >
          New API
        </a>{' '}
        © {currentYear}{' '}
        <a
          href='https://github.com/QuantumNous'
          target='_blank'
          rel='noreferrer'
        >
          QuantumNous
        </a>{' '}
        | 基于{' '}
        <a
          href='https://github.com/songquanpeng/one-api/releases/tag/v0.5.4'
          target='_blank'
          rel='noreferrer'
        >
          One API v0.5.4
        </a>
      </p>
    </div>
  );
}

function AboutPage({ status, user }) {
  const [content, setContent] = useState('');
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    const cached = localStorage.getItem('about') || '';
    if (cached) setContent(cached);
    axios
      .get('/api/about')
      .then((res) => {
        if (cancelled) return;
        if (!res.data?.success) {
          setError(res.data?.message || '关于内容加载失败');
          return;
        }
        const raw = res.data.data || '';
        return renderMarkupContent(raw).then((html) => {
          if (cancelled) return;
          setContent(html);
          if (raw) localStorage.setItem('about', html);
          else localStorage.removeItem('about');
        });
      })
      .catch(() => setError('关于内容加载失败'))
      .finally(() => {
        if (!cancelled) setLoaded(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <PublicFrame status={status} user={user} className='about-page'>
      <main className='public-main'>
        <PageHero
          eyebrow='About'
          title='关于本站'
          description='查看平台说明、服务信息和项目 attribution。'
        />
        {error && <Message type='error'>{error}</Message>}
        {!loaded && !content ? (
          <div className='public-content-card public-loading-card'>
            <Loader2 size={18} className='spin' />
            正在加载关于内容
          </div>
        ) : content ? (
          isRemoteUrl(content) ? (
            <iframe title='About' src={content} className='about-frame' />
          ) : (
            <article
              className='public-content-card markup-content'
              dangerouslySetInnerHTML={{ __html: content }}
            />
          )
        ) : (
          <AboutFallback />
        )}
      </main>
    </PublicFrame>
  );
}

function normalizeMarkdownLinks(content) {
  return DOC_LINK_REPLACEMENTS.reduce(
    (current, [pattern, replacement]) => current.replace(pattern, replacement),
    content,
  );
}

function getActiveDoc(pathname) {
  const normalizedPath = normalizePathname(pathname);
  const matched = [...DOCS]
    .sort((a, b) => b.path.length - a.path.length)
    .find(
      (doc) =>
        normalizedPath === doc.path ||
        normalizedPath.startsWith(`${doc.path}/`),
    );
  return matched || DOCS[0];
}

function MarkdownArticle({ content }) {
  const [html, setHtml] = useState('');
  useEffect(() => {
    let cancelled = false;
    renderMarkupContent(content).then((nextHtml) => {
      if (!cancelled) setHtml(nextHtml);
    });
    return () => {
      cancelled = true;
    };
  }, [content]);

  return (
    <article
      className='docs-article markup-content'
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

function DocsPage({ status, user }) {
  const activeDoc = getActiveDoc(window.location.pathname);
  const content = useMemo(
    () => normalizeMarkdownLinks(activeDoc.content),
    [activeDoc.content],
  );

  return (
    <PublicFrame status={status} user={user} className='docs-page'>
      <main className='public-main docs-main'>
        <PageHero
          eyebrow='NAN Docs'
          title='平台使用文档'
          description='新手先看平台教程和配置速查；遇到错误直接查常见报错排查。'
          actions={
            <a
              className='public-secondary-action'
              href='https://docs.newapi.pro/zh/docs'
              target='_blank'
              rel='noreferrer'
            >
              上游文档
              <ExternalLink size={15} />
            </a>
          }
        />

        <div className='docs-layout'>
          <aside className='docs-sidebar'>
            {DOC_GROUPS.map((group) => (
              <div className='docs-nav-group' key={group}>
                <div className='docs-nav-title'>{group}</div>
                {DOCS.filter((doc) => doc.group === group).map((doc) => (
                  <a
                    key={doc.key}
                    className={doc.key === activeDoc.key ? 'active' : ''}
                    href={doc.path}
                  >
                    <BookOpen size={15} />
                    <span>{doc.title}</span>
                  </a>
                ))}
              </div>
            ))}
          </aside>
          <section className='docs-content'>
            <header className='docs-content-head'>
              <div>
                <h2>{activeDoc.title}</h2>
                <p>{activeDoc.description}</p>
              </div>
            </header>
            <MarkdownArticle content={content} />
          </section>
        </div>
      </main>
    </PublicFrame>
  );
}

function base64UrlToBuffer(base64url) {
  if (!base64url) return new ArrayBuffer(0);
  const padding = '='.repeat((4 - (base64url.length % 4)) % 4);
  const base64 = (base64url + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  const buffer = new ArrayBuffer(rawData.length);
  const array = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; i += 1) {
    array[i] = rawData.charCodeAt(i);
  }
  return buffer;
}

function bufferToBase64Url(buffer) {
  if (!buffer) return '';
  const array = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < array.byteLength; i += 1) {
    binary += String.fromCharCode(array[i]);
  }
  return window
    .btoa(binary)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/g, '');
}

function prepareCredentialRequestOptions(payload) {
  const options =
    payload?.publicKey ||
    payload?.PublicKey ||
    payload?.response ||
    payload?.Response;
  if (!options) throw new Error('无法解析 Passkey 登录参数');
  const publicKey = {
    ...options,
    challenge: base64UrlToBuffer(options.challenge),
  };
  if (Array.isArray(options.allowCredentials)) {
    publicKey.allowCredentials = options.allowCredentials.map((item) => ({
      ...item,
      id: base64UrlToBuffer(item.id),
    }));
  }
  return publicKey;
}

function buildAssertionResult(assertion) {
  if (!assertion) return null;
  const { response } = assertion;
  return {
    id: assertion.id,
    rawId: bufferToBase64Url(assertion.rawId),
    type: assertion.type,
    authenticatorAttachment: assertion.authenticatorAttachment,
    response: {
      authenticatorData: bufferToBase64Url(response.authenticatorData),
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
      signature: bufferToBase64Url(response.signature),
      userHandle: response.userHandle
        ? bufferToBase64Url(response.userHandle)
        : null,
    },
    clientExtensionResults: assertion.getClientExtensionResults?.() ?? {},
  };
}

async function getOAuthState() {
  let path = '/api/oauth/state';
  const affCode = localStorage.getItem('aff');
  if (affCode) path += `?aff=${encodeURIComponent(affCode)}`;
  const res = await axios.get(path);
  if (res.data?.success) return res.data.data;
  throw new Error(res.data?.message || '无法获取 OAuth 状态');
}

async function startOAuth(provider) {
  try {
    await axios.get('/api/user/logout');
  } catch {
    // Logout is best-effort before OAuth.
  }
  localStorage.removeItem('user');
  const state = await getOAuthState();
  if (!state) return;

  if (provider.type === 'github') {
    window.location.href = `https://github.com/login/oauth/authorize?client_id=${provider.clientId}&state=${state}&scope=user:email`;
    return;
  }
  if (provider.type === 'discord') {
    const redirectUri = `${window.location.origin}/oauth/discord`;
    window.location.href = `https://discord.com/oauth2/authorize?client_id=${provider.clientId}&redirect_uri=${encodeURIComponent(redirectUri)}&response_type=code&scope=identify+openid&state=${state}`;
    return;
  }
  if (provider.type === 'linuxdo') {
    window.location.href = `https://connect.linux.do/oauth2/authorize?response_type=code&client_id=${provider.clientId}&state=${state}`;
    return;
  }
  if (provider.type === 'oidc') {
    const url = new URL(provider.authUrl);
    url.searchParams.set('client_id', provider.clientId);
    url.searchParams.set(
      'redirect_uri',
      `${window.location.origin}/oauth/oidc`,
    );
    url.searchParams.set('response_type', 'code');
    url.searchParams.set('scope', 'openid profile email');
    url.searchParams.set('state', state);
    window.location.href = url.toString();
    return;
  }
  if (provider.type === 'custom') {
    const url = new URL(provider.authUrl);
    url.searchParams.set('client_id', provider.clientId);
    url.searchParams.set(
      'redirect_uri',
      `${window.location.origin}/oauth/${provider.slug}`,
    );
    url.searchParams.set('response_type', 'code');
    url.searchParams.set('scope', provider.scopes || 'openid profile email');
    url.searchParams.set('state', state);
    window.location.href = url.toString();
  }
}

function TurnstileBox({ siteKey, onToken, resetKey }) {
  const containerRef = useRef(null);
  const widgetRef = useRef(null);

  useEffect(() => {
    if (!siteKey) return undefined;
    let cancelled = false;
    const renderWidget = () => {
      if (
        cancelled ||
        !containerRef.current ||
        !window.turnstile ||
        widgetRef.current
      ) {
        return;
      }
      widgetRef.current = window.turnstile.render(containerRef.current, {
        sitekey: siteKey,
        callback: (token) => onToken(token || ''),
        'expired-callback': () => onToken(''),
        'error-callback': () => onToken(''),
      });
    };

    if (window.turnstile) {
      renderWidget();
    } else {
      const existing = document.querySelector('script[data-turnstile]');
      const script = existing || document.createElement('script');
      if (!existing) {
        script.src =
          'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
        script.async = true;
        script.defer = true;
        script.dataset.turnstile = 'true';
        document.head.appendChild(script);
      }
      script.addEventListener('load', renderWidget);
      return () => {
        cancelled = true;
        script.removeEventListener('load', renderWidget);
      };
    }
    return () => {
      cancelled = true;
    };
  }, [onToken, siteKey]);

  useEffect(() => {
    if (window.turnstile && widgetRef.current) {
      window.turnstile.reset(widgetRef.current);
      onToken('');
    }
  }, [onToken, resetKey]);

  return <div className='turnstile-box' ref={containerRef} />;
}

function AuthShell({ status, user, setUser, mode }) {
  const isRegister = mode === 'register';
  const [form, setForm] = useState({
    username: '',
    password: '',
    password2: '',
    email: '',
    verification_code: '',
    twoFactorCode: '',
  });
  const [agreed, setAgreed] = useState(false);
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState('info');
  const [loading, setLoading] = useState(false);
  const [verificationLoading, setVerificationLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [turnstileToken, setTurnstileToken] = useState('');
  const [turnstileReset, setTurnstileReset] = useState(0);
  const [twoFactor, setTwoFactor] = useState(false);
  const [oauthLoading, setOauthLoading] = useState('');

  useEffect(() => {
    const affCode = new URLSearchParams(window.location.search).get('aff');
    if (affCode) localStorage.setItem('aff', affCode);
  }, []);

  useEffect(() => {
    if (countdown <= 0) return undefined;
    const timer = window.setTimeout(
      () => setCountdown((prev) => prev - 1),
      1000,
    );
    return () => window.clearTimeout(timer);
  }, [countdown]);

  useEffect(() => {
    if (user) window.location.replace(getSafeLoginRedirectPath());
  }, [user]);

  const legalRequired =
    status.user_agreement_enabled || status.privacy_policy_enabled;
  const turnstileEnabled = Boolean(status.turnstile_check);
  const showRegister = !status.self_use_mode_enabled;

  const oauthProviders = useMemo(() => {
    const providers = [];
    if (status.github_oauth) {
      providers.push({
        key: 'github',
        type: 'github',
        label: 'GitHub',
        clientId: status.github_client_id,
      });
    }
    if (status.discord_oauth) {
      providers.push({
        key: 'discord',
        type: 'discord',
        label: 'Discord',
        clientId: status.discord_client_id,
      });
    }
    if (status.oidc_enabled) {
      providers.push({
        key: 'oidc',
        type: 'oidc',
        label: 'OIDC',
        clientId: status.oidc_client_id,
        authUrl: status.oidc_authorization_endpoint,
      });
    }
    if (status.linuxdo_oauth) {
      providers.push({
        key: 'linuxdo',
        type: 'linuxdo',
        label: 'Linux DO',
        clientId: status.linuxdo_client_id,
      });
    }
    (status.custom_oauth_providers || []).forEach((provider) => {
      providers.push({
        key: provider.slug,
        type: 'custom',
        label: provider.name,
        slug: provider.slug,
        clientId: provider.client_id,
        authUrl: provider.authorization_endpoint,
        scopes: provider.scopes,
      });
    });
    return providers.filter(
      (provider) => provider.clientId || provider.authUrl,
    );
  }, [status]);

  const setField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setMessage('');
  };

  const showMessage = (type, text) => {
    setMessageType(type);
    setMessage(text);
  };

  const storeUserAndRedirect = (data) => {
    writeJson('user', data);
    setUser(data);
    window.location.href = getSafeLoginRedirectPath();
  };

  const resetTurnstile = () => {
    setTurnstileToken('');
    setTurnstileReset((prev) => prev + 1);
  };

  const ensureCanSubmit = () => {
    if (legalRequired && !agreed) {
      showMessage('error', '请先阅读并同意用户协议和隐私政策');
      return false;
    }
    if (turnstileEnabled && !turnstileToken) {
      showMessage('error', '请稍后几秒重试，Turnstile 正在检查用户环境');
      return false;
    }
    return true;
  };

  const handleLogin = async (event) => {
    event.preventDefault();
    if (!ensureCanSubmit()) return;
    if (!form.username || !form.password) {
      showMessage('error', '请输入用户名和密码');
      return;
    }
    setLoading(true);
    try {
      const res = await axios.post(
        `/api/user/login?turnstile=${encodeURIComponent(turnstileToken)}`,
        { username: form.username, password: form.password },
      );
      if (res.data?.success) {
        if (res.data.data?.require_2fa) {
          setTwoFactor(true);
          showMessage('info', '请输入二步验证验证码');
        } else {
          storeUserAndRedirect(res.data.data);
        }
      } else {
        showMessage('error', res.data?.message || '登录失败');
      }
    } catch {
      showMessage('error', '登录失败，请重试');
    } finally {
      setLoading(false);
      if (turnstileEnabled) resetTurnstile();
    }
  };

  const handleTwoFactor = async (event) => {
    event.preventDefault();
    if (!form.twoFactorCode) {
      showMessage('error', '请输入验证码');
      return;
    }
    setLoading(true);
    try {
      const res = await axios.post('/api/user/login/2fa', {
        code: form.twoFactorCode,
      });
      if (res.data?.success) {
        storeUserAndRedirect(res.data.data);
      } else {
        showMessage('error', res.data?.message || '验证失败');
      }
    } catch {
      showMessage('error', '验证失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  const sendVerificationCode = async () => {
    if (!form.email) {
      showMessage('error', '请先填写邮箱');
      return;
    }
    if (turnstileEnabled && !turnstileToken) {
      showMessage('error', '请先完成 Turnstile 检查');
      return;
    }
    setVerificationLoading(true);
    try {
      const res = await axios.get(
        `/api/verification?email=${encodeURIComponent(form.email)}&turnstile=${encodeURIComponent(turnstileToken)}`,
      );
      if (res.data?.success) {
        showMessage('success', '验证码发送成功，请检查邮箱');
        setCountdown(30);
      } else {
        showMessage('error', res.data?.message || '发送验证码失败');
      }
    } catch {
      showMessage('error', '发送验证码失败，请重试');
    } finally {
      setVerificationLoading(false);
      if (turnstileEnabled) resetTurnstile();
    }
  };

  const handleRegister = async (event) => {
    event.preventDefault();
    if (!showRegister) {
      showMessage('error', '当前站点未开放注册');
      return;
    }
    if (!ensureCanSubmit()) return;
    if (form.password.length < 8) {
      showMessage('error', '密码长度不得小于 8 位');
      return;
    }
    if (form.password !== form.password2) {
      showMessage('error', '两次输入的密码不一致');
      return;
    }
    setLoading(true);
    try {
      const affCode =
        new URLSearchParams(window.location.search).get('aff') ||
        localStorage.getItem('aff') ||
        '';
      const payload = {
        username: form.username,
        password: form.password,
        email: form.email,
        verification_code: form.verification_code,
        aff_code: affCode,
      };
      const res = await axios.post(
        `/api/user/register?turnstile=${encodeURIComponent(turnstileToken)}`,
        payload,
      );
      if (res.data?.success) {
        localStorage.removeItem('aff');
        showMessage('success', '注册成功，正在前往登录');
        window.setTimeout(() => {
          window.location.href = '/login';
        }, 650);
      } else {
        showMessage('error', res.data?.message || '注册失败');
      }
    } catch {
      showMessage('error', '注册失败，请重试');
    } finally {
      setLoading(false);
      if (turnstileEnabled) resetTurnstile();
    }
  };

  const handlePasskeyLogin = async () => {
    if (legalRequired && !agreed) {
      showMessage('error', '请先阅读并同意用户协议和隐私政策');
      return;
    }
    if (!window.PublicKeyCredential) {
      showMessage('error', '当前浏览器不支持 Passkey');
      return;
    }
    setLoading(true);
    try {
      const beginRes = await axios.post('/api/user/passkey/login/begin');
      if (!beginRes.data?.success) {
        showMessage('error', beginRes.data?.message || '无法发起 Passkey 登录');
        return;
      }
      const assertion = await navigator.credentials.get({
        publicKey: prepareCredentialRequestOptions(beginRes.data.data),
      });
      const payload = buildAssertionResult(assertion);
      const finishRes = await axios.post(
        '/api/user/passkey/login/finish',
        payload,
      );
      if (finishRes.data?.success) {
        storeUserAndRedirect(finishRes.data.data);
      } else {
        showMessage('error', finishRes.data?.message || 'Passkey 登录失败');
      }
    } catch (error) {
      showMessage(
        'error',
        error?.name === 'AbortError'
          ? '已取消 Passkey 登录'
          : 'Passkey 登录失败',
      );
    } finally {
      setLoading(false);
    }
  };

  const handleOAuth = async (provider) => {
    if (legalRequired && !agreed) {
      showMessage('error', '请先阅读并同意用户协议和隐私政策');
      return;
    }
    setOauthLoading(provider.key);
    try {
      await startOAuth(provider);
    } catch (error) {
      showMessage('error', error.message || `${provider.label} 登录失败`);
      setOauthLoading('');
    }
  };

  const title = isRegister ? '创建账户' : twoFactor ? '二步验证' : '登录账户';
  const subtitle = isRegister
    ? '注册后即可创建 API Key 并接入统一模型网关'
    : twoFactor
      ? '请输入认证器应用中的验证码或备用码'
      : '进入控制台管理密钥、用量和账单';

  return (
    <PublicFrame status={status} user={user} className='auth-page'>
      <main className='auth-main'>
        <section className='auth-panel'>
          <div className='auth-copy'>
            <div className='auth-brand'>
              <img src={status.logo || '/logo.svg'} alt='' />
              <span>{status.system_name || 'New API'}</span>
            </div>
            <h1>统一管理你的 AI API 调用</h1>
            <p>进入控制台管理密钥、用量、账单与安全设置。</p>
            <div className='auth-points'>
              <span>
                <ShieldCheck size={16} />
                会话安全
              </span>
              <span>
                <KeyRound size={16} />
                API Key 管理
              </span>
              <span>
                <FileText size={16} />
                用量账单
              </span>
            </div>
          </div>

          <form
            className='auth-card'
            onSubmit={
              twoFactor
                ? handleTwoFactor
                : isRegister
                  ? handleRegister
                  : handleLogin
            }
          >
            <div className='auth-card-head'>
              <h2>{title}</h2>
              <p>{subtitle}</p>
            </div>

            {message && <Message type={messageType}>{message}</Message>}

            {!twoFactor && (
              <>
                <label className='auth-field'>
                  <span>用户名</span>
                  <div>
                    <User size={17} />
                    <input
                      autoComplete='username'
                      value={form.username}
                      onChange={(event) =>
                        setField('username', event.target.value)
                      }
                      placeholder='请输入用户名'
                    />
                  </div>
                </label>

                {isRegister && (
                  <label className='auth-field'>
                    <span>邮箱</span>
                    <div>
                      <Mail size={17} />
                      <input
                        type='email'
                        autoComplete='email'
                        value={form.email}
                        onChange={(event) =>
                          setField('email', event.target.value)
                        }
                        placeholder='用于接收验证码'
                      />
                    </div>
                  </label>
                )}

                <label className='auth-field'>
                  <span>密码</span>
                  <div>
                    <Lock size={17} />
                    <input
                      type='password'
                      autoComplete={
                        isRegister ? 'new-password' : 'current-password'
                      }
                      value={form.password}
                      onChange={(event) =>
                        setField('password', event.target.value)
                      }
                      placeholder='请输入密码'
                    />
                  </div>
                </label>

                {isRegister && (
                  <>
                    <label className='auth-field'>
                      <span>确认密码</span>
                      <div>
                        <Lock size={17} />
                        <input
                          type='password'
                          autoComplete='new-password'
                          value={form.password2}
                          onChange={(event) =>
                            setField('password2', event.target.value)
                          }
                          placeholder='再次输入密码'
                        />
                      </div>
                    </label>

                    {status.email_verification && (
                      <label className='auth-field'>
                        <span>邮箱验证码</span>
                        <div className='auth-code-row'>
                          <input
                            value={form.verification_code}
                            onChange={(event) =>
                              setField('verification_code', event.target.value)
                            }
                            placeholder='6 位验证码'
                          />
                          <button
                            type='button'
                            onClick={sendVerificationCode}
                            disabled={verificationLoading || countdown > 0}
                          >
                            {verificationLoading
                              ? '发送中'
                              : countdown > 0
                                ? `${countdown}s`
                                : '发送'}
                          </button>
                        </div>
                      </label>
                    )}
                  </>
                )}
              </>
            )}

            {twoFactor && (
              <label className='auth-field'>
                <span>验证码</span>
                <div>
                  <ShieldCheck size={17} />
                  <input
                    autoFocus
                    value={form.twoFactorCode}
                    onChange={(event) =>
                      setField('twoFactorCode', event.target.value)
                    }
                    placeholder='6 位验证码或 8 位备用码'
                  />
                </div>
              </label>
            )}

            {!twoFactor && legalRequired && (
              <label className='auth-check'>
                <input
                  type='checkbox'
                  checked={agreed}
                  onChange={(event) => setAgreed(event.target.checked)}
                />
                <span>
                  我已阅读并同意
                  {status.user_agreement_enabled && (
                    <a href='/user-agreement'> 用户协议</a>
                  )}
                  {status.privacy_policy_enabled && (
                    <a href='/privacy-policy'> 隐私政策</a>
                  )}
                </span>
              </label>
            )}

            {!twoFactor && turnstileEnabled && (
              <TurnstileBox
                siteKey={status.turnstile_site_key}
                resetKey={turnstileReset}
                onToken={setTurnstileToken}
              />
            )}

            <button className='auth-submit' type='submit' disabled={loading}>
              {loading && <Loader2 size={16} className='spin' />}
              {twoFactor ? '验证并登录' : isRegister ? '注册' : '登录'}
            </button>

            {twoFactor ? (
              <button
                className='auth-link-button'
                type='button'
                onClick={() => {
                  setTwoFactor(false);
                  setField('twoFactorCode', '');
                }}
              >
                返回登录
              </button>
            ) : (
              <>
                {!isRegister && status.passkey_login && (
                  <button
                    className='auth-outline-button'
                    type='button'
                    onClick={handlePasskeyLogin}
                    disabled={loading}
                  >
                    <KeyRound size={16} />
                    使用 Passkey 登录
                  </button>
                )}

                {oauthProviders.length > 0 && (
                  <div className='auth-oauth'>
                    <div className='auth-divider'>其他方式</div>
                    <div className='auth-oauth-grid'>
                      {oauthProviders.map((provider) => (
                        <button
                          type='button'
                          key={provider.key}
                          onClick={() => handleOAuth(provider)}
                          disabled={Boolean(oauthLoading)}
                        >
                          {provider.type === 'github' ? (
                            <Github size={16} />
                          ) : (
                            <ExternalLink size={16} />
                          )}
                          {oauthLoading === provider.key
                            ? '跳转中'
                            : provider.label}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                <div className='auth-switch'>
                  {isRegister ? (
                    <>
                      已有账户？ <a href='/login'>去登录</a>
                    </>
                  ) : (
                    <>
                      <a href='/reset'>忘记密码</a>
                      {!status.self_use_mode_enabled && (
                        <>
                          <span />
                          没有账户？ <a href='/register'>注册</a>
                        </>
                      )}
                    </>
                  )}
                </div>
              </>
            )}
          </form>
        </section>
      </main>
    </PublicFrame>
  );
}

export default function PublicPages({
  page,
  status: rawStatus,
  user,
  setUser,
}) {
  const status = { ...defaultStatus, ...rawStatus };

  if (page === 'pricing') return <PricingPage status={status} user={user} />;
  if (page === 'about') return <AboutPage status={status} user={user} />;
  if (page === 'docs') return <DocsPage status={status} user={user} />;
  if (page === 'login' || page === 'register') {
    return (
      <AuthShell status={status} user={user} setUser={setUser} mode={page} />
    );
  }
  return null;
}
