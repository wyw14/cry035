<script setup lang="ts">
import type { Plan } from '../../types/domain'
defineProps<{ items: Plan[] }>()
const emit = defineEmits<{ start: [plan: Plan]; suspend: [plan: Plan] }>()
const labels: Record<string, string> = { planned: '已计划', in_progress: '执行中', suspended: '暂停', overdue: '逾期', pending_review: '待复核', qualified: '合格', restricted: '限用', restored: '已恢复', cancelled: '已取消' }
const format = (value: string) => new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead><tr><th>停用窗口</th><th>设备</th><th>负责人</th><th>状态</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="item in items" :key="item.id">
          <td>{{ format(item.window.start) }} - {{ format(item.window.end) }}</td><td class="mono">{{ item.equipment_id }}</td>
          <td>{{ item.assignee }}</td><td><span class="status" :class="item.status">{{ labels[item.status] }}</span></td>
          <td class="actions">
            <button v-if="item.status === 'planned' || item.status === 'suspended'" class="text-button" @click="emit('start', item)">开始</button>
            <button v-if="item.status === 'planned' || item.status === 'in_progress'" class="text-button muted" @click="emit('suspend', item)">暂停</button>
          </td>
        </tr>
        <tr v-if="items.length === 0"><td colspan="5" class="empty">暂无保养计划</td></tr>
      </tbody>
    </table>
  </div>
</template>

