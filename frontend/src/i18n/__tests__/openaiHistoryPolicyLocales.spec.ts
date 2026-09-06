import { describe, expect, it } from 'vitest'
import en from '../locales/en'
import zh from '../locales/zh'

describe('OAuth history policy locale contract', () => {
  it('exposes the same nonempty keys in English and Chinese', () => {
    for (const key of [
      'rejectExternalHistory',
      'rejectExternalHistoryDesc',
      'rejectExternalHistoryInherited',
      'rejectExternalHistoryParentOnly'
    ] as const) {
      expect(en.admin.accounts.openai[key].trim()).not.toBe('')
      expect(zh.admin.accounts.openai[key].trim()).not.toBe('')
    }
  })
})
