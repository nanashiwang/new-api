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

export const normalizeChannelGroups = (groups) => {
  const values = Array.isArray(groups)
    ? groups
    : typeof groups === 'string'
      ? groups.split(',')
      : [];

  return Array.from(
    new Set(
      values
        .map((group) => (typeof group === 'string' ? group.trim() : ''))
        .filter(Boolean),
    ),
  );
};

export const mergeChannelFormValues = (formValues = {}, inputs = {}) => ({
  ...formValues,
  ...inputs,
  groups: normalizeChannelGroups(inputs.groups),
});

export const parseChannelJsonObject = (rawValue) => {
  if (!rawValue) return {};
  try {
    const parsed =
      typeof rawValue === 'string' ? JSON.parse(rawValue) : rawValue;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {};
    }
    return { ...parsed };
  } catch {
    return {};
  }
};

export const mergeChannelJsonObject = (rawValue, updates = {}) => ({
  ...parseChannelJsonObject(rawValue),
  ...updates,
});
