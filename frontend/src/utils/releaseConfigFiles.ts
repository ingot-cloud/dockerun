export interface ReleaseConfigFile { id: number; path: string; content: string }

export function releaseConfigFiles(configs: Record<string,string> = {}): ReleaseConfigFile[] {
  return Object.entries(configs).map(([path, content], index) => ({ id: index + 1, path, content }))
}

export function releaseConfigValues(files: ReleaseConfigFile[]): Record<string,string> {
  const paths = new Set<string>()
  const entries: [string,string][] = []
  files.forEach((file, index) => {
    const path = file.path
    if (!path.trim() || path === 'configs/') throw new Error(`请填写第 ${index + 1} 个配置文件的文件名`)
    if (!path.startsWith('configs/')) throw new Error(`配置文件「${path}」的文件名必须以 configs/ 开头`)
    if (path.includes('\\') || path.includes('\0') || path.split('/').some(part => !part || part === '.' || part === '..')) throw new Error(`配置文件「${path}」的路径无效，请使用 configs/ 下的文件名，可包含子目录`)
    if (paths.has(path)) throw new Error(`配置文件名「${path}」重复`)
    paths.add(path)
    entries.push([path, file.content])
  })
  for (const path of paths) {
    const parts = path.split('/')
    for (let index = 2; index < parts.length; index++) {
      const parent = parts.slice(0, index).join('/')
      if (paths.has(parent)) throw new Error(`配置文件「${parent}」与「${path}」的目录路径冲突`)
    }
  }
  return Object.fromEntries(entries)
}

export function releaseConfigDraft(files: ReleaseConfigFile[]): Array<{path:string;content:string}> {
  return files.map(({path, content}) => ({path, content})).sort((left, right) => left.path.localeCompare(right.path) || left.content.localeCompare(right.content))
}
