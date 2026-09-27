const trim = (value) => (typeof value === 'string' ? value.trim() : '');
const httpAddress = (value) => {
  const text = trim(value);
  try {
    const url = new URL(text);
    return ['http:', 'https:'].includes(url.protocol) &&
      !url.username &&
      !url.password
      ? text
      : '';
  } catch {
    return '';
  }
};

export function getApiAddresses(status, origin = '') {
  if (!status) return [];
  const configured =
    status.api_info_enabled !== false && Array.isArray(status.api_info)
      ? status.api_info
      : [];
  const seen = new Set();
  const addresses = configured.flatMap((item) => {
    const url = httpAddress(item?.url);
    if (!url || seen.has(url)) return [];
    seen.add(url);
    return [
      {
        url,
        label: trim(item.route) || url,
        description: trim(item.description),
      },
    ];
  });
  if (addresses.length) return addresses;
  const server = httpAddress(status.server_address);
  const url = server || httpAddress(origin);
  return url
    ? [
        {
          url,
          labelKey: server ? '默认 API 地址' : '当前域名',
          description: '',
        },
      ]
    : [];
}
