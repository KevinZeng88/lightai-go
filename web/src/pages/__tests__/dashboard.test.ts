/**
 * Dashboard aggregation unit tests.
 * Run: npx vitest run src/pages/__tests__/dashboard.test.ts
 */
import { describe, it, expect } from 'vitest'
import type { GPU } from '@/api/gpus'
import type { Node } from '@/api/nodes'

// ---- Pure aggregation functions (extracted for testability) ----

function sumTotalMemory(gpus: GPU[]): number {
  return gpus.reduce((s, x) => s + (x.memory_total_bytes || 0), 0)
}

function sumUsedMemory(gpus: GPU[]): number {
  return gpus.reduce((s, x) => s + (x.memory_used_bytes || 0), 0)
}

function sumFreeMemory(gpus: GPU[]): number {
  return gpus.reduce((s, x) => s + (x.memory_free_bytes || 0), 0)
}

function avgGpuUtilization(gpus: GPU[]): number {
  if (gpus.length === 0) return 0
  return gpus.reduce((s, x) => s + (x.gpu_utilization_percent || 0), 0) / gpus.length
}

function maxGpuUtilization(gpus: GPU[]): number {
  if (gpus.length === 0) return 0
  return Math.max(...gpus.map(x => x.gpu_utilization_percent || 0))
}

function maxMemoryUtilizationPercent(gpus: GPU[]): number {
  if (gpus.length === 0) return 0
  return Math.max(...gpus.map(x =>
    x.memory_total_bytes > 0 ? (x.memory_used_bytes / x.memory_total_bytes) * 100 : 0
  ))
}

function maxTemperature(gpus: GPU[]): number {
  if (gpus.length === 0) return 0
  return Math.max(...gpus.map(x => x.temperature_celsius ?? 0))
}

function topGpuUtilization(gpus: GPU[], n: number = 5): GPU[] {
  return [...gpus]
    .filter(g => g.gpu_utilization_percent != null)
    .sort((a, b) => (b.gpu_utilization_percent ?? 0) - (a.gpu_utilization_percent ?? 0))
    .slice(0, n)
}

function topMemoryUsage(gpus: GPU[], n: number = 5): GPU[] {
  return [...gpus]
    .filter(g => g.memory_total_bytes > 0)
    .sort((a, b) => {
      const pa = a.memory_total_bytes > 0 ? a.memory_used_bytes / a.memory_total_bytes : 0
      const pb = b.memory_total_bytes > 0 ? b.memory_used_bytes / b.memory_total_bytes : 0
      return pb - pa
    })
    .slice(0, n)
}

function abnormalGpus(gpus: GPU[]): GPU[] {
  return gpus.filter(g => g.health !== 'healthy' || g.status === 'unavailable')
}

function healthyGpuCount(gpus: GPU[]): number {
  return gpus.filter(g => g.health === 'healthy').length
}

function onlineNodeCount(nodes: Node[]): number {
  return nodes.filter(n => n.status === 'online').length
}

// ---- Helpers ----

function mkGpu(overrides: Partial<GPU> = {}): GPU {
  return {
    id: 'gpu-1',
    node_id: 'node-1',
    vendor: 'nvidia',
    index: 0,
    name: 'Test GPU',
    uuid: 'GPU-uuid',
    pci_bus_id: '0000:01:00.0',
    driver_version: '550.0',
    memory_total_bytes: 25651314688, // ~24 GiB
    memory_used_bytes: 0,
    memory_free_bytes: 25651314688,
    health: 'healthy',
    status: 'available',
    created_at: '2026-01-01',
    updated_at: '2026-01-01',
    ...overrides,
  }
}

function mkNode(overrides: Partial<Node> = {}): Node {
  return {
    id: 'node-1',
    agent_id: 'agent-1',
    hostname: 'host1',
    primary_ip: '10.0.0.1',
    advertised_address: '10.0.0.1',
    os: 'linux',
    arch: 'amd64',
    kernel: '6.6.0',
    agent_version: 'v0.1.9',
    metrics_enabled: true,
    metrics_scheme: 'http',
    metrics_port: 9090,
    metrics_path: '/metrics',
    status: 'online',
    tenant_id: 'tenant-1',
    created_at: '2026-01-01',
    updated_at: '2026-01-01',
    ...overrides,
  }
}

// ---- Tests ----

describe('Dashboard GPU memory aggregation', () => {
  it('sums 8 GPUs of 64 GiB each to ~512 GiB', () => {
    const GiB64 = 64 * 1024 * 1024 * 1024 // 68719476736
    const gpus: GPU[] = Array.from({ length: 8 }, (_, i) =>
      mkGpu({ id: `gpu-${i}`, memory_total_bytes: GiB64, memory_used_bytes: GiB64 / 2, memory_free_bytes: GiB64 / 2 })
    )
    expect(sumTotalMemory(gpus)).toBe(8 * GiB64) // 549755813888
    expect(sumUsedMemory(gpus)).toBe(8 * GiB64 / 2)
    expect(sumFreeMemory(gpus)).toBe(8 * GiB64 / 2)
  })

  it('handles zero GPUs', () => {
    expect(sumTotalMemory([])).toBe(0)
    expect(sumUsedMemory([])).toBe(0)
  })

  it('handles GPUs with missing memory_total_bytes', () => {
    const gpus = [mkGpu({ memory_total_bytes: 0, memory_used_bytes: 100 })]
    expect(sumTotalMemory(gpus)).toBe(0)
  })

  it('sums mixed-size GPUs correctly', () => {
    const GiB24 = 24 * 1024 * 1024 * 1024
    const GiB48 = 48 * 1024 * 1024 * 1024
    const gpus = [
      mkGpu({ id: 'a', memory_total_bytes: GiB24, memory_used_bytes: GiB24 / 2, memory_free_bytes: GiB24 / 2 }),
      mkGpu({ id: 'b', memory_total_bytes: GiB48, memory_used_bytes: GiB48 / 2, memory_free_bytes: GiB48 / 2 }),
    ]
    expect(sumTotalMemory(gpus)).toBe(GiB24 + GiB48)
    expect(sumUsedMemory(gpus)).toBe(GiB24 / 2 + GiB48 / 2)
  })
})

describe('Dashboard GPU utilization aggregation', () => {
  it('computes average utilization correctly', () => {
    const gpus = [
      mkGpu({ id: 'a', gpu_utilization_percent: 50 }),
      mkGpu({ id: 'b', gpu_utilization_percent: 100 }),
      mkGpu({ id: 'c', gpu_utilization_percent: 0 }),
    ]
    expect(avgGpuUtilization(gpus)).toBeCloseTo(50, 0)
  })

  it('max utilization is correct', () => {
    const gpus = [
      mkGpu({ id: 'a', gpu_utilization_percent: 30 }),
      mkGpu({ id: 'b', gpu_utilization_percent: 85 }),
    ]
    expect(maxGpuUtilization(gpus)).toBe(85)
  })

  it('max memory utilization percent is correct', () => {
    const GiB = 1024 * 1024 * 1024
    const gpus = [
      mkGpu({ id: 'a', memory_total_bytes: GiB, memory_used_bytes: GiB * 0.3 }),
      mkGpu({ id: 'b', memory_total_bytes: GiB, memory_used_bytes: GiB * 0.9 }),
    ]
    expect(maxMemoryUtilizationPercent(gpus)).toBeCloseTo(90, 0)
  })

  it('max temperature is correct', () => {
    const gpus = [
      mkGpu({ id: 'a', temperature_celsius: 45 }),
      mkGpu({ id: 'b', temperature_celsius: 72 }),
      mkGpu({ id: 'c', temperature_celsius: undefined }),
    ]
    expect(maxTemperature(gpus)).toBe(72)
  })
})

describe('Dashboard GPU Top-N sorting', () => {
  it('topUtilization returns top 5 sorted desc', () => {
    const gpus = Array.from({ length: 10 }, (_, i) =>
      mkGpu({ id: `gpu-${i}`, gpu_utilization_percent: i * 10 })
    )
    const top = topGpuUtilization(gpus, 5)
    expect(top.length).toBe(5)
    expect(top[0].gpu_utilization_percent).toBe(90)
    expect(top[4].gpu_utilization_percent).toBe(50)
  })

  it('topMemoryUsage returns top 5 sorted desc by ratio', () => {
    const GiB = 1024 * 1024 * 1024
    const gpus = Array.from({ length: 10 }, (_, i) =>
      mkGpu({
        id: `gpu-${i}`,
        memory_total_bytes: GiB * 10,
        memory_used_bytes: GiB * i,
      })
    )
    const top = topMemoryUsage(gpus, 5)
    expect(top.length).toBe(5)
    // GPU with index 9 has 90% used, should be first
    expect(top[0].memory_used_bytes / top[0].memory_total_bytes).toBeCloseTo(0.9, 1)
  })

  it('topUtilization ignores null utilization', () => {
    const gpus = [
      mkGpu({ id: 'a', gpu_utilization_percent: 50 }),
      mkGpu({ id: 'b', gpu_utilization_percent: undefined }),
      mkGpu({ id: 'c', gpu_utilization_percent: 80 }),
    ]
    const top = topGpuUtilization(gpus, 5)
    expect(top.length).toBe(2)
    expect(top[0].id).toBe('c')
  })
})

describe('Dashboard abnormal GPU filtering', () => {
  it('returns unhealthy or unavailable GPUs', () => {
    const gpus = [
      mkGpu({ id: 'a', health: 'healthy', status: 'available' }),
      mkGpu({ id: 'b', health: 'unhealthy', status: 'available' }),
      mkGpu({ id: 'c', health: 'healthy', status: 'unavailable' }),
      mkGpu({ id: 'd', health: 'degraded', status: 'available' }),
    ]
    const abnormal = abnormalGpus(gpus)
    expect(abnormal.length).toBe(3) // b, c, d
    expect(abnormal.map(g => g.id)).toEqual(['b', 'c', 'd'])
  })

  it('returns empty when all GPUs are healthy and available', () => {
    const gpus = [
      mkGpu({ id: 'a', health: 'healthy', status: 'available' }),
      mkGpu({ id: 'b', health: 'healthy', status: 'available' }),
    ]
    expect(abnormalGpus(gpus).length).toBe(0)
  })
})

describe('Dashboard node aggregation', () => {
  it('counts online nodes correctly', () => {
    const nodes = [
      mkNode({ id: 'a', status: 'online' }),
      mkNode({ id: 'b', status: 'offline' }),
      mkNode({ id: 'c', status: 'online' }),
    ]
    expect(onlineNodeCount(nodes)).toBe(2)
  })

  it('healthyGpuCount counts only healthy', () => {
    const gpus = [
      mkGpu({ id: 'a', health: 'healthy' }),
      mkGpu({ id: 'b', health: 'warning' }),
      mkGpu({ id: 'c', health: 'healthy' }),
    ]
    expect(healthyGpuCount(gpus)).toBe(2)
  })
})
