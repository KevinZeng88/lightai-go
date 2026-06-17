<template>
  <div class="gpus-page">
    <div class="page-header">
      <h2>{{ t('gpus.title') }} ({{ filteredGPUs.length }})</h2>
      <div class="header-actions">
        <el-button size="small" @click="resetWidths">{{ t('common.reset') }}</el-button>
        <el-button @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <div class="toolbar">
      <el-select v-model="vendorFilter" :placeholder="t('gpus.vendor')" clearable size="small" style="width: 120px">
        <el-option label="NVIDIA" value="nvidia" />
        <el-option label="MetaX" value="metax" />
      </el-select>
      <el-select v-model="healthFilter" :placeholder="t('gpus.health')" clearable size="small" style="width: 120px; margin-left: 8px">
        <el-option :label="t('status.healthy')" value="healthy" />
        <el-option :label="t('status.warning')" value="warning" />
        <el-option :label="t('status.unhealthy')" value="unhealthy" />
      </el-select>
      <el-input v-model="search" :placeholder="t('common.search')" clearable size="small" style="width: 240px; margin-left: 8px" />
    </div>

    <div class="table-wrap">
    <el-table :data="filteredGPUs" v-loading="loading" size="small" @row-click="openDetail" style="width: 100%">
      <el-table-column prop="health" :label="t('gpus.health')" :width="colWidth('health')">
        <template #header>
          <span>{{ t('gpus.health') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('health', $event)"></span>
        </template>
        <template #default="{ row }"><StatusTag :status="row.health" /></template>
      </el-table-column>
      <el-table-column prop="vendor" :label="t('gpus.vendor')" :width="colWidth('vendor')">
        <template #header>
          <span>{{ t('gpus.vendor') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('vendor', $event)"></span>
        </template>
        <template #default="{ row }">{{ row.vendor }}</template>
      </el-table-column>
      <el-table-column prop="name" :label="t('gpus.name')" :width="colWidth('name')" show-overflow-tooltip>
        <template #header>
          <span>{{ t('gpus.name') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('name', $event)"></span>
        </template>
      </el-table-column>
      <el-table-column prop="index" :label="t('gpus.index')" :width="colWidth('index')" align="center">
        <template #header>
          <span>{{ t('gpus.index') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('index', $event)"></span>
        </template>
      </el-table-column>
      <el-table-column :label="t('gpus.uuid')" :width="colWidth('uuid')">
        <template #header>
          <span>{{ t('gpus.uuid') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('uuid', $event)"></span>
        </template>
        <template #default="{ row }">
          <span class="mono" :title="row.uuid">{{ shortId(row.uuid) }}</span>
          <CopyButton :text="row.uuid" />
        </template>
      </el-table-column>
      <el-table-column :label="t('gpus.memory')" :width="colWidth('memory')">
        <template #header>
          <span>{{ t('gpus.memory') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('memory', $event)"></span>
        </template>
        <template #default="{ row }">
          <div class="mem-bar">
            <el-progress :percentage="memPercent(row)" :stroke-width="8" :show-text="false" style="width:60px" />
            <span class="mem-text">{{ formatGB(row.memory_used_bytes) }} / {{ formatGB(row.memory_total_bytes) }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('gpus.gpuUtilization')" :width="colWidth('gpuUtil')" align="right">
        <template #header>
          <span>{{ t('gpus.gpuUtilization') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('gpuUtil', $event)"></span>
        </template>
        <template #default="{ row }">{{ formatPercent(row.gpu_utilization_percent) }}</template>
      </el-table-column>
      <el-table-column :label="t('gpus.memoryUtilization')" :width="colWidth('memUtil')" align="right">
        <template #header>
          <span>{{ t('gpus.memoryUtilization') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('memUtil', $event)"></span>
        </template>
        <template #default="{ row }">{{ formatPercent(row.memory_utilization_percent) }}</template>
      </el-table-column>
      <el-table-column :label="t('gpus.temperature')" :width="colWidth('temp')" align="right">
        <template #header>
          <span>{{ t('gpus.temperature') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('temp', $event)"></span>
        </template>
        <template #default="{ row }">{{ formatCelsius(row.temperature_celsius) }}</template>
      </el-table-column>
      <el-table-column :label="t('gpus.powerDraw')" :width="colWidth('power')" align="right">
        <template #header>
          <span>{{ t('gpus.powerDraw') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('power', $event)"></span>
        </template>
        <template #default="{ row }">{{ formatWatts(row.power_draw_watts) }}</template>
      </el-table-column>
      <el-table-column :label="t('gpus.collectedAt')" :width="colWidth('collectedAt')">
        <template #header>
          <span>{{ t('gpus.collectedAt') }}</span>
          <span class="resize-handle" @mousedown.prevent="startResize('collectedAt', $event)"></span>
        </template>
        <template #default="{ row }"><span :title="formatDateTime(row.collected_at)">{{ formatRelativeTime(row.collected_at, locale) }}</span></template>
      </el-table-column>
      <template #empty>{{ t('gpus.noGpus') }}</template>
    </el-table>
    </div>

    <el-drawer v-model="drawerVisible" :title="t('gpus.detail')" size="550px">
      <template v-if="selectedGPU">
        <el-descriptions :column="1" border size="small" :title="t('gpus.deviceInfo')">
          <el-descriptions-item :label="t('gpus.index')">{{ selectedGPU.index }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.name')">{{ selectedGPU.name }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.vendor')">{{ selectedGPU.vendor }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.uuid')">
            <span class="mono">{{ selectedGPU.uuid }}</span>
            <CopyButton :text="selectedGPU.uuid" />
          </el-descriptions-item>
          <el-descriptions-item :label="t('gpus.pciBusId')">{{ selectedGPU.pci_bus_id || '--' }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.driverVersion')">{{ selectedGPU.driver_version || '--' }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.status')"><StatusTag :status="selectedGPU.status" /></el-descriptions-item>
          <el-descriptions-item :label="t('gpus.health')"><StatusTag :status="selectedGPU.health" /></el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" :title="t('gpus.memoryInfo')" style="margin-top: 16px">
          <el-descriptions-item :label="t('gpus.memoryTotal')">{{ formatGB(selectedGPU.memory_total_bytes) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.memoryUsed')">{{ formatGB(selectedGPU.memory_used_bytes) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.memoryFree')">{{ formatGB(selectedGPU.memory_free_bytes) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.memoryUtilization')">{{ formatPercent(selectedGPU.memory_utilization_percent) }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" :title="t('gpus.thermalInfo')" style="margin-top: 16px">
          <el-descriptions-item :label="t('gpus.gpuUtilization')">{{ formatPercent(selectedGPU.gpu_utilization_percent) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.temperature')">{{ formatCelsius(selectedGPU.temperature_celsius) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.powerDraw')">{{ formatWatts(selectedGPU.power_draw_watts) }}</el-descriptions-item>
          <el-descriptions-item :label="t('gpus.collectedAt')">{{ formatDateTime(selectedGPU.collected_at) }}</el-descriptions-item>
        </el-descriptions>

        <el-collapse style="margin-top: 16px">
          <el-collapse-item :title="t('common.rawJson')">
            <pre class="raw-json">{{ JSON.stringify(selectedGPU, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { fetchGPUs, type GPU } from '@/api/gpus'
import StatusTag from '@/components/StatusTag.vue'
import CopyButton from '@/components/CopyButton.vue'
import { formatDateTime, formatPercent, formatCelsius, formatWatts, shortId, formatRelativeTime, formatGB } from '@/utils/format'
import { useResizableColumns } from '@/composables/useResizableColumns'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t, locale } = useI18n()

const GPU_COLUMN_DEFAULTS = {
  health: 80, vendor: 80, name: 200, index: 55, uuid: 240,
  memory: 180, gpuUtil: 100, memUtil: 100, temp: 80, power: 70, collectedAt: 100,
}
const { colWidth, startResize, resetWidths } = useResizableColumns('gpuDevices', GPU_COLUMN_DEFAULTS, 50)

const gpus = ref<GPU[]>([])
const search = ref('')
const vendorFilter = ref('')
const healthFilter = ref('')
const drawerVisible = ref(false)
const selectedGPU = ref<GPU | null>(null)

const { loading, refresh } = useAutoRefresh(async () => {
  gpus.value = await fetchGPUs()
})

const filteredGPUs = computed(() => {
  let result = gpus.value
  if (vendorFilter.value) result = result.filter(g => g.vendor === vendorFilter.value)
  if (healthFilter.value) result = result.filter(g => g.health === healthFilter.value)
  if (search.value) {
    const q = search.value.toLowerCase()
    result = result.filter(g =>
      g.name.toLowerCase().includes(q) || g.uuid.toLowerCase().includes(q) || g.node_id.toLowerCase().includes(q)
    )
  }
  return result
})

function openDetail(row: GPU) {
  selectedGPU.value = row
  drawerVisible.value = true
}

function memPercent(gpu: GPU): number {
  if (!gpu.memory_total_bytes) return 0
  return Math.round((gpu.memory_used_bytes / gpu.memory_total_bytes) * 100)
}

</script>

<style scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;
}
.page-header h2 { margin: 0; }
.header-actions { display: flex; gap: 8px; }
.toolbar { margin-bottom: 12px; }
.table-wrap { overflow-x: auto; }
.mono { font-family: monospace; font-size: 12px; }
.mem-bar { display: flex; align-items: center; gap: 6px; }
.mem-text { font-size: 12px; white-space: nowrap; color: var(--el-text-color-secondary); }
.resize-handle {
  display: inline-block; width: 6px; height: 100%; cursor: col-resize;
  position: absolute; right: 0; top: 0; bottom: 0;
}
.resize-handle:hover { background: var(--el-color-primary-light-5); }
:deep(.el-table th) { position: relative; }
.raw-json { max-height: 400px; overflow: auto; font-size: 12px; }
</style>
