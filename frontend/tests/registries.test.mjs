import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/utils/registries.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { registryProvider, filterRegistries, sortRegistries, normalizeRegistryUrl, emptyRegistryForm, buildRegistryRequest, readRegistryListState, registryListQuery } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const registries = [
  { id: 1, name: 'registry-10', kind: 'external', url: 'https://registry-1.docker.io', username: 'deploy', is_default: true },
  { id: 2, name: 'registry-2', kind: 'external', url: 'https://private.example.test', username: '', is_default: false },
  { id: 3, name: 'internal', kind: 'builtin', url: 'https://internal.test', username: '', is_default: false },
]
test('provider inference uses exact hostnames and supports explicit ports', () => {
  assert.equal(registryProvider('https://registry-1.docker.io:443/'), 'dockerhub')
  assert.equal(registryProvider('ghcr.io'), 'github')
  assert.equal(registryProvider('https://quay.io'), 'quay')
  assert.equal(registryProvider('https://project.azurecr.io'), 'azure')
  assert.equal(registryProvider('https://quay.io.attacker.test'), 'custom')
})
test('search and combined filters exclude the builtin registry', () => {
  for (const search of [' REGISTRY-10 ', 'deploy', 'Docker Hub']) assert.deepEqual(filterRegistries(registries, search).map(item => item.id), [1])
  assert.deepEqual(filterRegistries(registries, '', 'custom', 'false', 'false').map(item => item.id), [2])
  assert.deepEqual(filterRegistries(registries).map(item => item.id), [1, 2])
  assert.equal(filterRegistries(registries, '', 'dockerhub', 'false').length, 0)
})
test('sorting is natural and does not mutate the input', () => {
  assert.deepEqual(sortRegistries(registries.slice(0, 2), 'name', false).map(item => item.id), [2, 1])
  assert.deepEqual(sortRegistries(registries.slice(0, 2), 'name', true).map(item => item.id), [1, 2])
  assert.equal(registries[0].id, 1)
})
test('registry URLs default to HTTPS and reject embedded secrets, paths and unsupported protocols', () => {
  assert.equal(normalizeRegistryUrl(' registry.example.test:5000/ '), 'https://registry.example.test:5000')
  assert.equal(normalizeRegistryUrl('http://localhost:5000/'), 'http://localhost:5000')
  for (const url of ['', 'ftp://registry.test', 'https://user:secret@registry.test', 'https://registry.test/team/app', 'https://registry.test?token=x', 'https://registry.test#x']) assert.throws(() => normalizeRegistryUrl(url))
})
test('create requires authentication credentials and agent pull accounts require readonly confirmation', () => {
  const form = emptyRegistryForm(); form.name = ' test '; form.url = 'registry.test'; form.authentication = true; form.username = 'user'
  assert.throws(() => buildRegistryRequest(form), /密码/)
  form.password = 'test-secret'
  assert.equal(buildRegistryRequest(form).name, 'test')
  form.pull_username = 'agent'; form.pull_password = 'test-pull'
  assert.throws(() => buildRegistryRequest(form), /只读/)
  form.pull_only_confirmed = true
  assert.equal(buildRegistryRequest(form).pull_username, 'agent')
})
test('edit preserves blank secrets and disabling authentication sends explicit clearing intent', () => {
  const form = emptyRegistryForm(); form.name = 'test'; form.url = 'registry.test'; form.authentication = true; form.username = 'user'
  assert.equal(buildRegistryRequest(form, true).password, '')
  form.authentication = false; form.password = 'stale-secret'
  const payload = buildRegistryRequest(form, true)
  assert.equal(payload.authentication, false); assert.equal(payload.username, ''); assert.equal(payload.password, '')
})
test('filter and pagination state roundtrips through detail links and rejects malformed values', () => {
  const state = readRegistryListState({ search: 'app', type: 'custom', authentication: 'true', defaults: 'false', page: '3', size: '25', sort: 'url', direction: 'desc' })
  assert.deepEqual(readRegistryListState(registryListQuery(state)), state)
  for (const page of ['-1', 'Infinity', '2.5', '9007199254740992']) assert.equal(readRegistryListState({ page }).page, 1)
  assert.equal(readRegistryListState({ size: '999', type: 'unknown', sort: 'unknown' }).size, 10)
})
