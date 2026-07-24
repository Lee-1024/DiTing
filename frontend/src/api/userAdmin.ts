import { apiClient } from './client';
import type { CreateUserPayload, ManagedUser, Role, UpdateUserPayload } from '../types/userAdmin';

// listUsers 查询并返回 list Users 列表。
export async function listUsers(): Promise<ManagedUser[]> {
  const response = await apiClient.get<ManagedUser[]>('/users');
  return response.data;
}

// createUser 创建新的 create User。
export async function createUser(payload: CreateUserPayload): Promise<ManagedUser> {
  const response = await apiClient.post<ManagedUser>('/users', payload);
  return response.data;
}

// updateUser 保存或更新 update User。
export async function updateUser(id: string, payload: UpdateUserPayload): Promise<ManagedUser> {
  const response = await apiClient.put<ManagedUser>(`/users/${id}`, payload);
  return response.data;
}

// resetUserPassword 重置 reset User Password 状态。
export async function resetUserPassword(id: string, password: string): Promise<void> {
  await apiClient.post(`/users/${id}/password`, { password });
}

// deleteUser 删除指定的 delete User。
export async function deleteUser(id: string): Promise<void> {
  await apiClient.delete(`/users/${id}`);
}

// listRoles 查询并返回 list Roles 列表。
export async function listRoles(): Promise<Role[]> {
  const response = await apiClient.get<Role[]>('/roles');
  return response.data;
}
