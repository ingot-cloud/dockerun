export type ApplicationDetailTab = 'environments' | 'releases' | 'configuration'
export function applicationDetailTab(value: unknown): ApplicationDetailTab {
  return value === 'releases' || value === 'configuration' ? value : 'environments'
}
export function applicationSetupStep(environments: {current_release_id?: number}[], targets: {current_release_id?: number}[] = []): number {
  if ([...environments, ...targets].some(item => item.current_release_id)) return 3
  return environments.length ? 2 : 1
}

export function environmentDeletionReason(environment: {deletion_blocked_reason?: string}): string {
  return typeof environment.deletion_blocked_reason === 'string' ? environment.deletion_blocked_reason : '环境任务状态尚未确认，请刷新后重试'
}
