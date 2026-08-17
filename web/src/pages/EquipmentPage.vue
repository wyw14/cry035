<script setup lang="ts">
import { computed } from 'vue'
import EquipmentTable from '../features/equipment/EquipmentTable.vue'
import { useWorkspaceStore } from '../stores/workspace'
const workspace = useWorkspaceStore()
const running = computed(() => workspace.equipment.filter((item) => item.status === 'running').length)
const limited = computed(() => workspace.equipment.filter((item) => item.status === 'limited').length)
</script>

<template>
  <section class="page-heading"><div><h2>设备总览</h2><p>楼宇内垂直运输设备的当前运行与保养状态</p></div></section>
  <section class="metrics">
    <div><span>设备总数</span><strong>{{ workspace.equipment.length }}</strong></div>
    <div><span>正常运行</span><strong>{{ running }}</strong></div>
    <div><span>限用设备</span><strong class="danger">{{ limited }}</strong></div>
    <div><span>未关闭严重缺陷</span><strong>{{ workspace.criticalDefects.length }}</strong></div>
  </section>
  <section class="content-section"><h3>设备台账</h3><EquipmentTable :items="workspace.equipment" /></section>
</template>

