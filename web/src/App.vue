<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import { useWorkspaceStore } from './stores/workspace'

const workspace = useWorkspaceStore()
onMounted(() => workspace.refresh())

const navigation = [
  ['/equipment', '设备总览'], ['/plans', '计划日历'], ['/execution', '执行表单'],
  ['/defects', '缺陷整改'], ['/reinspection', '复检'], ['/history', '设备时间线'], ['/settings', '配置']
]
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">VT</span>
        <div><strong>云梯保养</strong><small>安全运行中心</small></div>
      </div>
      <nav aria-label="主导航">
        <RouterLink v-for="item in navigation" :key="item[0]" :to="item[0]">{{ item[1] }}</RouterLink>
      </nav>
      <div class="sidebar-footer">
        <span class="status-dot"></span>
        本地服务已连接
      </div>
    </aside>
    <main>
      <header class="topbar">
        <div><span class="eyebrow">物业安全运营</span><h1>垂直运输设备保养平台</h1></div>
        <button class="icon-button" title="刷新数据" aria-label="刷新数据" @click="workspace.refresh">↻</button>
      </header>
      <p v-if="workspace.error" class="error-banner">{{ workspace.error }}</p>
      <RouterView />
    </main>
  </div>
</template>
