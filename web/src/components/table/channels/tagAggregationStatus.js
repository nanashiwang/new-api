const getTagAggregationProgressColor = (enabledPercent) => {
  if (enabledPercent >= 70) {
    return 'var(--semi-color-success)';
  }
  if (enabledPercent > 30) {
    return 'var(--semi-color-warning)';
  }
  return 'var(--semi-color-danger)';
};

const isEffectivelyEnabledChannel = (channel) =>
  channel?.status === 1 && channel?.effective_available !== false;

export const getTagAggregationStatus = (channels = []) => {
  const totalCount = Array.isArray(channels) ? channels.length : 0;
  const enabledCount = totalCount
    ? channels.filter(isEffectivelyEnabledChannel).length
    : 0;
  const disabledCount = Math.max(totalCount - enabledCount, 0);
  const progressPercent =
    totalCount > 0 ? Math.round((enabledCount / totalCount) * 100) : 0;

  return {
    totalCount,
    enabledCount,
    disabledCount,
    progressPercent,
    progressStroke: getTagAggregationProgressColor(progressPercent),
    isAllEnabled: totalCount > 0 && disabledCount === 0,
    isAllDisabled: totalCount > 0 && enabledCount === 0,
    isMixed: enabledCount > 0 && disabledCount > 0,
    countLabel: `${enabledCount}/${totalCount}`,
  };
};
