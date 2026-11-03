import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/templatePackages.ts',import.meta.url),'utf8')
const output=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}}).outputText
const {validateTemplateArchive,defaultTemplateImportChoices,validateTemplateImportChoices,templateArchiveFilename,templateSource,templateImportStatus}=await import(`data:text/javascript;base64,${Buffer.from(output).toString('base64')}`)
const entries=[{index:0,key:'redis',name:'Redis',version:'1.0.0',status:'new'},{index:1,key:'redis',name:'Redis',version:'2.0.0',status:'new_version'},{index:2,key:'mysql',name:'MySQL',version:'1.0.0',status:'identical'},{index:3,key:'app',name:'应用',version:'1.0.0',status:'conflict'}]
test('模板文件只接受不超过限制的非空 ZIP，扩展名不区分大小写',()=>{
  validateTemplateArchive({name:'templates.ZIP',size:32*1024*1024})
  assert.throws(()=>validateTemplateArchive({name:'template.json',size:100}),/zip/)
  assert.throws(()=>validateTemplateArchive({name:'templates.zip',size:0}),/不能为空/)
  assert.throws(()=>validateTemplateArchive({name:'templates.zip',size:32*1024*1024+1}),/32 MiB/)
})
test('默认导入新增模板及版本，相同内容与冲突项需要明确另存',()=>{
  const choices=defaultTemplateImportChoices(entries)
  assert.deepEqual(choices.map(item=>item.action),['import','import','skip','skip'])
  assert.equal(validateTemplateImportChoices(entries,choices),2)
  choices[3].action='import';assert.throws(()=>validateTemplateImportChoices(entries,choices),/冲突/)
  choices[3].action='rename';assert.equal(validateTemplateImportChoices(entries,choices),3)
})
test('另存保留版本，拒绝无效标识、过长名称及重复身份',()=>{
  const choices=defaultTemplateImportChoices(entries);choices[0].action='rename'
  for(const key of ['../path','custom-123','UPPER','a'.repeat(101),'a--b'])assert.throws(()=>validateTemplateImportChoices(entries,[{...choices[0],key}]),/标识/)
  assert.throws(()=>validateTemplateImportChoices(entries,[{...choices[0],name:' '}]),/名称/)
  assert.throws(()=>validateTemplateImportChoices(entries,[{...choices[0],name:'中'.repeat(34)}]),/过长/)
  assert.throws(()=>validateTemplateImportChoices(entries,[choices[0],choices[0]]),/选择无效/)
  assert.throws(()=>validateTemplateImportChoices(entries,[{...choices[2]}]),/没有待导入/)
  assert.equal(validateTemplateImportChoices(entries,[{...choices[0],key:'shared-copy'},{...choices[1],action:'rename',key:'shared-copy'}]),2)
})
test('来源、冲突状态和下载文件名使用清晰且安全的显示值',()=>{
  assert.equal(templateSource('imported'),'导入模板')
  assert.equal(templateImportStatus('conflict'),'版本内容冲突')
  assert.equal(templateArchiveFilename(),'dockerun-templates.zip')
  assert.equal(templateArchiveFilename('team-redis','v1/unsafe\r\n'), 'team-redis-v1-unsafe--.zip')
})
