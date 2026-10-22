import request from './request'
import type { ContainerDetails, ContainerStats } from '@/utils/containers'
import type { DockerNetwork, NetworkCreateRequest } from '@/utils/networks'
import type { DockerVolume, VolumeCreateRequest } from '@/utils/volumes'

export const dockerApi = {
  // 容器相关
  containers: {
    list: (all = true, signal?: AbortSignal) => request.get('/containers', { params: { all }, signal }),
    inspect: (id: string, signal?: AbortSignal) => request.get<unknown, ContainerDetails>(`/containers/${id}`, { signal }),
    start: (id: string) => request.post(`/containers/${id}/start`),
    stop: (id: string) => request.post(`/containers/${id}/stop`),
    restart: (id: string) => request.post(`/containers/${id}/restart`),
    kill: (id: string) => request.post(`/containers/${id}/kill`),
    pause: (id: string) => request.post(`/containers/${id}/pause`),
    resume: (id: string) => request.post(`/containers/${id}/resume`),
    rename: (id: string, name: string) => request.post(`/containers/${id}/rename`, { name }),
    updateRestart: (id: string, name: string, maximum_retry_count = 0) => request.put(`/containers/${id}/restart-policy`, { name, maximum_retry_count }),
    commit: (id: string, image: string) => request.post(`/containers/${id}/commit`, { image }, { timeout: 300000 }),
    remove: (id: string, force = false) => request.delete(`/containers/${id}`, { params: { force } }),
    logs: (id: string, tail = '100', options: { since?: number; timestamps?: boolean; signal?: AbortSignal } = {}) => request.get<unknown, string>(`/containers/${id}/logs`, { params: { tail, since: options.since, timestamps: options.timestamps ?? true }, responseType: 'text', signal: options.signal }),
    stats: (id: string, signal?: AbortSignal) => request.get<unknown, ContainerStats>(`/containers/${id}/stats`, { signal }),
    top: (id: string, signal?: AbortSignal) => request.get<unknown, { Titles: string[]; Processes: string[][] }>(`/containers/${id}/top`, { signal }),
    console: (id: string, options: { mode: 'exec' | 'attach'; command: string[]; user: string }) => request.post<unknown, { ticket: string }>(`/containers/${id}/console`, options)
  },

  // 镜像相关
  images: {
    list: () => request.get('/images'),
    inspect: (id: string) => request.get(`/images/${id}`),
    pull: (imageName: string) => request.post('/images/pull', { image_name: imageName }),
    remove: (id: string, force = false) => request.delete(`/images/${id}`, { params: { force } }),
    tag: (source: string, target: string) => request.post('/images/tag', { source, target }),
    load: (path: string) => request.post('/images/load', { path }),
    save: (id: string, path: string) => request.post('/images/save', { image_id: id, path }),
    loadFile: (formData: FormData) => request.post('/images/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }),
    saveFile: (id: string) => request.get(`/images/${id}/export`, { 
      responseType: 'blob'
    })
  },

  // 网络相关
  networks: {
    list: (signal?: AbortSignal) => request.get<unknown, DockerNetwork[] | null>('/networks', { signal }),
    inspect: (id: string, signal?: AbortSignal) => request.get<unknown, DockerNetwork>(`/networks/${encodeURIComponent(id)}`, { signal }),
    create: (params: NetworkCreateRequest) => request.post<unknown, { Id: string; Warning?: string }>('/networks', params),
    remove: (id: string) => request.delete(`/networks/${id}`),
    connect: (id: string, containerId: string) => request.post(`/networks/${id}/connect`, { container_id: containerId }),
    disconnect: (id: string, containerId: string, force = false) => request.post(`/networks/${id}/disconnect`, { container_id: containerId, force })
  },

  // 卷相关
  volumes: {
    list: (signal?: AbortSignal) => request.get<unknown, { Volumes: DockerVolume[] | null; Warnings?: string[] | null }>('/volumes', { signal }),
    inspect: (name: string, signal?: AbortSignal) => request.get<unknown, DockerVolume>(`/volumes/${encodeURIComponent(name)}`, { signal }),
    create: (params: VolumeCreateRequest) => request.post<unknown, DockerVolume>('/volumes', params),
    remove: (name: string, force = false) => request.delete(`/volumes/${encodeURIComponent(name)}`, { params: { force } })
  }
}

