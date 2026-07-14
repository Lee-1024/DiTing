import dayjs from 'dayjs';

export function formatLocalDateTime(value?: string | null) {
  if (!value) {
    return '-';
  }
  const parsed = dayjs(value);
  if (!parsed.isValid()) {
    return value;
  }
  return parsed.format('YYYY-MM-DD HH:mm:ss');
}
