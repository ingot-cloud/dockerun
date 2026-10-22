import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/users.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { filterUsers, sortUsers, effectiveRegistryAccess, isLastEnabledAdmin, emptyUserForm, validateUsername, validatePassword, authorizationChanged, buildUserRequest, userDate, userError } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const users = [
  { id: 1, username: 'admin', is_admin: true, enabled: true, registry_access: 'none', created_at: '2026-10-20T00:00:00Z' },
  { id: 2, username: 'developer-10', is_admin: false, enabled: true, registry_access: 'read_write', created_at: '2026-10-21T00:00:00Z' },
  { id: 3, username: 'developer-2', is_admin: false, enabled: false, registry_access: 'read_only', created_at: '2026-10-22T00:00:00Z' },
  { id: 4, username: 'disabled-admin', is_admin: true, enabled: false, registry_access: 'read_only', created_at: '' },
]

test('search ignores case and surrounding whitespace; role, state and effective permission filters combine', () => {
  assert.deepEqual(filterUsers(users, ' DEVELOPER-10 ', 'member', 'enabled', 'read_write').map(user => user.id), [2])
  assert.deepEqual(filterUsers(users, '', 'admin', 'enabled', 'all').map(user => user.id), [1])
  assert.deepEqual(filterUsers(users, '', '', '', 'read_only').map(user => user.id), [3])
  assert.equal(filterUsers(users, 'no-match').length, 0)
  assert.equal(effectiveRegistryAccess(users[0]), 'all')
})
test('sorting handles numeric names and effective permissions without mutating input', () => {
  assert.deepEqual(sortUsers(users, 'username', false).map(user => user.id), [1, 3, 2, 4])
  assert.deepEqual(sortUsers(users, 'access', true).map(user => user.id), [1, 4, 2, 3])
  assert.deepEqual(sortUsers(users, 'created_at', true).map(user => user.id), [3, 2, 1, 4])
  assert.equal(users[1].id, 2)
})
test('a disabled administrator does not count as a recovery account', () => {
  assert.equal(isLastEnabledAdmin(users[0], users), true)
  assert.equal(isLastEnabledAdmin(users[3], users), false)
  assert.equal(isLastEnabledAdmin(users[0], [...users, { ...users[3], enabled: true }]), false)
  for (const key of ['enabled', 'is_admin']) {
    const form = emptyUserForm(users[0]); form[key] = false
    assert.throws(() => buildUserRequest(form, 'authorization', users, users[0]), /最后一个可用管理员/)
  }
})
test('password validation follows backend UTF-8 byte limits and preserves literal spaces', () => {
  assert.match(validatePassword('12345'), /6–72/)
  assert.equal(validatePassword('123456'), '')
  assert.equal(validatePassword('中'.repeat(24)), '')
  assert.match(validatePassword('中'.repeat(25)), /6–72/)
  assert.equal(validatePassword(' '.repeat(6)), '')
  assert.match(validatePassword('a'.repeat(73)), /6–72/)
})
test('username validation trims creation input, rejects duplicates and checks byte limits', () => {
  assert.ok(validateUsername('   ', users))
  assert.match(validateUsername(' admin ', users), /已存在/)
  assert.equal(validateUsername('中'.repeat(33), users), '')
  assert.match(validateUsername('中'.repeat(34), users), /100 字节/)
})
test('creation sends only supported fields and requires matching confirmation', () => {
  const form = emptyUserForm(); form.username = ' new-user '; form.password = 'secret '; form.confirmPassword = form.password
  assert.deepEqual(buildUserRequest(form, 'create', users), { username: 'new-user', password: 'secret ', is_admin: false, registry_access: 'none' })
  form.confirmPassword = 'secret'
  assert.throws(() => buildUserRequest(form, 'create', users), /不一致/)
})
test('password reset cannot accidentally change authorization or account state', () => {
  const form = emptyUserForm(users[0]); form.password = 'new-secret'; form.confirmPassword = form.password; form.enabled = false
  assert.deepEqual(buildUserRequest(form, 'password', users, users[0]), { password: 'new-secret' })
  assert.throws(() => buildUserRequest(form, 'password', users), /不存在/)
})
test('unchanged authorization does not revoke sessions; real changes omit passwords', () => {
  const form = emptyUserForm(users[1])
  assert.equal(authorizationChanged(form, users[1]), false)
  assert.throws(() => buildUserRequest(form, 'authorization', users, users[1]), /未发生更改/)
  form.registry_access = 'read_only'; form.password = 'must-not-be-sent'
  assert.deepEqual(buildUserRequest(form, 'authorization', users, users[1]), { is_admin: false, enabled: true, registry_access: 'read_only' })
})
test('a second enabled administrator permits demotion; unsupported permission values fail', () => {
  const form = emptyUserForm(users[0]); form.is_admin = false
  assert.equal(buildUserRequest(form, 'authorization', [...users, { ...users[3], enabled: true }], users[0]).is_admin, false)
  form.registry_access = 'unknown'
  assert.throws(() => buildUserRequest(form, 'authorization', users, users[0]), /有效的仓库权限/)
})
test('invalid and absent dates render a placeholder', () => {
  for (const value of ['', 'invalid', '0000-00-00']) assert.equal(userDate(value), '—')
  assert.match(userDate(users[0].created_at), /2026/)
})

test('feedback preserves server errors and explains network failures', () => {
  assert.equal(userError({ response: { status: 409, data: { error: '不能停用或降级最后一个可用管理员' } } }), '不能停用或降级最后一个可用管理员')
  assert.match(userError({ code: 'ERR_NETWORK' }), /无法连接服务/)
  assert.match(userError({ code: 'ECONNABORTED' }), /请求超时/)
})
