export interface ManagedUser {
  id: string;
  username: string;
  displayName: string;
  email: string;
  status: 'active' | 'disabled';
  roles: string[];
  createdAt: string;
  updatedAt: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateUserPayload {
  username: string;
  password: string;
  displayName: string;
  email: string;
  status: 'active' | 'disabled';
  roles: string[];
}

export interface UpdateUserPayload {
  displayName: string;
  email: string;
  status: 'active' | 'disabled';
  roles: string[];
}
