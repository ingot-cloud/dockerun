import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=ts.transpileModule(await readFile(new URL('../src/utils/composeProject.ts',import.meta.url),'utf8'),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}}).outputText
const {composeProjectName,validateComposeProjectName,refreshProjectNameDraft,projectNameChanges}=await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`)
const env={id:5,application_id:7,compose_project_name:''}
test('留空沿用自动命名，自定义名称只裁剪首尾空格',()=>{
  assert.equal(composeProjectName(env),'dockerun-a7-e5')
  assert.equal(composeProjectName(env,' redis_test-1 '),'redis_test-1')
  assert.equal(composeProjectName({}), '')
  for(const name of ['redis-test','1redis','a_b',''])assert.equal(validateComposeProjectName(name,env),name)
  for(const name of ['Redis','-redis','_redis','redis.test','有中文','a'.repeat(101),'dockerun-a1-e2'])assert.throws(()=>validateComposeProjectName(name,env))
})
test('锁定后保留原名称，不能清空或改名',()=>{
  const locked={...env,compose_project_name:'redis-prod',compose_project_name_locked:true}
  assert.equal(validateComposeProjectName('redis-prod',locked),'redis-prod')
  assert.throws(()=>validateComposeProjectName('',locked),/已固定/)
  assert.throws(()=>validateComposeProjectName('redis-new',locked),/已固定/)
  assert.equal(validateComposeProjectName('dockerun-a7-e5',{...env,compose_project_name_locked:true}),'dockerun-a7-e5')
})
test('多环境草稿独立保留，刷新不覆盖修改，还原和保存后清除修改状态',()=>{
  const drafts={5:refreshProjectNameDraft(undefined,env)}
  assert.deepEqual(projectNameChanges(drafts),{})
  drafts[5].name='redis-test'
  drafts[6]=refreshProjectNameDraft(undefined,{...env,id:6})
  drafts[5]=refreshProjectNameDraft(drafts[5],env)
  assert.deepEqual(projectNameChanges(drafts),{5:'redis-test'})
  drafts[5].name=' '
  assert.deepEqual(projectNameChanges(drafts),{})
  drafts[5]=refreshProjectNameDraft(undefined,{...env,compose_project_name:'redis-saved'})
  assert.deepEqual(projectNameChanges(drafts),{})
  drafts[5].name=''
  assert.deepEqual(projectNameChanges(drafts),{5:''})
})
