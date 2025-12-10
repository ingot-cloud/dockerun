import request from './request'

export interface DockerInfo {
  os: string
  arch: string
  docker_version: string
  docker_api_version: string
  compose_version: string
  server_version: string
  server_os: string
  server_arch: string
  total_memory: string
  cpus: number
  kernel_version: string
  operating_system: string
}

export const systemApi = {
  checkDockerStatus: () => request.get('/system/docker-status'),
  getSystemInfo: () => request.get<DockerInfo>('/system/info')
}

