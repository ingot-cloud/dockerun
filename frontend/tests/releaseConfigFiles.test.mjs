import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/releaseConfigFiles.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {releaseConfigFiles,releaseConfigValues,releaseConfigDraft}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('配置逐项回填，保留空文件、多行、换行格式、空白和特殊字符',()=>{
  const original={'configs/app.yaml':'  host: ${DB_HOST}\r\nmessage: "中文=a$b"\r\n','configs/empty.conf':'','configs/nested/app.json':'{\n  "path": "C:\\\\data"\n}\n'}
  const files=releaseConfigFiles(original)
  assert.deepEqual(releaseConfigValues(files),original)
  assert.deepEqual(JSON.parse(JSON.stringify(releaseConfigValues(files))),original)
  files[0].content='changed'
  assert.equal(original['configs/app.yaml'],'  host: ${DB_HOST}\r\nmessage: "中文=a$b"\r\n')
  assert.deepEqual(releaseConfigFiles(),[])
  assert.deepEqual(releaseConfigValues([]),{})
})
test('支持新增、重命名、删除和子目录，删除项不会重新提交',()=>{
  const files=releaseConfigFiles({'configs/old.conf':'old','configs/keep.conf':'keep'})
  files[1].path='configs/中文目录/新 配置.conf'
  files.push({id:3,path:'configs/new.txt',content:'  exact content  '})
  files.shift()
  assert.deepEqual(releaseConfigValues(files),{'configs/中文目录/新 配置.conf':'keep','configs/new.txt':'  exact content  '})
})
test('文件名必填且位于 configs/，拒绝无效目录段、空字符和反斜杠',()=>{
  const file=path=>[{id:1,path,content:''}]
  for(const path of ['',' ','configs/'])assert.throws(()=>releaseConfigValues(file(path)),/请填写第 1 个配置文件的文件名/)
  for(const path of ['app.yaml','/configs/app.yaml','../app.yaml'])assert.throws(()=>releaseConfigValues(file(path)),/必须以 configs\//)
  for(const path of ['configs/../app.yaml','configs/a/../../app.yaml','configs/./app.yaml','configs//app.yaml','configs/folder/','configs/a\\b.conf','configs/file\0.conf'])assert.throws(()=>releaseConfigValues(file(path)),/路径无效/)
  assert.deepEqual(releaseConfigValues(file('configs/.hidden.conf')),{'configs/.hidden.conf':''})
})
test('重复文件名与文件/目录冲突在提交前明确报错，不覆盖内容',()=>{
  assert.throws(()=>releaseConfigValues([{id:1,path:'configs/app.yaml',content:'first'},{id:2,path:'configs/app.yaml',content:'second'}]),/文件名.*重复/)
  for(const paths of [['configs/nginx','configs/nginx/site.conf'],['configs/nginx/site.conf','configs/nginx']])assert.throws(()=>releaseConfigValues(paths.map((path,index)=>({id:index,path,content:''}))),/目录路径冲突/)
})
test('草稿与版本标识忽略行编号及文件排序，仍识别内容、文件名和无效草稿变化',()=>{
  const files=releaseConfigFiles({'configs/a':'a','configs/b':'b'})
  const baseline=releaseConfigDraft(files)
  const reordered=files.toReversed().map(file=>({...file,id:file.id+10}))
  assert.deepEqual(releaseConfigDraft(reordered),baseline)
  reordered[0].content='different'
  assert.notDeepEqual(releaseConfigDraft(reordered),baseline)
  reordered[0].content='b';reordered[0].path='configs/c'
  assert.notDeepEqual(releaseConfigDraft(reordered),baseline)
  reordered.push({id:99,path:'configs/',content:''})
  assert.equal(releaseConfigDraft(reordered).length,3)
})
