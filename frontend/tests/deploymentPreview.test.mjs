import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { parse } from 'yaml'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/deploymentPreview.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const compiled=outputText.replaceAll("'yaml'",JSON.stringify(new URL('../node_modules/yaml/dist/index.js',import.meta.url).href))
const {deploymentComposePreview}=await import(`data:text/javascript;base64,${Buffer.from(compiled).toString('base64')}`)
test('解析后的紧凑 JSON 转成 YAML，概览显示实际名称与关键配置，保持原始数据及脱敏内容',()=>{
  const raw={name:'dockerun-a1-e1',services:{redis:{image:'registry.test/redis:8.2.0',container_name:'ingot-redis',ports:[{host_ip:'127.0.0.1',published:'16379',target:6379,protocol:'tcp'},{target:16379,protocol:'tcp'}],networks:{dockerun_external_ingot:{}},volumes:[{type:'volume',source:'data',target:'/data',read_only:true}],configs:[{source:'redis-config',target:'/etc/redis.conf'}],restart:'unless-stopped',healthcheck:{test:['CMD-SHELL','redis-cli ping'],interval:10e9,timeout:'5s',retries:12},environment:{REDIS_PASSWORD:'******'},command:['sh','-c','echo $$REDIS_PASSWORD']}},networks:{dockerun_external_ingot:{name:'ingot-network',external:true}},volumes:{data:{name:'dockerun-a1-e1_data'}},configs:{'redis-config':{file:'/tmp/preview-123/configs/redis.conf'}}}
  const json=JSON.stringify(raw)
  const result=deploymentComposePreview(json,[{reference:'registry.test/redis:8.2.0',digest:'sha256:locked'}])
  assert.deepEqual(parse(result.yaml),raw)
  assert.ok(result.yaml.includes('\n  redis:\n'))
  assert.ok(result.yaml.includes('$$REDIS_PASSWORD'))
  assert.equal(JSON.stringify(raw),json)
  assert.deepEqual(result.services[0],{name:'redis',containerName:'ingot-redis',image:'registry.test/redis:8.2.0',digest:'sha256:locked',imageID:'',ports:['127.0.0.1:16379 → 6379/TCP','容器端口 16379/TCP'],networks:['ingot-network（外部网络）'],mounts:['dockerun-a1-e1_data → /data（只读）'],configs:['redis-config → /etc/redis.conf'],restart:'除非手动停止，否则重启',healthcheck:'已配置 · 间隔 10 秒 · 超时 5s · 重试 12 次',environmentCount:1})
  assert.ok(!JSON.stringify(result.services).includes('preview-123'))
  assert.ok(!JSON.stringify(result.services).includes('REDIS_PASSWORD'))
})
test('当前版本的原始 YAML 保留变量，不用新环境参数插值；兼容短端口、短卷和默认网络',()=>{
  const source='services:\n  app:\n    image: registry.test/app:v1\n    container_name: ${APP_NAME}\n    ports: ["${PORT}:8080"]\n    volumes: ["data:/data"]\n'
  const preview=deploymentComposePreview(source,null)
  assert.equal(preview.services[0].containerName,'${APP_NAME}')
  assert.deepEqual(preview.services[0].ports,['${PORT} → 8080'])
  assert.deepEqual(preview.services[0].networks,['应用默认网络'])
  assert.deepEqual(parse(preview.yaml),parse(source))
})
test('镜像名称与锁定元数据逐项对应，不将下个版本摘要套用到当前版本，不猜测多个同 ID 名称',()=>{
  const source='services:\n  old:\n    image: registry.test/app:v1\n  new:\n    image: registry.test/app:v2\n  id:\n    image: sha256:shared\n'
  const images=[{reference:'registry.test/app:v2',digest:'sha256:new'},{reference:'a:v1',id:'sha256:shared'},{reference:'b:v1',id:'sha256:shared'}]
  const result=deploymentComposePreview(source,images)
  assert.equal(result.services[0].digest,'')
  assert.equal(result.services[1].digest,'sha256:new')
  assert.equal(result.services[2].image,'sha256:shared')
  assert.equal(result.services[2].imageID,'sha256:shared')
})
test('停用健康检查和网络模式明确展示，缺失或无效配置不能生成空的成功概览',()=>{
  const result=deploymentComposePreview('services:\n  app:\n    image: nginx\n    network_mode: host\n    healthcheck: {disable: true}\n')
  assert.deepEqual(result.services[0].networks,['网络模式：host'])
  assert.equal(result.services[0].healthcheck,'已停用')
  for(const source of ['','{"services":','services: {}','services: {app: null}','not compose'])assert.throws(()=>deploymentComposePreview(source))
})
