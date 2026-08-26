import { describe, expect, it } from 'vitest'
import {
  platformAccentBarClass,
  platformAccentColor,
  platformBadgeClass,
  platformBadgeLightClass,
  platformBorderClass,
  platformBorderStrongClass,
  platformButtonClass,
  platformDiscountClass,
  platformGradientClass,
  platformGradientSubtextClass,
  platformGradientTextClass,
  platformIconClass,
  platformTextClass,
} from '@/utils/platformColors'

describe('platformColors', () => {
  it('uses the site primary palette for OpenAI accents', () => {
    const classes = [
      platformAccentBarClass('openai'),
      platformBadgeClass('openai'),
      platformBadgeLightClass('openai'),
      platformBorderClass('openai'),
      platformBorderStrongClass('openai'),
      platformButtonClass('openai'),
      platformDiscountClass('openai'),
      platformGradientClass('openai'),
      platformGradientSubtextClass('openai'),
      platformGradientTextClass('openai'),
      platformIconClass('openai'),
      platformTextClass('openai'),
    ].join(' ')

    expect(classes).toContain('primary-')
    expect(classes).not.toMatch(/green|emerald|teal/)
    expect(platformAccentColor('openai')).toBe('#3c80e6')
  })

  it('uses the site primary color for unknown platform accents', () => {
    expect(platformAccentColor('unknown')).toBe('#3c80e6')
  })
})
