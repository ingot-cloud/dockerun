import type { TemplateParameter } from '@/api/applicationTemplates'
import { environmentParameterPairs, type EnvironmentParameterPair } from '@/utils/applicationDetail'

export interface DeploymentParameterDraft {
  pairs: EnvironmentParameterPair[]
  savedParameters: Record<string, string>
  baseline: string
}

function pairSnapshot(pairs: EnvironmentParameterPair[]): string {
  return JSON.stringify(pairs.map(({key, value}) => [key, value]).sort(([leftKey, leftValue], [rightKey, rightValue]) => leftKey!.localeCompare(rightKey!) || leftValue!.localeCompare(rightValue!)))
}

export function createDeploymentParameterDraft(parameters: Record<string, string> = {}, schema: TemplateParameter[] = []): DeploymentParameterDraft {
  const pairs = environmentParameterPairs(parameters, schema)
  return { pairs, savedParameters: { ...parameters }, baseline: pairSnapshot(pairs) }
}

export function deploymentParametersChanged(draft?: DeploymentParameterDraft): boolean {
  return !!draft && pairSnapshot(draft.pairs) !== draft.baseline
}

export function deploymentParameterDraftsSnapshot(drafts: Record<string, DeploymentParameterDraft>): Record<string, string> {
  return Object.fromEntries(Object.entries(drafts).filter(([, draft]) => deploymentParametersChanged(draft)).map(([id, draft]) => [id, pairSnapshot(draft.pairs)]))
}
