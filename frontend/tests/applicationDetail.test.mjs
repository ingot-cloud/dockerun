import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/applicationDetail.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {applicationDetailTab,applicationSetupStep,environmentDeletionReason}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
test('应用详情视图可从网址恢复，异常参数回到部署环境',()=>{
  for(const value of [undefined,null,'bad',['releases'],'environments']) assert.equal(applicationDetailTab(value),'environments')
  assert.equal(applicationDetailTab('releases'),'releases')
  assert.equal(applicationDetailTab('configuration'),'configuration')
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
