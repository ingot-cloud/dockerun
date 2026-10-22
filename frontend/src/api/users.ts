import request from './request'
import type { ManagedUser, RegistryAccess } from '@/utils/users'

interface CreateUserRequest { username: string; password: string; is_admin: boolean; registry_access: RegistryAccess }
interface UpdateUserRequest { password?: string; is_admin?: boolean; enabled?: boolean; registry_access?: RegistryAccess }
export const usersApi = {
  list: (signal?: AbortSignal): Promise<ManagedUser[] | null> => request.get('/users', { signal, silentError: true }),
  create: (data: CreateUserRequest): Promise<ManagedUser> => request.post('/users', data, { silentError: true }),
  update: (id: number, data: UpdateUserRequest): Promise<ManagedUser> => request.patch(`/users/${id}`, data, { silentError: true }),
}
