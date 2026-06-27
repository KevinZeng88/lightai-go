<template>
  <el-collapse
    :model-value="activeNames"
    data-testid="config-edit-section"
    :data-section-key="section.key"
    @update:model-value="activeNames = normalizeActive($event)"
  >
    <el-collapse-item :name="section.key">
      <template #title>
        <span>{{ sectionI18nLabel }}</span>
        <el-tag v-if="section.key === 'advanced_raw'" size="small" effect="plain" class="section-tag">{{ $t('configEdit.sections.advancedRaw') }}</el-tag>
      </template>
      <p v-if="section.description" class="section-description">{{ section.description }}</p>
      <div class="section-fields">
        <ConfigField
          v-for="field in sortedFields(section)"
          :key="field.key"
          :field="field"
          :readonly="readonly"
          @change="$emit('change')"
        />
      </div>
    </el-collapse-item>
  </el-collapse>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfigField from './ConfigField.vue'
import { sortedFields, type ConfigEditSection } from '@/utils/configEditView'

const { t } = useI18n()

// Map section.key to i18n key. Falls back to section.label (from backend) if no mapping exists.
const SECTION_I18N_MAP: Record<string, string> = {
  basic: 'configEdit.sections.basic',
  model_serving: 'configEdit.sections.modelServing',
  advanced_parameters: 'configEdit.sections.advancedParameters',
  expert_parameters: 'configEdit.sections.expertParameters',
  backend_runtime: 'configEdit.sections.backendRuntime',
  container_resources: 'configEdit.sections.containerResources',
  devices_mounts: 'configEdit.sections.devicesMounts',
  environment: 'configEdit.sections.environment',
  service: 'configEdit.sections.service',
  health_check: 'configEdit.sections.healthCheck',
  advanced_raw: 'configEdit.sections.advancedRaw',
}

const props = defineProps<{
  section: ConfigEditSection
  readonly?: boolean
}>()

defineEmits<{ change: [] }>()

const sectionI18nLabel = computed(() => {
  const i18nKey = SECTION_I18N_MAP[props.section.key]
  if (i18nKey) {
    const translated = t(i18nKey)
    // t() returns the key itself if no translation found
    if (translated !== i18nKey) return translated
  }
  // Fallback to backend-provided label (may be English)
  return props.section.label
})

const activeNames = ref<string[]>(props.section.collapsed ? [] : [props.section.key])

watch(() => props.section.key, () => {
  activeNames.value = props.section.collapsed ? [] : [props.section.key]
})

function normalizeActive(value: string | string[]) {
  return Array.isArray(value) ? value : [value]
}
</script>

<style scoped>
.section-tag { margin-left: 8px; }
.section-description { margin: 0 0 10px; color: var(--el-text-color-secondary); font-size: 12px; }
.section-fields { display: grid; grid-template-columns: 1fr; gap: 10px; }
</style>
