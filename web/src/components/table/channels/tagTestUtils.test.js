import { describe, expect, test } from 'bun:test';

import {
  buildTagTestSummary,
  collectTagTestModels,
  resolveTagTestTargets,
  shouldPromptEnableChannelAfterManualTest,
} from './tagTestUtils.js';

describe('tagTestUtils', () => {
  test('collectTagTestModels deduplicates and preserves order', () => {
    expect(
      collectTagTestModels([
        { models: 'gpt-4o, claude-3-5-sonnet' },
        { models: 'claude-3-5-sonnet, gpt-4o-mini , ' },
        { models: '' },
        null,
      ]),
    ).toEqual(['gpt-4o', 'claude-3-5-sonnet', 'gpt-4o-mini']);
  });

  test('shouldPromptEnableChannelAfterManualTest matches disabled and temporarily unavailable channels', () => {
    expect(
      shouldPromptEnableChannelAfterManualTest({
        status: 2,
        effective_available: false,
      }),
    ).toBe(true);
    expect(
      shouldPromptEnableChannelAfterManualTest({
        status: 3,
        effective_available: false,
      }),
    ).toBe(true);
    expect(
      shouldPromptEnableChannelAfterManualTest({
        status: 1,
        effective_available: false,
      }),
    ).toBe(true);
    expect(
      shouldPromptEnableChannelAfterManualTest({
        status: 1,
        effective_available: true,
      }),
    ).toBe(false);
  });

  test('buildTagTestSummary returns compact tone-aware summaries', () => {
    expect(buildTagTestSummary('vip', 5, 5)).toEqual({
      message: 'vip 5/5',
      tone: 'success',
      successCount: 5,
      totalCount: 5,
    });
    expect(buildTagTestSummary('vip', 0, 5)).toEqual({
      message: 'vip 0/5',
      tone: 'error',
      successCount: 0,
      totalCount: 5,
    });
    expect(buildTagTestSummary('vip', 2, 5)).toEqual({
      message: 'vip 2/5',
      tone: 'info',
      successCount: 2,
      totalCount: 5,
    });
  });

  test('resolveTagTestTargets filters only selected channels for specified scope', () => {
    const channels = [
      { id: 1, name: 'alpha' },
      { id: 2, name: 'beta' },
      { id: 3, name: 'gamma' },
    ];

    expect(resolveTagTestTargets(channels, 'all', [])).toEqual(channels);
    expect(resolveTagTestTargets(channels, 'specified', ['2', 3])).toEqual([
      { id: 2, name: 'beta' },
      { id: 3, name: 'gamma' },
    ]);
  });
});
