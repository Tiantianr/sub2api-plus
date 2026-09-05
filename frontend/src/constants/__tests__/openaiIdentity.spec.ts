import { describe, expect, it } from 'vitest'
import {
  DEFAULT_PI_USER_AGENT,
  DEFAULT_CODEX_TUI_USER_AGENT,
  isPiUserAgent,
  isCodexTuiUserAgent
} from '@/constants/openaiIdentity'

describe('openaiIdentity constants and helpers', () => {
  it('defines the expected default user agent strings without version for Pi', () => {
    expect(DEFAULT_PI_USER_AGENT).toBe('pi (darwin 24.1.0; arm64)')
    expect(DEFAULT_CODEX_TUI_USER_AGENT).toBe('codex-tui/0.144.0 (Mac OS X 14.0; arm64) iTerm')
  })

  it('correctly identifies Pi user agents including legacy formatted ones', () => {
    expect(isPiUserAgent(DEFAULT_PI_USER_AGENT)).toBe(true)
    expect(isPiUserAgent('pi/0.85.0 (darwin 24.1.0; arm64)')).toBe(true)
    expect(isPiUserAgent('  pi (linux 6.1.0; x86_64)  ')).toBe(true)
    expect(isPiUserAgent('pi')).toBe(true)
    expect(isPiUserAgent('')).toBe(false)
    expect(isPiUserAgent(null)).toBe(false)
    expect(isPiUserAgent(undefined)).toBe(false)
    expect(isPiUserAgent('pipeline/1.0')).toBe(false)
    expect(isPiUserAgent('pip/1.0')).toBe(false)
    expect(isPiUserAgent('pie')).toBe(false)
    expect(isPiUserAgent('codex-tui/0.144.0 (Mac OS X 14.0; arm64) iTerm')).toBe(false)
  })

  it('correctly identifies Codex TUI user agents', () => {
    expect(isCodexTuiUserAgent(DEFAULT_CODEX_TUI_USER_AGENT)).toBe(true)
    expect(isCodexTuiUserAgent('  codex-tui/0.144.0  ')).toBe(true)
    expect(isCodexTuiUserAgent('codex-tui')).toBe(true)
    expect(isCodexTuiUserAgent('')).toBe(false)
    expect(isCodexTuiUserAgent(null)).toBe(false)
    expect(isCodexTuiUserAgent(undefined)).toBe(false)
    expect(isCodexTuiUserAgent('codex-tui-custom/1.0')).toBe(false)
    expect(isCodexTuiUserAgent(DEFAULT_PI_USER_AGENT)).toBe(false)
  })
})
