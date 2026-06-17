export function formatBytes(bytes: number | undefined | null): string {
  if (bytes == null || bytes === 0) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

export function formatPercent(val: number | undefined | null): string {
  if (val == null) return 'N/A'
  return val.toFixed(1) + '%'
}

export function formatCelsius(val: number | undefined | null): string {
  if (val == null) return 'N/A'
  return val.toFixed(1) + ' °C'
}

export function formatWatts(val: number | undefined | null): string {
  if (val == null) return 'N/A'
  return val.toFixed(1) + ' W'
}

export function formatDateTime(iso: string | undefined | null): string {
  if (!iso) return 'N/A'
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
}

export function shortId(id: string | undefined | null, prefix = 8, suffix = 6): string {
  if (!id) return 'N/A'
  if (id.length <= prefix + suffix + 3) return id
  return id.slice(0, prefix) + '...' + id.slice(-suffix)
}

export function formatRelativeTime(iso: string | undefined | null, locale?: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '-'

  const diff = Date.now() - d.getTime()
  const s = Math.floor(diff / 1000)
  const isZh = locale === 'zh-CN'

  // Future time (slight clock skew — treat as just now).
  if (s < 0) {
    if (s > -5) return isZh ? '<1秒前' : '<1s ago'
    return isZh ? '时间异常' : 'time anomaly'
  }

  // < 1 second
  if (s < 1) return isZh ? '<1秒前' : '<1s ago'

  // 1-59 seconds
  if (s < 60) return isZh ? `${s}秒前` : `${s}s ago`

  // 1-59 minutes
  const m = Math.floor(s / 60)
  if (m < 60) return isZh ? `${m}分钟前` : `${m}m ago`

  // 1-24 hours
  const h = Math.floor(m / 60)
  if (h < 24) return isZh ? `${h}小时前` : `${h}h ago`

  // > 24 hours: show absolute date/time.
  const pad = (n: number) => String(n).padStart(2, '0')
  const now = new Date()

  // Different year → YYYY-MM-DD HH:mm:ss
  if (d.getFullYear() !== now.getFullYear()) {
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }

  // Same year → MM-DD HH:mm:ss
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export function formatGB(bytes: number | undefined | null): string {
  if (bytes == null) return 'N/A'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}
