import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/codeLanguage.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {codeLanguage,deploymentFileLanguage}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
test('部署与配置文件按文件名识别语法，未知文件保留为纯文本',()=>{
  for(const [name,language] of Object.entries({'deploy.sh':'shell','compose.yaml':'yaml','configs/APP.YML':'yaml','configs/app.json':'json','configs/.env.production':'ini','configs/app.env':'ini','configs/java.properties':'ini','Dockerfile':'dockerfile','build/Dockerfile.prod':'dockerfile','configs/app.xml':'xml','C:\\configs\\app.json':'json','configs/无后缀':'plaintext','configs/empty.txt':'plaintext'}))assert.equal(codeLanguage(name),language,name)
  assert.equal(codeLanguage(),'plaintext')
})
test('历史脚本与发布文件共用语言识别，脚本文件按实际后缀高亮',()=>{
  for(const [name,language] of Object.entries({'init.sql':'sql','migrate.py':'python','Main.java':'java','main.go':'go','deploy.ps1':'powershell','notes.markdown':'markdown','image.dockerfile':'dockerfile'}))assert.equal(codeLanguage(name),language)
  assert.equal(deploymentFileLanguage({file_name:'migrate.py',file_type:'script'}),'python')
  assert.equal(deploymentFileLanguage({file_name:'unknown',file_path:'configs/app.yaml',file_type:'config'}),'yaml')
  assert.equal(deploymentFileLanguage({file_name:'unknown',file_type:'script'}),'shell')
  assert.equal(deploymentFileLanguage({file_name:'stack.conf',file_type:'compose'}),'yaml')
  assert.equal(deploymentFileLanguage({file_name:'settings.env',file_type:'config'}),'ini')
  assert.equal(deploymentFileLanguage({file_name:'unknown',file_type:'other'}),'plaintext')
})
