// Display the same partition as billing. The total includes the TTL splits.
// Legacy logs may omit the total; only then may it be derived from the splits.
export const getCacheCreationBreakdown = (record = {}) => {
  const count = (value) => {
    const n = Number(value);
    return Number.isFinite(n) && n >= 0 ? n : 0;
  };
  const fiveMinute = count(record.cache_creation_tokens_5m);
  const oneHour = count(record.cache_creation_tokens_1h);
  const split = fiveMinute + oneHour;
  const reported = record.cache_creation_tokens;
  const hasTotal = reported !== undefined && reported !== null;
  const total = hasTotal ? count(reported) : split;
  return {
    total,
    fiveMinute,
    oneHour,
    unclassified: Math.max(0, total - split),
    conflict: hasTotal && split > total,
  };
};
