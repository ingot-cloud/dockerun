import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/releases.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {defaultReleaseRegistry,automationIssue,imageRepository,copyScripts,hasScripts,releaseDeletionReason,deploymentDraftFromRelease,deploymentParameterReferences}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
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
test('命令自动发布直接使用保存的命令，要求引用触发镜像，不依赖首次发布',()=>{
  const env={registry_id:3,repository:'team/app'}
  for(const command of ['docker run "$DOCKERUN_IMAGE"','docker run "${DOCKERUN_IMAGE}"','docker run "${DOCKERUN_IMAGE:-$IMAGE}"']){
    assert.equal(automationIssue(env,[],{definition:{execution:{mode:'command',command}}}),'')
  }
  for(const command of ['docker run nginx:test','echo "$DOCKERUN_IMAGE_TAG"','echo "$DOCKERUN_IMAGES"']){
    assert.match(automationIssue(env,[],{definition:{execution:{mode:'command',command}}}),/DOCKERUN_IMAGE/)
  }
  assert.match(automationIssue({...env,repository:''},[]),/镜像名称/)
})
test('自动镜像的手动回退参数仍被识别，系统变量不作为缺失参数',()=>{
  assert.deepEqual(deploymentParameterReferences('docker run --name "$APP_NAME" "${DOCKERUN_IMAGE:-$IMAGE}"; echo "$DOCKERUN_IMAGE_TAG" "$WORK_DIR" "${PORT:-8080}" "$IMAGE"'),['APP_NAME','IMAGE','PORT'])
})
test('保存后的在线 Compose 可配置自动发布，本机和发布包来源有明确限制',()=>{
  const env={registry_id:3,repository:'team/app'}
  const definition={source:'registry',registry_id:3,compose:'services: {}'}
  assert.equal(automationIssue(env,[],{definition}),'')
  assert.match(automationIssue(env,[],{definition:{...definition,registry_id:4}}),/仓库/)
  assert.match(automationIssue(env,[],{definition:{...definition,source:'local'}}),/在线部署配置/)
})
test('基于历史版本生成可编辑草稿，保留命令和配置原文且不修改历史快照',()=>{
  const version={source:'command',execution:{mode:'command',command:'echo old',work_dir:'$WORK_DIR',timeout_seconds:30},configs:JSON.stringify({'configs/.env':' exact\r\n'}),scripts:{before:'echo before',after:'echo after'}}
  const stored=JSON.stringify(version)
  const draft=deploymentDraftFromRelease(version)
  assert.equal(draft.execution.command,'echo old')
  assert.equal(draft.configs['configs/.env'],' exact\r\n')
  assert.equal(draft.source,'direct')
  draft.execution.command='new';draft.scripts.before='new';draft.configs['configs/.env']='new'
  assert.equal(JSON.stringify(version),stored)
  const legacy=deploymentDraftFromRelease({source:'registry',scripts:{deploy:'echo legacy'},execution:{mode:''}})
  assert.equal(legacy.execution.mode,'compose')
  assert.equal(legacy.execution.command,'echo legacy')
  assert.equal(legacy.scripts.deploy,'')
})
