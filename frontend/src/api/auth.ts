import { apiClient } from './client';
import type { LoginResponse } from '../types/auth';

export async function login(username: string, password: string): Promise<LoginResponse> {
  const response = await apiClient.post<LoginResponse>('/auth/login', { username, password });
  return response.data;
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  await apiClient.post('/auth/password', { oldPassword, newPassword });
}
