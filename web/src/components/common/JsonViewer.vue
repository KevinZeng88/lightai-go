<template>
  <div class="json-viewer" :class="{ 'json-viewer--fullscreen': isFullscreen }">
    <div class="json-viewer__toolbar">
      <span v-if="title" class="json-viewer__title">{{ title }}</span>
      <div class="json-viewer__actions">
        <el-input
          v-if="searchable"
          v-model="searchTerm"
          size="small"
          :placeholder="t('common.search') || 'Search...'"
          clearable
          style="width:160px"
        />
        <el-button size="small" @click="toggleWrap">{{ wrapped ? 'No Wrap' : 'Wrap' }}</el-button>
        <el-button size="small" @click="doCopy">{{ t('common.copy') || 'Copy' }}</el-button>
        <el-button v-if="allowDownload" size="small" @click="doDownload">Download</el-button>
        <el-button size="small" @click="toggleFullscreen">{{ isFullscreen ? 'Exit' : 'Expand' }}</el-button>
      </div>
    </div>
    <div
      ref="contentRef"
      class="json-viewer__content"
      :style="{ maxHeight: isFullscreen ? 'none' : (maxHeight || '400px'), whiteSpace: wrapped ? 'pre-wrap' : 'pre' }"
    >
      <template v-if="searchTerm && filteredLines.length > 0">
        <div v-for="(line, i) in filteredLines" :key="i" class="json-viewer__line" v-html="line" />
      </template>
      <template v-else-if="searchTerm && filteredLines.length === 0">
        <div class="json-viewer__empty">No matches</div>
      </template>
      <template v-else>
        <pre class="json-viewer__pre">{{ formatted }}</pre>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  value: unknown
  title?: string
  defaultExpanded?: boolean
  maxHeight?: string
  readonly?: boolean
  allowDownload?: boolean
  searchable?: boolean
}>()

const searchTerm = ref('')
const wrapped = ref(true)
const isFullscreen = ref(false)
const contentRef = ref<HTMLElement | null>(null)

const formatted = computed(() => {
  if (props.value == null) return ''
  if (typeof props.value === 'string') {
    try {
      return JSON.stringify(JSON.parse(props.value), null, 2)
    } catch {
      return props.value
    }
  }
  try {
    return JSON.stringify(props.value, null, 2)
  } catch {
    return String(props.value)
  }
})

const rawText = computed(() => {
  if (props.value == null) return ''
  if (typeof props.value === 'string') {
    try { JSON.parse(props.value); return props.value } catch { return props.value }
  }
  try { return JSON.stringify(props.value) } catch { return String(props.value) }
})

const filteredLines = computed(() => {
  if (!searchTerm.value) return []
  const term = searchTerm.value.toLowerCase()
  const lines = formatted.value.split('\n')
  return lines
    .filter(line => line.toLowerCase().includes(term))
    .map(line => {
      const escaped = line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      const regex = new RegExp(`(${searchTerm.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
      return escaped.replace(regex, '<mark>$1</mark>')
    })
})

function toggleWrap() { wrapped.value = !wrapped.value }

async function doCopy() {
  try {
    await navigator.clipboard.writeText(formatted.value)
    ElMessage.success(t('common.copied') || 'Copied')
  } catch {
    ElMessage.error(t('common.copyFailed') || 'Copy failed')
  }
}

function doDownload() {
  const blob = new Blob([formatted.value], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'diagnostic.json'
  a.click()
  URL.revokeObjectURL(url)
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}
</script>

<style scoped>
.json-viewer {
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
}
.json-viewer--fullscreen {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: var(--el-bg-color);
  border-radius: 0;
  display: flex;
  flex-direction: column;
}
.json-viewer__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);
  gap: 8px;
  flex-shrink: 0;
}
.json-viewer__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.json-viewer__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.json-viewer__content {
  overflow: auto;
  padding: 12px;
  background: #111827;
  color: #e5e7eb;
  font-size: 12px;
  line-height: 1.5;
}
.json-viewer--fullscreen .json-viewer__content {
  flex: 1;
  max-height: none !important;
}
.json-viewer__pre {
  margin: 0;
  font-family: inherit;
  white-space: inherit;
}
.json-viewer__line {
  padding: 1px 0;
}
.json-viewer__line :deep(mark) {
  background: #fbbf24;
  color: #111827;
  border-radius: 2px;
}
.json-viewer__empty {
  color: #9ca3af;
  padding: 12px;
  text-align: center;
}
</style>
