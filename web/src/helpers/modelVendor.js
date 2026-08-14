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

const MIMO_MODEL_PATTERN = /(^|[/:._-])mimo($|[/:._-])/i;
const XIAOMI_MODEL_PATTERN = /(^|[/:._-])xiaomi($|[/:._-])/i;

// MiMo models are commonly published as mimo-* or under a xiaomi/* namespace.
// Segment matching avoids classifying unrelated names such as "mimosa".
export const isMiMoModel = (modelName) => {
  const normalizedName = String(modelName ?? '').trim();
  return (
    MIMO_MODEL_PATTERN.test(normalizedName) ||
    XIAOMI_MODEL_PATTERN.test(normalizedName)
  );
};
