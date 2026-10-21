import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

// 用项目已有的 TypeScript 编译器加载纯逻辑，无额外测试依赖。
const source = await readFile(new URL('../src/utils/localImages.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } })
const utilsURL = `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
const { ImageStreamParser, filterImages, sortImages, imageTags, imageArchiveError, imageExportName, formatImageSize, imageUseCount, imageEnvironment, imageLayerCommand, readImageListState, imageListQuery, formatImageDate } = await import(utilsURL)
const images = [
  { Id: 'sha256:abc123', RepoTags: ['nginx:latest', 'registry.local/nginx:1'], RepoDigests: ['nginx@sha256:123456'], Size: 2000, Created: 2 },
  { Id: 'sha256:def456', RepoTags: null, Size: 1000, Created: 3 },
  { Id: 'sha256:xyz789', RepoTags: ['<none>:<none>'], Size: 0, Created: 1 },
]

test('search matches tags, full/short ID, digest and untagged images', () => {
  assert.equal(filterImages(images, ' NGINX ').length, 1)
  assert.equal(filterImages(images, 'def456')[0].Id, 'sha256:def456')
  assert.equal(filterImages(images, 'sha256:123456').length, 1)
  assert.equal(filterImages(images, '未标记').length, 2)
  assert.equal(filterImages(images, 'missing').length, 0)
  assert.equal(filterImages(images, ' ').length, 3)
})
test('sorts numbers numerically without mutating the source', () => {
  assert.deepEqual(sortImages(images, 'Size', false).map(image => image.Size), [0, 1000, 2000])
  assert.deepEqual(sortImages(images, 'Created', true).map(image => image.Created), [3, 2, 1])
  assert.equal(images[0].Created, 2)
  assert.equal(sortImages(images, 'RepoTags', false).length, 3)
})
test('image usage matches immutable IDs and includes stopped containers', () => {
  const containers = [
    {Id:'running', ImageID:'sha256:abc123', State:'running'},
    {Id:'stopped', ImageID:'sha256:abc123', State:'exited'},
    {Id:'retagged', ImageID:'sha256:def456', Image:'nginx:latest', State:'created'},
  ]
  assert.equal(imageUseCount('sha256:abc123', containers), 2)
  assert.equal(imageUseCount('def456', containers), 1)
  assert.equal(imageUseCount('sha256:xyz789', containers), 0)
  assert.equal(imageUseCount('sha256:xyz789', []), 0)
  assert.equal(imageUseCount('sha256:xyz789', null), null)
  assert.equal(imageUseCount('sha256:xyz789', [{Id:'unknown'}]), null)
  assert.equal(imageUseCount('sha256:abc123', [...containers, {Id:'unknown'}]), 2)
})
test('detail fields preserve equals signs and format real Docker history', () => {
  assert.deepEqual(imageEnvironment(['TOKEN=a=b=c','EMPTY=','BARE']), [{name:'TOKEN',value:'a=b=c'},{name:'EMPTY',value:''},{name:'BARE',value:''}])
  assert.deepEqual(imageEnvironment(null), [])
  assert.equal(imageLayerCommand('/bin/sh -c #(nop)  CMD ["nginx"]'), 'CMD ["nginx"]')
  assert.equal(imageLayerCommand('/bin/sh -c echo hello'), 'RUN echo hello')
  assert.equal(imageLayerCommand('COPY . /app # buildkit'), 'COPY . /app # buildkit')
  assert.equal(imageLayerCommand(''), '—')
  assert.equal(formatImageDate('invalid'), '—')
  assert.equal(formatImageDate(new Date(2026, 9, 21, 10, 2, 3).getTime() / 1000), '2026-10-21 10:02:03')
})
test('returning from details preserves validated search, sorting and pagination', () => {
  const state = {search:'nginx',page:3,size:50,sort:'Size',descending:false}
  assert.deepEqual(readImageListState(imageListQuery(state)), state)
  assert.deepEqual(readImageListState({page:-2,size:10000,sort:'invalid',search:['bad']}), {search:'',page:1,size:25,sort:'Created',descending:true})
})
test('archive names and decimal units agree with the list', () => {
  assert.deepEqual(imageTags(images[2]), [])
  assert.equal(imageExportName([images[0]]), 'nginx_latest.tar')
  assert.equal(imageExportName([images[1]]), 'def456.tar')
  assert.equal(imageExportName(images), 'docker-images-3.tar')
  assert.equal(formatImageSize(3400000000), '3.4 GB')
  assert.equal(formatImageSize(0), '0 B')
  assert.equal(formatImageSize(-1), '0 B')
})
test('import validates archive types and rejects empty files', () => {
  for (const name of ['image.tar', 'image.TAR.GZ', 'image.tgz', 'image.tar.bz2', 'image.tar.xz', 'image.tar.zst']) {
    assert.equal(imageArchiveError({name, size:1}), '')
  }
  assert.match(imageArchiveError({name:'image.txt', size:1}), /归档/)
  assert.match(imageArchiveError({name:'image.tar', size:0}), /文件为空/)
})
test('Docker stream handles arbitrary chunks, progress and the final record', () => {
  const logs = []
  const parser = new ImageStreamParser('docker', message => logs.push(message))
  for (const character of '{"id":"layer","status":"Downloading"}\r\n{"status":"Status: Downloaded newer image for nginx:latest"}') parser.push(character)
  parser.finish()
  assert.equal(logs[0], 'layer：下载中')
  assert.equal(logs.at(-1), '镜像拉取完成')
})
test('HTTP 200 containing Docker error or errorDetail fails', () => {
  for (const line of ['{"error":"access denied"}', '{"errorDetail":{"message":"no space left"}}']) {
    const parser = new ImageStreamParser('docker', () => {})
    assert.throws(() => parser.push(`${line}\n`), /access denied|no space left/)
  }
})
test('incomplete or malformed Docker stream cannot report success', () => {
  for (const line of ['', '{"status":"Downloading"}\n', '{"status":', 'null']) {
    const parser = new ImageStreamParser('docker', () => {})
    parser.push(line)
    assert.throws(() => parser.finish(), /中断|不完整/)
  }
})
test('SSE imports require a complete event and handle split CRLF boundaries', () => {
  const logs = []
  const parser = new ImageStreamParser('sse', message => logs.push(message))
  const stream = 'event: log\r\ndata: {"message":"Starting image import..."}\r\n\r\nevent: complete\r\ndata: {"message":"Image import completed successfully"}\r\n\r\n'
  for (let offset = 0; offset < stream.length; offset += 3) parser.push(stream.slice(offset, offset + 3))
  parser.finish()
  assert.deepEqual(logs, ['正在导入镜像…', '镜像导入完成'])
})
test('SSE errors, missing complete and malformed data fail', () => {
  const error = new ImageStreamParser('sse', () => {})
  assert.throws(() => error.push('event: error\ndata: {"message":"invalid archive"}\n\n'), /invalid archive/)
  const incomplete = new ImageStreamParser('sse', () => {})
  incomplete.push('event: log\ndata: {"message":"working"}\n\n')
  assert.throws(() => incomplete.finish(), /中断/)
  const malformed = new ImageStreamParser('sse', () => {})
  assert.throws(() => malformed.push('event: complete\ndata: oops\n\n'), /不完整/)
})

test('transfer API sends auth, force flags and a single multi-image export request', async () => {
  const apiSource = await readFile(new URL('../src/api/localImages.ts', import.meta.url), 'utf8')
  const apiJS = ts.transpileModule(apiSource, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ES2020 } }).outputText
    .replace(/import.*from ['"]@\/stores\/auth['"];?/, 'const useAuthStore = () => ({token:"test-token",logout(){}});')
    .replace(/import.*from ['"]@\/router['"];?/, 'const router = {replace(){}};')
    .replace(/['"]@\/utils\/localImages['"]/, JSON.stringify(utilsURL))
  const { localImagesApi } = await import(`data:text/javascript;base64,${Buffer.from(apiJS).toString('base64')}`)
  const originalFetch = globalThis.fetch
  const calls = []
  globalThis.fetch = async (path, options) => {
    calls.push({path, options})
    assert.equal(options.headers.get('Authorization'), 'Bearer test-token')
    if (path === '/api/containers?all=true') return new Response('[{"Id":"stopped","ImageID":"sha256:abc123","State":"exited"}]')
    if (path.endsWith('/history')) return new Response('[{"Id":"layer","Created":1,"CreatedBy":"CMD nginx","Size":0}]')
    if (path === '/api/images/sha256%3Aabc123') return new Response('{"Id":"sha256:abc123","Config":{"Env":["A=B=C"]}}')
    if (path.endsWith('/pull')) return new Response('{"status":"Status: Image is up to date for nginx:latest"}\n')
    if (path.endsWith('/import')) return new Response('event: complete\ndata: {"message":"镜像导入完成"}\n\n')
    if (path.endsWith('/export')) return new Response('tar-archive', {headers:{'Content-Type':'application/x-tar'}})
    if (options.method === 'DELETE') return new Response('{"message":"镜像删除成功"}')
    return new Response(JSON.stringify(images))
  }
  try {
    const signal = new AbortController().signal
    assert.equal((await localImagesApi.list(signal)).length, 3)
    assert.equal((await localImagesApi.containers(signal))[0].State, 'exited')
    assert.equal((await localImagesApi.inspect(images[0].Id, signal)).Id, images[0].Id)
    assert.equal((await localImagesApi.history(images[0].Id, signal))[0].CreatedBy, 'CMD nginx')
    await localImagesApi.tag(images[0].Id, 'app:latest', signal)
    assert.deepEqual(JSON.parse(calls.at(-1).options.body), {source:images[0].Id,target:'app:latest'})
    await localImagesApi.remove(images[0].Id, false, signal)
    assert.match(calls.at(-1).path, /force=false$/)
    await localImagesApi.remove(images[0].Id, true, signal)
    assert.match(calls.at(-1).path, /force=true$/)
    await localImagesApi.pull(' nginx:latest ', () => {}, signal)
    assert.deepEqual(JSON.parse(calls.at(-1).options.body), {image_name:'nginx:latest'})
    await localImagesApi.import(new File(['archive'], 'image.tar'), () => {}, signal)
    assert.ok(calls.at(-1).options.body instanceof FormData)
    assert.equal(calls.at(-1).options.headers.has('Content-Type'), false)
    assert.equal(await (await localImagesApi.export(images, signal)).text(), 'tar-archive')
    assert.deepEqual(JSON.parse(calls.at(-1).options.body).image_ids, images.map(image => image.Id))
    await localImagesApi.export([images[0]], signal)
    assert.match(calls.at(-1).path, /sha256%3Aabc123\/export$/)
    globalThis.fetch = async () => new Response('{"error":"镜像仍被保留版本引用"}', {status:409})
    await assert.rejects(localImagesApi.remove(images[0].Id, true, signal), /保留版本/)
    globalThis.fetch = async () => new Response('event: error\ndata: {"message":"导入归档损坏"}\n\n')
    await assert.rejects(localImagesApi.import(new File(['bad'], 'bad.tar'), () => {}, signal), /归档损坏/)
  } finally { globalThis.fetch = originalFetch }
})
