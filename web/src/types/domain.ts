export type OperatingStatus = 'running' | 'maintenance' | 'limited' | 'stopped'
export type PlanStatus = 'planned' | 'in_progress' | 'suspended' | 'overdue' | 'pending_review' | 'qualified' | 'restricted' | 'restored' | 'cancelled'

export interface Equipment {
  id: string
  code: string
  name: string
  building_id: string
  model_id: string
  location: string
  responsible_unit: string
  status: OperatingStatus
  version: number
}

export interface Plan {
  id: string
  equipment_id: string
  program_id: string
  window: { start: string; end: string }
  assignee: string
  status: PlanStatus
  version: number
  restriction_key?: string
}

export interface Defect {
  id: string
  equipment_id: string
  plan_id: string
  checklist_item_id: string
  level: 'minor' | 'major' | 'critical'
  description: string
  status: 'open' | 'rectifying' | 'ready_for_reinspection' | 'closed'
  due_at: string
  version: number
  restriction_key: string
}

export interface Alert {
  id: string
  equipment_id: string
  plan_id: string
  level: string
  message: string
  created_at: string
}

export interface Program {
  id: string
  name: string
  version: number
  statutory_cycle_days: number
  applicable_categories: string[]
  active: boolean
}

export interface Page<T> { items: T[]; page: number; page_size: number; total: number }
