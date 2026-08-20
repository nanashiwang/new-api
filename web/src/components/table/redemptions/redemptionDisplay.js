export function getRedemptionDisplayName(record, translate = (key) => key) {
  const name = record?.name || '';
  const userId = Number(record?.user_id);

  if (
    record?.funding_source !== 'wallet' ||
    !Number.isSafeInteger(userId) ||
    userId <= 0
  ) {
    return name;
  }

  return `${name}（${translate('用户 ID')}: ${userId}）`;
}
