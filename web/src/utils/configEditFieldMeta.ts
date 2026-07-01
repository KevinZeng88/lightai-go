import type { ConfigEditField } from './configEditView'

export type TranslateFn = (key: string, ...args: any[]) => string

export function resolveConfigFieldLabel(field: ConfigEditField, t: TranslateFn): string {
  const translated = firstTranslation(t, [
    field.label_i18n_key,
    field.title_i18n_key,
    `configEdit.labels.${field.key}`,
    field.semantic_key ? `configEdit.labels.${field.semantic_key}` : '',
    field.internal_key ? `configEdit.labels.${field.internal_key}` : '',
  ])
  if (translated) return translated

  const direct = firstNonEmpty(field.label, field.title)
  if (direct && !isGenericConfigLabel(direct)) return direct

  const fallbackKey = field.semantic_key || field.key || field.internal_key || ''
  const fallback = humanizeConfigKey(fallbackKey)
  if (isDevMode() && fallbackKey) return `${fallback} (${fallbackKey})`
  return fallback
}

export function resolveConfigFieldHelp(field: ConfigEditField, t: TranslateFn): string {
  const translated = firstTranslation(t, [
    field.help_i18n_key,
    field.description_i18n_key,
    field.tooltip_i18n_key,
    `configEdit.descriptions.${field.key}`,
    field.semantic_key ? `configEdit.descriptions.${field.semantic_key}` : '',
    field.internal_key ? `configEdit.descriptions.${field.internal_key}` : '',
  ])
  if (translated) return translated

  const direct = firstNonEmpty(field.help, field.description)
  if (direct && !isGenericConfigLabel(direct)) return direct
  return fallbackHelp(field, t)
}

export function resolveConfigFieldTooltip(field: ConfigEditField, t: TranslateFn): string {
  const lines: string[] = []
  const showTechnical = field.view === 'developer'
  const technical = field.cli_flag || field.env_key || field.technical_key || field.internal_key || field.key
  if (showTechnical && technical) lines.push(technical)

  const help = resolveConfigFieldHelp(field, t)
  if (help) lines.push(help)

  if (showTechnical && field.technical_key && field.technical_key !== technical) {
    lines.push(`${t('configEdit.fields.technicalKey')}: ${field.technical_key}`)
  }
  return lines.join('\n')
}

function fallbackHelp(field: ConfigEditField, t: TranslateFn): string {
  const type = field.type || (Array.isArray(field.value) ? 'array' : typeof field.value)
  const section = field.section || 'configuration'
  const key = field.semantic_key || field.key || field.internal_key || 'configuration'
  const translated = t('configEdit.descriptions.fallback', { key: humanizeConfigKey(key), type, section })
  if (translated && translated !== 'configEdit.descriptions.fallback') return translated
  return `${humanizeConfigKey(key)} (${type}, ${section})`
}

export function humanizeConfigKey(key: string): string {
  const cleaned = key
    .replace(/^backend\.arg\./, '')
    .replace(/^model_runtime\./, '')
    .replace(/^launcher\./, '')
    .replace(/^runtime\./, '')
    .replace(/^service\./, '')
    .replace(/^deployment\./, '')
    .replace(/\./g, ' ')
    .replace(/_/g, ' ')
    .trim()
  if (!cleaned) return 'Configuration'
  return cleaned.charAt(0).toUpperCase() + cleaned.slice(1)
}

function firstTranslation(t: TranslateFn, keys: Array<string | undefined>): string {
  for (const key of keys) {
    if (!key) continue
    const translated = t(key)
    if (translated && translated !== key) return translated
  }
  return ''
}

function firstNonEmpty(...values: Array<string | undefined>): string {
  for (const value of values) {
    if (value && value.trim()) return value.trim()
  }
  return ''
}

function isGenericConfigLabel(value: string): boolean {
  const normalized = value.trim().toLowerCase()
  return normalized === '配置项' || normalized === 'config' || normalized === 'configuration'
}

function isDevMode(): boolean {
  return Boolean((import.meta as any).env?.DEV)
}
