export type StatusType = 'success' | 'warning' | 'danger' | 'info' | ''

export function getStatusType(status: string): StatusType {
  switch (status) {
    case 'online':
    case 'healthy':
    case 'available':
    case 'ready':
    case 'running':
    case 'success':
    case 'active':
      return 'success'
    case 'warning':
    case 'pending':
    case 'degraded':
    case 'starting':
    case 'stopping':
    case 'reserved':
    case 'leased':
    case 'syncing':
      return 'warning'
    case 'error':
    case 'failed':
    case 'unhealthy':
      return 'danger'
    case 'offline':
    case 'unavailable':
    case 'unknown':
    case 'disabled':
    case 'inactive':
    case 'stopped':
    case 'deleted':
    case 'released':
      return 'info'
    default:
      return 'info'
  }
}

export function translateStatus(status: string, t: (key: string) => string): string {
  const key = `status.${status}`
  const translated = t(key)
  // If the i18n key doesn't exist, vue-i18n returns the key itself
  return translated === key || translated.startsWith('status.') ? status : translated
}

const STATUS_REASON_MAP: Record<string, string> = {
  'runtime verified for node': 'runtime.statusReason.verifiedForNode',
  'docker image is not present on node': 'runtime.statusReason.missingImage',
  'node has no matching GPU vendor': 'runtime.statusReason.noMatchingGPU',
  'docker availability has not been verified': 'runtime.statusReason.dockerNotVerified',
  'Huawei/Ascend runtime is a template only': 'runtime.statusReason.templateOnly',
}

export function translateStatusReason(reason: string, t: (key: string, params?: any) => string): string {
  if (!reason) return ''
  const key = STATUS_REASON_MAP[reason]
  if (key) {
    const translated = t(key)
    return translated === key ? reason : translated
  }
  // Fallback: try to find by prefix match
  for (const [pattern, mappedKey] of Object.entries(STATUS_REASON_MAP)) {
    if (reason.toLowerCase().includes(pattern.toLowerCase())) {
      const translated = t(mappedKey)
      return translated === mappedKey ? reason : translated
    }
  }
  return reason
}
