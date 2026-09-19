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

// Provider-declared names are diagnostics, never proof of the underlying model.
const normalize = (value) =>
  typeof value === 'string' ? value.trim().toLowerCase() : '';

// Mirror relay/common/response_model.go; both consume the same test fixtures.
function matchesModel(returned, expected) {
  if (!expected) return false;
  let candidate = returned;
  if (!expected.includes('/') && candidate.includes('/')) {
    const parts = candidate.split('/');
    if (parts.some((part) => !part.trim() || part === '.' || part === '..'))
      return false;
    candidate = parts.at(-1);
  }
  if (candidate === expected) return true;
  if (!candidate.startsWith(`${expected}-`)) return false;
  const suffix = candidate.slice(expected.length + 1);
  // Only calendar snapshots, not arbitrary model variants, are compatible.
  const date = suffix.match(/^(\d{4})-?(\d{2})-?(\d{2})$/);
  if (!date || (suffix.length !== 8 && suffix.length !== 10)) return false;
  const iso = `${date[1]}-${date[2]}-${date[3]}T00:00:00.000Z`;
  const parsed = new Date(iso);
  return !Number.isNaN(parsed.getTime()) && parsed.toISOString() === iso;
}

export function isResponseModelMismatch(observation) {
  const returned = normalize(observation?.returned_model);
  if (!returned) return false;
  return ![observation?.requested_model, observation?.upstream_model].some(
    (name) => matchesModel(returned, normalize(name)),
  );
}

export function getResponseModelInfo(other, requestedModel) {
  const observation = other?.response_model;
  if (
    !observation ||
    typeof observation.returned_model !== 'string' ||
    !observation.returned_model.trim()
  )
    return null;
  const text = (value, fallback) =>
    typeof value === 'string' && value.trim() ? value : fallback;
  const requested = text(observation.requested_model, requestedModel);
  const upstream = text(
    observation.upstream_model,
    text(other?.upstream_model_name, requested),
  );
  return {
    requested,
    upstream,
    returned: observation.returned_model,
    mismatch: isResponseModelMismatch({
      requested_model: requested,
      upstream_model: upstream,
      returned_model: observation.returned_model,
    }),
  };
}
