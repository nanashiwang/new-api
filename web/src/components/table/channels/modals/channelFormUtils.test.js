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

import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import {
  mergeChannelFormValues,
  mergeChannelJsonObject,
  normalizeChannelGroups,
} from './channelFormUtils.js';

describe('channelFormUtils', () => {
  test('does not assign a default group to a new channel', () => {
    assert.deepEqual(normalizeChannelGroups([]), []);
    assert.deepEqual(normalizeChannelGroups(''), []);
  });

  test('keeps loaded values when stale form defaults are submitted', () => {
    const merged = mergeChannelFormValues(
      {
        groups: ['default'],
        priority: 0,
        auto_ban: true,
        settings: '',
      },
      {
        groups: ['OpenAI · 企业专属', 'OpenAI · 优质'],
        priority: 5,
        auto_ban: false,
        settings: '{"allow_speed":true}',
      },
    );

    assert.deepEqual(merged.groups, ['OpenAI · 企业专属', 'OpenAI · 优质']);
    assert.equal(merged.priority, 5);
    assert.equal(merged.auto_ban, false);
    assert.equal(merged.settings, '{"allow_speed":true}');
  });

  test('normalizes and deduplicates explicitly selected groups', () => {
    assert.deepEqual(
      normalizeChannelGroups([
        ' OpenAI · 优质 ',
        'OpenAI · 优质',
        'Claude · 优质',
      ]),
      ['OpenAI · 优质', 'Claude · 优质'],
    );
  });

  test('preserves settings unknown to the current editor', () => {
    const merged = mergeChannelJsonObject(
      JSON.stringify({
        claude_image_transport_mode: 'bridge',
        future_setting: 'keep-me',
      }),
      { force_format: true },
    );

    assert.deepEqual(merged, {
      claude_image_transport_mode: 'bridge',
      future_setting: 'keep-me',
      force_format: true,
    });
  });

  test('preserves other settings while changing one known option', () => {
    const merged = mergeChannelJsonObject(
      JSON.stringify({
        allow_speed: true,
        upstream_model_update_last_removed_models: ['old-model'],
      }),
      { allow_service_tier: true },
    );

    assert.deepEqual(merged, {
      allow_speed: true,
      upstream_model_update_last_removed_models: ['old-model'],
      allow_service_tier: true,
    });
  });
});
