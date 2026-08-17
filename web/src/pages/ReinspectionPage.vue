<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../services/api'
import { useWorkspaceStore } from '../stores/workspace'
const workspace = useWorkspaceStore()
const selectedPlanID = ref('')
const comment = ref('')
const passed = ref(true)
const lastReinspectionID = ref('')
const restrictedPlans = computed(() => workspace.plans.filter((item) => item.status === 'restricted'))
const selectedPlan = computed(() => restrictedPlans.value.find((item) => item.id === selectedPlanID.value))
const readyDefects = computed(() => workspace.defects.filter((item) => item.plan_id === selectedPlanID.value && item.status === 'ready_for_reinspection'))
async function submit() { if (!selectedPlan.value) return; const result = await api.reinspect(selectedPlan.value, readyDefects.value.map((item) => item.id), passed.value, comment.value); lastReinspectionID.value = result.item.id; await workspace.refresh() }
async function approve() { if (!lastReinspectionID.value) return; await api.reviewReinspection(lastReinspectionID.value); await workspace.refresh() }
</script>

<template>
  <section class="page-heading"><div><h2>复检</h2><p>复检必须关联本次限用周期和已完成整改的缺陷</p></div></section>
  <section class="form-layout">
    <label>限用计划<select v-model="selectedPlanID"><option value="">请选择</option><option v-for="plan in restrictedPlans" :key="plan.id" :value="plan.id">{{ plan.id }} · {{ plan.equipment_id }}</option></select></label>
    <div class="selected-list"><span v-for="item in readyDefects" :key="item.id">{{ item.checklist_item_id }}</span><p v-if="selectedPlanID && readyDefects.length === 0" class="empty">尚无可复检的已整改缺陷</p></div>
    <label>复检结论<select v-model="passed"><option :value="true">合格</option><option :value="false">不合格</option></select></label>
    <label>复检意见<textarea v-model="comment" rows="4" placeholder="填写实测结果和复检说明"></textarea></label>
    <div class="button-row"><button class="primary-button" :disabled="!selectedPlan || readyDefects.length === 0" @click="submit">提交复检</button><button class="secondary-button" :disabled="!lastReinspectionID" @click="approve">安全主管复核</button></div>
  </section>
</template>

