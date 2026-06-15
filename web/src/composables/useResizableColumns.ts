import { ref, onMounted, watch } from 'vue'

const STORAGE_PREFIX = 'lightai.tablePrefs.'

export interface ColumnWidths {
  [key: string]: number
}

export function useResizableColumns(tableKey: string, defaults: ColumnWidths, minWidth = 80) {
  const widths = ref<ColumnWidths>({ ...defaults })
  const storageKey = STORAGE_PREFIX + tableKey

  // Load saved widths from localStorage.
  onMounted(() => {
    try {
      const saved = localStorage.getItem(storageKey)
      if (saved) {
        const parsed = JSON.parse(saved)
        if (parsed && typeof parsed === 'object') {
          // Merge saved with defaults (in case new columns were added).
          widths.value = { ...defaults, ...parsed }
        }
      }
    } catch {
      // Corrupted data — keep defaults.
    }
  })

  // Persist widths to localStorage on change.
  watch(widths, (val) => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(val))
    } catch {
      // localStorage full or unavailable — silently skip.
    }
  }, { deep: true })

  // Reset to defaults.
  function resetWidths() {
    widths.value = { ...defaults }
    try {
      localStorage.removeItem(storageKey)
    } catch { /* */ }
  }

  // Start column resize drag.
  function startResize(columnKey: string, event: MouseEvent) {
    const startX = event.clientX
    const startWidth = widths.value[columnKey] ?? defaults[columnKey] ?? 150

    function onMouseMove(e: MouseEvent) {
      const delta = e.clientX - startX
      const newWidth = Math.max(minWidth, startWidth + delta)
      widths.value = { ...widths.value, [columnKey]: newWidth }
    }

    function onMouseUp() {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }

  // Column width as CSS value.
  function colWidth(key: string): string {
    const w = widths.value[key] ?? defaults[key]
    return w ? w + 'px' : 'auto'
  }

  return { widths, colWidth, startResize, resetWidths }
}

// Helper: group GPUs by node_id.
export function groupGpusByNodeId<T extends { node_id: string }>(gpus: T[]): Map<string, T[]> {
  const map = new Map<string, T[]>()
  for (const gpu of gpus) {
    const list = map.get(gpu.node_id) || []
    list.push(gpu)
    map.set(gpu.node_id, list)
  }
  return map
}
