<template>
  <AppLayout>
    <div class="mx-auto max-w-[1800px] pb-28" :inert="showPreview ? true : undefined">
      <header class="mb-5 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-2xl font-semibold text-gray-950 dark:text-white">
            {{ t('admin.oauthAccess.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
            {{ t('admin.oauthAccess.accountSummary', { restricted: restrictedCount, defaults: defaultCount }) }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary btn-icon"
          :disabled="loading.accounts || dirty"
          :aria-label="t('common.refresh')"
          :title="t('common.refresh')"
          @click="loadAll"
        >
          <Icon name="refresh" size="md" :class="loading.accounts && 'animate-spin'" />
        </button>
      </header>

      <div v-if="loadError && draftAccounts.length === 0" role="alert" class="mb-5 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
        <p>{{ loadError }}</p>
        <button type="button" class="btn btn-secondary btn-sm mt-3" @click="loadAll">
          {{ t('common.retry') }}
        </button>
      </div>

      <div v-else-if="loadError" role="alert" class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
        <span>{{ loadError }}</span>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="dirty" @click="loadAll">{{ t('common.retry') }}</button>
      </div>

      <section v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-center gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <div class="w-full sm:w-72">
            <SearchInput
              v-model="filters.search"
              :placeholder="t('admin.oauthAccess.filters.search')"
              :aria-label="t('admin.oauthAccess.filters.search')"
              @search="applySearch"
            />
          </div>
          <label class="sr-only" for="oauth-access-status">{{ t('admin.oauthAccess.filters.status') }}</label>
          <select id="oauth-access-status" v-model="filters.status" class="input w-full sm:w-40" @change="applyFilters">
            <option value="">{{ t('admin.oauthAccess.filters.all') }}</option>
            <option value="active">{{ t('admin.oauthAccess.filters.active') }}</option>
            <option value="disabled">{{ t('admin.oauthAccess.filters.disabled') }}</option>
          </select>
          <label class="sr-only" for="oauth-access-filter">{{ t('admin.oauthAccess.filters.access') }}</label>
          <select id="oauth-access-filter" v-model="filters.access" class="input w-full sm:w-52" @change="applyFilters">
            <option value="all">{{ t('admin.oauthAccess.filters.all') }}</option>
            <option value="none">{{ t('admin.oauthAccess.filters.none') }}</option>
            <option value="granted">{{ t('admin.oauthAccess.filters.granted') }}</option>
          </select>

          <div v-if="selectedUserIDs.size > 0 || selectedAccountIDs.size > 0" class="ml-auto flex flex-wrap items-center gap-2">
            <span class="text-xs text-gray-500 dark:text-dark-300">
              {{ t('admin.oauthAccess.batch.users', { count: selectedUserIDs.size }) }}
            </span>
            <span class="text-xs text-gray-500 dark:text-dark-300">
              {{ t('admin.oauthAccess.batch.accounts', { count: selectedAccountIDs.size }) }}
            </span>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="!canApplyBatch" @click="applyBatch(true)">
              <Icon name="check" size="sm" />
              {{ t('admin.oauthAccess.batch.grant') }}
            </button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="!canApplyBatch" @click="applyBatch(false)">
              <Icon name="x" size="sm" />
              {{ t('admin.oauthAccess.batch.revoke') }}
            </button>
          </div>
        </div>

        <div v-if="draftAccounts.length === 0 && !loading.accounts" class="px-4 py-12 text-center text-sm text-gray-500 dark:text-dark-300">
          {{ t('admin.oauthAccess.empty.accounts') }}
        </div>

        <template v-else>
          <div class="overflow-x-auto">
            <table class="min-w-max table-fixed border-collapse text-left text-sm">
              <thead class="bg-gray-50 dark:bg-dark-850">
                <tr class="border-b border-gray-200 dark:border-dark-700">
                  <th class="sticky left-0 z-20 w-12 min-w-12 bg-gray-50 px-3 py-3 text-center dark:bg-dark-850">
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="allPageUsersSelected"
                      :indeterminate="somePageUsersSelected"
                      :aria-checked="somePageUsersSelected ? 'mixed' : allPageUsersSelected"
                      :aria-label="t('admin.oauthAccess.columns.select')"
                      @change="togglePageUsers"
                    />
                  </th>
                  <th class="w-48 min-w-48 bg-gray-50 px-4 py-3 font-semibold text-gray-700 dark:bg-dark-850 dark:text-dark-100 sm:sticky sm:left-12 sm:z-20 sm:w-64 sm:min-w-64">
                    {{ t('admin.oauthAccess.columns.user') }}
                  </th>
                  <th class="w-64 min-w-64 px-4 py-3 font-semibold text-gray-700 dark:text-dark-100">
                    {{ t('admin.oauthAccess.columns.groups') }}
                  </th>
                  <th class="w-56 min-w-56 px-4 py-3 font-semibold text-gray-700 dark:text-dark-100">
                    {{ t('admin.oauthAccess.columns.effective') }}
                  </th>
                  <th
                    v-for="account in draftAccounts"
                    :key="account.id"
                    class="w-56 min-w-56 border-l border-gray-200 px-3 py-3 align-top dark:border-dark-700"
                  >
                    <div class="flex items-start gap-2">
                      <input
                        type="checkbox"
                        class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:opacity-40"
                        :checked="selectedAccountIDs.has(account.id)"
                        :disabled="account.mode !== 'restricted'"
                        :aria-label="account.name"
                        @change="toggleAccountSelection(account.id)"
                      />
                      <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-2">
                          <span class="min-w-0 truncate font-semibold text-gray-900 dark:text-white" :title="account.name">{{ account.name }}</span>
                          <span class="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-200">
                            {{ accountTypeLabel(account.type) }}
                          </span>
                          <span v-if="account.status !== 'active'" class="shrink-0 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-800 dark:bg-amber-950/50 dark:text-amber-200">
                            {{ t('admin.oauthAccess.account.inactive') }}
                          </span>
                        </div>
                        <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400" :title="groupLabels(account.group_ids)">
                          {{ groupLabels(account.group_ids) }}
                        </p>
                      </div>
                    </div>

                    <div class="mt-3 grid grid-cols-2 overflow-hidden rounded-md border border-gray-300 dark:border-dark-600">
                      <button
                        type="button"
                        class="h-8 px-2 text-xs font-medium transition-colors"
                        :class="account.mode === 'public' ? 'bg-primary-600 text-white' : 'bg-white text-gray-600 dark:bg-dark-800 dark:text-dark-200'"
                        :aria-pressed="account.mode === 'public'"
                        :aria-label="`${account.name}: ${t('admin.oauthAccess.account.public')}`"
                        :data-test="`mode-public-${account.id}`"
                        @click="changeMode(account, 'public')"
                      >
                        {{ t('admin.oauthAccess.account.public') }}
                      </button>
                      <button
                        type="button"
                        class="h-8 border-l border-gray-300 px-2 text-xs font-medium transition-colors dark:border-dark-600"
                        :class="account.mode === 'restricted' ? 'bg-primary-600 text-white' : 'bg-white text-gray-600 dark:bg-dark-800 dark:text-dark-200'"
                        :aria-pressed="account.mode === 'restricted'"
                        :aria-label="`${account.name}: ${t('admin.oauthAccess.account.restricted')}`"
                        :data-test="`mode-restricted-${account.id}`"
                        @click="changeMode(account, 'restricted')"
                      >
                        {{ t('admin.oauthAccess.account.restricted') }}
                      </button>
                    </div>

                    <div class="mt-3 space-y-2">
                      <span class="block text-xs text-gray-500 dark:text-dark-300">
                        {{ account.mode === 'restricted' ? t('admin.oauthAccess.account.grants', { count: account.granted_user_ids.length }) : t('admin.oauthAccess.account.public') }}
                      </span>
                      <label class="flex min-h-7 items-center justify-between gap-2 text-xs text-gray-600 dark:text-dark-200">
                        <span>{{ t('admin.oauthAccess.account.defaultNewUsers') }}</span>
                        <Toggle
                          :model-value="account.default_for_new_users"
                          :disabled="account.mode !== 'restricted'"
                          :aria-label="`${account.name}: ${t('admin.oauthAccess.account.defaultNewUsers')}`"
                          :data-test="`default-${account.id}`"
                          @update:model-value="account.default_for_new_users = $event"
                        />
                      </label>
                    </div>
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
                <tr v-if="loading.users">
                  <td :colspan="4 + draftAccounts.length" class="px-4 py-12 text-center text-gray-500 dark:text-dark-300">
                    {{ t('common.loading') }}
                  </td>
                </tr>
                <tr v-else-if="userError">
                  <td :colspan="4 + draftAccounts.length" role="alert" class="px-4 py-10 text-center text-red-600 dark:text-red-300">
                    {{ userError }}
                  </td>
                </tr>
                <tr v-else-if="users.items.length === 0">
                  <td :colspan="4 + draftAccounts.length" class="px-4 py-12 text-center text-gray-500 dark:text-dark-300">
                    {{ t('admin.oauthAccess.empty.users') }}
                  </td>
                </tr>
                <tr v-for="user in users.items" v-else :key="user.id" class="bg-white hover:bg-gray-50 dark:bg-dark-800 dark:hover:bg-dark-750">
                  <td class="sticky left-0 z-10 w-12 min-w-12 bg-inherit px-3 py-3 text-center">
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="selectedUserIDs.has(user.id)"
                      :aria-label="user.email"
                      @change="toggleUserSelection(user.id)"
                    />
                  </td>
                  <td class="w-48 min-w-48 bg-inherit px-4 py-3 sm:sticky sm:left-12 sm:z-10 sm:w-64 sm:min-w-64">
                    <p class="truncate font-medium text-gray-900 dark:text-white" :title="user.email">{{ user.email }}</p>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">#{{ user.id }} · {{ statusLabel(user.status) }}</p>
                  </td>
                  <td class="w-64 min-w-64 px-4 py-3 align-top">
                    <div class="flex flex-wrap gap-1">
                      <span v-for="groupID in user.api_key_group_ids" :key="`key-${groupID}`" class="rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-950/40 dark:text-blue-200">
                        {{ groupName(groupID) }}
                      </span>
                      <span v-for="groupID in user.subscription_group_ids" :key="`sub-${groupID}`" class="rounded bg-emerald-50 px-1.5 py-0.5 text-xs text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200">
                        {{ groupName(groupID) }}
                      </span>
                      <span v-if="user.api_key_group_ids.length + user.subscription_group_ids.length === 0" class="text-xs text-gray-400">{{ t('admin.oauthAccess.empty.groups') }}</span>
                    </div>
                  </td>
                  <td class="w-56 min-w-56 px-4 py-3 align-top">
                    <div class="flex flex-wrap gap-1">
                      <span v-for="name in effectiveAccountNames(user)" :key="name" class="max-w-48 truncate rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-100" :title="name">
                        {{ name }}
                      </span>
                      <span v-if="effectiveAccountNames(user).length === 0" class="text-xs font-medium text-red-600 dark:text-red-300">{{ t('admin.oauthAccess.empty.effective') }}</span>
                    </div>
                  </td>
                  <td v-for="account in draftAccounts" :key="`${user.id}-${account.id}`" class="w-56 min-w-56 border-l border-gray-200 px-3 py-3 text-center dark:border-dark-700">
                    <span v-if="account.mode === 'public'" class="inline-flex h-7 items-center rounded bg-emerald-50 px-2 text-xs font-medium text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200">
                      {{ t('admin.oauthAccess.account.public') }}
                    </span>
                    <input
                      v-else
                      type="checkbox"
                      class="h-5 w-5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="hasGrant(account, user.id)"
                      :aria-label="`${user.email}: ${account.name}`"
                      :data-test="`grant-${user.id}-${account.id}`"
                      @change="toggleGrant(account, user.id)"
                    />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <Pagination
            v-if="!loading.users && users.total > 0"
            :total="users.total"
            :page="users.page"
            :page-size="users.limit"
            :show-jump="true"
            @update:page="changePage"
            @update:page-size="changePageSize"
          />
        </template>
      </section>
    </div>

    <div class="fixed inset-x-0 bottom-0 z-30 border-t border-gray-200 bg-white/95 px-4 py-3 shadow-[0_-10px_28px_rgba(15,23,42,0.08)] backdrop-blur dark:border-dark-700 dark:bg-dark-900/95 lg:left-64" :inert="showPreview ? true : undefined">
      <div class="mx-auto flex max-w-[1800px] flex-wrap items-center justify-between gap-3">
        <span class="text-sm" :class="dirty ? 'text-amber-700 dark:text-amber-300' : 'text-gray-500 dark:text-dark-400'">
          {{ dirty ? t('admin.oauthAccess.saveBar.dirty', { count: changes.length }) : t('admin.oauthAccess.saveBar.synced') }}
        </span>
        <div class="flex items-center gap-3">
          <span v-if="draftLimitError" role="alert" class="text-sm text-red-600 dark:text-red-300">{{ draftLimitError }}</span>
          <button type="button" class="btn btn-secondary" :disabled="!dirty || loading.preview || loading.save" @click="resetDraft">
            {{ t('admin.oauthAccess.saveBar.reset') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="!dirty || Boolean(draftLimitError) || loading.preview || loading.save" data-test="preview-save" @click="openPreview">
            {{ loading.preview ? t('common.loading') : t('admin.oauthAccess.saveBar.preview') }}
          </button>
        </div>
      </div>
    </div>

    <BaseDialog :show="showPreview" :title="t('admin.oauthAccess.preview.title')" width="wide" @close="closePreview">
      <div v-if="preview" class="space-y-4">
        <p class="text-sm text-gray-700 dark:text-dark-100">
          {{ t('admin.oauthAccess.preview.summary', { accounts: preview.accounts.length, added: preview.grant_added_count, removed: preview.grant_removed_count }) }}
        </p>
        <div class="max-h-64 divide-y divide-gray-200 overflow-y-auto rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
          <div v-for="impact in preview.accounts" :key="impact.account_id" class="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5 text-sm">
            <span class="font-medium text-gray-900 dark:text-white">{{ impact.account_name }}</span>
            <span class="text-gray-500 dark:text-dark-300">
              {{ modeLabel(impact.old_mode) }} → {{ modeLabel(impact.new_mode) }} · +{{ impact.grant_added_count }} / -{{ impact.grant_removed_count }}
            </span>
            <span v-if="impact.old_default_for_new_users !== impact.new_default_for_new_users" class="w-full text-xs text-gray-500 dark:text-dark-300">
              {{ t('admin.oauthAccess.preview.defaultChange', {
                old: defaultLabel(impact.old_default_for_new_users),
                next: defaultLabel(impact.new_default_for_new_users),
              }) }}
            </span>
          </div>
        </div>
        <div role="status" class="rounded-lg border px-4 py-3 text-sm" :class="preview.users_losing_all_access_count > 0 ? 'border-red-300 bg-red-50 text-red-800 dark:border-red-900 dark:bg-red-950/30 dark:text-red-200' : 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-200'">
          {{ preview.users_losing_all_access_count > 0 ? t('admin.oauthAccess.preview.losingAccess', { count: preview.users_losing_all_access_count }) : t('admin.oauthAccess.preview.noLoss') }}
        </div>
        <div v-if="preview.users_losing_all_access.length > 0" class="max-h-40 overflow-y-auto text-sm text-gray-700 dark:text-dark-100">
          <p v-for="user in preview.users_losing_all_access" :key="user.id" class="border-b border-gray-100 px-1 py-2 last:border-0 dark:border-dark-700">
            {{ user.email }} <span class="text-gray-400">#{{ user.id }}</span>
          </p>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="loading.save" @click="closePreview">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loading.save" data-test="apply-save" @click="saveChanges">
          {{ loading.save ? t('common.saving') : t('admin.oauthAccess.preview.confirm') }}
        </button>
      </template>
    </BaseDialog>
    <TotpStepUpDialog :controller="stepUp" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores/app'
import { isStepUpBlocked, isStepUpCancelled, stepUpBlockReason, useStepUp } from '@/composables/useStepUp'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'
import openAIOAuthAccessAPI from './api'
import type {
  OAuthAccessAccount,
  OAuthAccessGroup,
  OAuthAccessMode,
  OAuthAccessPreview,
  OAuthAccessUser,
  OAuthAccessUserPage,
} from './api'
import {
  buildOAuthAccessChanges,
  cloneOAuthAccessAccounts,
  hasOAuthAccessGrant,
  setOAuthAccessGrant,
  setOAuthAccessMode,
} from './viewModel'

const { t } = useI18n()
const appStore = useAppStore()
const stepUp = useStepUp()
const serverAccounts = ref<OAuthAccessAccount[]>([])
const draftAccounts = ref<OAuthAccessAccount[]>([])
const groups = ref<OAuthAccessGroup[]>([])
const users = reactive<OAuthAccessUserPage>({ items: [], total: 0, page: 1, limit: 50, pages: 1 })
const filters = reactive({ search: '', status: '', access: 'all' })
const selectedUserIDs = ref(new Set<number>())
const selectedAccountIDs = ref(new Set<number>())
const preview = ref<OAuthAccessPreview | null>(null)
const previewChanges = ref<ReturnType<typeof buildOAuthAccessChanges>>([])
const showPreview = ref(false)
const loadError = ref('')
const userError = ref('')
const loading = reactive({ accounts: false, users: false, preview: false, save: false })

const changes = computed(() => buildOAuthAccessChanges(serverAccounts.value, draftAccounts.value))
const dirty = computed(() => changes.value.length > 0)
const draftLimitError = computed(() => {
  if (changes.value.length > 25) return t('admin.oauthAccess.errors.tooManyChangedAccounts')
  if (draftAccounts.value.some((account) => account.mode === 'restricted' && account.granted_user_ids.length > 1000)) {
    return t('admin.oauthAccess.errors.tooManyGrantedUsers')
  }
  return ''
})
const restrictedCount = computed(() => draftAccounts.value.filter((account) => account.mode === 'restricted').length)
const defaultCount = computed(() => draftAccounts.value.filter((account) => account.mode === 'restricted' && account.default_for_new_users).length)
const allPageUsersSelected = computed(() => users.items.length > 0 && users.items.every((user) => selectedUserIDs.value.has(user.id)))
const somePageUsersSelected = computed(() => !allPageUsersSelected.value && users.items.some((user) => selectedUserIDs.value.has(user.id)))
const canApplyBatch = computed(() => selectedUserIDs.value.size > 0 && selectedAccountIDs.value.size > 0)
const groupByID = computed(() => new Map(groups.value.map((group) => [group.id, group.name])))
let usersRequestSequence = 0

function errorMessage(error: unknown, fallbackKey: string): string {
  const code = extractApiErrorCode(error)
  if (code) {
    const key = `admin.oauthAccess.errors.${code}`
    const translated = t(key)
    if (translated !== key) return translated
  }
  return extractApiErrorMessage(error, t(fallbackKey))
}

async function loadAll(): Promise<boolean> {
  loading.accounts = true
  loadError.value = ''
  try {
    const [accounts, groupItems] = await Promise.all([
      openAIOAuthAccessAPI.listAccounts(),
      openAIOAuthAccessAPI.listGroups(),
    ])
    serverAccounts.value = cloneOAuthAccessAccounts(accounts)
    draftAccounts.value = cloneOAuthAccessAccounts(accounts)
    groups.value = groupItems
    selectedAccountIDs.value = new Set()
    selectedUserIDs.value = new Set()
    await loadUsers()
    return true
  } catch (error) {
    loadError.value = errorMessage(error, 'admin.oauthAccess.errors.load')
    if (draftAccounts.value.length > 0) appStore.showError(loadError.value)
    return false
  } finally {
    loading.accounts = false
  }
}

async function loadUsers() {
  const requestSequence = ++usersRequestSequence
  loading.users = true
  userError.value = ''
  try {
    const result = await openAIOAuthAccessAPI.listUsers({
      search: filters.search.trim(),
      status: filters.status,
      access: filters.access,
      page: users.page,
      limit: users.limit,
    })
    if (requestSequence === usersRequestSequence) Object.assign(users, result)
  } catch (error) {
    if (requestSequence === usersRequestSequence) userError.value = errorMessage(error, 'admin.oauthAccess.errors.users')
  } finally {
    if (requestSequence === usersRequestSequence) loading.users = false
  }
}

function applySearch() {
  users.page = 1
  selectedUserIDs.value = new Set()
  void loadUsers()
}

function applyFilters() {
  users.page = 1
  selectedUserIDs.value = new Set()
  void loadUsers()
}

function changePage(page: number) {
  users.page = page
  void loadUsers()
}

function changePageSize(limit: number) {
  users.limit = limit
  users.page = 1
  void loadUsers()
}

function toggleUserSelection(userID: number) {
  const next = new Set(selectedUserIDs.value)
  if (next.has(userID)) next.delete(userID)
  else next.add(userID)
  selectedUserIDs.value = next
}

function togglePageUsers() {
  const next = new Set(selectedUserIDs.value)
  const select = !allPageUsersSelected.value
  for (const user of users.items) {
    if (select) next.add(user.id)
    else next.delete(user.id)
  }
  selectedUserIDs.value = next
}

function toggleAccountSelection(accountID: number) {
  const account = draftAccounts.value.find((item) => item.id === accountID)
  if (!account || account.mode !== 'restricted') return
  const next = new Set(selectedAccountIDs.value)
  if (next.has(accountID)) next.delete(accountID)
  else next.add(accountID)
  selectedAccountIDs.value = next
}

function changeMode(account: OAuthAccessAccount, mode: OAuthAccessMode) {
  setOAuthAccessMode(account, mode)
  if (mode === 'public') {
    const next = new Set(selectedAccountIDs.value)
    next.delete(account.id)
    selectedAccountIDs.value = next
  }
}

function toggleGrant(account: OAuthAccessAccount, userID: number) {
  setOAuthAccessGrant(account, userID, !hasOAuthAccessGrant(account, userID))
}

function hasGrant(account: OAuthAccessAccount, userID: number): boolean {
  return hasOAuthAccessGrant(account, userID)
}

function applyBatch(granted: boolean) {
  for (const account of draftAccounts.value) {
    if (!selectedAccountIDs.value.has(account.id) || account.mode !== 'restricted') continue
    for (const userID of selectedUserIDs.value) setOAuthAccessGrant(account, userID, granted)
  }
}

function resetDraft() {
  draftAccounts.value = cloneOAuthAccessAccounts(serverAccounts.value)
  selectedAccountIDs.value = new Set()
  preview.value = null
  previewChanges.value = []
}

async function openPreview() {
  if (!dirty.value || draftLimitError.value) return
  loading.preview = true
  try {
    previewChanges.value = changes.value.map((change) => ({ ...change, granted_user_ids: [...change.granted_user_ids] }))
    preview.value = await openAIOAuthAccessAPI.preview(previewChanges.value)
    showPreview.value = true
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.oauthAccess.errors.preview'))
  } finally {
    loading.preview = false
  }
}

function closePreview() {
  if (loading.save) return
  showPreview.value = false
  preview.value = null
  previewChanges.value = []
}

async function saveChanges() {
  if (previewChanges.value.length === 0) return
  loading.save = true
  try {
    const payload = previewChanges.value.map((change) => ({ ...change, granted_user_ids: [...change.granted_user_ids] }))
    const result = await stepUp.run(() => openAIOAuthAccessAPI.apply(payload))
    showPreview.value = false
    preview.value = null
    previewChanges.value = []
    if (result.accounts.length > 0) {
      serverAccounts.value = cloneOAuthAccessAccounts(result.accounts)
      draftAccounts.value = cloneOAuthAccessAccounts(result.accounts)
      selectedAccountIDs.value = new Set()
      selectedUserIDs.value = new Set()
      await loadUsers()
    } else {
      applyCommittedPoliciesLocally(payload)
      if (!await loadAll()) appStore.showWarning(t('admin.oauthAccess.messages.savedReloadFailed'))
    }
    appStore.showSuccess(t('admin.oauthAccess.messages.saved'))
  } catch (error) {
    if (isStepUpCancelled(error)) return
    if (isStepUpBlocked(error)) {
      appStore.showError(stepUpBlockReason(error) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN'
        ? t('stepUp.adminApiKeyForbidden')
        : t('stepUp.notEnabled'))
      return
    }
    const code = extractApiErrorCode(error)
    appStore.showError(code === 'OPENAI_OAUTH_ACCESS_REVISION_CONFLICT'
      ? t('admin.oauthAccess.messages.conflict')
      : errorMessage(error, 'admin.oauthAccess.errors.save'))
    showPreview.value = false
    preview.value = null
    previewChanges.value = []
  } finally {
    loading.save = false
  }
}

function groupName(groupID: number): string {
  return groupByID.value.get(groupID) || `#${groupID}`
}

function applyCommittedPoliciesLocally(payload: ReturnType<typeof buildOAuthAccessChanges>) {
  const byID = new Map(payload.map((change) => [change.account_id, change]))
  for (const collection of [serverAccounts.value, draftAccounts.value]) {
    for (const account of collection) {
      const change = byID.get(account.id)
      if (!change) continue
      account.mode = change.mode
      account.default_for_new_users = change.default_for_new_users
      account.granted_user_ids = [...change.granted_user_ids]
      account.revision = change.expected_revision + 1
    }
  }
  selectedAccountIDs.value = new Set()
  selectedUserIDs.value = new Set()
}

function statusLabel(status: string): string {
  if (status === 'active') return t('admin.oauthAccess.filters.active')
  if (status === 'disabled') return t('admin.oauthAccess.filters.disabled')
  return status
}

function modeLabel(mode: OAuthAccessMode): string {
  return t(`admin.oauthAccess.account.${mode}`)
}

function accountTypeLabel(type: string): string {
  return type === 'apikey'
    ? t('admin.oauthAccess.account.apiKey')
    : t('admin.oauthAccess.account.oauth')
}

function defaultLabel(enabled: boolean): string {
  return t(enabled ? 'admin.oauthAccess.preview.enabled' : 'admin.oauthAccess.preview.disabled')
}

function groupLabels(groupIDs: number[]): string {
  if (groupIDs.length === 0) return t('admin.oauthAccess.empty.groups')
  return groupIDs.map(groupName).join(', ')
}

function effectiveAccountNames(user: OAuthAccessUser): string[] {
  const keyGroups = new Set(user.api_key_group_ids)
  return draftAccounts.value
    .filter((account) => account.group_ids.some((groupID) => keyGroups.has(groupID)))
    .filter((account) => account.mode === 'public' || hasOAuthAccessGrant(account, user.id))
    .map((account) => account.name)
}

onMounted(() => {
  window.addEventListener('beforeunload', confirmBrowserLeave)
  void loadAll()
})

onUnmounted(() => window.removeEventListener('beforeunload', confirmBrowserLeave))

onBeforeRouteLeave(() => !dirty.value || window.confirm(t('admin.oauthAccess.saveBar.discardConfirm')))

function confirmBrowserLeave(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}
</script>
