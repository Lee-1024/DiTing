export function formatJSON(value?: string) {
  if (!value) {
    return '{}';
  }
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

export function compactNumber(value?: number) {
  return new Intl.NumberFormat('zh-CN').format(value ?? 0);
}
