import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/unsavedChanges.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { draftSnapshot, createUnsavedChangesGuard } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const draft = (state = {}) => ({ active: () => state.active ?? true, dirty: () => state.dirty ?? true, busy: () => state.busy ?? false })

test('比较实际草稿：忽略对象键顺序，保留数组顺序、脚本及空值差异', () => {
  const initial = { name: '生产', scripts: { after: '', before: 'echo before' }, targets: [1, 2] }
  const original = draftSnapshot(initial)
  assert.equal(original, draftSnapshot({ targets: [1, 2], scripts: { before: 'echo before', after: '' }, name: '生产' }))
  initial.scripts.after = 'echo changed'
  assert.notEqual(draftSnapshot(initial), original)
  initial.scripts.after = ''
  assert.equal(draftSnapshot(initial), original)
  assert.notEqual(draftSnapshot({ targets: [2, 1] }), draftSnapshot({ targets: [1, 2] }))
  assert.equal(new Set([undefined, null, '', false, 0].map(draftSnapshot)).size, 5)
})

test('未修改和未打开的弹窗不提示，提交中的活动表单直接阻止离开', async () => {
  let prompts = 0
  const guard = createUnsavedChangesGuard(async () => { prompts++; return true })
  guard.register(draft({ dirty: false }))
  guard.register(draft({ active: false, busy: true }))
  assert.equal(await guard.canLeave(), true)
  guard.register(draft({ busy: true }))
  assert.equal(await guard.canLeave(), false)
  assert.equal(prompts, 0)
})

test('取消保留草稿；确认只授权当次离开，不自动标记已保存', async () => {
  let accepted = false, prompts = 0
  const state = { dirty: true }
  const guard = createUnsavedChangesGuard(async () => { prompts++; return accepted })
  guard.register(draft(state))
  assert.equal(await guard.canLeave(), false)
  assert.equal(guard.dirty(), true)
  accepted = true
  assert.equal(await guard.canLeave(), true)
  assert.equal(guard.dirty(), true)
  assert.equal(await guard.canLeave(), true)
  assert.equal(prompts, 3)
  state.dirty = false
  assert.equal(await guard.canLeave(), true)
  assert.equal(prompts, 3)
})

test('嵌套草稿与重复导航共用一次待确认结果', async () => {
  let prompts = 0, answer
  const guard = createUnsavedChangesGuard(() => { prompts++; return new Promise(resolve => { answer = resolve }) })
  const parent = draft(), child = draft()
  guard.register(parent); guard.register(child)
  const first = guard.canLeave(), second = guard.canLeave(), close = guard.canLeave([child])
  await Promise.resolve()
  assert.equal(prompts, 1)
  answer(false)
  assert.deepEqual(await Promise.all([first, second, close]), [false, false, false])
})

test('关闭子弹窗只检查其自身，路由离开检查所有草稿', async () => {
  let prompts = 0
  const guard = createUnsavedChangesGuard(async () => { prompts++; return true })
  const child = draft({ dirty: false })
  guard.register(draft()); guard.register(child)
  assert.equal(await guard.canLeave([child]), true)
  assert.equal(prompts, 0)
  assert.equal(await guard.canLeave(), true)
  assert.equal(prompts, 1)
})

test('等待确认期间开始提交，仍不能离开', async () => {
  let answer
  const state = { busy: false }
  const guard = createUnsavedChangesGuard(() => new Promise(resolve => { answer = resolve }))
  guard.register(draft(state))
  const result = guard.canLeave()
  await Promise.resolve()
  state.busy = true
  answer(true)
  assert.equal(await result, false)
})

test('卸载后清理草稿；关闭确认或确认失败按取消处理', async () => {
  const guard = createUnsavedChangesGuard(async () => { throw new Error('取消') })
  const unregister = guard.register(draft())
  assert.equal(await guard.canLeave(), false)
  unregister()
  assert.equal(guard.size, 0)
  assert.equal(guard.dirty(), false)
  assert.equal(await guard.canLeave(), true)
})
