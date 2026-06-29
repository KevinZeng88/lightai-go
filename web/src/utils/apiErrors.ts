import type { ApiError } from '@/api/client'

type Translate = (key: string, params?: Record<string, any>) => string

export function apiErrorMessage(error: unknown, t: Translate, fallbackKey = 'common.requestFailed'): string {
  const apiError = error as Partial<ApiError> & { data?: any, message?: string, status?: number, code?: string }
  const data = apiError?.data || {}
  const code = data.code || apiError.code
  if (typeof code === 'string' && code) {
    const key = `apiErrors.${code}`
    const translated = t(key)
    if (translated !== key) return translated
  }
  if (apiError.status === 401) return t('common.sessionExpired')
  if (apiError.status === 403) return t('common.permissionDenied')
  if (apiError.status === 404) return t('apiErrors.not_found')
  if (apiError.status && apiError.status >= 500) return t('apiErrors.server_error')
  return t(fallbackKey)
}
