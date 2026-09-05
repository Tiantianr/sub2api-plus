<template>
  <div class="flex items-center gap-1.5">
    <button
      type="button"
      @click="emit('update:modelValue', '')"
      class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
      :class="!modelValue?.trim()
        ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
        : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'"
    >
      {{ defaultLabel || t('admin.settings.gatewayForwarding.channelDefault') }}
    </button>
    <button
      type="button"
      @click="emit('update:modelValue', DEFAULT_PI_USER_AGENT)"
      class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
      :class="isPiUserAgent(modelValue)
        ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
        : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'"
    >
      {{ piLabel || t('admin.settings.gatewayForwarding.channelPiAgent') }}
    </button>
    <button
      type="button"
      @click="emit('update:modelValue', DEFAULT_CODEX_TUI_USER_AGENT)"
      class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
      :class="isCodexTuiUserAgent(modelValue)
        ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
        : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'"
    >
      {{ codexTuiLabel || t('admin.settings.gatewayForwarding.channelCodexTui') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  DEFAULT_PI_USER_AGENT,
  DEFAULT_CODEX_TUI_USER_AGENT,
  isPiUserAgent,
  isCodexTuiUserAgent,
} from '@/constants/openaiIdentity'

defineProps<{
  modelValue?: string | null
  defaultLabel?: string
  piLabel?: string
  codexTuiLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const { t } = useI18n()
</script>
