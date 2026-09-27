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
const tokens = (value) =>
  value.match(/\s+|[A-Za-z_]\w*|\d+(?:\.\d+)?|./gu) || [];

export function expressionDiff(value, baseline) {
  const text = String(value ?? '');
  const parts = tokens(text);
  if (typeof baseline !== 'string' || baseline === text) {
    return [{ text, changed: false }];
  }
  const previous = tokens(baseline);
  if (parts.length === previous.length) {
    return parts.map((part, i) => ({
      text: part,
      changed: part !== previous[i],
    }));
  }
  // Linear bounded work, even for very long expressions. Preserve shared ends.
  let start = 0;
  while (
    start < parts.length &&
    start < previous.length &&
    parts[start] === previous[start]
  )
    start++;
  let end = parts.length,
    oldEnd = previous.length;
  while (
    end > start &&
    oldEnd > start &&
    parts[end - 1] === previous[oldEnd - 1]
  ) {
    end--;
    oldEnd--;
  }
  return parts.map((part, i) => ({
    text: part,
    changed: end === start || (i >= start && i < end),
  }));
}
