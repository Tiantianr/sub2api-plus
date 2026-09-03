import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get },
}))

import { getLogInput } from '@/api/admin/riskControl'

describe('admin Content Moderation input detail API', () => {
  beforeEach(() => {
    get.mockReset()
    get.mockResolvedValue({ data: { id: 42, content: 'complete audited content', complete: true } })
  })

  it('loads one record through the administrator detail endpoint', async () => {
    const result = await getLogInput(42)

    expect(get).toHaveBeenCalledWith('/admin/risk-control/logs/42/input')
    expect(result).toEqual({ id: 42, content: 'complete audited content', complete: true })
  })
})
