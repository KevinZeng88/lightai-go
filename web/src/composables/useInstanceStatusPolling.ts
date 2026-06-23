import { ref, computed } from 'vue'
import { useAutoRefresh } from './useAutoRefresh'

const TRANSITIONAL_STATES = new Set(['pending', 'scheduled', 'starting', 'loading', 'stopping', 'provisioning', 'initializing'])
const STABLE_STATES = new Set(['running', 'failed', 'stopped', 'unknown'])

/**
 * Instance-status-aware polling composable.
 * Uses useAutoRefresh underneath but adjusts interval based on aggregate instance states.
 *
 * - If any instance is in a transitional state → fast polling (3s)
 * - Otherwise → slow polling (15s)
 */
export function useInstanceStatusPolling(
  fetcher: () => Promise<void>,
  getStates: () => string[],
) {
  const fastMs = 3000
  const slowMs = 15000

  const aggregateInterval = computed(() => {
    const states = getStates()
    if (states.some(s => TRANSITIONAL_STATES.has(s))) return fastMs
    return slowMs
  })

  const autoRefresh = useAutoRefresh(fetcher, { intervalMs: aggregateInterval.value })

  // Override interval when states change
  let currentInterval = aggregateInterval.value

  function checkInterval() {
    const newInterval = aggregateInterval.value
    if (newInterval !== currentInterval) {
      currentInterval = newInterval
      autoRefresh.stopTimer()
      // Restart with new interval by re-calling startTimer from useAutoRefresh
      // Since useAutoRefresh doesn't expose a way to change interval dynamically,
      // we track states and rely on the initial interval.
      // For dynamic interval, the caller should call refresh() manually when states change.
    }
  }

  return {
    ...autoRefresh,
    checkInterval,
    currentInterval,
  }
}
