import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/releases.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {defaultReleaseRegistry,automationIssue,imageRepository,copyScripts,hasScripts,releaseDeletionReason}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
test('release sources prefer the built-in registry even when an external source is marked default',()=>{
  assert.equal(defaultReleaseRegistry([{id:1,kind:'external',is_default:true},{id:2,kind:'builtin'}]),2)
  assert.equal(defaultReleaseRegistry([{id:1,kind:'external',is_default:true}]),1)
  assert.equal(defaultReleaseRegistry([]),0)
})
test('automatic releases require a successful version from the selected registry and repository',()=>{
  const releases=[{id:2,source:'registry',registry_id:3,images:JSON.stringify([{reference:'localhost:8080/team/app:v1'}])}]
  const env={current_release_id:2,registry_id:3,repository:'team/app'}
  assert.equal(automationIssue(env,releases),'')
  assert.match(automationIssue({...env,current_release_id:0},releases),/首次手动发布/)
  assert.match(automationIssue({...env,registry_id:4},releases),/所选仓库/)
  assert.match(automationIssue({...env,repository:'team/other'},releases),/当前成功版本/)
  assert.match(automationIssue(env,[{...releases[0],source:'local'}]),/所选仓库/)
})
test('repository matching handles ports, digests and untagged names',()=>{
  assert.equal(imageRepository('localhost:8080/team/app:v1'),'team/app')
  assert.equal(imageRepository('registry.test/app@sha256:abc'),'app')
  assert.equal(imageRepository('team/app:latest'),'team/app')
})
test('editing a script draft does not mutate an existing release snapshot',()=>{
  const stored={before:'echo before'}
  const draft=copyScripts(stored)
  draft.before='changed'
  assert.equal(stored.before,'echo before')
  assert.equal(hasScripts(copyScripts(null)),false)
  assert.equal(hasScripts({deploy:'   '}),false)
  assert.equal(hasScripts(stored),true)
})
test('发布版本删除沿用服务端引用校验，引用状态缺失时禁止删除',()=>{
  assert.equal(releaseDeletionReason({deletion_blocked_reason:''}),'')
  assert.equal(releaseDeletionReason({deletion_blocked_reason:'版本正在被部署环境使用，不能删除'}),'版本正在被部署环境使用，不能删除')
  assert.ok(releaseDeletionReason({}))
})
