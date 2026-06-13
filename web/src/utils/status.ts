export type StatusType = 'success' | 'warning' | 'danger' | 'info' | ''

export function getStatusType(status: string): StatusType {
  switch (status) {
    case 'online':
    case 'healthy':
    case 'available':
    case 'ready':
    case 'running':
      return 'success'
    case 'warning':
    case 'pending':
    case 'degraded':
      return 'warning'
    case 'error':
    case 'failed':
    case 'unhealthy':
      return 'danger'
    case 'offline':
    case 'unavailable':
    case 'unknown':
    case 'disabled':
      return 'info'
    case 'starting':
    case 'syncing':
      return ''
    default:
      return 'info'
  }
}

export function translateStatus(status: string, t: (key: string) => string): string {
  const key = `status.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}
