// Display-only comparison. Never evaluate or rewrite a billing expression.
const tokens = (value) =>
  value.match(/\s+|[A-Za-z_]\w*|\d+(?:\.\d+)?|./gu) || [];

export function expressionDiff(value, baseline) {
  const text = String(value ?? '');
  const parts = tokens(text);
  if (typeof baseline !== 'string' || baseline === text) {
    return [{ text, changed: false }];
  }
  const previous = tokens(baseline);
  if (parts.length === previous.length) {
    return parts.map((part, i) => ({
      text: part,
      changed: part !== previous[i],
    }));
  }
  // Linear bounded work, even for very long expressions. Preserve shared ends.
  let start = 0;
  while (
    start < parts.length &&
    start < previous.length &&
    parts[start] === previous[start]
  )
    start++;
  let end = parts.length,
    oldEnd = previous.length;
  while (
    end > start &&
    oldEnd > start &&
    parts[end - 1] === previous[oldEnd - 1]
  ) {
    end--;
    oldEnd--;
  }
  return parts.map((part, i) => ({
    text: part,
    changed: end === start || (i >= start && i < end),
  }));
}
