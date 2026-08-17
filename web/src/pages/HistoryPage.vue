<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../services/api'
import { useWorkspaceStore } from '../stores/workspace'
const workspace = useWorkspaceStore()
const equipmentID = ref('')
const history = ref<any>(null)
async function load() { if (!equipmentID.value) return; history.value = (await api.history(equipmentID.value)).item }
</script>

<template>
  <section class="page-heading"><div><h2>设备时间线</h2><p>计划、缺陷、费用与审计事件按时间追溯</p></div><a v-if="equipmentID" class="secondary-button link-button" :href="`/api/v1/equipment/${equipmentID}/export.csv`">导出 CSV</a></section>
  <section class="filter-band"><select v-model="equipmentID" @change="load"><option value="">选择设备</option><option v-for="item in workspace.equipment" :key="item.id" :value="item.id">{{ item.code }} · {{ item.name }}</option></select></section>
  <section v-if="history" class="timeline">
    <div v-for="event in history.events" :key="event.id"><time>{{ new Date(event.occurred_at).toLocaleString('zh-CN') }}</time><strong>{{ event.action }}</strong><span>{{ event.actor }}</span></div>
    <div v-for="service in history.services" :key="service.id"><time>{{ new Date(service.serviced_at).toLocaleString('zh-CN') }}</time><strong>{{ service.vendor_name }}</strong><span>¥ {{ (service.amount_cents / 100).toFixed(2) }}</span></div>
    <p v-if="history.events.length + history.services.length === 0" class="empty">暂无时间线记录</p>
  </section>
  <section v-else class="empty-state">选择设备查看完整历史</section>
</template>

