<script setup lang="ts">
import type { Equipment } from '../../types/domain'
defineProps<{ items: Equipment[] }>()
const labels: Record<string, string> = { running: '运行', maintenance: '保养中', limited: '限用', stopped: '停用' }
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead><tr><th>设备编号</th><th>设备名称</th><th>位置</th><th>责任单位</th><th>状态</th></tr></thead>
      <tbody>
        <tr v-for="item in items" :key="item.id">
          <td class="mono">{{ item.code }}</td><td>{{ item.name }}</td><td>{{ item.location }}</td>
          <td>{{ item.responsible_unit }}</td><td><span class="status" :class="item.status">{{ labels[item.status] }}</span></td>
        </tr>
        <tr v-if="items.length === 0"><td colspan="5" class="empty">暂无设备</td></tr>
      </tbody>
    </table>
  </div>
</template>

