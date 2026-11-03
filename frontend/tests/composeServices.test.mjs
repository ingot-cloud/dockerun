import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { parse } from 'yaml'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/composeServices.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const yamlURL=new URL('../node_modules/yaml/dist/index.js',import.meta.url).href
const compiled=outputText.replaceAll("'yaml'",JSON.stringify(yamlURL))
const {readComposeServices,setComposeContainerName,setComposeServiceNetworks,restoreComposeFormatting,composeNetworkUnavailableReason}=await import(`data:text/javascript;base64,${Buffer.from(compiled).toString('base64')}`)
const compose=`# 服务说明
services:
  redis:
    image: \${REDIS_IMAGE:?请填写镜像}
    command: [sh, -c, 'exec redis-server --requirepass "$$REDIS_PASSWORD"']
    environment:
      REDIS_PASSWORD: \${REDIS_PASSWORD}
    volumes: ['redis-data:/data']
    configs:
      - source: redis-config
        target: /etc/redis.conf
  web:
    image: nginx:stable
configs:
  redis-config:
    file: ./configs/redis.conf
volumes:
  redis-data: {}
`

test('按服务回填，留空沿用自动名称与默认网络，读取及无变化写入不改写原文',()=>{
  const result=readComposeServices(compose)
  assert.deepEqual(result.services.map(({name,containerName,networks})=>({name,containerName,networks})),[
    {name:'redis',containerName:'',networks:[]},{name:'web',containerName:'',networks:[]},
  ])
  assert.equal(setComposeContainerName(compose,'redis',''),compose)
  assert.equal(setComposeServiceNetworks(compose,'redis',[]),compose)
  assert.equal(result.networks[0].label,'应用默认网络（自动创建）')
})
test('自定义容器名称写回 Compose，保留其他服务、注释、密码变量和非敏感配置文件',()=>{
  const updated=setComposeContainerName(compose,'redis','\${REDIS_CONTAINER_NAME}')
  const expected=parse(compose)
  expected.services.redis.container_name='${REDIS_CONTAINER_NAME}'
  assert.deepEqual(parse(updated),expected)
  assert.match(updated,/# 服务说明/)
  assert.match(updated,/\$\$REDIS_PASSWORD/)
  const reset=setComposeContainerName(updated,'redis','')
  assert.deepEqual(parse(reset),parse(compose))
  assert.equal(readComposeServices(updated).services[0].containerName,'${REDIS_CONTAINER_NAME}')
})
test('选择网络管理中的网络生成外部网络声明，保留已选网络的别名和固定地址',()=>{
  const original=`services:
  api:
    image: nginx
    networks:
      private:
        aliases: [backend]
        ipv4_address: 172.30.0.2
networks:
  private:
    driver: bridge
    ipam:
      config: [{subnet: 172.30.0.0/24}]
`
  const changed=setComposeServiceNetworks(original,'api',['compose:private','external:ingot-network'])
  const raw=parse(changed)
  const added=Object.keys(raw.networks).find(key=>key!=='private')
  assert.deepEqual(raw.networks[added],{name:'ingot-network',external:true,'x-dockerun-managed':true})
  assert.deepEqual(raw.services.api.networks.private,parse(original).services.api.networks.private)
  assert.equal(raw.services.api.networks[added],null)
  assert.deepEqual(raw.networks.private,parse(original).networks.private)
  assert.deepEqual(readComposeServices(changed).services[0].networks,['compose:private',`compose:${added}`])
  assert.equal(readComposeServices(changed).networks.find(network=>network.value===`compose:${added}`).label,'ingot-network（外部网络）')
  const restored=setComposeServiceNetworks(changed,'api',['compose:private'])
  assert.deepEqual(parse(restored),parse(original))
})
test('多个服务复用同一外部网络，未解除全部引用前不删除声明，用户自己的声明始终保留',()=>{
  const first=setComposeServiceNetworks(compose,'redis',['external:team.net'])
  const second=setComposeServiceNetworks(first,'web',['external:team.net'])
  const key=Object.keys(parse(first).networks)[0]
  assert.equal(Object.keys(parse(second).networks).length,1)
  const third=setComposeServiceNetworks(second,'redis',[])
  assert.ok(parse(third).networks[key])
  const fourth=setComposeServiceNetworks(third,'web',[])
  assert.deepEqual(parse(fourth),parse(compose))
  const manual='services:\n  app:\n    image: redis\n    networks: [shared]\nnetworks:\n  shared: {external: true, name: shared}\n'
  const reused=setComposeServiceNetworks(manual,'app',['external:shared'])
  assert.deepEqual(Object.keys(parse(reused).networks),['shared'])
  assert.ok(parse(setComposeServiceNetworks(reused,'app',[])).networks.shared)
})
test('表单改回原值时恢复原始 Compose 格式，不误报未保存；真实修改及手写注释保留',()=>{
  const named=setComposeContainerName(compose,'redis','ingot-redis')
  assert.equal(restoreComposeFormatting(named,compose),named)
  const restored=setComposeContainerName(named,'redis','')
  assert.equal(restoreComposeFormatting(restored,compose),compose)
  const networked=setComposeServiceNetworks(compose,'redis',['external:ingot-network'])
  assert.equal(restoreComposeFormatting(setComposeServiceNetworks(networked,'redis',[]),compose),compose)
  const manual=compose+'# 手写的新说明\n'
  const changed=setComposeContainerName(manual,'redis','ingot-redis')
  assert.equal(restoreComposeFormatting(setComposeContainerName(changed,'redis',''),manual),manual)
})
test('网络键冲突时创建独立键，远程输入按网络名保存而非本机 ID',()=>{
  const original=compose+'networks:\n  dockerun_external_shared: {driver: bridge}\n'
  const changed=setComposeServiceNetworks(original,'redis',['shared'])
  const raw=parse(changed)
  assert.equal(raw.networks.dockerun_external_shared.driver,'bridge')
  assert.equal(raw.networks.dockerun_external_shared_2.name,'shared')
  assert.deepEqual(raw.services.redis.networks,['dockerun_external_shared_2'])
  const emptyDefinition=setComposeServiceNetworks(compose+'networks:\n','redis',['external:shared'])
  assert.equal(Object.values(parse(emptyDefinition).networks)[0].name,'shared')
})
test('网络模式、锚点、合并与不完整 YAML 明确限制编辑，不覆盖原配置',()=>{
  const host='services:\n  app:\n    image: nginx\n    network_mode: host\n'
  assert.match(readComposeServices(host).services[0].networkEditingReason,/网络模式 host/)
  assert.throws(()=>setComposeServiceNetworks(host,'app',['external:test']),/网络模式/)
  assert.equal(parse(setComposeContainerName(host,'app','gateway')).services.app.container_name,'gateway')
  const aliases='services:\n  first: &base\n    image: nginx\n  second: *base\n'
  for(const name of ['first','second'])assert.throws(()=>setComposeContainerName(aliases,name,'custom'),/锚点/)
  const merged='x-base: &base\n  image: nginx\nservices:\n  app:\n    <<: *base\n'
  assert.throws(()=>setComposeContainerName(merged,'app','custom'),/合并/)
  for(const invalid of ['services: [','services: []','services: {}','services:\n  app: null'])assert.throws(()=>readComposeServices(invalid))
  assert.throws(()=>setComposeContainerName(compose,'missing','custom'),/服务已变化/)
  assert.throws(()=>setComposeServiceNetworks(compose,'redis',['compose:missing']),/网络已变化/)
})
test('普通自定义 bridge 可选，系统网络、入口及未开放独立容器连接的 Swarm 网络不可选',()=>{
  assert.equal(composeNetworkUnavailableReason({Name:'team',Driver:'bridge',Attachable:false}),'')
  assert.equal(composeNetworkUnavailableReason({Name:'overlay',Driver:'overlay',Scope:'swarm',Attachable:true}),'')
  for(const network of [{Name:'bridge',Driver:'bridge'},{Name:'host',Driver:'host'},{Name:'none',Driver:'null'},{Name:'ingress',Ingress:true},{Name:'overlay',Scope:'swarm',Attachable:false}])assert.notEqual(composeNetworkUnavailableReason(network),'')
})
