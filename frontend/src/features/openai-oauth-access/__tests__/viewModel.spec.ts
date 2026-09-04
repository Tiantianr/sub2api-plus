import { describe, expect, it } from 'vitest'
import type { OAuthAccessAccount } from '../api'
import {
  buildOAuthAccessChanges,
  cloneOAuthAccessAccounts,
  hasOAuthAccessGrant,
  setOAuthAccessGrant,
  setOAuthAccessMode,
} from '../viewModel'

const account = (): OAuthAccessAccount => ({
  id: 7,
  name: 'OAuth A',
  type: 'oauth',
  status: 'active',
  group_ids: [2],
  mode: 'restricted',
  default_for_new_users: false,
  revision: 4,
  granted_user_ids: [10],
})

describe('OpenAI OAuth access view model', () => {
  it('tracks only changed account policies and keeps the server revision', () => {
    const server = [account()]
    const draft = cloneOAuthAccessAccounts(server)
    expect(buildOAuthAccessChanges(server, draft)).toEqual([])
    setOAuthAccessGrant(draft[0], 11, true)
    expect(buildOAuthAccessChanges(server, draft)).toEqual([{
      account_id: 7,
      expected_revision: 4,
      mode: 'restricted',
      default_for_new_users: false,
      granted_user_ids: [10, 11],
    }])
  })

  it('removes restored cell edits from dirty state', () => {
    const server = [account()]
    const draft = cloneOAuthAccessAccounts(server)
    setOAuthAccessGrant(draft[0], 10, false)
    expect(buildOAuthAccessChanges(server, draft)).toHaveLength(1)
    setOAuthAccessGrant(draft[0], 10, true)
    expect(buildOAuthAccessChanges(server, draft)).toEqual([])
  })

  it('clears hidden grants and defaults when an account becomes public', () => {
    const draft = account()
    draft.default_for_new_users = true
    setOAuthAccessMode(draft, 'public')
    expect(draft).toMatchObject({ mode: 'public', default_for_new_users: false, granted_user_ids: [] })
    setOAuthAccessGrant(draft, 12, true)
    expect(hasOAuthAccessGrant(draft, 12)).toBe(false)
  })

  it('treats nullable arrays from older API responses as empty', () => {
    const legacy = { ...account(), group_ids: null, granted_user_ids: null } as unknown as OAuthAccessAccount
    const [draft] = cloneOAuthAccessAccounts([legacy])
    expect(draft.group_ids).toEqual([])
    expect(draft.granted_user_ids).toEqual([])
    setOAuthAccessGrant(draft, 12, true)
    expect(draft.granted_user_ids).toEqual([12])
  })
})
