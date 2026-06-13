<template>
  <div>
    <h2>Grafana</h2>
    <el-descriptions :column="1" border style="margin-top: 16px">
      <el-descriptions-item label="URL">{{ grafUrl }}</el-descriptions-item>
      <el-descriptions-item label="Status">{{ status }}</el-descriptions-item>
      <el-descriptions-item label="Default Login">admin / lightai (dev only)</el-descriptions-item>
    </el-descriptions>
    <div style="margin-top: 16px">
      <el-button type="primary" @click="openGrafana">Open Grafana</el-button>
      <el-tag v-if="!running" type="danger" style="margin-left: 8px">Not Running</el-tag>
    </div>
    <el-alert v-if="!running" type="info" :closable="false" style="margin-top: 12px"
      title="Grafana is not running. Start it with: bash scripts/observability-up.sh" />

    <h4 style="margin-top: 24px">Dashboards</h4>
    <el-space wrap>
      <el-button v-for="d in dashboards" :key="d.uid" @click="openUrl(d.url)" size="small">
        {{ d.name }}
      </el-button>
    </el-space>

    <el-alert type="info" :closable="false" style="margin-top: 12px"
      title="iframe embedding will be available in a future release. Use external link for now." />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const grafUrl = 'http://127.0.0.1:13000'
const status = ref('Checking...')
const running = ref(false)

const dashboards = [
  { name: 'LightAI Overview', uid: 'lightai-overview', url: grafUrl + '/d/lightai-overview' },
  { name: 'GPU Resources', uid: 'lightai-gpu-resources', url: grafUrl + '/d/lightai-gpu-resources' },
  { name: 'Agent Health', uid: 'lightai-agent-health', url: grafUrl + '/d/lightai-agent-health' },
]

onMounted(async () => {
  try {
    await fetch(grafUrl + '/api/health')
    status.value = 'Running'
    running.value = true
  } catch {
    status.value = 'Not running'
    running.value = false
  }
})

function openGrafana() { window.open(grafUrl, '_blank') }
function openUrl(url: string) { window.open(url, '_blank') }
</script>
