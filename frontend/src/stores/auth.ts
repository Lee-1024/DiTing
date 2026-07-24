import type { LoginUser } from '../types/auth';

const tokenKey = 'diting_token';
const userKey = 'diting_user';

// getToken 获取 get Token 数据。
export function getToken(): string {
  return localStorage.getItem(tokenKey) ?? '';
}

// getUser 获取 get User 数据。
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

// saveSession 保存或更新 save Session。
export function saveSession(token: string, user: LoginUser) {
  localStorage.setItem(tokenKey, token);
  localStorage.setItem(userKey, JSON.stringify(user));
}

// clearSession 处理 clear Session 相关逻辑。
export function clearSession() {
  localStorage.removeItem(tokenKey);
  localStorage.removeItem(userKey);
}
