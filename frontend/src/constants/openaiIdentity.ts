/**
 * OpenAI wire identity presets.
 *
 * The Pi Agent User-Agent follows the reference implementation in
 * @earendil-works/pi-ai/dist/utils/pi-user-agent.js:
 * `nodeOs ? \`pi (${nodeOs.platform()} ${nodeOs.release()}; ${nodeOs.arch()})\` : "pi (browser)"`
 * It deliberately does NOT include a package version.
 */
export const DEFAULT_PI_USER_AGENT = 'pi (darwin 24.1.0; arm64)'
export const DEFAULT_CODEX_TUI_USER_AGENT = 'codex-tui/0.144.0 (Mac OS X 14.0; arm64) iTerm'

export function isPiUserAgent(ua?: string | null): boolean {
  if (!ua) return false
  const trimmed = ua.trim()
  return (
    trimmed === 'pi' ||
    trimmed.startsWith('pi ') ||
    trimmed.startsWith('pi/') ||
    trimmed.startsWith('pi(')
  )
}

export function isCodexTuiUserAgent(ua?: string | null): boolean {
  if (!ua) return false
  const trimmed = ua.trim()
  return (
    trimmed === 'codex-tui' ||
    trimmed.startsWith('codex-tui/') ||
    trimmed.startsWith('codex-tui ') ||
    trimmed.startsWith('codex-tui(')
  )
}
