import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/applicationDetail.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {applicationDetailTab,applicationSetupStep,environmentDeletionReason,environmentTargetConfiguration,environmentParameterPairs,environmentParameterValues,isSensitiveEnvironmentParameter,readOnlyEnvironmentParameters,applicationRuntimeImages,runtimeImageLocation}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
test('应用详情视图可从网址恢复，异常参数回到部署环境',()=>{
  for(const value of [undefined,null,'bad',['releases'],'environments']) assert.equal(applicationDetailTab(value),'environments')
  assert.equal(applicationDetailTab('releases'),'releases')
  assert.equal(applicationDetailTab('configuration'),'configuration')
})
test('版本部署只读展示环境参数，敏感名称及模板密码脱敏，保留已保存的值和原对象',()=>{
  const parameters={PORT:' 8080 ',PASSWORD:'server-secret',CUSTOM:'template-secret',EMPTY:''}
  assert.deepEqual(readOnlyEnvironmentParameters(parameters,[{key:'CUSTOM',type:'password'}]),{PORT:' 8080 ',PASSWORD:'******',CUSTOM:'******',EMPTY:''})
  assert.equal(parameters.PASSWORD,'server-secret')
  assert.equal(parameters.CUSTOM,'template-secret')
  assert.deepEqual(readOnlyEnvironmentParameters(),{})
})
test('首次创建指引随环境及实际主机发布状态推进',()=>{
  assert.equal(applicationSetupStep([]),1)
  assert.equal(applicationSetupStep([{current_release_id:0}]),2)
  assert.equal(applicationSetupStep([{current_release_id:3}]),3)
  assert.equal(applicationSetupStep([{current_release_id:0}],[{current_release_id:4}]),3)
  assert.equal(applicationSetupStep([{current_release_id:0}],[{current_release_id:0}]),2)
})
test('部署环境删除遵循服务端任务保护，任务状态缺失时禁止删除',()=>{
  assert.equal(environmentDeletionReason({deletion_blocked_reason:''}),'')
  assert.equal(environmentDeletionReason({deletion_blocked_reason:'环境存在排队、执行或恢复中的部署任务，结束后才可删除'}),'环境存在排队、执行或恢复中的部署任务，结束后才可删除')
  assert.ok(environmentDeletionReason({}))
})
test('统一部署目标兼容单机和按选择顺序逐台部署，不修改选择内容',()=>{
  assert.deepEqual(environmentTargetConfiguration([2]),{target_id:2,target_ids:'[]'})
  const targets=[3,1,2]
  assert.deepEqual(environmentTargetConfiguration(targets),{target_id:3,target_ids:'[3,1,2]'})
  assert.deepEqual(targets,[3,1,2])
  assert.deepEqual(environmentTargetConfiguration([2],true),{target_id:2,target_ids:'[]'})
})
test('部署目标必须有效且不重复，单机模板拒绝多个目标',()=>{
  assert.throws(()=>environmentTargetConfiguration([]),/请选择部署目标/)
  for(const targets of [[0],[-1],[1.5],[NaN],[Number.MAX_SAFE_INTEGER+1]])assert.throws(()=>environmentTargetConfiguration(targets),/有效的部署目标/)
  assert.throws(()=>environmentTargetConfiguration([2,2]),/不能重复/)
  assert.throws(()=>environmentTargetConfiguration([1,2],true),/仅支持单台/)
})
test('运行参数回填不修改原配置，保存保留空值、空白、等号及特殊字符',()=>{
  const original={DB_HOST:'test-db',PASSWORD:'  p=a$$;"中文"\nnext  ',EMPTY:'',ZERO:'0'}
  const pairs=environmentParameterPairs(original)
  assert.deepEqual(environmentParameterValues(pairs),original)
  pairs[0].value='prod-db'
  assert.equal(original.DB_HOST,'test-db')
  const updated=environmentParameterValues([...pairs.filter(pair=>pair.key!=='ZERO'),{id:99,key:' APP_PROFILE ',value:'production'}])
  assert.equal(updated.DB_HOST,'prod-db')
  assert.equal(updated.PASSWORD,original.PASSWORD)
  assert.equal(updated.APP_PROFILE,'production')
  assert.equal(Object.hasOwn(updated,'ZERO'),false)
  assert.deepEqual(environmentParameterPairs(),[])
})
test('运行参数拒绝重复、缺失或无效名称，空行不产生无名参数',()=>{
  const pair=(key,value='')=>({id:1,key,value})
  assert.deepEqual(environmentParameterValues([pair('',''),pair('  ','')]),{})
  assert.throws(()=>environmentParameterValues([pair('','value')]),/第 1 项.*名称/)
  assert.throws(()=>environmentParameterValues([pair('A'),pair(' A ')]),/「A」重复/)
  for(const name of ['1PORT','DB-HOST','A B','数据库'])assert.throws(()=>environmentParameterValues([pair(name)]),/名称.*无效/)
  assert.deepEqual(environmentParameterValues([pair('A','1'),pair('a','2')]),{A:'1',a:'2'})
})
test('运行参数中的原型同名键作为普通配置保存，不改变对象原型',()=>{
  const values=environmentParameterValues([{id:1,key:'__proto__',value:'literal'},{id:2,key:'constructor',value:'ordinary'}])
  assert.equal(Object.getPrototypeOf(values),Object.prototype)
  assert.equal(Object.hasOwn(values,'__proto__'),true)
  assert.equal(values.__proto__,'literal')
  assert.equal(values.constructor,'ordinary')
})
test('密码和令牌等运行参数默认隐藏，普通地址和端口使用明文输入',()=>{
  for(const key of ['DB_PASSWORD','db_passwd','CLIENT_SECRET','AUTH_TOKEN','CREDENTIAL','PRIVATE_KEY','API_KEY','ACCESS_KEY'])assert.equal(isSensitiveEnvironmentParameter(key),true)
  for(const key of ['DB_HOST','PORT','APP_PROFILE',''])assert.equal(isSensitiveEnvironmentParameter(key),false)
})
test('模板参数按定义回填固定行，缺失必填项保留空值，附加参数保持可编辑',()=>{
  const schema=[{key:'IMAGE',label:'镜像',type:'image',required:true},{key:'PASSWORD',label:'密码',type:'password',required:true}]
  const original={IMAGE:'registry.test/app:v1',APP_PROFILE:'test'}
  const pairs=environmentParameterPairs(original,schema)
  assert.deepEqual(pairs.map(({key,value,schemaKey})=>({key,value,schemaKey})),[
    {key:'IMAGE',value:original.IMAGE,schemaKey:'IMAGE'},
    {key:'PASSWORD',value:'',schemaKey:'PASSWORD'},
    {key:'APP_PROFILE',value:'test',schemaKey:undefined},
  ])
  assert.deepEqual(environmentParameterValues(pairs,schema),{...original,PASSWORD:''})
  assert.deepEqual(original,{IMAGE:'registry.test/app:v1',APP_PROFILE:'test'})
  assert.throws(()=>environmentParameterValues([...pairs,{id:99,key:'PASSWORD',value:'extra'}],schema),/重复/)
  assert.throws(()=>environmentParameterValues([{...pairs[0],key:'OTHER_IMAGE'}],schema),/模板参数名称.*不能修改/)
})
test('Nacos 自动计算的客户端端口不作为附加参数编辑或覆盖',()=>{
  const schema=[{key:'NACOS_PORT',label:'服务端口',type:'port',required:true}]
  const pairs=environmentParameterPairs({NACOS_PORT:'8848',NACOS_GRPC_PORT:'9848',CUSTOM:'keep'},schema)
  assert.deepEqual(environmentParameterValues(pairs,schema),{NACOS_PORT:'8848',CUSTOM:'keep'})
  assert.throws(()=>environmentParameterValues([...pairs,{id:99,key:'NACOS_GRPC_PORT',value:'1234'}],schema),/自动计算/)
  assert.deepEqual(environmentParameterValues(environmentParameterPairs({NACOS_GRPC_PORT:'1234'})),{NACOS_GRPC_PORT:'1234'})
})

test('运行镜像使用容器核对结果，按环境和主机展示真实 Image ID',()=>{
  const environments=[{id:2,name:'测试环境'}]
  const container={id:'container-full-id',name:'test-nginx',image:'registry.test/nginx:test',image_id:'sha256:actual-running-image',state:'running',checked_at:'2026-11-01T04:00:00Z'}
  const states=[{environment_id:2,target_id:1,containers:[container]},{environment_id:2,target_id:3,containers:[{name:'test-nginx',state:'missing',image_id:''}]},{environment_id:999,target_id:1,containers:[container]}]
  const rows=applicationRuntimeImages(environments,states,[{id:1,name:'本机'}])
  assert.equal(rows.length,2)
  assert.deepEqual(rows[0],{...container,environment_id:2,environment_name:'测试环境',target_id:1,target_name:'本机'})
  assert.equal(rows[1].target_name,'主机 #3')
  assert.equal(rows[1].state,'missing')
  assert.equal(rows[1].image_id,'')
  assert.equal(states[0].containers[0],container)
})

test('尚无容器核对结果时不从发布版本推测运行镜像',()=>{
  assert.deepEqual(applicationRuntimeImages([{id:2,name:'测试环境'}],[{environment_id:2,target_id:1,current_release_id:5}]),[])
  assert.deepEqual(applicationRuntimeImages(),[])
})


test('运行镜像按实际 Image ID 打开本机详情，不使用可变 tag',()=>{
  const imageId='sha256:'+'a'.repeat(64)
  assert.deepEqual(runtimeImageLocation({target_id:1,image_id:imageId,image:'registry.test/nginx:test'}),{name:'ImageDetail',params:{id:imageId}})
})

test('远程镜像及缺失 Image ID 时不生成指向本机的错误详情链接',()=>{
  assert.equal(runtimeImageLocation({target_id:2,image_id:'sha256:remote'}),undefined)
  for(const image_id of [undefined,'','   '])assert.equal(runtimeImageLocation({target_id:1,image_id}),undefined)
  assert.equal(runtimeImageLocation({image_id:'sha256:unknown-host'}),undefined)
})
