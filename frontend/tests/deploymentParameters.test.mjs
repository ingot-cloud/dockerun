import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const compile=source=>ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}}).outputText
const moduleURL=source=>`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
const detail=moduleURL(compile(await readFile(new URL('../src/utils/applicationDetail.ts',import.meta.url),'utf8')))
const source=compile(await readFile(new URL('../src/utils/deploymentParameters.ts',import.meta.url),'utf8')).replaceAll("'@/utils/applicationDetail'",JSON.stringify(detail))
const {createDeploymentParameterDraft,deploymentParametersChanged,deploymentParameterDraftsSnapshot}=await import(moduleURL(source))
test('发布参数草稿独立于原环境，修改后再还原不误报未保存',()=>{
  const parameters={PASSWORD:'  value=secret  ',PORT:'8080'}
  const draft=createDeploymentParameterDraft(parameters)
  assert.equal(deploymentParametersChanged(draft),false)
  draft.pairs[0].value='changed'
  assert.equal(parameters.PASSWORD,'  value=secret  ')
  assert.equal(draft.savedParameters.PASSWORD,parameters.PASSWORD)
  assert.equal(deploymentParametersChanged(draft),true)
  draft.pairs[0].value=parameters.PASSWORD
  draft.pairs.reverse().forEach(pair=>pair.id+=100)
  assert.equal(deploymentParametersChanged(draft),false)
})
test('新增环境的初始草稿不触发保护，跨环境的未保存修改分别保留',()=>{
  const drafts={1:createDeploymentParameterDraft({PORT:'8080'})}
  assert.deepEqual(deploymentParameterDraftsSnapshot(drafts),{})
  drafts[2]=createDeploymentParameterDraft({PORT:'9090'})
  assert.deepEqual(deploymentParameterDraftsSnapshot(drafts),{})
  drafts[1].pairs[0].value='8081'
  const first=deploymentParameterDraftsSnapshot(drafts)
  drafts[2].pairs[0].value='9091'
  assert.deepEqual(Object.keys(deploymentParameterDraftsSnapshot(drafts)),['1','2'])
  drafts[2]=createDeploymentParameterDraft({PORT:'9091'})
  assert.deepEqual(deploymentParameterDraftsSnapshot(drafts),first)
})
test('重复参数和未填名称的输入保留草稿，不会被比较逻辑覆盖',()=>{
  const draft=createDeploymentParameterDraft({PORT:'8080'})
  draft.pairs.push({id:2,key:'PORT',value:'8080'})
  assert.equal(deploymentParametersChanged(draft),true)
  draft.pairs[1].key=''
  assert.equal(deploymentParametersChanged(draft),true)
  draft.pairs.pop()
  assert.equal(deploymentParametersChanged(draft),false)
})
