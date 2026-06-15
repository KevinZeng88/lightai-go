import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

/**
 * Unified auto-refresh composable.
 *
 * - immediate refresh on mount
 * - interval refresh
 * - pauses when document is hidden
 * - immediately refreshes on window focus
 * - guards against in-flight re-entry
 * - stops on route leave / component unmount
 * - preserves old data on error
 */
export function useAutoRefresh(
  fetcher: () => Promise<void>,
  opts: { intervalMs?: number } = {},
) {
  const { intervalMs = 5000 } = opts

  const loading = ref(false)
  const lastUpdate = ref('')
  const refreshError = ref(false)
  let timer: ReturnType<typeof setInterval> | null = null
  let inflight = false

  const router = useRouter()

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
    timer = setInterval(refresh, intervalMs)
  }

  function stopTimer() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  // Visibility: pause when hidden, refresh on focus.
  function onVisibilityChange() {
    if (document.hidden) {
      stopTimer()
    } else {
      refresh() // immediate refresh on becoming visible
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
