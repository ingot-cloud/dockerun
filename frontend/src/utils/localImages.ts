export interface LocalImage {
  Id: string
  RepoTags: string[] | null
  RepoDigests?: string[] | null
  Size: number
  Created: number
}

export type ImageSortKey = 'Id' | 'RepoTags' | 'Size' | 'Created'

export interface ImageContainer {
  Id: string
  ImageID?: string
  Names?: string[]
  State?: string
}

export interface LocalImageDetails {
  Id: string
  RepoTags: string[] | null
  RepoDigests?: string[] | null
  Size: number
  Created: string
  Author?: string
  DockerVersion?: string
  Os?: string
  Architecture?: string
  Variant?: string
  Config?: {
    Cmd?: string[] | null
    Entrypoint?: string[] | null
    Env?: string[] | null
    ExposedPorts?: Record<string, unknown> | null
    Labels?: Record<string, string> | null
    Volumes?: Record<string, unknown> | null
    User?: string
    WorkingDir?: string
    Healthcheck?: { Test?: string[] | null }
  } | null
}

export interface ImageHistoryLayer {
  Id: string
  Created: number
  CreatedBy: string
  Size: number
  Comment?: string
  Tags?: string[] | null
}

/** 包含已停止的容器；容器信息未知时，不把镜像误标为未使用。 */
export function imageUseCount(id: string, containers: ImageContainer[] | null): number | null {
  if (!containers) return null
  const normalize = (value: string) => value.replace(/^sha256:/, '')
  const count = containers.filter(container => container.ImageID && normalize(container.ImageID) === normalize(id)).length
  return count || !containers.some(container => !container.ImageID) ? count : null
}

export function imageEnvironment(values: string[] | null | undefined): { name: string; value: string }[] {
  return (values || []).map(value => {
    const index = value.indexOf('=')
    return index < 0 ? { name: value, value: '' } : { name: value.slice(0, index), value: value.slice(index + 1) }
  })
}

export function imageLayerCommand(command: string): string {
  if (command.startsWith('/bin/sh -c #(nop) ')) return command.slice(18).trim()
  if (command.startsWith('/bin/sh -c ')) return `RUN ${command.slice(11)}`
  return command || '—'
}

export interface ImageListState {
  search: string
  page: number
  size: number
  sort: ImageSortKey
  descending: boolean
}

export function readImageListState(query: Record<string, unknown>): ImageListState {
  const page = Number(query.page)
  const size = Number(query.size)
  const keys = ['Id', 'RepoTags', 'Size', 'Created']
  return {
    search: typeof query.search === 'string' ? query.search : '',
    page: Number.isSafeInteger(page) && page > 0 ? page : 1,
    size: [10, 25, 50, 100].includes(size) ? size : 25,
    sort: keys.includes(String(query.sort)) ? query.sort as ImageSortKey : 'Created',
    descending: query.order !== 'asc',
  }
}

export function imageListQuery(state: ImageListState): Record<string, string> {
  return { search: state.search, page: String(state.page), size: String(state.size), sort: state.sort, order: state.descending ? 'desc' : 'asc' }
}

export function formatImageDate(value: string | number): string {
  const date = new Date(typeof value === 'number' ? value * 1000 : value)
  if (Number.isNaN(date.getTime())) return '—'
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

export function imageTags(image: LocalImage): string[] {
  return (image.RepoTags || []).filter(tag => tag !== '<none>:<none>')
}

export function filterImages(images: LocalImage[], search: string): LocalImage[] {
  const query = search.trim().toLowerCase()
  if (!query) return images
  return images.filter(image => [image.Id, ...imageTags(image), ...(image.RepoDigests || []),
    ...(imageTags(image).length ? [] : ['未标记'])]
    .some(value => value.toLowerCase().includes(query)))
}

export function sortImages(images: LocalImage[], key: ImageSortKey, descending: boolean): LocalImage[] {
  const value = (image: LocalImage) => key === 'RepoTags' ? imageTags(image).join(',') : image[key]
  return [...images].sort((a, b) => {
    const left = value(a)
    const right = value(b)
    const comparison = typeof left === 'number' && typeof right === 'number'
      ? left - right : String(left).localeCompare(String(right))
    return (descending ? -comparison : comparison) || a.Id.localeCompare(b.Id)
  })
}

export function formatImageSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1000)), 4)
  return `${Number((bytes / 1000 ** index).toFixed(1))} ${['B', 'KB', 'MB', 'GB', 'TB'][index]}`
}

export function imageExportName(images: LocalImage[]): string {
  if (images.length !== 1) return `docker-images-${images.length}.tar`
  const image = images[0]
  const name = imageTags(image)[0] || image.Id.replace(/^sha256:/, '').slice(0, 12)
  return `${name.replace(/[^a-zA-Z0-9_.-]/g, '_').slice(0, 120)}.tar`
}

export function imageArchiveError(file: { name: string; size: number }): string {
  if (!/\.(tar|tar\.gz|tgz|tar\.bz2|tar\.xz|tar\.zst)$/i.test(file.name)) {
    return '请选择 .tar 或受支持的压缩镜像归档文件'
  }
  return file.size ? '' : '文件为空，请重新选择'
}

export function translateImageProgress(message: string): string {
  const translations: Record<string, string> = {
    'Pulling fs layer': '准备镜像层', 'Waiting': '等待中', 'Downloading': '下载中',
    'Verifying Checksum': '校验中', 'Download complete': '下载完成',
    'Extracting': '解压中', 'Pull complete': '镜像层就绪', 'Already exists': '镜像层已存在',
    'Starting image import...': '正在导入镜像…',
    'Image import completed successfully': '镜像导入完成',
  }
  if (translations[message]) return translations[message]
  if (message.startsWith('Pulling from ')) return `正在从 ${message.slice(13)} 拉取`
  if (message.startsWith('Digest: ')) return `镜像摘要：${message.slice(8)}`
  if (message.startsWith('Status: ')) return '镜像拉取完成'
  if (message.startsWith('Loaded image: ')) return `已导入镜像：${message.slice(14)}`
  if (message.startsWith('Loaded image ID: ')) return `已导入镜像：${message.slice(17)}`
  return message
}

/** Docker JSON Lines 与导入 SSE 的增量解析；失败和不完整的流不能当作成功。 */
export class ImageStreamParser {
  private buffer = ''
  private event = ''
  private data: string[] = []
  private complete = false

  constructor(private format: 'docker' | 'sse', private onProgress: (message: string) => void) {}

  push(chunk: string) {
    this.buffer += chunk
    let newline: number
    while ((newline = this.buffer.indexOf('\n')) !== -1) {
      const line = this.buffer.slice(0, newline).replace(/\r$/, '')
      this.buffer = this.buffer.slice(newline + 1)
      this.line(line)
    }
  }

  finish() {
    if (this.buffer.trim()) this.line(this.buffer.replace(/\r$/, ''))
    if (this.format === 'sse') this.dispatch()
    if (!this.complete) throw new Error('操作尚未完成，连接已中断，请刷新镜像列表后重试')
  }

  private line(line: string) {
    if (this.format === 'docker') {
      if (!line.trim()) return
      const message = this.parse(line)
      if (message.error || message.errorDetail?.message) {
        throw new Error(message.error || message.errorDetail.message)
      }
      if (message.status) {
        if (/^Status: /.test(message.status)) this.complete = true
        this.onProgress(`${message.id ? `${message.id}：` : ''}${translateImageProgress(message.status)}${message.progress ? ` ${message.progress}` : ''}`)
      }
      return
    }
    if (!line) { this.dispatch(); return }
    if (line.startsWith('event:')) this.event = line.slice(6).trim()
    if (line.startsWith('data:')) this.data.push(line.slice(5).trimStart())
  }

  private dispatch() {
    if (!this.data.length) { this.event = ''; return }
    const message = this.parse(this.data.join('\n'))
    if (this.event === 'error') throw new Error(message.message || '镜像导入失败')
    if (this.event === 'complete') this.complete = true
    if (message.message) this.onProgress(translateImageProgress(message.message.trim()))
    this.event = ''
    this.data = []
  }

  private parse(text: string) {
    try {
      const value = JSON.parse(text)
      if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error()
      return value
    } catch { throw new Error('镜像操作返回的数据不完整，请刷新列表后重试') }
  }
}
