// formatJSON 格式化 format JSON 以便界面展示。
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

// compactNumber 处理 compact Number 相关逻辑。
export function compactNumber(value?: number) {
  return new Intl.NumberFormat('zh-CN').format(value ?? 0);
}
