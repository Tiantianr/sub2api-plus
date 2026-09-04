<template>
  <div class="space-y-4">
    <!-- Quick Amount Buttons -->
    <div>
      <div class="mb-3 flex items-center gap-2">
        <label class="block text-sm font-bold text-slate-800 dark:text-white">
          {{ t('payment.quickAmounts') }}
        </label>
        <span v-if="rateHint" class="rounded bg-slate-100 px-2 py-0.5 text-xs font-normal text-slate-500 dark:bg-dark-700 dark:text-slate-400">
          {{ rateHint }}
        </span>
      </div>
      <div class="grid grid-cols-3 gap-2 sm:gap-3">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          :class="[
            'rounded-xl border-2 py-2.5 text-center text-base font-medium transition-all duration-200',
            modelValue === amt && !isCustomActive
              ? 'border-blue-600 bg-blue-50 text-blue-700 shadow-sm dark:border-blue-500 dark:bg-blue-950/40 dark:text-blue-300'
              : 'border-slate-100 bg-white text-slate-700 hover:border-blue-200 hover:bg-blue-50/50 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(amt)"
        >
          {{ currencySymbol }} {{ amt }}
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
    <div>
      <label class="mb-3 block text-sm font-bold text-slate-800 dark:text-white">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 font-medium text-slate-400 dark:text-dark-500">
          {{ currencySymbol }}
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          :class="[
            'w-full rounded-xl border-2 py-2.5 pl-8 pr-4 font-medium text-slate-800 outline-none transition-all duration-200 dark:bg-dark-800 dark:text-white',
            isCustomActive
              ? 'border-blue-600 bg-white shadow-sm focus:ring-4 focus:ring-blue-600/10 dark:border-blue-500'
              : 'border-slate-200 bg-slate-50 focus:border-blue-400 focus:bg-white dark:border-dark-600 dark:bg-dark-800'
          ]"
          @input="handleInput"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  currencySymbol?: string
  rateHint?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500],
  min: 0,
  max: 0,
  currencySymbol: '¥',
  rateHint: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

const isCustomActive = computed(() => {
  if (!customText.value) return false
  const num = parseFloat(customText.value)
  return !isNaN(num) && !props.amounts.includes(num)
})

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = ''
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  if (!AMOUNT_PATTERN.test(val)) return
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && !props.amounts.includes(v) && String(v) !== customText.value) {
    customText.value = String(v)
  } else if (v !== null && props.amounts.includes(v) && customText.value !== '' && Number(customText.value) !== v) {
    customText.value = ''
  }
}, { immediate: true })
</script>
