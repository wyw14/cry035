import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../services/api'
import type { Alert, Defect, Equipment, Plan } from '../types/domain'

export const useWorkspaceStore = defineStore('workspace', () => {
  const equipment = ref<Equipment[]>([])
  const plans = ref<Plan[]>([])
  const defects = ref<Defect[]>([])
  const alerts = ref<Alert[]>([])
  const loading = ref(false)
  const error = ref('')

  const activePlans = computed(() => plans.value.filter((item) => !['restored', 'cancelled'].includes(item.status)))
  const criticalDefects = computed(() => defects.value.filter((item) => item.level === 'critical' && item.status !== 'closed'))

  async function refresh() {
    loading.value = true
    error.value = ''
    try {
      const [equipmentPage, planPage, defectPage, alertPage] = await Promise.all([
        api.equipment(), api.plans(), api.defects(), api.alerts()
      ])
      equipment.value = equipmentPage.items
      plans.value = planPage.items
      defects.value = defectPage.items
      alerts.value = alertPage.items
    } catch (reason) {
      error.value = reason instanceof Error ? reason.message : '加载失败'
    } finally {
      loading.value = false
    }
  }

  return { equipment, plans, defects, alerts, loading, error, activePlans, criticalDefects, refresh }
})

