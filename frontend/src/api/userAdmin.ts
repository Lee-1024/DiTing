import { apiClient } from './client';
import type { CreateUserPayload, ManagedUser, Role, UpdateUserPayload } from '../types/userAdmin';

export async function listUsers(): Promise<ManagedUser[]> {
  const response = await apiClient.get<ManagedUser[]>('/users');
  return response.data;
}

export async function createUser(payload: CreateUserPayload): Promise<ManagedUser> {
  const response = await apiClient.post<ManagedUser>('/users', payload);
  return response.data;
}

export async function updateUser(id: string, payload: UpdateUserPayload): Promise<ManagedUser> {
  const response = await apiClient.put<ManagedUser>(`/users/${id}`, payload);
  return response.data;
}

export async function resetUserPassword(id: string, password: string): Promise<void> {
  await apiClient.post(`/users/${id}/password`, { password });
}

export async function deleteUser(id: string): Promise<void> {
  await apiClient.delete(`/users/${id}`);
}

export async function listRoles(): Promise<Role[]> {
  const response = await apiClient.get<Role[]>('/roles');
  return response.data;
}
