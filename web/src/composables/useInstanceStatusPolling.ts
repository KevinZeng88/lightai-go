import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'

const TRANSITIONAL_STATES = new Set(['pending', 'scheduled', 'starting', 'loading', 'stopping', 'provisioning', 'initializing'])

/**
 * Instance-status-aware polling composable.
 *
 * - transitional states → fast polling (3s)
 * - stable states → slow polling (15s)
 * - document hidden → pause
 * - window focus → immediate refresh
 * - route leave → stop
 * - preserves old data on error
 */
export function useInstanceStatusPolling(
  fetcher: () => Promise<void>,
  getStates: () => string[],
  opts: { fastMs?: number; slowMs?: number } = {},
) {
  const { fastMs = 3000, slowMs = 15000 } = opts

  const loading = ref(false)
  const lastUpdate = ref('')
  const refreshError = ref(false)
  let timer: ReturnType<typeof setInterval> | null = null
  let inflight = false
  let currentIntervalMs = slowMs

  const router = useRouter()

  const intervalMs = computed(() => {
    const states = getStates()
    if (states.some(s => TRANSITIONAL_STATES.has(s))) return fastMs
    return slowMs
  })

  async function refresh() {
    if (inflight) return
    inflight = true
    try {
      await fetcher()
      lastUpdate.value = new Date().toISOString()
      refreshError.value = false
    } catch {
      refreshError.value = true
    } finally {
      inflight = false
    }
  }

  async function initialLoad() {
    loading.value = true
    await refresh()
    loading.value = false
  }

  function startTimer() {
    stopTimer()
    currentIntervalMs = intervalMs.value
    timer = setInterval(refresh, currentIntervalMs)
  }

  function stopTimer() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  // Restart timer when interval changes (state transitions).
  watch(intervalMs, (newMs) => {
    if (newMs !== currentIntervalMs && timer) {
      startTimer()
    }
  })

  // Visibility: pause when hidden, refresh on focus.
  function onVisibilityChange() {
    if (document.hidden) {
      stopTimer()
    } else {
      refresh()
      startTimer()
    }
  }

  function onFocus() {
    if (!document.hidden) {
      refresh()
    }
  }

  // Route leave guard: stop timer.
  const removeRouteGuard = router.beforeEach(() => {
    stopTimer()
  })

  onMounted(() => {
    initialLoad()
    startTimer()
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('focus', onFocus)
  })

  onUnmounted(() => {
    stopTimer()
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener('focus', onFocus)
    removeRouteGuard()
  })

  return {
    loading,
    lastUpdate,
    refreshError,
    refresh,
    startTimer,
    stopTimer,
  }
}
