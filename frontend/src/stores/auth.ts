import type { LoginUser } from '../types/auth';

const tokenKey = 'diting_token';
const userKey = 'diting_user';

export function getToken(): string {
  return localStorage.getItem(tokenKey) ?? '';
}

export function getUser(): LoginUser | undefined {
  const raw = localStorage.getItem(userKey);
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw) as LoginUser;
  } catch {
    return undefined;
  }
}

export function saveSession(token: string, user: LoginUser) {
  localStorage.setItem(tokenKey, token);
  localStorage.setItem(userKey, JSON.stringify(user));
}

export function clearSession() {
  localStorage.removeItem(tokenKey);
  localStorage.removeItem(userKey);
}
