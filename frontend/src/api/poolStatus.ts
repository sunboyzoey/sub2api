import { apiClient } from './client'
import type { PoolStatusSummary } from '@/types'

export async function getPoolStatus(): Promise<PoolStatusSummary> {
  const { data } = await apiClient.get<PoolStatusSummary>('/settings/pool-status')
  return data
}
