import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/networks.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { buildNetworkRequest, emptyNetworkForm, filterNetworks, sortNetworks, networkAddresses, canRemoveNetwork, networkListQuery, readNetworkListState } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const networks = [
  { Id: 'a', Name: 'app-10', Driver: 'bridge', Labels: { 'com.docker.compose.project': 'shop', team: 'platform' }, IPAM: { Driver: 'default', Config: [{ Subnet: '172.20.0.0/16', Gateway: '172.20.0.1' }, { Subnet: 'fd20::/64', Gateway: 'fd20::1' }, { Subnet: '172.21.0.0/16' }] } },
  { Id: 'b', Name: 'app-2', Driver: 'overlay', Attachable: true, IPAM: null },
  { Id: 'c', Name: 'host', Driver: 'host' },
]
test('search covers names, IDs, stacks, labels and both address families; filters combine', () => {
  for (const search of [' APP-10 ', 'shop', 'platform', '172.20.0.1', 'fd20::']) assert.equal(filterNetworks(networks, search)[0].Id, 'a')
  assert.deepEqual(filterNetworks(networks, '', 'overlay', 'custom', 'true').map(network => network.Id), ['b'])
  assert.equal(filterNetworks(networks, '', 'bridge', 'system').length, 0)
  assert.equal(filterNetworks(networks, '', '', 'system')[0].Name, 'host')
})
test('network addresses separate IPv4/IPv6 and retain every IPAM pool', () => {
  assert.equal(networkAddresses(networks[0], 4, 'Subnet'), '172.20.0.0/16, 172.21.0.0/16')
  assert.equal(networkAddresses(networks[0], 6, 'Gateway'), 'fd20::1')
  assert.equal(networkAddresses(networks[1], 4, 'Subnet'), '')
})
test('natural sorting is stable, handles missing IPAM and preserves the input', () => {
  assert.deepEqual(sortNetworks(networks, 'name', false).map(network => network.Name), ['app-2', 'app-10', 'host'])
  assert.equal(sortNetworks(networks, 'ipv6Subnet', true)[0].Id, 'a')
  assert.equal(networks[0].Name, 'app-10')
})
test('system and ingress networks cannot be selected for deletion', () => {
  for (const Name of ['bridge', 'host', 'none']) assert.equal(canRemoveNetwork({ Name }), false)
  assert.equal(canRemoveNetwork({ Name: 'ingress', Ingress: true }), false)
  assert.equal(canRemoveNetwork(networks[0]), true)
})
test('query state survives detail navigation and rejects malformed pagination', () => {
  const state = readNetworkListState({ search: 'app', driver: 'overlay', type: 'custom', attachable: 'true', page: '2', size: '25', sort: 'ipv6Subnet', direction: 'desc' })
  assert.deepEqual(readNetworkListState(networkListQuery(state)), state)
  for (const page of ['-1', 'Infinity', '2.5', '9007199254740992']) assert.equal(readNetworkListState({ page }).page, 1)
  assert.equal(readNetworkListState({ size: '999', sort: 'unknown' }).size, 10)
})
test('automatic addressing omits IPAM instead of submitting unused top-level fields', () => {
  const form = emptyNetworkForm(); form.name = ' my-network '
  assert.deepEqual(buildNetworkRequest(form), { name: 'my-network', driver: 'bridge', options: {}, labels: {}, internal: false, attachable: false, enable_ipv6: false })
})
test('creation sends both address families, range, exclusions, options, labels and flags in Docker shape', () => {
  const form = emptyNetworkForm(); form.name = 'test-network'; form.enableIPv6 = true; form.internal = true; form.attachable = true
  form.ipv4 = { subnet: '172.20.0.0/16', gateway: '172.20.0.1', range: '172.20.10.0/24', excluded: [{ key: 'reserved', value: '172.20.0.10' }] }
  form.ipv6 = { subnet: 'fd20::/64', gateway: 'fd20::1', range: 'fd20::/80', excluded: [{ key: 'reserved', value: 'fd20::10' }] }
  form.options = [{ key: 'com.docker.network.driver.mtu', value: '1400' }]; form.labels = [{ key: 'owner', value: 'platform' }]
  const payload = buildNetworkRequest(form)
  assert.deepEqual(payload.ipam.Config, [{ Subnet: '172.20.0.0/16', Gateway: '172.20.0.1', IPRange: '172.20.10.0/24', AuxiliaryAddresses: { reserved: '172.20.0.10' } }, { Subnet: 'fd20::/64', Gateway: 'fd20::1', IPRange: 'fd20::/80', AuxiliaryAddresses: { reserved: 'fd20::10' } }])
  assert.equal(payload.enable_ipv6, true); assert.equal(payload.internal, true); assert.equal(payload.attachable, true)
  assert.equal(payload.options['com.docker.network.driver.mtu'], '1400'); assert.deepEqual(payload.labels, { owner: 'platform' })
  assert.equal('subnet' in payload, false); assert.equal('gateway' in payload, false)
})
test('invalid and out-of-subnet IPv4 addresses and ranges cannot be submitted', () => {
  for (const values of [{ subnet: '172.20.0.1/16' }, { subnet: '999.0.0.0/16' }, { subnet: '172.20.0.0/33' }, { gateway: '172.20.0.1' }, { subnet: '172.20.0.0/16', gateway: '172.21.0.1' }, { subnet: '172.20.0.0/16', range: '172.20.1.0/8' }, { subnet: '172.20.0.0/16', range: '172.20.1.1/24' }, { subnet: '172.20.0.0/16', excluded: [{ key: 'bad', value: '172.21.0.10' }] }]) {
    const form = emptyNetworkForm(); form.name = 'test'; Object.assign(form.ipv4, values)
    assert.throws(() => buildNetworkRequest(form), /IPv4/)
  }
})
test('IPv6 validates compressed and embedded IPv4 addresses; disabled configuration is omitted', () => {
  const form = emptyNetworkForm(); form.name = 'test'; form.enableIPv6 = true; form.ipv6.subnet = 'fd20::/64'; form.ipv6.gateway = 'fd21::1'
  assert.throws(() => buildNetworkRequest(form), /IPv6/)
  form.ipv6.gateway = 'fd20::1'; assert.equal(buildNetworkRequest(form).ipam.Config.length, 1)
  form.ipv6.subnet = '::ffff:0:0/96'; form.ipv6.gateway = '::ffff:192.168.1.1'; assert.equal(buildNetworkRequest(form).ipam.Config[0].Gateway, '::ffff:192.168.1.1')
  form.enableIPv6 = false; form.ipv6.subnet = 'invalid'; assert.equal(buildNetworkRequest(form).ipam, undefined)
})
test('reserved names, duplicate option keys and missing macvlan parent fail before request', () => {
  for (const name of ['bridge', 'host', 'none', 'ingress', 'bad name', '-bad', '']) { const form = emptyNetworkForm(); form.name = name; assert.throws(() => buildNetworkRequest(form)) }
  const form = emptyNetworkForm(); form.name = 'test'; form.labels = [{ key: 'owner', value: 'one' }, { key: ' owner ', value: 'two' }]
  assert.throws(() => buildNetworkRequest(form), /重复/)
  form.labels = [{ key: '', value: 'orphan' }]; assert.throws(() => buildNetworkRequest(form), /名称/)
  form.labels = []; form.driver = 'macvlan'; assert.throws(() => buildNetworkRequest(form), /parent/)
  form.options = [{ key: 'parent', value: 'eth0' }]; assert.equal(buildNetworkRequest(form).options.parent, 'eth0')
})
