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

export const CC_SWITCH_APPS = {
  codex: {
    label: 'Codex',
    endpointTypes: ['openai-response'],
    preferredModels: ['gpt-5.6-sol'],
    appendV1: true,
  },
  claude: {
    label: 'Claude Code',
    endpointTypes: ['anthropic'],
    preferredModels: ['claude-opus-4-1', 'claude-sonnet-4-5'],
    appendV1: false,
  },
  gemini: {
    label: 'Gemini CLI',
    endpointTypes: ['gemini'],
    preferredModels: ['gemini-2.5-pro', 'gemini-2.5-flash'],
    appendV1: false,
  },
  opencode: {
    label: 'OpenCode',
    endpointTypes: ['openai', 'openai-response'],
    preferredModels: ['gpt-5.6-sol'],
    appendV1: true,
  },
  openclaw: {
    label: 'OpenClaw',
    endpointTypes: ['openai', 'openai-response'],
    preferredModels: ['gpt-5.6-sol'],
    appendV1: true,
  },
};

const NON_TEXT_MODEL_PATTERN =
  /(^|[-_/])(image|images|audio|speech|tts|whisper|ocr|embedding|rerank|video)([-_/]|$)/i;

export const normalizeBaseUrl = (value) => {
  const trimmed = String(value || '').trim();
  if (!trimmed) return '';
  try {
    const parsed = new URL(trimmed);
    if (!['http:', 'https:'].includes(parsed.protocol)) return '';
    parsed.username = '';
    parsed.password = '';
    parsed.search = '';
    parsed.hash = '';
    parsed.pathname = parsed.pathname.replace(/\/+$/, '').replace(/\/v1$/i, '');
    return parsed.toString().replace(/\/+$/, '');
  } catch (_) {
    return '';
  }
};

export const buildEndpoint = (baseUrl, app) => {
  const normalized = normalizeBaseUrl(baseUrl);
  if (!normalized) return '';
  return CC_SWITCH_APPS[app]?.appendV1 ? `${normalized}/v1` : normalized;
};

const normalizeModel = (model) => ({
  name: typeof model === 'string' ? model : String(model?.name || '').trim(),
  supported_endpoint_types: Array.isArray(model?.supported_endpoint_types)
    ? model.supported_endpoint_types
    : [],
});

export const getCompatibleModels = (models, app) => {
  const config = CC_SWITCH_APPS[app];
  if (!config) return [];
  return (Array.isArray(models) ? models : [])
    .map(normalizeModel)
    .filter((model) => model.name && !NON_TEXT_MODEL_PATTERN.test(model.name))
    .filter((model) =>
      model.supported_endpoint_types.some((type) =>
        config.endpointTypes.includes(type),
      ),
    );
};

export const chooseDefaultModel = (models, app, previousModel = '') => {
  const compatible = getCompatibleModels(models, app);
  for (const preferred of CC_SWITCH_APPS[app]?.preferredModels || []) {
    if (compatible.some((model) => model.name === preferred)) return preferred;
  }
  if (compatible.some((model) => model.name === previousModel)) {
    return previousModel;
  }
  return compatible[0]?.name || '';
};

export const buildCCSwitchDeepLink = ({
  app,
  name,
  baseUrl,
  apiKey,
  model,
  enabled = true,
}) => {
  if (!CC_SWITCH_APPS[app]) throw new Error('Unsupported CC Switch app');
  const endpoint = buildEndpoint(baseUrl, app);
  if (!endpoint || !apiKey || !model) {
    throw new Error('Incomplete CC Switch configuration');
  }
  const params = new URLSearchParams();
  params.set('resource', 'provider');
  params.set('app', app);
  params.set('name', String(name || 'new-api').trim() || 'new-api');
  params.set('endpoint', endpoint);
  params.set('model', model);
  params.set('apiKey', apiKey);
  params.set('homepage', normalizeBaseUrl(baseUrl));
  params.set('enabled', enabled ? 'true' : 'false');
  return `ccswitch://v1/import?${params.toString()}`;
};

export const getRouteOptions = (serverAddress, origin) => {
  const values = [serverAddress, origin].map(normalizeBaseUrl).filter(Boolean);
  if (values.includes('https://nan.meta-api.vip')) {
    values.unshift('https://cn.meta-api.vip');
  }
  return [...new Set(values)];
};
