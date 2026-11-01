import CryptoJS from 'crypto-js'
import request from './request'
import { api } from './workspace'

export interface OfflineMaterial {
  id: number
  name: string
  kind: 'bundle' | 'image'
  status: string
  manifest?: string
  error?: string
}

export async function uploadOfflineMaterial(file: File, kind: OfflineMaterial['kind'], report: (stage: string, progress: number | null) => void, signal?: AbortSignal): Promise<OfflineMaterial> {
  if (!file.size) throw new Error('文件为空，请选择有效的离线材料')
  const hasher = CryptoJS.algo.SHA256.create(), chunkSize = 8 * 1024 * 1024
  report('正在计算文件校验值…', null)
  for (let start = 0; start < file.size; start += chunkSize) {
    signal?.throwIfAborted()
    const bytes = new Uint8Array(await file.slice(start, start + chunkSize).arrayBuffer())
    hasher.update(CryptoJS.lib.WordArray.create(bytes as any))
  }
  const hash = hasher.finalize().toString()
  let { offset } = await api<{ offset: number }>('GET', `/bundles/uploads/${hash}`, undefined, signal)
  if (!Number.isSafeInteger(offset) || offset < 0 || offset > file.size) throw new Error('服务器返回的上传偏移无效，请重试')
  report('正在上传材料，可断点续传…', Math.floor(offset / file.size * 100))
  while (offset < file.size) {
    const end = Math.min(offset + chunkSize, file.size)
    const result = await request.put(`/bundles/uploads/${hash}`, file.slice(offset, end), {
      headers: { 'Content-Type': 'application/octet-stream', 'Content-Range': `bytes ${offset}-${end - 1}/${file.size}` },
      timeout: 120000, silentError: true, signal,
    }) as unknown as { offset: number }
    if (!Number.isSafeInteger(result.offset) || result.offset <= offset || result.offset > file.size) throw new Error('上传进度未推进，请重新选择同一文件重试')
    offset = result.offset
    report('正在上传材料，可断点续传…', Math.floor(offset / file.size * 100))
  }
  const material = await api<OfflineMaterial>('POST', `/bundles/uploads/${hash}/complete`, { kind, name: file.name }, signal)
  if ((material.kind === 'image') !== (kind === 'image')) throw new Error('此文件已按另一种材料类型导入，请选择对应的镜像来源')
  return material
}
