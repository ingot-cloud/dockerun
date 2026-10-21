import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/utils/containers.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { canContainerAction, containerIPs, containerMetrics, containerName, containerStack, containerStateLabel, filterContainers, sortContainers, publishedPorts, containerPortURL, parseContainerCommand, readContainerListState, containerListQuery } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const containers = [
  { Id: 'abc123', Names: ['/web-10'], Image: 'nginx:latest', State: 'running', Created: 10, Labels: { 'com.docker.compose.project': 'shop' }, NetworkSettings: { Networks: { frontend: { IPAddress: '172.18.0.2' }, backend: { IPAddress: '10.0.0.5' } } }, Ports: [{ PublicPort: 8080, PrivatePort: 80, Type: 'tcp' }] },
  { Id: 'def456', Names: ['/web-2'], Image: 'redis:7', State: 'exited', Created: 20, Labels: { 'com.docker.stack.namespace': 'cache' }, Ports: null },
  { Id: 'ghi789', Names: null, Image: 'alpine:latest', State: 'paused', Created: 5, NetworkSettings: { Networks: null } },
]

test('search includes Chinese state, ID, stacks, all networks and mapped ports', () => {
  for (const query of [' NGINX ', 'ABC123', 'shop', '10.0.0.5', '8080', '运行中']) assert.equal(filterContainers(containers, query, '')[0].Id, 'abc123')
  assert.equal(filterContainers(containers, 'cache', '')[0].Id, 'def456')
  assert.equal(filterContainers(containers, '', 'exited').length, 1)
  assert.equal(filterContainers(containers, 'nginx', 'exited').length, 0)
  assert.equal(filterContainers(containers, 'missing', '').length, 0)
})
test('names sort naturally and numeric dates sort without mutating source', () => {
  assert.deepEqual(sortContainers(containers.slice(0, 2), 'name', false).map(containerName), ['web-2', 'web-10'])
  assert.deepEqual(sortContainers(containers, 'Created', true).map(container => container.Created), [20, 10, 5])
  assert.equal(containers[0].Id, 'abc123')
  assert.equal(containerName(containers[2]), 'ghi789')
  assert.equal(containerStack(containers[2]), '')
  assert.equal(containerStateLabel('unknown'), '未知')
})
test('IP addresses include custom and IPv6 networks, remove duplicates, tolerate no network', () => {
  assert.deepEqual(containerIPs({ NetworkSettings: { Networks: { a: { IPAddress: '10.0.0.1', GlobalIPv6Address: 'fd00::1' }, b: { IPAddress: '10.0.0.1' } } } }), ['10.0.0.1', 'fd00::1'])
  assert.deepEqual(containerIPs(containers[2]), [])
})
test('bulk actions never start active containers or remove running/paused containers', () => {
  const expected = { running: [false, true, true, false], exited: [true, false, true, true], created: [true, false, false, true], paused: [false, false, false, false], restarting: [false, true, false, false], removing: [false, false, false, false], dead: [false, false, false, true] }
  for (const [State, eligibility] of Object.entries(expected)) assert.deepEqual(['start', 'stop', 'restart', 'remove'].map(action => canContainerAction({ State }, action)), eligibility, State)
})
test('metrics use CPU deltas, cgroup cache-adjusted memory and all network interfaces', () => {
  const stats = { cpu_stats: { cpu_usage: { total_usage: 300 }, system_cpu_usage: 1000, online_cpus: 4 }, precpu_stats: { cpu_usage: { total_usage: 100 }, system_cpu_usage: 500 }, memory_stats: { usage: 1000, limit: 2000, stats: { inactive_file: 200 } }, networks: { eth0: { rx_bytes: 100, tx_bytes: 200 }, eth1: { rx_bytes: 300, tx_bytes: 400 } }, pids_stats: { current: 5 } }
  assert.deepEqual(containerMetrics(stats), { cpu: 160, memory: 800, limit: 2000, memoryPercent: 40, received: 400, sent: 600, pids: 5, cache: 200, read: 0, written: 0 })
  stats.memory_stats.stats = { total_inactive_file: 300 }
  assert.equal(containerMetrics(stats).memory, 700)
  stats.memory_stats.stats = { inactive_file: 2000 }
  assert.equal(containerMetrics(stats).memory, 1000)
})
test('missing statistics produce unavailable CPU and no NaN or Infinity', () => {
  assert.deepEqual(containerMetrics({}), { cpu: null, memory: 0, limit: 0, memoryPercent: 0, received: 0, sent: 0, pids: null, cache: 0, read: 0, written: 0 })
  assert.equal(containerMetrics({ cpu_stats: { cpu_usage: { total_usage: 1 }, system_cpu_usage: 0 } }).cpu, null)
})


test('pause, resume and kill follow container state', () => {
  assert.equal(canContainerAction({ State: 'running' }, 'pause'), true)
  assert.equal(canContainerAction({ State: 'paused' }, 'resume'), true)
  assert.equal(canContainerAction({ State: 'running' }, 'resume'), false)
  assert.equal(canContainerAction({ State: 'paused' }, 'kill'), true)
  assert.equal(canContainerAction({ State: 'exited' }, 'kill'), false)
})
test('published ports deduplicate IPv4/IPv6 bindings and omit unexposed ports', () => {
  const fixture = { Ports: [{ IP: '0.0.0.0', PublicPort: 8181, PrivatePort: 8080, Type: 'tcp' }, { IP: '::', PublicPort: 8181, PrivatePort: 8080, Type: 'tcp' }, { PrivatePort: 8000, Type: 'tcp' }, { IP: '0.0.0.0', PublicPort: 9848, PrivatePort: 9848, Type: 'tcp' }] }
  assert.equal(publishedPorts(fixture).length, 2)
  assert.equal(containerPortURL(fixture.Ports[0], 'localhost'), 'http://localhost:8181')
  assert.equal(containerPortURL({ IP: 'fd00::1', PublicPort: 80 }, 'localhost'), 'http://[fd00::1]:80')
})
test('console command parser preserves quoted arguments without a host shell', () => {
  assert.deepEqual(parseContainerCommand('/bin/bash'), ['/bin/bash'])
  assert.deepEqual(parseContainerCommand(`/bin/sh -c 'printf "hello world"'`), ['/bin/sh', '-c', 'printf "hello world"'])
  assert.deepEqual(parseContainerCommand(`printf "" 'a b'`), ['printf', '', 'a b'])
  assert.throws(() => parseContainerCommand(`sh -c 'unfinished`), /不完整/)
  assert.throws(() => parseContainerCommand('   '), /请填写/)
})
test('details and quick pages preserve validated list filters, sort and page', () => {
  const state = { search: 'nginx', state: 'running', page: 3, size: 50, sort: 'Created', descending: true }
  assert.deepEqual(readContainerListState(containerListQuery(state)), state)
  assert.deepEqual(readContainerListState({ page: -1, size: 999, sort: 'invalid', state: 'invalid', search: ['no'] }), { search: '', state: '', page: 1, size: 25, sort: 'name', descending: false })
})
test('disk usage aggregates all devices and both Docker operation spellings', () => {
  const metrics = containerMetrics({ blkio_stats: { io_service_bytes_recursive: [{ op: 'Read', value: 12 }, { op: 'read', value: 4 }, { op: 'Write', value: 8 }, { op: 'Total', value: 100 }] } })
  assert.equal(metrics.read, 16)
  assert.equal(metrics.written, 8)
})
