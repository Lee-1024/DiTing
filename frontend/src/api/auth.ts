import { apiClient } from './client';
import type { LoginResponse } from '../types/auth';

// login 处理 login 相关逻辑。
export async function login(username: string, password: string): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>('/auth/login', { username, password });
  return response.data;
}

// changePassword 处理 change Password 相关逻辑。
export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  await apiClient.post('/auth/password', { oldPassword, newPassword });
}
