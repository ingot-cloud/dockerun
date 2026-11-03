export interface ProjectEnvironment {
  id?: number
  application_id?: number
  compose_project_name?: string
  compose_project_name_locked?: boolean
}

export function composeProjectName(environment?: ProjectEnvironment, name = environment?.compose_project_name || ''): string {
  return name.trim() || (environment?.id && environment.application_id ? `dockerun-a${environment.application_id}-e${environment.id}` : '')
}

export function validateComposeProjectName(name: string, environment?: ProjectEnvironment): string {
  const value = name.trim()
  if (value && (value.length > 100 || !/^[a-z0-9][a-z0-9_-]*$/.test(value))) throw new Error('Compose 项目名称须以小写字母或数字开头，仅包含小写字母、数字、连字符和下划线，最多 100 个字符')
  if (/^dockerun-a[0-9]+-e[0-9]+$/.test(value) && value !== composeProjectName(environment, '')) throw new Error('dockerun-a编号-e编号 为系统保留名称，请使用其他 Compose 项目名称')
  if (environment?.compose_project_name_locked && composeProjectName(environment, value) !== composeProjectName(environment)) throw new Error('此环境已提交部署任务，Compose 项目名称已固定，请刷新后保留原名称')
  return value
}

export interface ProjectNameDraft { name: string; saved: string }
export function refreshProjectNameDraft(draft: ProjectNameDraft | undefined, environment: ProjectEnvironment): ProjectNameDraft {
  return draft && draft.name.trim() !== draft.saved ? draft : { name: environment.compose_project_name || '', saved: environment.compose_project_name || '' }
}
export function projectNameChanges(drafts: Record<string, ProjectNameDraft>): Record<string, string> {
  return Object.fromEntries(Object.entries(drafts).filter(([, draft]) => draft.name.trim() !== draft.saved).map(([id, draft]) => [id, draft.name.trim()]))
}
