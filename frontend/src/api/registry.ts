import request from "./request";

export interface Registry {
  id: number;
  kind?: "external" | "builtin";
  name: string;
  url: string;
  pull_username?: string;
  pull_password?: string;
  pull_only_confirmed?: boolean;
  username: string;
  authentication?: boolean;
  password?: string;
  email: string;
  is_default: boolean;
  insecure: boolean;
  created_at: string;
  updated_at: string;
}

export interface RegistryImage {
  name: string;
  tags: string[];
  tags_error?: string;
}

export const registryApi = {
  // Registry 管理
  list: (signal?: AbortSignal) => request.get("/registries", { signal }) as Promise<Registry[]>,
  get: (id: number, signal?: AbortSignal) => request.get(`/registries/${id}`, { signal }) as Promise<Registry>,
  create: (data: Partial<Registry>) =>
    request.post("/registries", data) as Promise<Registry>,
  update: (id: number, data: Partial<Registry>) =>
    request.put(`/registries/${id}`, data) as Promise<Registry>,
  delete: (id: number) => request.delete(`/registries/${id}`) as Promise<void>,
  test: (data: {
    id?: number;
    url: string;
    username?: string;
    password?: string;
    insecure?: boolean;
    authentication?: boolean;
  }) => request.post("/registries/test", data) as Promise<{ message: string }>,

  // Registry 镜像管理
  images: {
    list: (id: number, page?: number, pageSize?: number, search?: string, signal?: AbortSignal) =>
      request.get(`/registries/${id}/images`, {
        params: { page, page_size: pageSize, search },
        signal,
      }) as Promise<{
        items: RegistryImage[];
        total: number;
        page: number;
        page_size: number;
        registry_name: string;
      }>,
    getTags: (id: number, name: string, signal?: AbortSignal) =>
      request.get(
        `/registries/${id}/images/${encodeURIComponent(name)}/tags`,
        { signal },
      ) as Promise<{ name: string; tags: string[] }>,
    delete: (id: number, name: string, tag: string) =>
      request.delete(`/registries/${id}/images/${encodeURIComponent(name)}`, {
        params: { tag },
      }) as Promise<void>,
  },
};
