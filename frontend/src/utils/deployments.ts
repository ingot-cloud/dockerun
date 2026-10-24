export interface DeploymentNames {
  id?: number | string; name?: string; application_name?: string; environment_name?: string; release_version?: string;
  legacy_script_name?: string; environment_id?: number; release_id?: number; target_id?: number; target_name?: string; user_id?: number; username?: string;
}
export function deploymentName(task: DeploymentNames): string {
  if (task.name?.trim()) return task.name
  if (task.legacy_script_name?.trim()) return `执行脚本 · ${task.legacy_script_name}`
  const names = [task.application_name, task.environment_name, task.release_version].filter(value => value?.trim())
  return names.length ? names.join(' · ') : task.id ? `部署任务 #${task.id}` : '部署详情'
}
export function deploymentEnvironment(task: DeploymentNames): string { return task.environment_name?.trim() ? task.environment_name : task.environment_id ? `未命名环境（#${task.environment_id}）` : '—' }
export function deploymentVersion(task: DeploymentNames): string { return task.release_version?.trim() ? task.release_version : task.release_id ? `未命名版本（#${task.release_id}）` : '—' }
export function deploymentTarget(task: DeploymentNames): string { return task.target_name?.trim() ? task.target_name : task.target_id === 1 ? '本机' : task.target_id ? `主机 #${task.target_id}` : '—' }
export function deploymentUser(task: DeploymentNames): string { return task.username?.trim() ? task.username : task.user_id ? `用户 #${task.user_id}` : '—' }
