import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/utils/builtin.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { registryConnectionHelp, registryLoginCommand, registryHost, imageReference, latestImageEvents, filterBuiltinRepositories, builtinPermission, builtinError, manifestSize } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('client references preserve ports and IPv6 and distinguish digests from tags', () => {
 assert.equal(registryHost('https://registry.example.test:5443/', 'fallback'), 'registry.example.test:5443')
 assert.equal(registryHost('http://[::1]:8080', 'fallback'), '[::1]:8080')
 assert.equal(registryHost('invalid', 'localhost:8080'), 'localhost:8080')
 assert.equal(imageReference('localhost:8080', 'team/app', 'v1'), 'localhost:8080/team/app:v1')
 assert.equal(imageReference('localhost:8080', 'team/app', 'sha256:abc'), 'localhost:8080/team/app@sha256:abc')
})
test('latest submission metadata is independent of event order', () => {
 const events = [{ id: 2, repository: 'team/app', reference: 'v2', created_at: '2026-10-22T02:00:00Z' }, { id: 1, repository: 'team/app', reference: 'v1', created_at: '2026-10-21T02:00:00Z' }, { id: 3, repository: 'team/app', reference: 'v3', created_at: '2026-10-22T02:00:00Z' }]
 assert.equal(latestImageEvents(events).get('team/app').reference, 'v3')
 assert.equal(events[0].id, 2)
})
test('repository search combines namespace filters with natural sorting without mutation', () => {
 const names = ['team/app10', 'team/app2', 'other/app', 'root']
 const latest = new Map([['team/app2', { created_at: '2026-10-22T02:00:00Z' }]])
 assert.deepEqual(filterBuiltinRepositories(names, ' APP ', 'team', 'name', false, latest), ['team/app2', 'team/app10'])
 assert.deepEqual(filterBuiltinRepositories(names, '', '__root__', 'name', false, latest), ['root'])
 assert.equal(filterBuiltinRepositories(names, '', '', 'updated', true, latest)[0], 'team/app2')
 assert.equal(names[0], 'team/app10')
})
test('accounts without access are never labeled readonly', () => {
 assert.equal(builtinPermission(false, 'none'), '无访问权限')
 assert.equal(builtinPermission(false, 'read_only'), '只读')
 assert.equal(builtinPermission(false, 'read_write'), '读写')
 assert.equal(builtinPermission(true, 'none'), '管理员')
})
test('structured registry errors and unavailable service states are readable', () => {
 assert.equal(builtinError({ response: { status: 409, data: { error: '当前发布版本受保护' } } }), '当前发布版本受保护')
 assert.equal(builtinError({ response: { status: 400, data: { catalog: { errors: [{ message: 'invalid pagination number' }] } } } }), 'invalid pagination number')
 assert.match(builtinError({ response: { status: 503 } }), /维护/)
 assert.match(builtinError({ response: { status: 404 } }), /已不存在/)
 assert.doesNotMatch(builtinError({ response: { status: 400 } }), /Request failed/)
})
test('content size includes image config and compressed layers but does not mislabel index sizes', () => {
 assert.equal(manifestSize({ manifest: { config: { size: 512 }, layers: [{ size: 512 }] } }), '1.0 KB')
 assert.equal(manifestSize({ manifest: { manifests: [{ size: 1024 }] } }), '多平台镜像')
})
test('Docker Desktop connection help preserves the port and registry token origin', () => {
 const result = registryConnectionHelp('http://localhost:8080/')
 assert.equal(result.http, true)
 assert.equal(result.loopback, true)
 assert.equal(result.desktopHost, 'host.docker.internal:8080')
 assert.equal(result.desktopUrl, 'http://host.docker.internal:8080')
 assert.deepEqual(JSON.parse(result.daemonConfig), { 'insecure-registries': ['host.docker.internal:8080'] })
 for (const url of ['http://127.0.0.1:8080', 'http://[::1]:8080', 'http://localhost.:8080']) assert.equal(registryConnectionHelp(url).loopback, true)
 assert.equal(registryConnectionHelp('https://registry.example.test').http, false)
 assert.equal(registryConnectionHelp('https://registry.example.test').loopback, false)
 assert.deepEqual(JSON.parse(registryConnectionHelp('http://registry.example.test:5000').daemonConfig), { 'insecure-registries': ['registry.example.test:5000'] })
 assert.equal(registryConnectionHelp('invalid').desktopUrl, '')
})
test('login commands use the account name and quote shell metacharacters without embedding a password', () => {
 assert.equal(registryLoginCommand('host.docker.internal:8080', 'admin'), 'docker login -u admin host.docker.internal:8080')
 assert.equal(registryLoginCommand('localhost:8080', ''), 'docker login localhost:8080')
 assert.equal(registryLoginCommand('localhost:8080', 'user; exit'), "docker login -u 'user; exit' localhost:8080")
 assert.equal(registryLoginCommand('localhost:8080', "user'name"), `docker login -u 'user'"'"'name' localhost:8080`)
})
