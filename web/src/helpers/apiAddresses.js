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
const trim = (value) => (typeof value === 'string' ? value.trim() : '');
const httpAddress = (value) => {
  const text = trim(value);
  try {
    const url = new URL(text);
    return ['http:', 'https:'].includes(url.protocol) &&
      !url.username &&
      !url.password
      ? text
      : '';
  } catch {
    return '';
  }
};

export function getApiAddresses(status, origin = '') {
  if (!status) return [];
  const configured =
    status.api_info_enabled !== false && Array.isArray(status.api_info)
      ? status.api_info
      : [];
  const seen = new Set();
  const addresses = configured.flatMap((item) => {
    const url = httpAddress(item?.url);
    if (!url || seen.has(url)) return [];
    seen.add(url);
    return [
      {
        url,
        label: trim(item.route) || url,
        description: trim(item.description),
      },
    ];
  });
  if (addresses.length) return addresses;
  const server = httpAddress(status.server_address);
  const url = server || httpAddress(origin);
  return url
    ? [
        {
          url,
          labelKey: server ? '默认 API 地址' : '当前域名',
          description: '',
        },
      ]
    : [];
}
