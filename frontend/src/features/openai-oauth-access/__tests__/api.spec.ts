import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), post: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: client }))

import openAIOAuthAccessAPI from '../api'

describe('OpenAI OAuth access API', () => {
  beforeEach(() => Object.values(client).forEach((mock) => mock.mockReset()))

  it('uses the independent administrator namespace', async () => {
    client.get.mockResolvedValue({ data: [] })
    await openAIOAuthAccessAPI.listAccounts()
    expect(client.get).toHaveBeenCalledWith('/admin/openai-oauth-access/accounts')

    client.get.mockResolvedValue({ data: { items: [], total: 0 } })
    await openAIOAuthAccessAPI.listUsers({ page: 2, limit: 50, access: 'none' })
    expect(client.get).toHaveBeenCalledWith('/admin/openai-oauth-access/users', {
      params: { page: 2, limit: 50, access: 'none' },
    })
  })

  it('previews before applying the same revision-bound payload', async () => {
    const changes = [{
      account_id: 7,
      expected_revision: 3,
      mode: 'restricted' as const,
      default_for_new_users: true,
      granted_user_ids: [10, 11],
    }]
    client.post.mockResolvedValue({ data: { accounts: [] } })
    client.put.mockResolvedValue({ data: { accounts: [] } })
    await openAIOAuthAccessAPI.preview(changes)
    await openAIOAuthAccessAPI.apply(changes)
    expect(client.post).toHaveBeenCalledWith('/admin/openai-oauth-access/preview', { changes })
    expect(client.put).toHaveBeenCalledWith('/admin/openai-oauth-access/policies', { changes })
  })
})
