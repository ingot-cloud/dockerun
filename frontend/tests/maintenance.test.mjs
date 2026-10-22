import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/utils/maintenance.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const { needsMaintenancePolling, maintenanceStorage, maintenanceTransport, filterMaintenanceJobs, filterMaintenanceAudit, maintenanceError } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const jobs = [
 { id: 1, status: 'queued', file: '', error: '', user_id: 4 },
 { id: 2, status: 'ready', file: '20261022.tar.gz', error: '', user_id: 4 },
 { id: 3, status: 'failed', file: '', error: '存储空间不足', user_id: 8 },
 { id: 4, status: 'success', file: '', error: '', user_id: 1 },
]
test('only active jobs or an active storage operation need polling regardless of the visible filter', () => {
 assert.equal(needsMaintenancePolling([], [], ''), false)
 assert.equal(needsMaintenancePolling(jobs.slice(1), [], ''), false)
 assert.equal(needsMaintenancePolling(jobs, [], ''), true)
 assert.equal(needsMaintenancePolling([], [{ status: 'running' }], ''), true)
 assert.equal(needsMaintenancePolling([], [], 'backup'), true)
})
test('storage units distinguish unavailable data from zero and label binary units correctly', () => {
 assert.equal(maintenanceStorage(null), '—')
 assert.equal(maintenanceStorage(undefined), '—')
 assert.equal(maintenanceStorage(NaN), '—')
 assert.equal(maintenanceStorage(-1), '—')
 assert.equal(maintenanceStorage(0), '0 B')
 assert.equal(maintenanceStorage(1024 ** 3), '1.0 GiB')
 assert.equal(maintenanceStorage(1024 ** 2 * 1.5), '1.5 MiB')
})
test('transport distinguishes direct TLS, HTTPS proxy entry and plain HTTP', () => {
 assert.equal(maintenanceTransport({ tls: true, public_url: 'https://registry.test' }), '服务端 TLS')
 assert.equal(maintenanceTransport({ tls: false, public_url: 'https://registry.test' }), 'HTTPS 入口 / 代理 TLS')
 assert.equal(maintenanceTransport({ tls: false, public_url: 'http://localhost:8080' }), 'HTTP')
})
test('task filters combine status and searchable failure metadata and include both completion states', () => {
 assert.deepEqual(filterMaintenanceJobs(jobs, '', 'complete').map(job => job.id), [2, 4])
 assert.deepEqual(filterMaintenanceJobs(jobs, ' 存储空间 ', 'failed').map(job => job.id), [3])
 assert.deepEqual(filterMaintenanceJobs(jobs, '', 'active').map(job => job.id), [1])
 assert.deepEqual(filterMaintenanceJobs(jobs, '20261022', '').map(job => job.id), [2])
 assert.equal(jobs[0].id, 1)
})
test('audit search covers translated actions, raw names, resources and users without hiding unknown actions', () => {
 const events = [{ action: 'registry.gc.complete', resource: 'job=5', user_id: 2 }, { action: 'backup.create', resource: 'job=6', user_id: 3 }, { action: 'custom.action', resource: 'future', user_id: 4 }]
 assert.equal(filterMaintenanceAudit(events, ' 清理 ', 'registry').length, 1)
 assert.equal(filterMaintenanceAudit(events, 'job=6', '').length, 1)
 assert.equal(filterMaintenanceAudit(events, 'future', '').length, 1)
 assert.equal(filterMaintenanceAudit(events, '清理', 'backup').length, 0)
})
test('maintenance errors preserve server reasons and handle unavailable downloads', () => {
 assert.equal(maintenanceError({ response: { status: 409, data: { error: '清理任务正在执行' } } }), '清理任务正在执行')
 assert.match(maintenanceError({ response: { status: 404 } }), /备份文件/)
 assert.match(maintenanceError({ code: 'ECONNABORTED' }), /刷新任务状态/)
 assert.doesNotMatch(maintenanceError({ response: { status: 500 } }), /Request failed/)
})
