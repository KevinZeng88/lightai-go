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
  return translated === key ? status : translated
}
