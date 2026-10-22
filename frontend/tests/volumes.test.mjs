import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/utils/volumes.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { volumeReferences, volumeUseCount, filterVolumes, sortVolumes, displayVolumeOption, readVolumeListState, volumeListQuery, emptyVolumeForm, buildVolumeRequest } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const volumes = [
  { Name: 'data-10', Driver: 'local', Mountpoint: '/var/lib/docker/volumes/data-10/_data', CreatedAt: '2026-01-01T00:00:00+08:00', Labels: { 'com.docker.compose.project': 'shop', owner: 'platform' } },
  { Name: 'data-2', Driver: 'plugin', Mountpoint: '/mnt/data-2', CreatedAt: '2025-12-31T17:00:00Z' },
  { Name: 'unused', Driver: 'local', Mountpoint: '/mnt/unused' },
]
const containers = [
  { Id: 'stopped', Names: ['/db'], State: 'exited', Mounts: [{ Type: 'volume', Name: 'data-10', Destination: '/data', RW: false }, { Type: 'volume', Name: 'data-10', Destination: '/backup', RW: true }] },
  { Id: 'running', Names: ['/web'], State: 'running', Mounts: [{ Type: 'volume', Name: 'data-2', Destination: '/app', RW: true }] },
  { Id: 'bind', Mounts: [{ Type: 'bind', Source: '/var/lib/docker/volumes/data-10/_data', Destination: '/bind' }] },
]
test('volume references match exact named volumes, include stopped containers and preserve each mount', () => {
  assert.deepEqual(volumeReferences(volumes[0], containers), [{ id: 'stopped', name: 'db', state: 'exited', destination: '/data', readOnly: true }, { id: 'stopped', name: 'db', state: 'exited', destination: '/backup', readOnly: false }])
  assert.equal(volumeUseCount(volumes[0], containers), 1)
  assert.equal(volumeUseCount(volumes[2], containers), 0)
  assert.equal(volumeReferences(volumes[1], containers)[0].name, 'web')
})
test('legacy mounts fall back to exact source paths and never match bind mounts or prefixes', () => {
  const old = [{ Id: 'legacy', Mounts: [{ Type: 'volume', Source: volumes[0].Mountpoint }] }]
  assert.equal(volumeUseCount(volumes[0], old), 1)
  assert.equal(volumeReferences(volumes[0], old)[0].readOnly, null)
  assert.equal(volumeUseCount(volumes[0], [{ Id: 'other', Mounts: [{ Type: 'volume', Name: 'data-100', Source: volumes[0].Mountpoint }] }]), 0)
})
test('failed or incomplete container metadata never marks a volume unused', () => {
  assert.equal(volumeUseCount(volumes[2], null), null)
  assert.equal(volumeUseCount(volumes[2], [{ Id: 'unknown' }]), null)
  assert.equal(volumeUseCount(volumes[0], [...containers, { Id: 'unknown' }]), 1)
  assert.equal(volumeUseCount(volumes[0], [{ Id: 'none', Mounts: null }]), 0)
})
test('search covers names, driver, path, stack and labels; usage and driver filters combine', () => {
  for (const search of [' DATA-10 ', 'SHOP', 'platform', '_data']) assert.equal(filterVolumes(volumes, containers, search)[0].Name, 'data-10')
  assert.deepEqual(filterVolumes(volumes, containers, '', 'local', 'unused').map(volume => volume.Name), ['unused'])
  assert.equal(filterVolumes(volumes, containers, '', '', 'used').length, 2)
  assert.equal(filterVolumes(volumes, null, '', '', 'unused').length, 0)
  assert.equal(filterVolumes(volumes, null, '', '', 'unknown').length, 3)
})
test('natural name and chronological date sorting do not mutate the input', () => {
  assert.deepEqual(sortVolumes(volumes, 'Name', false).map(volume => volume.Name), ['data-2', 'data-10', 'unused'])
  assert.equal(sortVolumes(volumes, 'CreatedAt', true)[0].Name, 'data-2')
  assert.equal(volumes[0].Name, 'data-10')
})
test('detail navigation preserves validated filters and pagination', () => {
  const state = readVolumeListState({ search: 'data', driver: 'local', usage: 'unused', page: '2', size: '25', sort: 'CreatedAt', direction: 'desc' })
  assert.deepEqual(readVolumeListState(volumeListQuery(state)), state)
  for (const page of ['-1', 'Infinity', '2.5', '9007199254740992']) assert.equal(readVolumeListState({ page }).page, 1)
  assert.equal(readVolumeListState({ size: '999', usage: 'invalid', sort: 'unknown' }).usage, '')
})
test('local volumes support generated names, labels and arbitrary driver options', () => {
  const form = emptyVolumeForm()
  assert.deepEqual(buildVolumeRequest(form), { name: '', driver: 'local', options: {}, labels: {} })
  form.name = ' my-data '; form.options = [{ key: 'type', value: 'tmpfs' }, { key: 'device', value: 'tmpfs' }, { key: 'o', value: 'size=100m,uid=1000' }]; form.labels = [{ key: 'owner', value: 'platform' }]
  assert.deepEqual(buildVolumeRequest(form), { name: 'my-data', driver: 'local', options: { type: 'tmpfs', device: 'tmpfs', o: 'size=100m,uid=1000' }, labels: { owner: 'platform' } })
})
test('NFS maps to Docker driver options and ignores hidden custom fields', () => {
  const form = emptyVolumeForm(); form.mode = 'nfs'; form.nfs.address = 'nas.example.com'; form.nfs.path = '/export/data'; form.options = [{ key: '', value: 'stale' }]
  const request = buildVolumeRequest(form)
  assert.deepEqual(request.options, { type: 'nfs4', device: ':/export/data', o: 'addr=nas.example.com,rw,noatime,rsize=8192,wsize=8192,tcp,timeo=14' })
  form.nfs.version = 'nfs'; form.nfs.options = 'ro'; assert.equal(buildVolumeRequest(form).options.type, 'nfs')
  form.nfs.path = 'relative'; assert.throws(() => buildVolumeRequest(form), /绝对路径/)
})
test('CIFS maps share and version, preserves password whitespace, and masks credentials on display', () => {
  const form = emptyVolumeForm(); form.mode = 'cifs'; form.cifs = { address: '192.0.2.1', share: '/shared/data', version: '3.0', username: 'demo', password: ' example ' }
  assert.deepEqual(buildVolumeRequest(form).options, { type: 'cifs', device: '//192.0.2.1/shared/data', o: 'addr=192.0.2.1,vers=3.0,username=demo,password= example ' })
  assert.equal(displayVolumeOption('o', buildVolumeRequest(form).options.o).includes('example'), false)
  assert.equal(displayVolumeOption('password', 'example'), '••••••')
  assert.equal(displayVolumeOption('o', 'password=example,uid=1000'), 'password=••••••,uid=1000')
  form.cifs.password = 'comma,password'; assert.throws(() => buildVolumeRequest(form), /逗号/)
})
test('invalid names, remote settings and duplicate option keys fail before submission', () => {
  for (const name of ['a', 'bad name', '-bad', '../outside']) { const form = emptyVolumeForm(); form.name = name; assert.throws(() => buildVolumeRequest(form), /卷名称/) }
  const form = emptyVolumeForm(); form.options = [{ key: 'type', value: 'tmpfs' }, { key: ' type ', value: 'nfs' }]; assert.throws(() => buildVolumeRequest(form), /重复/)
  form.options = [{ key: '', value: 'orphan' }]; assert.throws(() => buildVolumeRequest(form), /名称/)
  form.mode = 'nfs'; form.nfs.address = 'https://nas.example.com'; assert.throws(() => buildVolumeRequest(form), /服务器/)
  form.nfs.address = 'nas.example.com'; form.nfs.path = '/export'; form.nfs.options = 'addr=other'; assert.throws(() => buildVolumeRequest(form), /addr/)
  form.driver = 'plugin'; assert.throws(() => buildVolumeRequest(form), /local/)
})
