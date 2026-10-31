import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/containerLogs.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { parseContainerLogs, sliceLogSegments, containerLogLevel, isLogViewportAtBottom, captureLogScrollAnchor, restoreLogScrollAnchor } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('ANSI 颜色、字重和重置保留正文，搜索与复制得到纯文本', () => {
  const [line] = parseContainerLogs('\x1b[1;31mERROR\x1b[22;39m 服务启动失败\n')
  assert.equal(line.text, 'ERROR 服务启动失败')
  assert.equal(line.level, 'error')
  assert.deepEqual(line.segments, [{ text: 'ERROR', style: { fontWeight: '700', color: '#f08080' } }, { text: ' 服务启动失败', style: {} }])
})

test('多行颜色持续到重置，空行与重复日志拥有稳定标识', () => {
  const lines = parseContainerLogs('\x1b[33m警告\r\n继续\x1b[0m\n\n重复\n重复\n')
  assert.equal(lines.length, 5)
  assert.equal(lines[1].segments[0].style.color, '#e6c48e')
  assert.equal(lines[2].text, '')
  assert.notEqual(lines[3].key, lines[4].key)
  assert.deepEqual(lines[3].segments[0].style, {})
  assert.deepEqual(parseContainerLogs(''), [])
})

test('支持明亮色、背景色、256 色与真彩色并忽略无效颜色', () => {
  const [line] = parseContainerLogs('\x1b[94;48;5;235m蓝色\x1b[38;5;196m红色\x1b[38;2;12;120;240m真彩\x1b[38;5;999m保留\x1b[0m默认')
  assert.deepEqual(line.segments.map(segment => segment.style.color), ['#a6c8ff', 'rgb(255, 0, 0)', 'rgb(12, 120, 240)', 'rgb(12, 120, 240)', undefined])
  assert.equal(line.segments[0].style.backgroundColor, 'rgb(38, 38, 38)')
  assert.deepEqual(line.segments[4].style, {})
})

test('终端控制与超链接被清理，HTML 原文保留为普通文本', () => {
  const [line] = parseContainerLogs('\x1b[2K\x1b]0;窗口标题\x07\x1b]8;;https://example.com\x1b\\链接\x1b]8;;\x1b\\ <img src=x onerror=alert(1)>')
  assert.equal(line.text, '链接 <img src=x onerror=alert(1)>')
  assert.deepEqual(line.segments.map(segment => segment.style), [{}, {}])
})

test('隐藏时间戳时保留跨片段正文的 ANSI 样式', () => {
  const [line] = parseContainerLogs('\x1b[36m2026-10-30T00:00:00Z \x1b[31mERROR\x1b[0m 失败')
  assert.deepEqual(sliceLogSegments(line.segments, 21), [{ text: 'ERROR', style: { color: '#f08080' } }, { text: ' 失败', style: {} }])
})

test('识别常见日志级别，普通资源值不误判', () => {
  for (const [text, level] of [['[WARN] 磁盘空间不足', 'warning'], ['level=error 连接失败', 'error'], ['{"level":"debug","msg":"请求"}', 'debug'], ['2026-10-30 INFO 已启动', 'info'], ['[PANIC] 崩溃', 'error'], ['GET /error-report HTTP/1.1', ''], ['user=info@example.com', '']]) assert.equal(containerLogLevel(text), level, text)
})

test('只有到达底部才继续跟随，兼容滚动位置的小数误差和短日志', () => {
  assert.equal(isLogViewportAtBottom({ scrollHeight: 1000, clientHeight: 300, scrollTop: 700 }), true)
  assert.equal(isLogViewportAtBottom({ scrollHeight: 1000, clientHeight: 300, scrollTop: 699.5 }), true)
  assert.equal(isLogViewportAtBottom({ scrollHeight: 1000, clientHeight: 300, scrollTop: 690 }), false)
  assert.equal(isLogViewportAtBottom({ scrollHeight: 200, clientHeight: 300, scrollTop: 0 }), true)
})

function mockViewport(heights, scrollTop = 0) {
  const viewport = {
    scrollTop, scrollLeft: 12, clientTop: 1,
    getBoundingClientRect: () => ({ top: 100 }),
    children: { item(index) {
      if (index < 0 || index >= heights.length) return null
      return { getBoundingClientRect() {
        const top = 101 + heights.slice(0, index).reduce((sum, height) => sum + height, 0) - viewport.scrollTop
        return { top, bottom: top + heights[index] }
      } }
    } },
  }
  return viewport
}
const keyedLines = (...keys) => keys.map(key => ({ key }))

test('新增日志时保留第一条可见日志和行内阅读偏移', () => {
  const heights = [20, 40, 20], viewport = mockViewport(heights, 27)
  const anchor = captureLogScrollAnchor(viewport, keyedLines('a', 'b', 'c'))
  assert.equal(anchor.key, 'b')
  assert.equal(anchor.offset, -7)
  heights.push(80)
  restoreLogScrollAnchor(viewport, keyedLines('a', 'b', 'c', 'd'), anchor)
  assert.equal(viewport.scrollTop, 27)
  assert.equal(viewport.scrollLeft, 12)
})

test('日志尾部窗口移除旧行或换行高度变化时，仍停留在原日志位置', () => {
  const heights = [20, 40, 20], viewport = mockViewport(heights, 27)
  const anchor = captureLogScrollAnchor(viewport, keyedLines('a', 'b', 'c'))
  heights.splice(0, heights.length, 60, 20, 40)
  restoreLogScrollAnchor(viewport, keyedLines('b', 'c', 'd'), anchor)
  assert.equal(viewport.scrollTop, 7)
  assert.equal(captureLogScrollAnchor(viewport, keyedLines('b', 'c', 'd')).key, 'b')
})

test('阅读的日志已移除或日志为空时安全恢复滚动位置', () => {
  const viewport = mockViewport([20, 40, 20], 27)
  const anchor = captureLogScrollAnchor(viewport, keyedLines('a', 'b', 'c'))
  viewport.scrollTop = 0
  restoreLogScrollAnchor(viewport, keyedLines('d', 'e', 'f'), anchor)
  assert.equal(viewport.scrollTop, 27)
  assert.equal(captureLogScrollAnchor(mockViewport([]), []).key, undefined)
})
