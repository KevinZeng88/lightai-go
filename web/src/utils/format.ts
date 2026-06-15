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

export function formatRelativeTime(iso: string | undefined | null): string {
  if (!iso) return 'N/A'
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return s + 's ago'
  const m = Math.floor(s / 60)
  if (m < 60) return m + 'm ago'
  const h = Math.floor(m / 60)
  if (h < 24) return h + 'h ago'
  const d = Math.floor(h / 24)
  return d + 'd ago'
}

export function formatGB(bytes: number | undefined | null): string {
  if (bytes == null) return 'N/A'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}
