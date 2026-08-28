import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

const here = dirname(fileURLToPath(import.meta.url))
const read = (path: string) => readFileSync(resolve(here, path), 'utf8')
const localeKeys = (value: unknown, prefix = ''): string[] => {
  if (!value || typeof value !== 'object') return [prefix]
  return Object.entries(value).flatMap(([key, child]) => localeKeys(child, prefix ? `${prefix}.${key}` : key)).sort()
}

describe('OpenAI OAuth access integration surface', () => {
  it('registers an administrator route hidden from simple mode', () => {
    const router = read('../../../router/index.ts')
    const start = router.indexOf("path: '/admin/openai-oauth-access'")
    const route = router.slice(start, router.indexOf("path: '/admin/groups'", start))
    expect(route).toContain('requiresAuth: true')
    expect(route).toContain('requiresAdmin: true')
    expect(router.slice(router.indexOf('const restrictedPaths'))).toContain("'/admin/openai-oauth-access'")

    const sidebar = read('../../../components/layout/AppSidebar.vue')
    expect(sidebar).toContain("path: '/admin/openai-oauth-access'")
    expect(sidebar).toContain("t('nav.oauthAccess')")
    expect(sidebar).toContain('hideInSimpleMode: true')
  })

  it('keeps locales symmetric and the matrix horizontally scrollable and labeled', () => {
    expect(localeKeys(zh.admin.oauthAccess)).toEqual(localeKeys(en.admin.oauthAccess))
    expect(zh.nav.oauthAccess).toBeTruthy()
    expect(en.nav.oauthAccess).toBeTruthy()
    const view = read('../OpenAIOAuthAccessView.vue')
    expect(view).toContain('overflow-x-auto')
    expect(view).toContain('sticky left-0')
    expect(view).toContain('sm:sticky sm:left-12')
    expect(view).toContain(':aria-label')
    expect(view).toContain('role="status"')
  })
})
