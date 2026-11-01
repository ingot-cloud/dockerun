export function codeLanguage(filename = ''): string {
  const name = filename.replace(/\\/g, '/').split('/').pop()?.toLowerCase() || ''
  if (name === 'dockerfile' || name.startsWith('dockerfile.') || name.endsWith('.dockerfile')) return 'dockerfile'
  if (name === '.env' || name.startsWith('.env.') || name.endsWith('.env')) return 'ini'
  const extension = name.split('.').pop()
  const languages: Record<string,string> = {
    sh:'shell', bash:'shell', zsh:'shell', yaml:'yaml', yml:'yaml', json:'json', jsonc:'json',
    ini:'ini', conf:'ini', properties:'ini', xml:'xml', html:'html', htm:'html', css:'css',
    js:'javascript', mjs:'javascript', ts:'typescript', md:'markdown', markdown:'markdown', txt:'plaintext',
    py:'python', java:'java', go:'go', sql:'sql', ps1:'powershell', config:'ini',
  }
  return languages[extension || ''] || 'plaintext'
}

export function deploymentFileLanguage(file:{file_type?:string;file_name?:string;file_path?:string}):string {
  if(file.file_type==='compose')return 'yaml'
  const language=codeLanguage(file.file_name) !== 'plaintext'?codeLanguage(file.file_name):codeLanguage(file.file_path)
  return language==='plaintext' && file.file_type==='script'?'shell':language
}
