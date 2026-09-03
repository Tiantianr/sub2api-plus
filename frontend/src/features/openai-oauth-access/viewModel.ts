import type { OAuthAccessAccount, OAuthAccessMode, OAuthAccessPolicyChange } from './api'

export function cloneOAuthAccessAccounts(accounts: OAuthAccessAccount[]): OAuthAccessAccount[] {
  return (accounts ?? []).map((account) => ({
    ...account,
    group_ids: [...(account.group_ids ?? [])],
    granted_user_ids: [...(account.granted_user_ids ?? [])].sort((a, b) => a - b),
  }))
}

export function buildOAuthAccessChanges(
  serverAccounts: OAuthAccessAccount[],
  draftAccounts: OAuthAccessAccount[],
): OAuthAccessPolicyChange[] {
  const serverByID = new Map(serverAccounts.map((account) => [account.id, account]))
  return draftAccounts.flatMap((draft) => {
    const server = serverByID.get(draft.id)
    if (!server || accountFingerprint(server) === accountFingerprint(draft)) return []
    return [{
      account_id: draft.id,
      expected_revision: server.revision,
      mode: draft.mode,
      default_for_new_users: draft.mode === 'restricted' && draft.default_for_new_users,
      granted_user_ids: draft.mode === 'restricted' ? uniqueSortedIDs(draft.granted_user_ids) : [],
    }]
  })
}

export function setOAuthAccessMode(account: OAuthAccessAccount, mode: OAuthAccessMode): void {
  account.mode = mode
  if (mode === 'public') {
    account.default_for_new_users = false
    account.granted_user_ids = []
  }
}

export function setOAuthAccessGrant(account: OAuthAccessAccount, userID: number, granted: boolean): void {
  if (account.mode !== 'restricted' || userID <= 0) return
  const ids = new Set(account.granted_user_ids ?? [])
  if (granted) ids.add(userID)
  else ids.delete(userID)
  account.granted_user_ids = [...ids].sort((a, b) => a - b)
}

export function hasOAuthAccessGrant(account: OAuthAccessAccount, userID: number): boolean {
  return account.mode === 'restricted' && (account.granted_user_ids ?? []).includes(userID)
}

function accountFingerprint(account: OAuthAccessAccount): string {
  return JSON.stringify([
    account.mode,
    account.mode === 'restricted' && account.default_for_new_users,
    account.mode === 'restricted' ? uniqueSortedIDs(account.granted_user_ids ?? []) : [],
  ])
}

function uniqueSortedIDs(ids: number[]): number[] {
  return [...new Set(ids.filter((id) => id > 0))].sort((a, b) => a - b)
}
