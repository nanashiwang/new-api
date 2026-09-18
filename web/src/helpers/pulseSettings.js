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

export function buildPulseSettingsUpdate(
  config,
  savedConfig,
  secrets,
  clearSecrets,
) {
  const changes = {};
  for (const [key, value] of Object.entries(config)) {
    const normalized = String(value ?? '').trim();
    if (normalized !== String(savedConfig[key] ?? '').trim()) {
      changes[key] = normalized;
    }
  }
  for (const [key, value] of Object.entries(secrets)) {
    if (value.trim() && !clearSecrets.includes(key)) {
      changes[key] = value.trim();
    }
  }
  return { config: changes, clear_secrets: clearSecrets };
}

export function pulseQuotaToUnits(value, quotaPerUnit) {
  if (
    !/^\d+$/.test(String(value)) ||
    !Number.isSafeInteger(Number(value)) ||
    Number(value) <= 0 ||
    !Number.isFinite(quotaPerUnit) ||
    quotaPerUnit <= 0
  ) {
    return null;
  }
  return Number(value) / quotaPerUnit;
}
