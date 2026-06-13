<template>
  <div>
    <h2>Prometheus</h2>
    <el-descriptions :column="1" border style="margin-top: 16px">
      <el-descriptions-item label="URL">{{ promUrl }}</el-descriptions-item>
      <el-descriptions-item label="Status">{{ status }}</el-descriptions-item>
      <el-descriptions-item label="Scrape Targets">Discovered via Server /metrics/targets.</el-descriptions-item>
    </el-descriptions>
    <div style="margin-top: 16px">
      <el-button type="primary" @click="openProm">Open Prometheus</el-button>
      <el-tag v-if="!running" type="danger" style="margin-left: 8px">Not Running</el-tag>
    </div>
    <el-alert v-if="!running" type="info" :closable="false" style="margin-top: 12px"
      title="Prometheus is not running. Start it with: bash scripts/start-observability.sh" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const promUrl = `http://${window.location.hostname}:19090`
const status = ref('Checking...')
const running = ref(false)

onMounted(async () => {
  try {
    const resp = await fetch('/api/observability/status')
    const data = await resp.json()
    if (data.prometheus?.ready) { status.value = 'Running'; running.value = true }
    else { status.value = 'Not running'; running.value = false }
  } catch { status.value = 'Cannot check'; running.value = false }
})

function openProm() { window.open(promUrl, '_blank') }
</script>
