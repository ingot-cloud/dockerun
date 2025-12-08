import request from './request'

export interface InstallDockerParams {
  method: 'online' | 'offline-rpm' | 'offline-deb' | 'offline-binary'
  package_path?: string
  distribution?: string
}

export const systemApi = {
  checkDockerStatus: () => request.get('/system/docker-status'),
  getSystemInfo: () => request.get('/system/info'),
  installDocker: (data: InstallDockerParams) => request.post('/system/install-docker', data)
}

