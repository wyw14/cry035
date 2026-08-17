<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../services/api'
import type { Program } from '../types/domain'
const programs = ref<Program[]>([])
onMounted(async () => { programs.value = (await api.programs()).items })
const roles = [['安全主管', '配置、复核、恢复运行'], ['计划员', '设备档案与保养排程'], ['维保人员', '执行、证据、整改'], ['复核员', '复核与复检'], ['审计查看者', '只读查询与导出']]
</script>

<template>
  <section class="page-heading"><div><h2>配置</h2><p>保养项目、法定周期和服务端角色边界</p></div></section>
  <section class="settings-grid">
    <div><h3>保养项目库</h3><ul class="plain-list"><li v-for="item in programs" :key="item.id"><strong>{{ item.name }}</strong><span>每 {{ item.statutory_cycle_days }} 天 · v{{ item.version }}</span></li></ul></div>
    <div><h3>角色权限</h3><ul class="plain-list"><li v-for="item in roles" :key="item[0]"><strong>{{ item[0] }}</strong><span>{{ item[1] }}</span></li></ul></div>
  </section>
</template>

