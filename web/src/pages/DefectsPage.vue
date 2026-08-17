<script setup lang="ts">
import { api } from '../services/api'
import { useWorkspaceStore } from '../stores/workspace'
import type { Defect } from '../types/domain'
const workspace = useWorkspaceStore()
async function rectify(item: Defect) { const action = window.prompt('填写整改措施'); if (!action) return; await api.rectify(item, action); await workspace.refresh() }
</script>

<template>
  <section class="page-heading"><div><h2>缺陷整改</h2><p>按缺陷等级跟踪整改期限与复检准备</p></div></section>
  <section class="content-section"><div class="table-wrap"><table><thead><tr><th>等级</th><th>设备</th><th>检查项</th><th>期限</th><th>状态</th><th>操作</th></tr></thead><tbody>
    <tr v-for="item in workspace.defects" :key="item.id"><td><span class="severity" :class="item.level">{{ item.level }}</span></td><td>{{ item.equipment_id }}</td><td>{{ item.checklist_item_id }}</td><td>{{ new Date(item.due_at).toLocaleDateString('zh-CN') }}</td><td>{{ item.status }}</td><td><button v-if="item.status !== 'closed'" class="text-button" @click="rectify(item)">登记整改</button></td></tr>
    <tr v-if="workspace.defects.length === 0"><td colspan="6" class="empty">暂无缺陷记录</td></tr>
  </tbody></table></div></section>
</template>
