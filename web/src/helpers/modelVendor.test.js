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

import { describe, expect, test } from 'bun:test';
import { isMiMoModel } from './modelVendor';

describe('isMiMoModel', () => {
  test.each([
    'mimo-v2-flash',
    'MiMo-VL-7B',
    'xiaomi/mimo-v2-flash',
    'vendor.xiaomi-mimo',
  ])('recognizes %s as MiMo', (modelName) => {
    expect(isMiMoModel(modelName)).toBe(true);
  });

  test.each(['mimosa', 'notmimo-model', 'qwen-tts', 'gpt-5.5', '', null])(
    'does not misclassify %s as MiMo',
    (modelName) => {
      expect(isMiMoModel(modelName)).toBe(false);
    },
  );
});
