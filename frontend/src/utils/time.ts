import dayjs from 'dayjs';

const backendDateTimePattern = /^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?$/;

export function formatLocalDateTime(value?: string | null) {
  if (!value) {
    return '-';
  }
  const normalized = backendDateTimePattern.test(value) ? `${value.replace(' ', 'T')}Z` : value;
  const parsed = dayjs(normalized);
  if (!parsed.isValid()) {
    return value;
  }
  return parsed.format('YYYY-MM-DD HH:mm:ss');
}
