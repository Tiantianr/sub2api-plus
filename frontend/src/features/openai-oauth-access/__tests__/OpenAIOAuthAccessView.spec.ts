import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpenAIOAuthAccessView from '../OpenAIOAuthAccessView.vue'

const mocks = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listUsers: vi.fn(),
  listGroups: vi.fn(),
  preview: vi.fn(),
  apply: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  stepUpRun: vi.fn(),
}))

vi.mock('../api', () => ({ default: mocks }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError }) }))
vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ visible: { value: false }, blockedReason: { value: '' }, run: mocks.stepUpRun }),
  isStepUpBlocked: () => false,
  isStepUpCancelled: () => false,
  stepUpBlockReason: () => '',
}))
vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return { ...actual, onBeforeRouteLeave: vi.fn() }
})
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => key.replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`)),
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const BaseDialogStub = defineComponent({
  props: ['show', 'title'],
  emits: ['close'],
  template: '<div v-if="show" data-test="preview-dialog"><slot /><slot name="footer" /></div>',
})
const PaginationStub = defineComponent({
  props: ['total', 'page', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<div data-test="pagination" />',
})

function mountView() {
  return mount(OpenAIOAuthAccessView, {
    global: {
      stubs: { AppLayout: AppLayoutStub, BaseDialog: BaseDialogStub, Pagination: PaginationStub, TotpStepUpDialog: true },
    },
  })
}

describe('OpenAIOAuthAccessView', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
    mocks.listAccounts.mockResolvedValue([{
      id: 1,
      name: 'OAuth A',
      status: 'active',
      group_ids: [10],
      mode: 'public',
      default_for_new_users: false,
      revision: 0,
      granted_user_ids: [],
    }])
    mocks.listUsers.mockResolvedValue({
      items: [{
        id: 101,
        email: 'user@example.com',
        status: 'active',
        api_key_group_ids: [10],
        subscription_group_ids: [],
        granted_account_ids: [],
        effective_account_ids: [1],
      }],
      total: 1,
      page: 1,
      limit: 50,
      pages: 1,
    })
    mocks.listGroups.mockResolvedValue([{ id: 10, name: 'Public' }])
    mocks.stepUpRun.mockImplementation((action: () => Promise<unknown>) => action())
    mocks.preview.mockResolvedValue({
      accounts: [{ account_id: 1, account_name: 'OAuth A', old_mode: 'public', new_mode: 'restricted', old_default_for_new_users: false, new_default_for_new_users: false, granted_user_count: 1, grant_added_count: 1, grant_removed_count: 0 }],
      grant_added_count: 1,
      grant_removed_count: 0,
      users_losing_all_access_count: 0,
      users_losing_all_access: [],
    })
    mocks.apply.mockResolvedValue({
      accounts: [{ id: 1, name: 'OAuth A', status: 'active', group_ids: [10], mode: 'restricted', default_for_new_users: false, revision: 1, granted_user_ids: [] }],
      account_count: 1,
      grant_added_count: 0,
      grant_removed_count: 0,
    })
  })

  it('keeps edits local and previews the exact revision-bound policy', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="mode-restricted-1"]').trigger('click')
    await wrapper.get('[data-test="grant-101-1"]').trigger('change')
    await wrapper.get('[data-test="preview-save"]').trigger('click')
    await flushPromises()

    expect(mocks.apply).not.toHaveBeenCalled()
    expect(mocks.preview).toHaveBeenCalledWith([{
      account_id: 1,
      expected_revision: 0,
      mode: 'restricted',
      default_for_new_users: false,
      granted_user_ids: [101],
    }])
    expect(wrapper.find('[data-test="preview-dialog"]').exists()).toBe(true)
  })

  it('preserves the draft when atomic save reports a revision conflict', async () => {
    mocks.apply.mockRejectedValue({ status: 409, reason: 'OPENAI_OAUTH_ACCESS_REVISION_CONFLICT' })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="mode-restricted-1"]').trigger('click')
    await wrapper.get('[data-test="preview-save"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="apply-save"]').trigger('click')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('admin.oauthAccess.messages.conflict')
    expect(wrapper.get('[data-test="preview-save"]').attributes()).not.toHaveProperty('disabled')
    expect(wrapper.text()).toContain('admin.oauthAccess.saveBar.dirty')
  })

  it('applies the immutable preview payload through step-up retry', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="mode-restricted-1"]').trigger('click')
    await wrapper.get('[data-test="preview-save"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-test="mode-public-1"]').trigger('click')
    await wrapper.get('[data-test="apply-save"]').trigger('click')
    await flushPromises()

    expect(mocks.stepUpRun).toHaveBeenCalledOnce()
    expect(mocks.apply).toHaveBeenCalledWith([{
      account_id: 1,
      expected_revision: 0,
      mode: 'restricted',
      default_for_new_users: false,
      granted_user_ids: [],
    }])
  })
})
