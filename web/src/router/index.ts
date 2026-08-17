import { createRouter, createWebHistory } from 'vue-router'
import EquipmentPage from '../pages/EquipmentPage.vue'
import PlansPage from '../pages/PlansPage.vue'
import ExecutionPage from '../pages/ExecutionPage.vue'
import DefectsPage from '../pages/DefectsPage.vue'
import ReinspectionPage from '../pages/ReinspectionPage.vue'
import HistoryPage from '../pages/HistoryPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/equipment' },
    { path: '/equipment', component: EquipmentPage },
    { path: '/plans', component: PlansPage },
    { path: '/execution', component: ExecutionPage },
    { path: '/defects', component: DefectsPage },
    { path: '/reinspection', component: ReinspectionPage },
    { path: '/history', component: HistoryPage },
    { path: '/settings', component: SettingsPage }
  ]
})

