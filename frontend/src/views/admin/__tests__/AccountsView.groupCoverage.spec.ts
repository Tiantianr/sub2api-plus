import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token', isSimpleMode: false })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params?.name ? `${key}:${params.name}` : key
    })
  }
})

const EditAccountModalStub = defineComponent({
  name: 'EditAccountModal',
  emits: ['updated'],
  template: '<div data-test="edit-account-modal" />'
})

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: { props: ['data'], template: '<div data-test="data-table" />' },
        AccountTableActions: { template: '<div><slot name="after" /></div>' },
        AccountTableFilters: true,
        AccountBulkActionsBar: true,
        Pagination: true,
        ConfirmDialog: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: true,
        EditAccountModal: EditAccountModalStub,
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: true,
        HelpTooltip: true,
        Icon: true,
        Teleport: true
      }
    }
  })
}

describe('admin AccountsView group account coverage', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockReset().mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    listWithEtag.mockReset().mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockReset().mockResolvedValue({ stats: {} })
    getAllProxies.mockReset().mockResolvedValue([])
    getAllGroups.mockReset().mockResolvedValue([])
  })

  it('lists active groups with no schedulable accounts and distinguishes the reason', async () => {
    getAllGroups.mockResolvedValue([
      { id: 7, name: 'unbound', account_count: 0, active_account_count: 0 },
      { id: 8, name: 'disabled-only', account_count: 2, active_account_count: 0 },
      { id: 9, name: 'healthy', account_count: 2, active_account_count: 1 }
    ])

    const wrapper = mountView()
    await flushPromises()

    const warning = wrapper.get('[data-test="group-coverage-warning"]')
    const items = warning.findAll('[data-test="group-coverage-item"]')
    expect(items).toHaveLength(2)
    expect(warning.text()).toContain('admin.accounts.groupCoverageWarning')
    expect(warning.text()).toContain('unbound')
    expect(warning.text()).toContain('admin.accounts.groupCoverageUnbound')
    expect(warning.text()).toContain('disabled-only')
    expect(warning.text()).toContain('admin.accounts.groupCoverageUnavailable')
    expect(warning.text()).not.toContain('healthy')
  })

  it('does not render a warning when every active group has a schedulable account', async () => {
    getAllGroups.mockResolvedValue([
      { id: 9, name: 'healthy', account_count: 2, active_account_count: 1 }
    ])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="group-coverage-warning"]').exists()).toBe(false)
  })

  it('filters the account table when an uncovered group is selected', async () => {
    getAllGroups.mockResolvedValue([
      { id: 7, name: 'unbound', account_count: 0, active_account_count: 0 }
    ])

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="group-coverage-item"]').trigger('click')
    await flushPromises()

    expect(listAccounts).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({ group: '7' }),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })

  it('refreshes group availability after an account update', async () => {
    getAllGroups
      .mockResolvedValueOnce([{ id: 7, name: 'production', account_count: 1, active_account_count: 1 }])
      .mockResolvedValueOnce([{ id: 7, name: 'production', account_count: 1, active_account_count: 0 }])

    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-test="group-coverage-warning"]').exists()).toBe(false)

    wrapper.getComponent(EditAccountModalStub).vm.$emit('updated', {
      id: 42,
      name: 'closed account',
      status: 'inactive',
      schedulable: true,
      platform: 'openai',
      type: 'oauth'
    })
    await flushPromises()

    expect(getAllGroups).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-test="group-coverage-warning"]').text()).toContain('production')
  })
})
