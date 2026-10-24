import type { LocationQuery, RouteLocationRaw } from 'vue-router'

interface WorkspaceRoute {
  name?: string | symbol | null
  path: string
  query: LocationQuery
  params: Record<string, unknown>
  meta: { title?: unknown }
}

export interface WorkspaceCrumb {
  label: string
  to?: RouteLocationRaw
}

export function workspaceParents(route: WorkspaceRoute): WorkspaceCrumb[] {
  const { query, params } = route
  const list = (label: string, name: string): WorkspaceCrumb => ({ label, to: { name, query } })
  const detail = (label: string, name: string): WorkspaceCrumb => ({ label, to: { name, params: { id: String(params.id) }, query } })
  switch (route.name) {
    case 'ApplicationDetail': return [list('应用', 'Applications')]
    case 'DeploymentWizard': return [list('应用', 'Applications'), detail('应用详情', 'ApplicationDetail')]
    case 'ApplicationTemplateCreate': return [list('应用模板', 'ApplicationTemplates')]
    case 'DeploymentDetail': return [list('部署记录', 'Deployments')]
    case 'ImageDetail': return [list('本地镜像', 'Images')]
    case 'ContainerDetail': return [list('容器', 'Containers')]
    case 'ContainerLogs': case 'ContainerInspect': case 'ContainerStats': case 'ContainerExec': case 'ContainerAttach':
      return [list('容器', 'Containers'), detail('容器详情', 'ContainerDetail')]
    case 'NetworkCreate': case 'NetworkDetail': return [list('网络', 'Networks')]
    case 'VolumeCreate': case 'VolumeDetail': return [list('卷', 'Volumes')]
    case 'RegistryCreate': case 'RegistryDetail': return [list('外部仓库', 'Registry')]
    case 'RegistryImages': return [list('外部仓库', 'Registry'), detail('仓库详情', 'RegistryDetail')]
    default: return []
  }
}

export function workspaceBreadcrumbs(route: WorkspaceRoute, admin: boolean): WorkspaceCrumb[] {
  const home = admin ? 'Overview' : 'Builtin'
  return [
    { label: 'Dockerun', ...(route.name === home ? {} : { to: { name: home } }) },
    ...workspaceParents(route),
    { label: String(route.meta.title || '工作台') },
  ]
}

interface ResolvedWorkspaceRoute {
  path: string
  matched: unknown[]
  meta: { requiresAuth?: unknown; title?: unknown }
}

export function workspaceBack(
  route: WorkspaceRoute,
  previous: unknown,
  resolve: (path: string) => ResolvedWorkspaceRoute,
): { to: RouteLocationRaw; label: string; history: boolean } | undefined {
  const parents = workspaceParents(route)
  const parent = parents[parents.length - 1]
  if (!parent?.to) return
  if (typeof previous === 'string' && previous.startsWith('/') && !previous.startsWith('//')) {
    try {
      const target = resolve(previous)
      if (target.matched.length && target.meta.requiresAuth && target.path !== route.path) {
        return { to: previous, label: `返回${target.meta.title || '上一页'}`, history: true }
      }
    } catch { /* 无法识别的历史地址按直接访问处理。 */ }
  }
  return { to: parent.to, label: `返回${parent.label}`, history: false }
}
