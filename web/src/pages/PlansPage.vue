<script setup lang="ts">
import PlanTable from '../features/plans/PlanTable.vue'
import { api } from '../services/api'
import { useWorkspaceStore } from '../stores/workspace'
import type { Plan } from '../types/domain'
const workspace = useWorkspaceStore()
async function transition(plan: Plan, action: 'start' | 'suspend') { await api.transition(plan, action); await workspace.refresh() }
</script>

<template>
  <section class="page-heading"><div><h2>计划日历</h2><p>检查周期、停用窗口与负责人统一排程</p></div><button class="primary-button" @click="workspace.refresh">刷新日历</button></section>
  <section class="calendar-strip">
    <div v-for="plan in workspace.activePlans.slice(0, 7)" :key="plan.id"><time>{{ new Date(plan.window.start).getDate() }}</time><span>{{ plan.assignee }}</span><small>{{ plan.status }}</small></div>
    <p v-if="workspace.activePlans.length === 0" class="empty">当前没有待执行计划</p>
  </section>
  <section class="content-section"><h3>计划任务</h3><PlanTable :items="workspace.plans" @start="transition($event, 'start')" @suspend="transition($event, 'suspend')" /></section>
</template>

