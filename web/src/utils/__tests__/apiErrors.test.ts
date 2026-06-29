import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/client'
import { apiErrorMessage } from '../apiErrors'

describe('apiErrorMessage', () => {
  it('uses stable error code i18n instead of raw backend English message', () => {
    const error = new ApiError(409, {
      code: 'display_name_exists',
      error: 'display_name already exists in runtime templates',
      message: 'display_name already exists in runtime templates',
    })
    const t = (key: string) => ({
      'apiErrors.display_name_exists': '显示名称已存在，请换一个名称',
      'common.requestFailed': '请求失败',
    })[key] || key

    expect(apiErrorMessage(error, t)).toBe('显示名称已存在，请换一个名称')
  })
})
