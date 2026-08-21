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

const WEEKDAY_VALUES = {
  Sun: 0,
  Mon: 1,
  Tue: 2,
  Wed: 3,
  Thu: 4,
  Fri: 5,
  Sat: 6,
};

function getZonedTimeParts(evaluationAt, timezone) {
  const requestedTimezone = String(timezone || '').trim() || 'UTC';
  const options = {
    timeZone: requestedTimezone,
    hour: 'numeric',
    minute: 'numeric',
    weekday: 'short',
    month: 'numeric',
    day: 'numeric',
    hourCycle: 'h23',
  };

  let formatter;
  try {
    formatter = new Intl.DateTimeFormat('en-US-u-ca-gregory-nu-latn', options);
  } catch {
    formatter = new Intl.DateTimeFormat('en-US-u-ca-gregory-nu-latn', {
      ...options,
      timeZone: 'UTC',
    });
  }

  const values = {};
  for (const part of formatter.formatToParts(evaluationAt)) {
    if (part.type !== 'literal') values[part.type] = part.value;
  }

  return {
    hour: Number(values.hour),
    minute: Number(values.minute),
    weekday: WEEKDAY_VALUES[values.weekday] ?? 0,
    month: Number(values.month),
    day: Number(values.day),
  };
}

export function evalBillingExprLocally(
  exprStr,
  p,
  c,
  extraTokenValues,
  evaluationAt = new Date(),
) {
  try {
    let matchedTier = '';
    const tierFn = (name, value) => {
      matchedTier = name;
      return value;
    };
    const cacheReadTokens = extraTokenValues.cacheReadTokens || 0;
    const cacheCreateTokens = extraTokenValues.cacheCreateTokens || 0;
    const cacheCreate1hTokens = extraTokenValues.cacheCreate1hTokens || 0;
    const len = p + cacheReadTokens + cacheCreateTokens + cacheCreate1hTokens;
    const at =
      evaluationAt instanceof Date ? evaluationAt : new Date(evaluationAt);
    const timeValue = (timezone, field) =>
      getZonedTimeParts(at, timezone)[field];

    const env = {
      p,
      c,
      len,
      nil: null,
      tier: tierFn,
      max: Math.max,
      min: Math.min,
      abs: Math.abs,
      ceil: Math.ceil,
      floor: Math.floor,
      header: () => '',
      param: () => null,
      has: (source, substring) =>
        source != null &&
        substring !== '' &&
        String(source).includes(String(substring)),
      hour: (timezone) => timeValue(timezone, 'hour'),
      minute: (timezone) => timeValue(timezone, 'minute'),
      weekday: (timezone) => timeValue(timezone, 'weekday'),
      month: (timezone) => timeValue(timezone, 'month'),
      day: (timezone) => timeValue(timezone, 'day'),
    };

    for (const [name, value] of Object.entries(extraTokenValues)) {
      env[name] = value || 0;
    }

    const fn = new Function(
      ...Object.keys(env),
      `"use strict"; return (${exprStr});`,
    );
    return { cost: fn(...Object.values(env)), matchedTier, error: null };
  } catch (e) {
    return { cost: 0, matchedTier: '', error: e.message };
  }
}
