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

const endpointFields = {
  authorization_endpoint: 'oidc.authorization_endpoint',
  token_endpoint: 'oidc.token_endpoint',
  userinfo_endpoint: 'oidc.user_info_endpoint',
};

// OIDC's enable switch is saved separately by the existing settings form.
export async function prepareOIDCSettingsUpdate(inputs, baseline, discover) {
  const values = { ...inputs };
  const wellKnown = (values['oidc.well_known'] || '').trim();
  values['oidc.well_known'] = wellKnown;
  const changed = wellKnown !== (baseline['oidc.well_known'] || '').trim();
  const missingEndpoint = Object.values(endpointFields).some(
    (key) => !values[key]?.trim(),
  );
  const discovered = Boolean(wellKnown && (changed || missingEndpoint));

  if (discovered) {
    if (!/^https?:\/\//.test(wellKnown)) {
      throw new Error('Well-Known URL 必须以 http:// 或 https:// 开头');
    }
    try {
      const metadata = await discover(wellKnown);
      for (const [field, key] of Object.entries(endpointFields)) {
        if (typeof metadata?.[field] !== 'string' || !metadata[field].trim()) {
          throw new Error('Incomplete OIDC discovery metadata');
        }
        values[key] = metadata[field].trim();
      }
    } catch {
      throw new Error(
        '获取 OIDC 配置失败，请检查网络状况和 Well-Known URL 是否正确',
      );
    }
  }

  const fields = [
    'oidc.well_known',
    'oidc.client_id',
    'oidc.client_secret',
    ...Object.values(endpointFields),
  ];
  const options = fields
    .filter((key) => {
      if (key === 'oidc.client_secret' && !values[key]) return false;
      return (baseline[key] ?? '') !== (values[key] ?? '');
    })
    .map((key) => ({ key, value: values[key] ?? '' }));
  return { values, options, discovered };
}
