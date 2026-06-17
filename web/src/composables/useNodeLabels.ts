import { ref, computed } from 'vue'
import { apiClient } from '@/api/client'

const nodeMap = ref<Map<string, string>>(new Map())
let loaded = false

export function useNodeLabels() {
  async function loadNodes() {
    if (loaded) return
    try {
      const ns = await apiClient.get('/nodes')
      nodeMap.value = new Map((ns || []).map((n: any) => [n.id, n.hostname || n.id]))
      loaded = true
    } catch { nodeMap.value = new Map() }
  }

  function nodeLabel(nodeId: string): string {
    const host = nodeMap.value.get(nodeId)
    return host ? `${host} (${nodeId.slice(0, 12)})` : nodeId
  }

  const nodes = computed(() => {
    const result: { id: string; label: string }[] = []
    for (const [id, host] of nodeMap.value) {
      result.push({ id, label: `${host} (${id.slice(0, 12)})` })
    }
    return result
  })

  return { loadNodes, nodeLabel, nodes, nodeMap }
}
