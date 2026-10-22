import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/management.ts',import.meta.url),'utf8')
const {outputText}=ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ES2020}})
const {readListState,filterAndSort,parseObject,parseArray,hostState,tokenState,statusLabel,formatDate}=await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
test('list state validates query values and isolates embedded lists',()=>{
  assert.deepEqual(readListState({env_search:'test',env_page:'2',env_size:'25',env_sort:'created_at',env_direction:'desc',search:'ignored'},'env_',['name','created_at']),{search:'test',filter:'',page:2,size:25,sort:'created_at',descending:true})
  for(const page of ['-1','Infinity','2.5','9007199254740992'])assert.equal(readListState({page}).page,1)
  assert.equal(readListState({size:'999',sort:'invalid',filter:'invalid'}).size,10)
  assert.equal(readListState({filter:'failed'},'',['name'],['failed']).filter,'failed')
  assert.equal(readListState({},'',['created_at']).descending,true)
  assert.equal(readListState({direction:'asc'},'',['created_at']).descending,false)
})
test('search combines fields, sorting uses natural names and dates, original data is retained',()=>{
  const items=[{id:2,name:'app-10',description:'Production',created_at:'2026-10-22'},{id:1,name:'app-2',description:'Test',created_at:'2026-10-21'}]
  assert.deepEqual(filterAndSort(items,' TEST ',['name','description'],'name',false).map(item=>item.id),[1])
  assert.deepEqual(filterAndSort(items,'',['name'],'name',false).map(item=>item.id),[1,2])
  assert.deepEqual(filterAndSort(items,'',['name'],'created_at',true).map(item=>item.id),[2,1])
  assert.equal(items[0].name,'app-10')
})
test('environment parameters must be an object and invalid JSON errors are Chinese',()=>{
  for(const text of ['[]','null','1','invalid'])assert.throws(()=>parseObject(text,'环境参数'),/环境参数.*JSON/)
  assert.deepEqual(parseObject('{"PORT":"8080"}','环境参数'),{PORT:'8080'})
  assert.deepEqual(parseArray('not-json'),[])
  assert.deepEqual(parseArray('[1,0,-1,2,"3"]'),[1,2])
})
test('disabled agents are never displayed online; stale heartbeat is offline',()=>{
  const now=Date.parse('2026-10-22T10:00:00Z')
  assert.equal(hostState({id:2,enabled:false,last_seen_at:'2026-10-22T09:59:59Z'},now),'disabled')
  assert.equal(hostState({id:2,enabled:true,last_seen_at:'2026-10-22T09:59:59Z'},now),'online')
  assert.equal(hostState({id:2,enabled:true,last_seen_at:'2026-10-22T09:00:00Z'},now),'offline')
  assert.equal(hostState({id:1,enabled:true},now),'local')
})
test('expired and malformed token lifetimes cannot be labelled valid',()=>{
  const now=Date.parse('2026-10-22T10:00:00Z')
  assert.equal(tokenState({revoked:false,expires_at:'2026-10-22T10:00:00Z'},now),'expired')
  assert.equal(tokenState({revoked:false,expires_at:'invalid'},now),'expired')
  assert.equal(tokenState({revoked:true,expires_at:'2099-01-01'},now),'revoked')
  assert.equal(tokenState({revoked:false,expires_at:'2099-01-01'},now),'ready')
})
test('unknown statuses remain explicit and invalid dates use a placeholder',()=>{
  assert.match(statusLabel('unexpected'),/未知状态/)
  assert.equal(formatDate('invalid'),'—')
  assert.equal(formatDate(''),'—')
})
