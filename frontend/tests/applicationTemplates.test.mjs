import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const compile=source=>ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}}).outputText
const moduleURL=source=>`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
const releases=moduleURL(compile(await readFile(new URL('../src/utils/releases.ts',import.meta.url),'utf8')))
const source=compile(await readFile(new URL('../src/utils/applicationTemplates.ts',import.meta.url),'utf8')).replaceAll("'@/utils/releases'",JSON.stringify(releases)).replaceAll('"@/utils/releases"',JSON.stringify(releases))
const {templateValues,validateTemplateValues,maskTemplateParameters,retargetTemplateImages,templateEndpoints}=await import(moduleURL(source))
const schema=[{key:'IMAGE',label:'镜像',type:'image',default:'team/app:v1',required:true},{key:'PASSWORD',label:'密码',type:'password',default:'must-not-copy',required:true},{key:'PORT',label:'端口',type:'port',default:'8080',required:true}]
const template={definition:{parameters:schema}}
test('template defaults use selected registry and never prefill passwords',()=>{
  const values=templateValues(template,{url:'https://registry.test:5443'})
  assert.deepEqual(values,{IMAGE:'registry.test:5443/team/app:v1',PASSWORD:'',PORT:'8080'})
  assert.throws(()=>validateTemplateValues(schema,values),/请填写密码/)
  values.PASSWORD='literal $secret'
  validateTemplateValues(schema,values)
  assert.throws(()=>validateTemplateValues(schema,{...values,PORT:'65536'}),/端口/)
})
test('switching registry preserves edited image paths, versions and credentials',()=>{
  const values={IMAGE:'old.test/team/custom:v3',PASSWORD:'keep-me',PORT:'3306'}
  assert.deepEqual(retargetTemplateImages(schema,values,{url:'https://old.test'},{url:'https://new.test'}),{...values,IMAGE:'new.test/team/custom:v3'})
  assert.equal(values.IMAGE,'old.test/team/custom:v3')
})
test('preview masks named secret fields while retaining connection configuration',()=>{
  const values=maskTemplateParameters({PASSWORD:'secret',PORT:'8080',NACOS_IDENTITY_VALUE:'secret' },[{key:'NACOS_IDENTITY_VALUE',type:'password'}])
  assert.deepEqual(values,{PASSWORD:'******',PORT:'8080',NACOS_IDENTITY_VALUE:'******'})
})
test('Nacos token and derived port are validated before creating an application',()=>{
  assert.throws(()=>validateTemplateValues([],{NACOS_AUTH_TOKEN:'invalid'}),/Base64/)
  assert.throws(()=>validateTemplateValues([],{NACOS_PORT:'65000'}),/gRPC/)
  validateTemplateValues([],{NACOS_AUTH_TOKEN:btoa('a'.repeat(32)),NACOS_PORT:'8848'})
})
test('endpoints contain only configured host, port and path',()=>{
  assert.deepEqual(templateEndpoints({endpoints:[{name:'控制台',port:'CONSOLE_PORT',protocol:'http',path:'/console'}]},{CONSOLE_PORT:'8181',PASSWORD:'secret'},'localhost'),[{name:'控制台',address:'http://localhost:8181/console'}])
})
