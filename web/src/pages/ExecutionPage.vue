<script setup lang="ts">
import { computed, ref } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
const workspace = useWorkspaceStore()
const selected = ref('')
const plans = computed(() => workspace.plans.filter((item) => item.status === 'in_progress' || item.status === 'overdue'))
const readings = ref([{ name: '层门门锁联动', result: 'pass', notes: '' }, { name: '制动器间隙', result: 'pass', notes: '' }])
</script>

<template>
  <section class="page-heading"><div><h2>执行表单</h2><p>按计划清单记录实测结果与现场证据</p></div></section>
  <section class="form-layout">
    <label>执行计划<select v-model="selected"><option value="">请选择</option><option v-for="plan in plans" :key="plan.id" :value="plan.id">{{ plan.id }} · {{ plan.assignee }}</option></select></label>
    <div class="checklist"><div v-for="row in readings" :key="row.name" class="check-row"><strong>{{ row.name }}</strong><select v-model="row.result"><option value="pass">合格</option><option value="fail">不合格</option></select><input v-model="row.notes" placeholder="实测值或备注" /></div></div>
    <label class="upload-zone">现场照片<input type="file" accept="image/png,image/jpeg,image/webp" /></label>
    <button class="primary-button" :disabled="!selected">提交待复核</button>
  </section>
</template>

