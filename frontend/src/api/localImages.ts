import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import { ImageStreamParser, type ImageContainer, type ImageHistoryLayer, type LocalImage, type LocalImageDetails } from '@/utils/localImages'

async function imageRequest(path: string, options: RequestInit = {}, base = '/api/images') {
  const auth = useAuthStore()
  const headers = new Headers(options.headers)
  if (auth.token) headers.set('Authorization', `Bearer ${auth.token}`)
  let response: Response
  try {
    response = await fetch(`${base}${path}`, { ...options, headers })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new Error('无法连接服务器，请检查网络后重试')
  }
  if (!response.ok) {
    if (response.status === 401) {
      auth.logout()
      void router.replace('/login')
      throw new Error('登录已过期，请重新登录')
    }
    const body = await response.json().catch(() => null)
    if (body?.error === 'API endpoint not found') throw new Error('请求的功能暂不可用，请稍后重试')
    throw new Error(body?.error || `镜像操作失败（${response.status}）`)
  }
  return response
}

async function transfer(path: string, body: BodyInit, format: 'docker' | 'sse',
  onProgress: (message: string) => void, signal: AbortSignal) {
  const response = await imageRequest(path, {
    method: 'POST', body, signal,
    headers: format === 'docker' ? { 'Content-Type': 'application/json' } : undefined,
  })
  if (!response.body) throw new Error('无法读取镜像操作进度，请重试')
  const parser = new ImageStreamParser(format, onProgress)
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      parser.push(decoder.decode(value, { stream: true }))
    }
    parser.push(decoder.decode())
    parser.finish()
  } finally {
    await reader.cancel().catch(() => undefined)
    reader.releaseLock()
  }
}

export const localImagesApi = {
  async inspect(id: string, signal: AbortSignal): Promise<LocalImageDetails> {
    const response = await imageRequest(`/${encodeURIComponent(id)}`, { signal, cache: 'no-store' })
    return response.json()
  },
  async history(id: string, signal: AbortSignal): Promise<ImageHistoryLayer[]> {
    const response = await imageRequest(`/${encodeURIComponent(id)}/history`, { signal, cache: 'no-store' })
    const data = await response.json()
    if (data === null) return []
    if (!Array.isArray(data)) throw new Error('镜像层数据格式错误，请重试')
    return data
  },
  async containers(signal: AbortSignal): Promise<ImageContainer[]> {
    const response = await imageRequest('?all=true', { signal, cache: 'no-store' }, '/api/containers')
    const data = await response.json()
    if (data === null) return []
    if (!Array.isArray(data)) throw new Error('容器使用状态数据格式错误，请重试')
    return data
  },
  async tag(source: string, target: string, signal: AbortSignal) {
    await imageRequest('/tag', {
      method: 'POST', signal, headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source, target }),
    })
  },
  async list(signal: AbortSignal): Promise<LocalImage[]> {
    const response = await imageRequest('', { signal, cache: 'no-store' })
    const data = await response.json()
    if (data === null) return []
    if (!Array.isArray(data)) throw new Error('镜像列表数据格式错误，请重试')
    return data
  },
  async remove(id: string, force: boolean, signal: AbortSignal) {
    await imageRequest(`/${encodeURIComponent(id)}?force=${force}`, { method: 'DELETE', signal })
  },
  pull(name: string, onProgress: (message: string) => void, signal: AbortSignal) {
    return transfer('/pull', JSON.stringify({ image_name: name.trim() }), 'docker', onProgress, signal)
  },
  import(file: File, onProgress: (message: string) => void, signal: AbortSignal) {
    const data = new FormData()
    data.append('file', file)
    return transfer('/import', data, 'sse', onProgress, signal)
  },
  async export(images: LocalImage[], signal: AbortSignal) {
    const response = images.length === 1
      ? await imageRequest(`/${encodeURIComponent(images[0].Id)}/export`, { signal })
      : await imageRequest('/export', {
        method: 'POST', signal, headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ image_ids: images.map(image => image.Id) }),
      })
    const archive = await response.blob()
    if (!archive.size) throw new Error('镜像归档为空，请重试')
    return archive
  },
}
