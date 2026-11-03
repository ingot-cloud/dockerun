import type { TemplateImportChoice, TemplateImportEntry } from '@/api/applicationTemplates'

export const MAX_TEMPLATE_ARCHIVE_SIZE=32*1024*1024
export function templateSource(source:string){return ({builtin:'内置模板',custom:'自建模板',imported:'导入模板'} as Record<string,string>)[source] || '未知来源'}
export function templateImportStatus(status:string){return ({new:'新增模板',new_version:'新增版本',identical:'已存在相同内容',conflict:'版本内容冲突'} as Record<string,string>)[status] || '状态未知'}
export function validateTemplateArchive(file:{name:string;size:number}){
  if(!/\.zip$/i.test(file.name))throw new Error('请选择 .zip 格式的模板包')
  if(!file.size)throw new Error('模板 ZIP 不能为空')
  if(file.size>MAX_TEMPLATE_ARCHIVE_SIZE)throw new Error('模板 ZIP 不能超过 32 MiB')
}
export function defaultTemplateImportChoices(entries:TemplateImportEntry[]):TemplateImportChoice[]{
  return entries.map(entry=>({index:entry.index,action:['identical','conflict'].includes(entry.status)?'skip':'import',key:`${entry.key.slice(0,90).replace(/-+$/,'')}-copy`,name:`${entry.name}（副本）`}))
}
export function validateTemplateImportChoices(entries:TemplateImportEntry[],choices:TemplateImportChoice[]){
  const byIndex=new Map(entries.map(entry=>[entry.index,entry])),indexes=new Set<number>(),identities=new Set<string>()
  let count=0
  for(const choice of choices){
    const entry=byIndex.get(choice.index)
    if(!entry || indexes.has(choice.index))throw new Error('模板选择无效，请重新预览')
    indexes.add(choice.index)
    if(choice.action==='skip')continue
    if(!['import','rename'].includes(choice.action))throw new Error('请选择有效的导入方式')
    if(choice.action==='import' && entry.status==='conflict')throw new Error(`${entry.name} 的版本内容冲突，请另存或跳过`)
    let key=entry.key
    if(choice.action==='rename'){
      key=choice.key || ''
      if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(key)||key.length>100||key.startsWith('custom-'))throw new Error(`${entry.name} 的新标识应为小写字母、数字及连字符，不能以 custom- 开头`)
      if(!choice.name?.trim())throw new Error('请填写另存模板的名称')
      if(new TextEncoder().encode(choice.name.trim()).length>100)throw new Error('另存模板名称过长，请缩短名称')
    }
    const identity=`${key}\0${entry.version}`
    if(identities.has(identity))throw new Error('本次导入存在相同的模板标识和版本，请调整标识或跳过重复项')
    identities.add(identity)
    if(choice.action==='rename'||entry.status!=='identical')count++
  }
  if(!count)throw new Error('没有待导入的模板，请选择导入或另存为新模板')
  return count
}
export function templateArchiveFilename(key?:string,version?:string){
  return key?`${key}${version?`-${version}`:''}`.replace(/[\\/:*?"<>|\u0000-\u001f]/g,'-')+'.zip':'dockerun-templates.zip'
}
