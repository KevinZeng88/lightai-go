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
