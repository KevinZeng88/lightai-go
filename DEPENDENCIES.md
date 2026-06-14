# LightAI Go — Dependency Inventory (RC1)

## Go Modules (from go.mod)

| Module | Version | License |
|--------|---------|---------|
| github.com/google/uuid | v1.6.0 | BSD-3 |
| github.com/mattn/go-sqlite3 | v1.14.45 | MIT |
| github.com/prometheus/client_golang | v1.23.2 | Apache-2.0 |
| github.com/prometheus/common | v0.66.1 | Apache-2.0 |
| github.com/shirou/gopsutil/v4 | v4.26.5 | BSD-3 |
| golang.org/x/crypto | v0.53.0 | BSD-3 |
| golang.org/x/time | v0.15.0 | BSD-3 |
| gopkg.in/yaml.v3 | v3.0.1 | MIT |

## npm Packages (from web/package.json)

| Package | Purpose |
|---------|---------|
| vue 3 | Frontend framework (MIT) |
| vite | Build tool (MIT) |
| element-plus | UI component library (MIT) |
| pinia | State management (MIT) |
| vue-router 4 | Routing (MIT) |
| vue-i18n | Internationalization (MIT) |
| typescript | Type checking (Apache-2.0) |

Full list: `web/package-lock.json`

## External Binaries (bundled)

| Component | Version | License | Source |
|-----------|---------|---------|--------|
| Prometheus | 3.12.0 | Apache-2.0 | github.com/prometheus/prometheus |
| Grafana | 13.0.2 | AGPLv3 | github.com/grafana/grafana |

## Build Image Toolchain

| Component | Version |
|-----------|---------|
| OS | Rocky Linux 8 |
| glibc | 2.28 |
| Go | 1.26.4 |
| Node.js | 26.1.0 |
| npm | 11.13.0 |

## SBOM Note

Full SBOM generation (SPDX/CycloneDX) is deferred to CI pipeline (P2-009).
This document serves as a manual dependency inventory for RC1.
