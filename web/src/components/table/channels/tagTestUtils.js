export const collectTagTestModels = (channels = []) => {
  const seen = new Set();
  const models = [];

  channels.forEach((channel) => {
    const rawModels = channel?.models || '';
    rawModels
      .split(',')
      .map((model) => model.trim())
      .filter(Boolean)
      .forEach((model) => {
        if (seen.has(model)) {
          return;
        }
        seen.add(model);
        models.push(model);
      });
  });

  return models;
};

export const shouldPromptEnableChannelAfterManualTest = (channel) => {
  if (!channel) {
    return false;
  }
  return channel.status !== 1 || channel.effective_available === false;
};

export const buildTagTestSummary = (tagLabel, successCount, totalCount) => {
  const safeTagLabel = String(tagLabel || '').trim() || '-';
  const tone =
    successCount === totalCount ? 'success' : successCount === 0 ? 'error' : 'info';

  return {
    message: `${safeTagLabel} ${successCount}/${totalCount}`,
    tone,
    successCount,
    totalCount,
  };
};

export const resolveTagTestTargets = (
  channels = [],
  scope = 'all',
  selectedChannelIds = [],
) => {
  if (scope !== 'specified') {
    return channels;
  }

  const selectedIdSet = new Set(
    (selectedChannelIds || []).map((channelId) => Number(channelId)),
  );
  return channels.filter((channel) => selectedIdSet.has(Number(channel?.id)));
};
