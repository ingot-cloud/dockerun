import type { CSSProperties } from 'vue'

export type LogStyle = Pick<CSSProperties, 'color' | 'backgroundColor' | 'fontWeight' | 'fontStyle' | 'textDecoration'>
export interface LogSegment { text: string; style: LogStyle }
export type LogLevel = 'error' | 'warning' | 'info' | 'debug' | ''

const palette = ['#89929e', '#f08080', '#9acb8c', '#e6c48e', '#82aaff', '#c792ea', '#80cbc4', '#d6d6d6', '#999999', '#ff9b9b', '#b5e3a8', '#ffe0a3', '#a6c8ff', '#dfb1ff', '#a0e7e0', '#ffffff']

function indexedColor(index: number) {
  if (!Number.isInteger(index) || index < 0 || index > 255) return undefined
  if (index < 16) return palette[index]
  if (index >= 232) { const grey = 8 + (index - 232) * 10; return `rgb(${grey}, ${grey}, ${grey})` }
  const value = index - 16
  const channel = (number: number) => number === 0 ? 0 : 55 + number * 40
  return `rgb(${channel(Math.floor(value / 36))}, ${channel(Math.floor(value / 6) % 6)}, ${channel(value % 6)})`
}

function applyAnsiStyle(style: LogStyle, parameters: string) {
  const codes = parameters.split(';').map(Number)
  for (let index = 0; index < codes.length; index++) {
    const code = codes[index]
    if (code === 0) { for (const key of Object.keys(style) as (keyof LogStyle)[]) delete style[key] }
    else if (code === 1) style.fontWeight = '700'
    else if (code === 22) delete style.fontWeight
    else if (code === 3) style.fontStyle = 'italic'
    else if (code === 23) delete style.fontStyle
    else if (code === 4) style.textDecoration = 'underline'
    else if (code === 24) delete style.textDecoration
    else if (code === 39) delete style.color
    else if (code === 49) delete style.backgroundColor
    else if (code >= 30 && code <= 37) style.color = palette[code - 30]
    else if (code >= 90 && code <= 97) style.color = palette[code - 90 + 8]
    else if (code >= 40 && code <= 47) style.backgroundColor = palette[code - 40]
    else if (code >= 100 && code <= 107) style.backgroundColor = palette[code - 100 + 8]
    else if (code === 38 || code === 48) {
      const property = code === 38 ? 'color' : 'backgroundColor'
      const mode = codes[++index]
      if (mode === 5) {
        const color = indexedColor(codes[++index])
        if (color) style[property] = color
      } else if (mode === 2) {
        const channels = codes.slice(index + 1, index + 4)
        if (channels.length === 3 && channels.every(channel => Number.isInteger(channel) && channel >= 0 && channel <= 255)) style[property] = `rgb(${channels.join(', ')})`
        index += 3
      }
    }
  }
}

export function containerLogLevel(text: string): LogLevel {
  // 只识别显式级别字段，避免把 URL、业务字段或正文中的单词误判成日志级别。
  const token = text.match(/(?:^|\s|\[|\|)(TRACE|DEBUG|INFO|NOTICE|WARN(?:ING)?|ERROR|FATAL|CRITICAL|PANIC)(?=$|[\s\]:|])/i)?.[1]
    || text.match(/["']?level["']?\s*[:=]\s*["']?(trace|debug|info|notice|warn(?:ing)?|error|fatal|critical|panic)\b/i)?.[1]
  if (!token) return ''
  if (/^(error|fatal|critical|panic)$/i.test(token)) return 'error'
  if (/^warn/i.test(token)) return 'warning'
  return /^(trace|debug)$/i.test(token) ? 'debug' : 'info'
}

export function parseContainerLogs(logs: string) {
  if (!logs) return []
  const style: LogStyle = {}, occurrences = new Map<string, number>()
  return logs.replace(/\r\n/g, '\n').replace(/\n$/, '').split('\n').map((raw, index) => {
    const segments: LogSegment[] = []
    // 丢弃光标、标题、超链接等终端控制序列；正文始终由 Vue 文本节点渲染。
    const controls = /\x1b\][\s\S]*?(?:\x07|\x1b\\)|(?:\x1b\[|\x9b)[0-?]*[ -/]*[@-~]|\x1b[@-_]|[\x00-\x08\x0b-\x1f\x7f-\x9f]/g
    let cursor = 0
    for (const match of raw.matchAll(controls)) {
      if (match.index > cursor) segments.push({ text: raw.slice(cursor, match.index), style: { ...style } })
      const sgr = match[0].match(/^(?:\x1b\[|\x9b)([\d;]*)m$/)
      if (sgr) applyAnsiStyle(style, sgr[1])
      cursor = match.index + match[0].length
    }
    if (cursor < raw.length) segments.push({ text: raw.slice(cursor), style: { ...style } })
    const text = segments.map(segment => segment.text).join('')
    const count = occurrences.get(raw) || 0
    occurrences.set(raw, count + 1)
    return { key: `${raw}\0${count}`, number: index + 1, text, segments, level: containerLogLevel(text) }
  })
}

export function sliceLogSegments(segments: LogSegment[], start: number): LogSegment[] {
  const result: LogSegment[] = []
  let offset = 0
  for (const segment of segments) {
    const end = offset + segment.text.length
    if (end > start) result.push({ text: segment.text.slice(Math.max(0, start - offset)), style: segment.style })
    offset = end
  }
  return result
}

interface LogScrollAnchor {
  key?: string
  offset: number
  scrollTop: number
  scrollLeft: number
}

export function isLogViewportAtBottom(viewport: Pick<HTMLElement, 'scrollHeight' | 'scrollTop' | 'clientHeight'>) {
  return viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight <= 2
}

export function captureLogScrollAnchor(viewport: HTMLElement, lines: readonly { key: string }[]): LogScrollAnchor {
  const top = viewport.getBoundingClientRect().top + viewport.clientTop
  let start = 0, end = lines.length
  // 日志可能很多，用二分定位第一条仍在视口中的日志。
  while (start < end) {
    const middle = Math.floor((start + end) / 2)
    const element = viewport.children.item(middle)
    if (element && element.getBoundingClientRect().bottom <= top) start = middle + 1
    else end = middle
  }
  const element = start < lines.length ? viewport.children.item(start) : null
  return { key: element ? lines[start].key : undefined, offset: element ? element.getBoundingClientRect().top - top : 0, scrollTop: viewport.scrollTop, scrollLeft: viewport.scrollLeft }
}

export function restoreLogScrollAnchor(viewport: HTMLElement, lines: readonly { key: string }[], anchor: LogScrollAnchor) {
  const index = anchor.key === undefined ? -1 : lines.findIndex(line => line.key === anchor.key)
  const element = index >= 0 ? viewport.children.item(index) : null
  // 获取最近若干行时，旧行会从头部移除；按日志标识恢复阅读位置。
  viewport.scrollTop = element
    ? viewport.scrollTop + element.getBoundingClientRect().top - viewport.getBoundingClientRect().top - viewport.clientTop - anchor.offset
    : anchor.scrollTop
  viewport.scrollLeft = anchor.scrollLeft
}
