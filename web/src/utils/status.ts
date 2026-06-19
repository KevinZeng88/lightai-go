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
    case 'needs_check':
      return 'warning'
    case 'error':
    case 'failed':
    case 'unhealthy':
    case 'unsupported_device':
    case 'missing_image':
      return 'danger'
    case 'offline':
    case 'unavailable':
    case 'unknown':
    case 'disabled':
    case 'inactive':
    case 'stopped':
    case 'deleted':
    case 'released':
    case 'template_only':
      return 'info'
    default:
      return 'info'
  }
}

export function translateStatus(status: string, t: (key: string) => string): string {
  if (!status) return '?'
  const key = `status.${status}`
  const translated = t(key)
  if (translated === key || (typeof translated === 'string' && translated.startsWith('status.'))) {
    return status
  }
  return translated
}

const STATUS_REASON_MAP: Record<string, string> = {
  'runtime verified for node': 'runtime.statusReason.verifiedForNode',
  'docker image is not present on node': 'runtime.statusReason.missingImage',
  'node has no matching GPU vendor': 'runtime.statusReason.noMatchingGPU',
  'docker availability has not been verified': 'runtime.statusReason.dockerNotVerified',
  'Huawei/Ascend runtime is a template only': 'runtime.statusReason.templateOnly',
  'node is offline': 'runtime.statusReason.nodeOffline',
  'awaiting agent verification of Docker and image availability': 'runtime.statusReason.awaitingAgentCheck',
  'node has no advertised address or metrics port': 'runtime.statusReason.agentUnreachable',
  'agent unreachable': 'runtime.statusReason.agentUnreachable',
}

export function translateStatusReason(reason: string, t: (key: string, params?: any) => string): string {
  if (!reason) return ''
  // Exact match
  const key = STATUS_REASON_MAP[reason]
  if (key) {
    const translated = t(key)
    if (typeof translated === 'string' && translated !== key && !translated.startsWith('runtime.')) {
      return translated
    }
  }
  // Substring match
  for (const [pattern, mappedKey] of Object.entries(STATUS_REASON_MAP)) {
    if (reason.toLowerCase().includes(pattern.toLowerCase())) {
      const translated = t(mappedKey)
      if (typeof translated === 'string' && translated !== mappedKey && !translated.startsWith('runtime.')) {
        return translated
      }
    }
  }
  return reason
}
