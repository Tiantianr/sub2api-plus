import { apiClient } from '@/api/client'

const basePath = '/admin/openai-oauth-access'

export type OAuthAccessMode = 'public' | 'restricted'
export type OAuthAccessAccountType = 'oauth' | 'apikey'

export interface OAuthAccessAccount {
  id: number
  name: string
  type: OAuthAccessAccountType
  status: string
  group_ids: number[]
  mode: OAuthAccessMode
  default_for_new_users: boolean
  revision: number
  granted_user_ids: number[]
}

export interface OAuthAccessUser {
  id: number
  email: string
  status: string
  api_key_group_ids: number[]
  subscription_group_ids: number[]
  granted_account_ids: number[]
  effective_account_ids: number[]
}

export interface OAuthAccessUserPage {
  items: OAuthAccessUser[]
  total: number
  page: number
  limit: number
  pages: number
}

export interface OAuthAccessGroup {
  id: number
  name: string
}

export interface OAuthAccessPolicyChange {
  account_id: number
  expected_revision: number
  mode: OAuthAccessMode
  default_for_new_users: boolean
  granted_user_ids: number[]
}

export interface OAuthAccessAccountImpact {
  account_id: number
  account_name: string
  old_mode: OAuthAccessMode
  new_mode: OAuthAccessMode
  old_default_for_new_users: boolean
  new_default_for_new_users: boolean
  granted_user_count: number
  grant_added_count: number
  grant_removed_count: number
}

export interface OAuthAccessAffectedUser {
  id: number
  email: string
  api_key_group_ids: number[]
}

export interface OAuthAccessPreview {
  accounts: OAuthAccessAccountImpact[]
  grant_added_count: number
  grant_removed_count: number
  users_losing_all_access_count: number
  users_losing_all_access: OAuthAccessAffectedUser[]
}

export interface OAuthAccessApplyResult {
  accounts: OAuthAccessAccount[]
  account_count: number
  grant_added_count: number
  grant_removed_count: number
}

export interface OAuthAccessUserQuery {
  search?: string
  status?: string
  access?: string
  page: number
  limit: number
}

export async function listOAuthAccessAccounts(): Promise<OAuthAccessAccount[]> {
  const { data } = await apiClient.get<OAuthAccessAccount[]>(`${basePath}/accounts`)
  return arrayValue(data).map(normalizeAccount)
}

export async function listOAuthAccessUsers(query: OAuthAccessUserQuery): Promise<OAuthAccessUserPage> {
  const { data } = await apiClient.get<OAuthAccessUserPage>(`${basePath}/users`, { params: query })
  return {
    ...data,
    items: arrayValue(data?.items).map(normalizeUser),
    total: data?.total ?? 0,
    page: data?.page ?? query.page,
    limit: data?.limit ?? query.limit,
    pages: data?.pages ?? 1,
  }
}

export async function previewOAuthAccessPolicies(changes: OAuthAccessPolicyChange[]): Promise<OAuthAccessPreview> {
  const { data } = await apiClient.post<OAuthAccessPreview>(`${basePath}/preview`, { changes })
  return {
    ...data,
    accounts: arrayValue(data?.accounts),
    grant_added_count: data?.grant_added_count ?? 0,
    grant_removed_count: data?.grant_removed_count ?? 0,
    users_losing_all_access_count: data?.users_losing_all_access_count ?? 0,
    users_losing_all_access: arrayValue(data?.users_losing_all_access).map((user) => ({
      ...user,
      api_key_group_ids: arrayValue(user.api_key_group_ids),
    })),
  }
}

export async function applyOAuthAccessPolicies(changes: OAuthAccessPolicyChange[]): Promise<OAuthAccessApplyResult> {
  const { data } = await apiClient.put<OAuthAccessApplyResult>(`${basePath}/policies`, { changes })
  return {
    ...data,
    accounts: arrayValue(data?.accounts).map(normalizeAccount),
    account_count: data?.account_count ?? 0,
    grant_added_count: data?.grant_added_count ?? 0,
    grant_removed_count: data?.grant_removed_count ?? 0,
  }
}

export async function listOAuthAccessGroups(): Promise<OAuthAccessGroup[]> {
  const { data } = await apiClient.get<OAuthAccessGroup[]>('/admin/groups/all', {
    params: { include_inactive: true },
  })
  return arrayValue(data)
}

function arrayValue<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

function normalizeAccount(account: OAuthAccessAccount): OAuthAccessAccount {
  return {
    ...account,
    type: account.type === 'apikey' ? 'apikey' : 'oauth',
    group_ids: arrayValue(account.group_ids),
    granted_user_ids: arrayValue(account.granted_user_ids),
  }
}

function normalizeUser(user: OAuthAccessUser): OAuthAccessUser {
  return {
    ...user,
    api_key_group_ids: arrayValue(user.api_key_group_ids),
    subscription_group_ids: arrayValue(user.subscription_group_ids),
    granted_account_ids: arrayValue(user.granted_account_ids),
    effective_account_ids: arrayValue(user.effective_account_ids),
  }
}

export const openAIOAuthAccessAPI = {
  listAccounts: listOAuthAccessAccounts,
  listUsers: listOAuthAccessUsers,
  preview: previewOAuthAccessPolicies,
  apply: applyOAuthAccessPolicies,
  listGroups: listOAuthAccessGroups,
}

export default openAIOAuthAccessAPI
