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

import { beforeAll, describe, expect, mock, test } from 'bun:test';

mock.module('../../../../helpers', () => ({
  API: {},
  showError: () => {},
  showSuccess: () => {},
}));

let buildModelState;
let buildPreviewRows;
let buildSummaryText;
let collectModelNames;
let isBasePricingUnset;
let serializeModel;

beforeAll(async () => {
  ({
    buildModelState,
    buildPreviewRows,
    buildSummaryText,
    collectModelNames,
    isBasePricingUnset,
    serializeModel,
  } = await import('./useModelPricingEditorState'));
});

const modelName = 'mimo-v2.5-asr';
const t = (key) => key;

const emptySourceMaps = () => ({
  ModelPrice: {},
  ModelRatio: {},
  CompletionRatio: {},
  CompletionRatioMeta: {},
  CacheRatio: {},
  CreateCacheRatio: {},
  ImageRatio: {},
  AudioRatio: {},
  AudioCompletionRatio: {},
  AudioDurationPrice: {},
  ModelBillingMode: {},
  ModelBillingExpr: {},
});

describe('audio duration pricing visual editor state', () => {
  test('includes models configured only through the hourly price map', () => {
    const sourceMaps = emptySourceMaps();
    sourceMaps.AudioDurationPrice[modelName] = 0.074;

    expect(collectModelNames([], sourceMaps)).toContain(modelName);
  });

  test('loads hourly audio pricing with the same priority as the backend', () => {
    const sourceMaps = emptySourceMaps();
    sourceMaps.AudioDurationPrice[modelName] = 0.074;
    sourceMaps.ModelPrice[modelName] = 1;
    sourceMaps.ModelRatio[modelName] = 2.5;

    const model = buildModelState(modelName, sourceMaps);

    expect(model.billingMode).toBe('per-audio-hour');
    expect(model.audioDurationPrice).toBe('0.074');
    expect(model.hasConflict).toBe(true);
    expect(isBasePricingUnset(model)).toBe(false);
  });

  test('keeps tiered expression billing ahead of an accidental hourly price', () => {
    const sourceMaps = emptySourceMaps();
    sourceMaps.AudioDurationPrice[modelName] = 0.074;
    sourceMaps.ModelBillingMode[modelName] = 'tiered_expr';
    sourceMaps.ModelBillingExpr[modelName] = 'tier("base", p * 5)';

    const model = buildModelState(modelName, sourceMaps);

    expect(model.billingMode).toBe('tiered_expr');
    expect(model.billingExpr).toBe('tier("base", p * 5)');
    expect(serializeModel(model, t).AudioDurationPrice).toBeNull();
  });

  test('serializes hourly audio pricing without retaining conflicting modes', () => {
    const sourceMaps = emptySourceMaps();
    sourceMaps.AudioDurationPrice[modelName] = 0.074;
    sourceMaps.ModelPrice[modelName] = 1;
    sourceMaps.ModelRatio[modelName] = 2.5;

    const serialized = serializeModel(
      buildModelState(modelName, sourceMaps),
      t,
    );

    expect(serialized.AudioDurationPrice).toBe(0.074);
    expect(serialized.ModelPrice).toBeNull();
    expect(serialized.ModelRatio).toBeNull();
    expect(serialized.AudioRatio).toBeNull();
  });

  test('switching back to token pricing removes the hourly price', () => {
    const sourceMaps = emptySourceMaps();
    sourceMaps.AudioDurationPrice[modelName] = 0.074;
    const model = {
      ...buildModelState(modelName, sourceMaps),
      billingMode: 'per-token',
      inputPrice: '5',
    };

    const serialized = serializeModel(model, t);

    expect(serialized.AudioDurationPrice).toBeNull();
    expect(serialized.ModelRatio).toBe(2.5);
  });

  test('shows the hourly unit in list summary and save preview', () => {
    const sourceMaps = emptySourceMaps();
    sourceMaps.AudioDurationPrice[modelName] = 0.074;
    const model = buildModelState(modelName, sourceMaps);

    expect(buildSummaryText(model, t)).toBe('音频输入价格 $0.074 / 小时');
    expect(buildPreviewRows(model, t)).toEqual([
      {
        key: 'AudioDurationPrice',
        label: 'AudioDurationPrice',
        value: '0.074 USD / 小时',
      },
    ]);
  });

  test('treats an empty hourly price as unset', () => {
    const sourceMaps = emptySourceMaps();
    const model = {
      ...buildModelState(modelName, sourceMaps),
      billingMode: 'per-audio-hour',
    };

    expect(isBasePricingUnset(model)).toBe(true);
  });
});
