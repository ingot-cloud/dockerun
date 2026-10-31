import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/deployments.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {deploymentName,deploymentEnvironment,deploymentVersion,deploymentTarget,deploymentUser,deploymentDeletionReason}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
test('部署名称优先使用业务名称和关联资源名称',()=>{
  assert.equal(deploymentName({id:1,name:'网关 · 生产 · v2'}),'网关 · 生产 · v2')
  assert.equal(deploymentName({id:1,application_name:'网关',environment_name:'生产',release_version:'v2'}),'网关 · 生产 · v2')
  assert.equal(deploymentName({id:2,legacy_script_name:'安装 Redis'}),'执行脚本 · 安装 Redis')
})
test('关联数据缺失时保留可定位的编号，不伪造资源名称',()=>{
  assert.equal(deploymentName({id:3}),'部署任务 #3')
  assert.equal(deploymentName({}),'部署详情')
  assert.equal(deploymentEnvironment({environment_id:4}),'未命名环境（#4）')
  assert.equal(deploymentVersion({release_id:5}),'未命名版本（#5）')
  assert.equal(deploymentEnvironment({}),'—')
  assert.equal(deploymentVersion({}),'—')
})
test('环境、版本、主机和用户名称覆盖技术编号',()=>{
  assert.equal(deploymentEnvironment({environment_id:1,environment_name:'生产'}),'生产')
  assert.equal(deploymentVersion({release_id:2,release_version:'v2026.10'}),'v2026.10')
  assert.equal(deploymentTarget({target_id:3,target_name:'上海主机'}),'上海主机')
  assert.equal(deploymentTarget({target_id:1}),'本机')
  assert.equal(deploymentUser({user_id:4,username:'张三'}),'张三')
  assert.equal(deploymentUser({user_id:4}),'用户 #4')
})
test('只有已结束且未恢复中的部署记录可删除',()=>{
  for(const status of ['queued','blocked','running','unknown','']) assert.ok(deploymentDeletionReason({status}))
  assert.ok(deploymentDeletionReason({}))
  for(const status of ['success','failed','needs_attention','cancelled','paused']){
    assert.equal(deploymentDeletionReason({status}),'')
    assert.ok(deploymentDeletionReason({status,recovery_status:'running'}))
  }
})
