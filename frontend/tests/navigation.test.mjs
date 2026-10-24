import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/navigation.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { workspaceParents, workspaceBreadcrumbs, workspaceBack } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const query = { search: '生产', page: '3', size: '25', order: 'desc' }
const route = (name, path = '/networks/new') => ({ name, path, params: { id: 'resource/with space' }, query, meta: { title: '当前页面' } })
const resolve = path => ({ path: path.split('?')[0], matched: [{}], meta: { requiresAuth: true, title: '来源页面' } })

test('创建和详情页的上级保留查询条件；一级页面不显示后退', () => {
  for (const [name, parent] of [
    ['NetworkCreate', 'Networks'], ['NetworkDetail', 'Networks'], ['VolumeCreate', 'Volumes'], ['VolumeDetail', 'Volumes'],
    ['RegistryCreate', 'Registry'], ['RegistryDetail', 'Registry'], ['ImageDetail', 'Images'],
    ['ApplicationDetail', 'Applications'], ['ApplicationTemplateCreate', 'ApplicationTemplates'], ['DeploymentDetail', 'Deployments'],
  ]) {
    const parents = workspaceParents(route(name))
    assert.equal(parents.length, 1)
    assert.deepEqual(parents[0].to, { name: parent, query })
  }
  assert.equal(workspaceBack(route('Networks', '/networks'), '/images', resolve), undefined)
})

test('三级页面的面包屑包含列表和对应详情，资源名称交给路由编码', () => {
  for (const [name, detail] of [['DeploymentWizard', 'ApplicationDetail'], ['RegistryImages', 'RegistryDetail'], ...['ContainerLogs', 'ContainerInspect', 'ContainerStats', 'ContainerExec', 'ContainerAttach'].map(name => [name, 'ContainerDetail'])]) {
    const parents = workspaceParents(route(name))
    assert.equal(parents.length, 2)
    assert.deepEqual(parents[1].to, { name: detail, params: { id: 'resource/with space' }, query })
  }
})

test('面包屑首页遵循角色；当前项不产生自链接', () => {
  assert.deepEqual(workspaceBreadcrumbs(route('DeploymentWizard'), true).map(item => item.label), ['Dockerun', '应用', '应用详情', '当前页面'])
  assert.deepEqual(workspaceBreadcrumbs(route('Account'), false)[0].to, { name: 'Builtin' })
  assert.equal(workspaceBreadcrumbs(route('Overview'), true)[0].to, undefined)
  assert.equal(workspaceBreadcrumbs(route('DeploymentWizard'), true).at(-1).to, undefined)
})

test('后退优先使用站内来源页及其原始查询，而非强制返回列表', () => {
  assert.deepEqual(workspaceBack(route('ImageDetail', '/images/sha256:abc'), '/containers/c1?search=api&page=2', resolve), {
    to: '/containers/c1?search=api&page=2', label: '返回来源页面', history: true,
  })
})

test('直接打开、外站、登录、未知路由和同页查询历史回退到可用父级', () => {
  const current = route('NetworkCreate')
  const fallback = { to: { name: 'Networks', query }, label: '返回网络', history: false }
  for (const previous of [undefined, null, 3, '', 'https://example.com', '//example.com', '/networks/new?search=old']) {
    assert.deepEqual(workspaceBack(current, previous, resolve), fallback)
  }
  assert.deepEqual(workspaceBack(current, '/login', path => ({ path, matched: [{}], meta: {} })), fallback)
  assert.deepEqual(workspaceBack(current, '/missing', path => ({ path, matched: [], meta: { requiresAuth: true } })), fallback)
  assert.deepEqual(workspaceBack(current, '/invalid', () => { throw new Error('无效路由') }), fallback)
})

test('直接打开三级页面时返回对应详情，且不丢失列表上下文', () => {
  const back = workspaceBack(route('ContainerInspect', '/containers/c1/inspect'), undefined, resolve)
  assert.deepEqual(back, { to: { name: 'ContainerDetail', params: { id: 'resource/with space' }, query }, label: '返回容器详情', history: false })
})
