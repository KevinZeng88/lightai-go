<template>
  <div>
    <h2>Prometheus</h2>
    <el-descriptions :column="1" border style="margin-top: 16px">
      <el-descriptions-item label="URL">{{ promUrl }}</el-descriptions-item>
      <el-descriptions-item label="Status">{{ status }}</el-descriptions-item>
      <el-descriptions-item label="Scrape Targets">
        Targets are discovered via <code>/metrics/targets</code> HTTP SD.
      </el-descriptions-item>
    </el-descriptions>
    <div style="margin-top: 16px">
      <el-button type="primary" @click="openProm">Open Prometheus</el-button>
      <el-tag v-if="!running" type="danger" style="margin-left: 8px">Not Running</el-tag>
    </div>
    <el-alert v-if="!running" type="info" :closable="false" style="margin-top: 12px"
      title="Prometheus is not running. Start it with: bash scripts/observability-up.sh" />
    <el-alert type="info" :closable="false" style="margin-top: 12px"
      title="Query API integration will be available in a future release." />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const promUrl = `http://${window.location.hostname}:9090`
const status = ref('Checking...')
const running = ref(false)

onMounted(async () => {
  try {
    await fetch(promUrl + '/-/healthy')
    status.value = 'Running'
    running.value = true
  } catch {
    status.value = 'Not running'
    running.value = false
  }
})

function openProm() { window.open(promUrl, '_blank') }
</script>
