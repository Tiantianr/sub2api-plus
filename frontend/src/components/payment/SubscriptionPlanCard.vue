<template>
  <div
    :class="[
      'group relative flex flex-col rounded-2xl bg-white p-5 transition-all duration-300 cursor-pointer dark:bg-dark-800 border-2',
      selected
        ? 'border-blue-600 shadow-xl shadow-blue-900/5'
        : 'border-slate-100 hover:border-blue-200 dark:border-dark-700 dark:hover:border-dark-500 shadow-sm hover:shadow-md'
    ]"
    @click="emit('select', plan)"
  >
    <!-- Header: Title & Platform / Duration Badges -->
    <div class="min-w-0">
      <h3
        :title="plan.name"
        class="h-12 text-base font-bold leading-6 text-slate-900 dark:text-white line-clamp-2 min-w-0 break-words [overflow-wrap:anywhere]"
      >
        {{ plan.name }}
      </h3>
      <div class="mt-1.5 flex items-center gap-2">
        <span :class="['shrink-0 rounded px-2 py-0.5 text-xs font-medium', badgeLightClass]">
          {{ pLabel }}
        </span>
        <span class="text-xs text-slate-500 dark:text-dark-400">/ {{ validitySuffix }}</span>
      </div>
    </div>

    <!-- Price Area -->
    <div class="mt-3">
      <div class="flex items-baseline gap-1">
        <span class="text-xl font-semibold text-slate-700 dark:text-slate-300">{{ planCurrencySymbol }}</span>
        <span :class="['text-3xl font-black tracking-tight text-slate-900 dark:text-white', textClass]">{{ plan.price }}</span>
        <span v-if="plan.currency" class="ml-1 text-sm font-medium text-slate-500 dark:text-dark-400">{{ plan.currency }}</span>
      </div>
      <div v-if="plan.original_price" class="mt-0.5 flex items-center gap-2 text-xs">
        <span class="text-slate-400 line-through dark:text-dark-500">
          {{ planCurrencySymbol }}{{ plan.original_price }}<template v-if="plan.currency"> {{ plan.currency }}</template>
        </span>
        <span class="rounded bg-red-50 px-1.5 py-0.5 text-xs font-bold text-red-600 dark:bg-red-950/40 dark:text-red-400">
          {{ discountText }}
        </span>
      </div>
    </div>

    <!-- Description -->
    <p
      v-if="plan.description"
      class="mt-3 h-9 text-xs leading-relaxed text-slate-500 dark:text-dark-400 line-clamp-2"
    >
      {{ plan.description }}
    </p>
    <div v-else class="mt-3 h-9"></div>

    <!-- Stats Metrics Box -->
    <div class="mt-4 flex justify-between rounded-xl bg-slate-50 p-3 text-sm dark:bg-dark-700/50">
      <div class="flex-1 text-center">
        <div class="mb-1 text-xs text-slate-500 dark:text-dark-400">{{ t('payment.planCard.rate') }}</div>
        <div class="font-bold text-slate-800 dark:text-white">{{ rateDisplay }}</div>
      </div>
      <div class="mx-2 w-px bg-slate-200 dark:bg-dark-600"></div>
      <div class="flex-1 text-center">
        <div class="mb-1 text-xs text-slate-500 dark:text-dark-400">{{ limitLabel }}</div>
        <div class="font-bold text-slate-800 dark:text-white">{{ limitValue }}</div>
      </div>
      <template v-if="monthlyLimitValue">
        <div class="mx-2 w-px bg-slate-200 dark:bg-dark-600"></div>
        <div class="flex-1 text-center">
          <div class="mb-1 text-xs text-slate-500 dark:text-dark-400">{{ t('payment.planCard.monthlyLimit') }}</div>
          <div class="font-bold text-slate-800 dark:text-white">{{ monthlyLimitValue }}</div>
        </div>
      </template>
      <template v-else-if="hasPeakRate">
        <div class="mx-2 w-px bg-slate-200 dark:bg-dark-600"></div>
        <div class="flex-1 text-center">
          <div class="mb-1 text-xs text-slate-500 dark:text-dark-400">{{ t('payment.planCard.peakRate') }}</div>
          <div class="text-xs font-semibold text-amber-700 dark:text-amber-300">{{ peakRateDisplay }}</div>
        </div>
      </template>
    </div>

    <!-- Antigravity Model Scopes (if applicable) -->
    <div v-if="modelScopeLabels.length > 0" class="mt-2.5 flex flex-wrap items-center justify-center gap-1">
      <span
        v-for="scope in modelScopeLabels"
        :key="scope"
        class="rounded bg-slate-200/80 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 dark:bg-dark-600 dark:text-gray-300"
      >
        {{ scope }}
      </span>
    </div>

    <!-- Features list -->
    <ul v-if="plan.features && plan.features.length > 0" class="mt-4 flex-1 space-y-2">
      <li v-for="feature in plan.features" :key="feature" class="flex items-start gap-2 text-xs text-slate-600 dark:text-gray-300">
        <div class="mt-0.5 shrink-0 rounded-full bg-blue-100 p-0.5 dark:bg-blue-950/80">
          <svg class="h-3 w-3 stroke-[3] text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        </div>
        <span class="leading-tight">{{ feature }}</span>
      </li>
    </ul>
    <div v-else class="flex-1"></div>

    <!-- Action Button -->
    <button
      type="button"
      :class="[
        'mt-5 w-full rounded-xl py-2.5 text-sm font-bold transition-all duration-200 active:scale-[0.98]',
        selected
          ? 'bg-blue-600 text-white shadow-md shadow-blue-600/20 hover:bg-blue-700'
          : 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600'
      ]"
      @click.stop="emit('select', plan)"
    >
      {{ isRenewal ? t('payment.renewNow') : t('payment.subscribeNow') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import { useAppStore } from '@/stores/app'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { planValiditySuffix } from './validity'
import { currencySymbol } from '@/components/payment/currency'
import {
  platformBadgeLightClass,
  platformTextClass,
  platformLabel,
} from '@/utils/platformColors'

const props = withDefaults(defineProps<{
  plan: SubscriptionPlan
  activeSubscriptions?: UserSubscription[]
  selected?: boolean
}>(), {
  selected: false,
})

const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()

const platform = computed(() => props.plan.group_platform || '')
const isRenewal = computed(() =>
  props.activeSubscriptions?.some(s => s.group_id === props.plan.group_id && s.status === 'active') ?? false
)

const badgeLightClass = computed(() => platformBadgeLightClass(platform.value))
const textClass = computed(() => platformTextClass(platform.value))
const pLabel = computed(() => platformLabel(platform.value))

const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= 0) return ''
  const pct = Math.round((1 - props.plan.price / props.plan.original_price) * 100)
  return pct > 0 ? `-${pct}%` : ''
})

const rateDisplay = computed(() => {
  const rate = props.plan.rate_multiplier ?? 1
  return `x${Number(rate.toPrecision(10))}`
})

const appStore = useAppStore()
const planCurrencySymbol = computed(() => currencySymbol(props.plan.currency || 'USD'))

const hasPeakRate = computed(() => groupHasPeakRate(props.plan))

const peakRateDisplay = computed(() => {
  return formatPeakRateWindow(props.plan, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
})

const limitLabel = computed(() => {
  if (props.plan.weekly_limit_usd != null) return t('payment.planCard.weeklyLimit')
  if (props.plan.daily_limit_usd != null) return t('payment.planCard.dailyLimit')
  if (props.plan.five_hour_limit_usd != null) return t('payment.planCard.fiveHourLimit')
  return t('payment.planCard.quota')
})

const limitValue = computed(() => {
  if (props.plan.weekly_limit_usd != null) return `$${props.plan.weekly_limit_usd}`
  if (props.plan.daily_limit_usd != null) return `$${props.plan.daily_limit_usd}`
  if (props.plan.five_hour_limit_usd != null) return `$${props.plan.five_hour_limit_usd}`
  return t('payment.planCard.unlimited')
})

const monthlyLimitValue = computed(() => {
  if (props.plan.monthly_limit_usd != null) return `$${props.plan.monthly_limit_usd}`
  return null
})

const MODEL_SCOPE_LABELS: Record<string, string> = {
  claude: 'Claude',
  gemini_text: 'Gemini',
  gemini_image: 'Imagen',
}

const modelScopeLabels = computed(() => {
  if (platform.value !== 'antigravity') return []
  const scopes = props.plan.supported_model_scopes
  if (!scopes || scopes.length === 0) return []
  return scopes.map(s => MODEL_SCOPE_LABELS[s] || s)
})

const validitySuffix = computed(() => planValiditySuffix(props.plan, t))
</script>
