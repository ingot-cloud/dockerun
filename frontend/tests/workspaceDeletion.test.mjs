import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/workspaceDeletion.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { templateDeletionID, templateDeletionReason, bundleDeletionReason, workspaceDeleteTargets } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const selectionSource = await readFile(new URL('../src/utils/resourceSelection.ts', import.meta.url), 'utf8')
const selectionOutput = ts.transpileModule(selectionSource, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } }).outputText
const { retainPageSelection } = await import(`data:text/javascript;base64,${Buffer.from(selectionOutput).toString('base64')}`)

test('只允许删除编号合法的自建模板，内置模板不能伪装成可删除项', () => {
  assert.equal(templateDeletionID({ source: 'custom', id: 'custom-42' }), 42)
  assert.equal(templateDeletionReason({ source: 'custom', id: 'custom-42' }), '')
  for (const item of [{ source: 'builtin', id: 'mysql' }, { source: 'builtin', id: 'custom-42' }, { source: 'unknown', id: 'custom-42' }]) {
    assert.equal(templateDeletionID(item), null)
    assert.match(templateDeletionReason(item), /内置模板不能删除/)
  }
})

test('拒绝零、负数、表达式、溢出和格式不完整的模板编号', () => {
  for (const id of ['custom-0', 'custom--1', 'custom-1 OR 1=1', '42', 'custom-1x', 'custom-4294967296', 'custom-01', 'custom-']) {
    assert.equal(templateDeletionID({ source: 'custom', id }), null, id)
    assert.match(templateDeletionReason({ source: 'custom', id }), /编号无效/)
  }
})

test('导入完成或失败后才能删除发布包，处理中和未知状态有明确原因', () => {
  for (const status of ['ready', 'failed']) assert.equal(bundleDeletionReason({ status }), '')
  for (const status of ['queued', 'validating', 'loading']) assert.match(bundleDeletionReason({ status }), /正在导入/)
  for (const status of [undefined, '', 'unexpected']) assert.match(bundleDeletionReason({ status }), /状态未知/)
})

test('批量删除固定名称与编号快照，去重并剔除无效编号', () => {
  const rows = [{ id: 1, name: '应用一' }, { id: 1, name: '重复记录' }, { id: 2, name: '应用二' }, { id: 0, name: '无效' }, { id: -1, name: '无效' }, { id: 4294967296, name: '溢出' }]
  const targets = workspaceDeleteTargets(rows, row => row.id, row => row.name)
  rows[0].name = '刷新后的名称'
  assert.deepEqual(targets, [{ id: 1, name: '应用一' }, { id: 2, name: '应用二' }])
  assert.equal(rows.length, 6)
})

test('批量模板删除仅包含合法自建版本，同名的不同版本独立保留', () => {
  const rows = [{ id: 'mysql', source: 'builtin', name: 'MySQL', version: '8' }, { id: 'custom-2', source: 'custom', name: '团队模板', version: 'v1' }, { id: 'custom-3', source: 'custom', name: '团队模板', version: 'v2' }, { id: 'custom-bad', source: 'custom', name: '无效', version: 'v3' }]
  assert.deepEqual(workspaceDeleteTargets(rows, templateDeletionID, row => `${row.name} · ${row.version}`), [{ id: 2, name: '团队模板 · v1' }, { id: 3, name: '团队模板 · v2' }])
})

test('勾选仅作用于当前页，刷新采用最新对象并移除失效或受保护的记录', () => {
  const selected = [{ id: 1, status: 'ready', name: '旧名称' }, { id: 2, status: 'ready' }, { id: 3, status: 'ready' }, { id: 4, status: 'failed' }]
  const current = [{ id: 1, status: 'ready', name: '新名称' }, { id: 2, status: 'loading' }, { id: 5, status: 'ready' }]
  const retained = retainPageSelection(selected, current, row => row.id, row => !bundleDeletionReason(row))
  assert.deepEqual(retained, [current[0]])
  assert.equal(retained[0], current[0])
  assert.equal(selected.length, 4)
})

test('普通刷新保留有效勾选，不能保留内置模板、重复编号或其他页面记录', () => {
  const old = [{ id: 'custom-1', source: 'custom' }, { id: 'mysql', source: 'builtin' }, { id: 'custom-8', source: 'custom' }]
  const current = [{ id: 'custom-1', source: 'custom' }, { id: 'custom-1', source: 'custom' }, { id: 'mysql', source: 'builtin' }]
  assert.deepEqual(retainPageSelection(old, current, row => row.id, row => !templateDeletionReason(row)), [current[0]])
  assert.deepEqual(retainPageSelection(old, [], row => row.id), [])
})

test('导入模板可按本机记录编号删除，内置模板保护保持不变',()=>{
  assert.equal(templateDeletionID({source:'imported',id:'custom-9'}),9)
  assert.equal(templateDeletionReason({source:'imported',id:'custom-9'}),'')
  assert.equal(templateDeletionID({source:'imported',id:'redis'}),null)
})
