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

const SSE_FIELD_PATTERN = /^(data|event|id|retry)(?::|$)/;
const HEARTBEAT_PATTERN = /^(ping|pong|heartbeat)$/i;

const isSSEEnvelope = (value) =>
  value
    .split(/\r\n|\r|\n/)
    .some((line) => line.startsWith(':') || SSE_FIELD_PATTERN.test(line));

const parseSSEEnvelope = (value) => {
  const dataLines = [];
  let eventType = 'message';

  value.split(/\r\n|\r|\n/).forEach((line) => {
    if (!line || line.startsWith(':')) return;

    const separatorIndex = line.indexOf(':');
    const field = separatorIndex === -1 ? line : line.slice(0, separatorIndex);
    let fieldValue =
      separatorIndex === -1 ? '' : line.slice(separatorIndex + 1);
    if (fieldValue.startsWith(' ')) fieldValue = fieldValue.slice(1);

    if (field === 'data') dataLines.push(fieldValue);
    if (field === 'event' && fieldValue) eventType = fieldValue;
  });

  return { data: dataLines.join('\n'), eventType };
};

const getStructuredErrorMessage = (value) => {
  if (!value || typeof value !== 'object') return '';

  if (typeof value.error?.message === 'string') {
    return value.error.message.trim();
  }
  if (typeof value.response?.error?.message === 'string') {
    return value.response.error.message.trim();
  }
  if (typeof value.message === 'string') return value.message.trim();
  if (typeof value.error === 'string') return value.error.trim();
  if (typeof value.detail === 'string') return value.detail.trim();
  return '';
};

export const splitPlaygroundSSEFrames = (value) => {
  if (typeof value !== 'string' || !isSSEEnvelope(value)) return [value];

  return value
    .split(/(?:\r\n|\r|\n){2}/)
    .filter((frame) => frame.trim() !== '');
};

export const parsePlaygroundSSEItem = (raw) => {
  if (raw !== null && typeof raw === 'object') {
    return {
      raw,
      data: raw,
      parsed: raw,
      error: null,
      serverError: getStructuredErrorMessage(raw),
      isDone: false,
      ignored: false,
      eventType: 'message',
    };
  }

  const rawText = raw == null ? '' : String(raw);
  const envelope = isSSEEnvelope(rawText)
    ? parseSSEEnvelope(rawText)
    : { data: rawText, eventType: 'message' };
  const data = envelope.data.trim();

  if (!data || HEARTBEAT_PATTERN.test(data)) {
    return {
      raw,
      data,
      parsed: null,
      error: null,
      serverError: '',
      isDone: false,
      ignored: true,
      eventType: envelope.eventType,
    };
  }

  if (data === '[DONE]') {
    return {
      raw,
      data,
      parsed: null,
      error: null,
      serverError: '',
      isDone: true,
      ignored: false,
      eventType: envelope.eventType,
    };
  }

  try {
    const parsed = JSON.parse(data);
    return {
      raw,
      data,
      parsed,
      error: null,
      serverError: getStructuredErrorMessage(parsed),
      isDone: false,
      ignored: false,
      eventType: envelope.eventType,
    };
  } catch (error) {
    return {
      raw,
      data,
      parsed: null,
      error: error.message,
      serverError: '',
      isDone: false,
      ignored: false,
      eventType: envelope.eventType,
    };
  }
};

export const parsePlaygroundSSEData = (items) => {
  if (!Array.isArray(items)) return [];

  return items.flatMap((item) =>
    splitPlaygroundSSEFrames(item)
      .map(parsePlaygroundSSEItem)
      .filter((entry) => !entry.ignored),
  );
};

export const extractPlaygroundErrorMessage = (value, fallback) => {
  if (value instanceof Error) {
    return extractPlaygroundErrorMessage(value.message, fallback);
  }

  const structuredMessage = getStructuredErrorMessage(value);
  if (structuredMessage) return structuredMessage;

  if (typeof value !== 'string') return fallback;

  const parsedItem = parsePlaygroundSSEItem(value);
  if (parsedItem.serverError) return parsedItem.serverError;
  if (parsedItem.parsed && typeof parsedItem.parsed === 'string') {
    return parsedItem.parsed.trim() || fallback;
  }
  if (parsedItem.error && parsedItem.data && !/^\s*</.test(parsedItem.data)) {
    return parsedItem.data;
  }
  return fallback;
};
