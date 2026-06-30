<template>
  <div
    class="wizard-action-bar"
    :class="[`layout-${layout}`, { 'is-sticky': layout !== 'inline' }]"
    data-testid="wizard-action-bar"
  >
    <div class="action-left">
      <el-button v-if="showCancel" @click="$emit('cancel')">{{ cancelLabel || $t('common.cancel') }}</el-button>
      <span v-if="nextDisabledReason && !canNext" class="disabled-reason" data-testid="wizard-disabled-reason">
        {{ nextDisabledReason }}
      </span>
    </div>
    <div class="action-right">
      <el-button v-if="showPrev && activeStep > 0 && canPrev" @click="$emit('prev')">
        {{ prevLabel || $t('common.prev') }}
      </el-button>
      <el-button
        v-for="action in secondaryActions || []"
        :key="action.key"
        :type="action.type || 'default'"
        :loading="!!action.loading"
        :disabled="!!action.disabled || !!action.loading"
        @click="$emit('secondary', action.key)"
      >
        {{ action.label }}
      </el-button>
      <el-tooltip :disabled="canNext || !nextDisabledReason" :content="nextDisabledReason" placement="top">
        <span>
          <el-button
            v-if="primaryLabel"
            type="primary"
            :loading="primaryLoading"
            :disabled="!canNext || primaryLoading"
            @click="$emit('primary')"
          >
            {{ primaryLabel }}
          </el-button>
        </span>
      </el-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
type SecondaryAction = {
  key: string
  label: string
  type?: 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'
  loading?: boolean
  disabled?: boolean
}

withDefaults(defineProps<{
  activeStep: number
  totalSteps: number
  canPrev?: boolean
  canNext?: boolean
  primaryLabel?: string
  primaryLoading?: boolean
  nextDisabledReason?: string
  secondaryActions?: SecondaryAction[]
  layout?: 'sticky-top' | 'sticky-bottom' | 'inline'
  showCancel?: boolean
  showPrev?: boolean
  cancelLabel?: string
  prevLabel?: string
}>(), {
  canPrev: true,
  canNext: true,
  primaryLabel: '',
  primaryLoading: false,
  nextDisabledReason: '',
  secondaryActions: () => [],
  layout: 'sticky-top',
  showCancel: true,
  showPrev: true,
  cancelLabel: '',
  prevLabel: '',
})

defineEmits<{
  cancel: []
  prev: []
  primary: []
  secondary: [key: string]
}>()
</script>

<style scoped>
.wizard-action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 10px 0;
  background: var(--el-bg-color);
  z-index: 5;
}

.wizard-action-bar.is-sticky {
  position: sticky;
}

.wizard-action-bar.layout-sticky-top {
  top: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.wizard-action-bar.layout-sticky-bottom {
  bottom: 0;
  border-top: 1px solid var(--el-border-color-lighter);
}

.action-left,
.action-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.action-right {
  margin-left: auto;
  justify-content: flex-end;
}

.disabled-reason {
  color: var(--el-color-warning);
  font-size: 12px;
}

@media (max-width: 640px) {
  .wizard-action-bar {
    align-items: stretch;
  }

  .action-left,
  .action-right {
    width: 100%;
  }

  .action-right {
    margin-left: 0;
  }
}
</style>
