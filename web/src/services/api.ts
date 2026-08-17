import type { Alert, Defect, Equipment, Page, Plan, Program } from '../types/domain'

const headers = {
  'Content-Type': 'application/json',
  'X-Actor': 'demo-safety-manager',
  'X-Role': 'safety_manager'
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { ...init, headers: { ...headers, ...init?.headers } })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ message: response.statusText }))
    throw new Error(payload.message ?? '请求失败')
  }
  return response.json() as Promise<T>
}

export const api = {
  equipment: () => request<Page<Equipment>>('/equipment?page_size=100'),
  plans: () => request<Page<Plan>>('/plans?page_size=100'),
  defects: () => request<Page<Defect>>('/defects?page_size=100'),
  alerts: () => request<Page<Alert>>('/alerts?page_size=100'),
  programs: () => request<Page<Program>>('/programs?page_size=100'),
  history: (equipmentID: string) => request<{ item: unknown }>(`/equipment/${equipmentID}/history`),
  createPlan: (payload: unknown, key: string) => request<{ item: Plan }>('/plans', {
    method: 'POST', headers: { 'Idempotency-Key': key }, body: JSON.stringify(payload)
  }),
  transition: (plan: Plan, action: 'start' | 'suspend' | 'cancel') => request<{ item: Plan }>(`/plans/${plan.id}/${action}`, {
    method: 'POST', body: JSON.stringify({ expected_version: plan.version })
  }),
  rectify: (defect: Defect, action: string) => request<{ item: Defect }>(`/defects/${defect.id}/rectify`, {
    method: 'POST', body: JSON.stringify({ expected_version: defect.version, action })
  }),
  reinspect: (plan: Plan, defectIDs: string[], passed: boolean, comment: string) => request<{ item: { id: string }; plan: Plan }>(`/plans/${plan.id}/reinspection`, {
    method: 'POST', body: JSON.stringify({ equipment_id: plan.equipment_id, defect_ids: defectIDs, restriction_key: plan.restriction_key, passed, comment })
  }),
  reviewReinspection: (id: string) => request<{ item: Plan }>(`/reinspections/${id}/review`, { method: 'POST', body: '{}' }),
  restore: (plan: Plan) => request<{ item: Plan }>(`/plans/${plan.id}/restore`, { method: 'POST', body: JSON.stringify({ expected_version: plan.version }) })
}
